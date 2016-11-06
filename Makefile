PLATFORM	= $(shell uname | tr 'A-Z' 'a-z')
ARCH		= amd64

.PHONY: build deps test clean release

build: clean test
	GOOS=darwin GOARCH=$(ARCH) gb build
	GOOS=linux GOARCH=$(ARCH) gb build

deps:
	gb vendor restore

test:
	gb test

clean:
	rm -rf bin

release: build
	$(eval version := $(shell bin/bub-$(PLATFORM)-$(ARCH) --version | tr ' ' '-'))
	find bin -type f -exec gzip --keep {} \;
	find bin -name *.gz \
		| sed -e "p;s#bin/bub#s3://s3bucket/contrib/$(version)#" \
		| xargs -n2 aws s3 cp
	git tag $(version)
	git push --tags

install: build
	rm -f /usr/local/bin/bub
	ln -s $(shell pwd)/bin/bub-$(PLATFORM)-$(ARCH) /usr/local/bin/bub
