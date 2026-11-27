# Topbase

[![CI](https://github.com/frontitle/Topbase/actions/workflows/ci.yml/badge.svg)](https://github.com/frontitle/Topbase/actions/workflows/ci.yml)
[![Version](https://img.shields.io/badge/version-0.2.5-111111.svg)](CHANGELOG.md)

English · [简体中文](README.zh-CN.md) · [Website](https://frontitle.github.io/Topbase/) · [Documentation](https://frontitle.github.io/Topbase/docs/)

Topbase is a self-hosted analytics platform. Import an Excel workbook or connect a business database, explore the data visually or in SQL, and share the results in a dashboard. Schedule a saved analysis to write its results to a warehouse table.

## What you can do

- Connect PostgreSQL, MySQL/MariaDB, ClickHouse, SQL Server, Oracle or SQLite. Test connections and configure TLS or an SSH tunnel.
- Filter, aggregate, sort and join data. Turn results into charts, metrics and tables.
- Arrange dashboards, link filters, embed a canvas and protect public links with a password or expiry date.
- Organize analyses in groups and use the CLI or MCP integration with the permissions granted to your account.

## Install

Choose one method. On first launch, open the server address in a browser and create your administrator account.

### Docker Compose

Install Docker with Compose, then run:

```bash
git clone https://github.com/frontitle/Topbase.git
cd Topbase
git checkout v0.2.5
cp .env.example .env
openssl rand -base64 32
```

In `.env`, set `TOPBASE_MASTER_KEY` to the generated key and choose a strong `TOPBASE_APP_DB_PASSWORD`. Keep the master key for future upgrades.

```bash
docker compose -f docker-compose.release.yml up -d
curl --fail http://127.0.0.1:8101/api/ready
```

Open `http://YOUR_SERVER:8101`. The default mapping is **8101 on the server → 8080 inside the container**. To use port 9000, set `TOPBASE_HTTP_PORT=9000` in `.env`, rerun the Compose command, and open `http://YOUR_SERVER:9000`. The release configuration stores application data in a PostgreSQL volume.

### Run directly on a server

Install the Go version specified in [go.mod](go.mod), then run:

```bash
git clone https://github.com/frontitle/Topbase.git
cd Topbase
git checkout v0.2.5
go build -o topbase ./cmd/topbase
TOPBASE_PORT=8080 TOPBASE_DATA_DIR=./data ./topbase
```

Open `http://YOUR_SERVER:8080`. This command selects port 8080 explicitly. The binary defaults to port 80. Keep the process running and preserve the data directory.

## Documentation and license

Read the [deployment guide](https://frontitle.github.io/Topbase/docs/deployment.html), [upgrade guide](https://frontitle.github.io/Topbase/docs/upgrading.html) and [release notes](https://github.com/frontitle/Topbase/releases/latest). Topbase is distributed under the [Apache License 2.0](LICENSE). See [NOTICE](NOTICE) for third-party notices, [CONTRIBUTING](CONTRIBUTING.md) for the development workflow and [SECURITY](SECURITY.md) to report a vulnerability.
