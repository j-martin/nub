.PHONY: build
build: clean
	GOOS=darwin gb build
	GOOS=linux gb build

.PHONY: deps
deps:
	gb vendor restore

.PHONY: clean
clean:
	rm -rf bin
