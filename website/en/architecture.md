# Architecture

The Go server serves the browser interface and HTTP API. Application services implement analysis, dashboards and warehouse workflows. Database adapters connect these services to the application store and remote data sources. Versioned SQL migrations ship with the binary.
