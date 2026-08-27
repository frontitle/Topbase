GO ?= go
WEB_DIR := internal/platform/httpapi/web
JS_FILES := $(shell find $(WEB_DIR) -type f -name '*.js' ! -name 'echarts.min.js' | sort)
JS_TESTS := $(shell find tests/js -type f -name '*.test.js' 2>/dev/null | sort)

.PHONY: help fmt fmt-check vet js-check test build check run

help:
	@echo "make fmt       format Go source"
	@echo "make check     run all required quality checks"
	@echo "make run       start Topbase on TOPBASE_ADDR (default :8080)"

fmt:
	@files="$$(gofmt -l cmd internal)"; if [ -n "$$files" ]; then gofmt -w $$files; fi

fmt-check:
	@files="$$(gofmt -l cmd internal)"; if [ -n "$$files" ]; then echo "Go files need formatting:"; echo "$$files"; exit 1; fi

vet:
	$(GO) vet ./...

js-check:
	@for file in $(JS_FILES); do node --check "$$file" || exit 1; done
	@if [ -n "$(JS_TESTS)" ]; then node --test $(JS_TESTS); fi

test:
	$(GO) test ./...

build:
	$(GO) build ./cmd/topbase

check: fmt-check vet js-check test build

run:
	$(GO) run ./cmd/topbase
