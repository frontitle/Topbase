GO ?= go
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo 0.1.0-alpha.0-dev)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_TIME ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w -X github.com/topbase/topbase/internal/buildinfo.Version=$(VERSION) -X github.com/topbase/topbase/internal/buildinfo.Commit=$(COMMIT) -X github.com/topbase/topbase/internal/buildinfo.BuildTime=$(BUILD_TIME)
WEB_DIR := internal/platform/httpapi/web
JS_FILES := $(shell find $(WEB_DIR) -type f -name '*.js' ! -name 'echarts.min.js' | sort)
JS_TESTS := $(shell find tests/js -type f -name '*.test.js' 2>/dev/null | sort)

.PHONY: help fmt fmt-check vet js-check test build check run backup

help:
	@echo "make fmt       format Go source"
	@echo "make check     run all required quality checks"
	@echo "make build     build versioned server and backup binaries into bin/"
	@echo "make run       start Topbase on TOPBASE_ADDR (default :8080)"
	@echo "make backup    create a consistent backup in backups/"

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
	@mkdir -p bin
	$(GO) build -trimpath -ldflags "$(LDFLAGS)" -o bin/topbase ./cmd/topbase
	$(GO) build -trimpath -ldflags "$(LDFLAGS)" -o bin/topbase-backup ./cmd/topbase-backup

check: fmt-check vet js-check test build

run:
	$(GO) run ./cmd/topbase

backup: build
	TOPBASE_DATA_DIR="$${TOPBASE_DATA_DIR:-data}" ./bin/topbase-backup
