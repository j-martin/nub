PLATFORM	= $(shell uname | tr 'A-Z' 'a-z')
ARCH		= amd64

.PHONY: build
build: clean test
	GOOS=darwin GOARCH=$(ARCH) gb build
	GOOS=linux GOARCH=$(ARCH) gb build

.PHONY: deps
deps:
	gb vendor restore

.PHONY: test
test:
	gb test

.PHONY: clean
clean:
	rm -rf bin

.PHONY: publish
publish: build
	$(eval version := $(shell bin/bub-$(PLATFORM)-$(ARCH) --version | tr ' ' '/'))
	find bin -type f -exec gzip {} \;
	aws s3 --recursive cp bin/ "s3://s3bucket/$(version)/"
	git tag $(version)
	git push --tags
