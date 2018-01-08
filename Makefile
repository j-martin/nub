PLATFORM	= $(shell uname | tr 'A-Z' 'a-z')
ARCH		= amd64
DEP		= ./.dep
DEP_VERSION	= 0.3.2
OUTPUT		= bin/bub

.PHONY: all deps test clean release fmt

all: deps test darwin linux

darwin:
	GOOS=darwin GOARCH=$(ARCH) go build -o "$(OUTPUT)-darwin-$(ARCH)"

linux:
	GOOS=linux GOARCH=$(ARCH) go build -o "$(OUTPUT)-linux-$(ARCH)"

dev:
	GOOS=darwin GOARCH=$(ARCH) go build -i -o "$(OUTPUT)-$(PLATFORM)-$(ARCH)"

$(DEP):
	curl --silent "https://s3.amazonaws.com/s3bucket/libs/golang/dep-$(PLATFORM)-amd64-$(DEP_VERSION).gz" \
		| gzip -d > "$(DEP)"
	chmod +x "$(DEP)"

deps: $(DEP)
	$(DEP) ensure --vendor-only

test:
	go test ./...

clean:
	rm -rf bin

release: all
	$(eval version := $(shell bin/bub-$(PLATFORM)-$(ARCH) --version | sed 's/ version /-/g'))
	git tag $(version)
	find bin -type f -exec gzip --keep {} \;
	find bin -type f -name *.gz \
		| sed -e "p;s#bin/bub#s3://s3bucket/contrib/$(version)#" \
		| xargs -n2 aws s3 cp
	find bin -type f -name *.gz -exec shasum -a 256 {} \;

install: deps dev
	rm -f /usr/local/bin/bub
	ln -s $(shell pwd)/bin/bub-$(PLATFORM)-$(ARCH) /usr/local/bin/bub

fmt:
	go fmt ./...
