PLATFORM	= $(shell uname | tr 'A-Z' 'a-z')
ARCH		= $(shell arch)
DEP		= ./.dep
DEP_VERSION	= 0.3.2
OUTPUT		= bin/nub

.PHONY: all dev deps test clean release fmt

all: clean deps test darwin linux

dev:
	GOOS=darwin GOARCH=$(ARCH) go build -o "$(OUTPUT)-$(PLATFORM)-$(ARCH)"

darwin:
	GOOS=darwin GOARCH=$(ARCH) go build -o "$(OUTPUT)-darwin-$(ARCH)"

linux:
	GOOS=linux GOARCH=$(ARCH) go build -o "$(OUTPUT)-linux-amd64"

deps:
	go mod download

test:
	echo $(S3_BUCKET)
	go test ./...

clean:
	rm -rf bin

release: all
	$(eval version := $(shell bin/nub-$(PLATFORM)-$(ARCH) --version | sed 's/ version /-/g'))
	git tag $(version)
	find bin -type f -exec gzip --keep {} \;
	find bin -type f -name *.gz \
		| sed -e "p;s#bin/nub#s3://$(S3_BUCKET)/contrib/$(version)#" \
		| xargs -n2 aws s3 cp
	find bin -type f -name *.gz -exec shasum -a 256 {} \;

install: dev
	rm -f /usr/local/bin/nub
	ln -s $(shell pwd)/bin/nub-$(PLATFORM)-$(ARCH) /usr/local/bin/nub

fmt:
	go fmt ./...
