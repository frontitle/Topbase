# Contributing to Topbase

English · [简体中文](CONTRIBUTING.zh-CN.md)

## Run the project

Install the Go version specified in [go.mod](go.mod) and Node.js 22 or newer for the browser checks, tests and website build. Run:

```bash
go mod download
make check
TOPBASE_ADDR=127.0.0.1:8080 go run ./cmd/topbase
```

The development server stores its data in `./data`. Set `TOPBASE_DATA_DIR` to a temporary directory for isolated tests.

## Make a change

1. Link the issue or describe the problem, expected behavior and how reviewers can verify it.
2. Cover the relevant domain and application behavior, implement the API and interface, then update documentation and tests together.
3. Keep dependencies flowing from `platform` to `app` to `core`. The Go server serves the browser interface and HTTP API. Application services implement analysis, dashboard and warehouse workflows. Database and integration adapters connect those services to external systems. Versioned SQL migrations ship with the binary.
4. Reuse the existing navigation, forms, dialogs, popovers, grids and chart renderers. Check keyboard navigation, focus restoration and narrow screens. Keep UI text localizable.
5. Run `make check`. For website changes, build both languages using the [website instructions](website/README.md).

## Pull request checklist

- [ ] The description states the expected behavior and how to verify it.
- [ ] Loading, empty results, errors, access denial and retries work where applicable.
- [ ] Reads and writes enforce object-level access on the server. Logs and responses protect passwords, private keys, session cookies and API keys.
- [ ] Database extensions use the existing interfaces. Published migrations remain intact.
- [ ] User-facing changes update the affected English and Chinese guides and include browser checks for the workflow.

## Tests

Core tests live in `internal/core/`; use-case tests in `internal/app/`; adapter tests in `internal/adapters/`; and HTTP contract tests in `internal/platform/httpapi/`. Frontend tests live in `tests/js/`. Tests should run independently of contributors’ local databases. Integration tests that require an external service should use explicit environment variables and skip when the service is unavailable.

## What belongs in the repository

Commit the source, tests, deployment files, user documentation and synthetic fixtures needed for the open-source product. Keep private planning, competitor materials, customer records, credentials, runtime backups and temporary working files out of Git. User guides are maintained in pairs under `website/guides/{en,zh}/`.

Describe changes to stored formats or public APIs and how an existing installation handles them. Follow the [security policy](SECURITY.md) when reporting a vulnerability. Topbase uses the [Apache License 2.0](LICENSE). Verify the license for any bundled third-party component and retain its applicable copyright and license notices.
