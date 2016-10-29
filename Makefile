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

.PHONY: release
release: build
	$(eval version := $(shell bin/bub-$(PLATFORM)-$(ARCH) --version | tr ' ' '-'))
	find bin -type f -exec gzip {} \;
	find bin -name *gz \
		| sed -e "p;s#bin/bub#s3://s3bucket/contrib/$(version)#" \
		| xargs -n2 aws s3 cp
	git tag $(version)
	git push --tags
