# Install and run Topbase

Choose Docker Compose to run the published image with PostgreSQL, or build the Go server to run it directly. On first launch, open Topbase in a browser and create the administrator account. You can then [build your first dashboard](getting-started.html) using six sample orders.

## Install with Docker Compose

1. Install Git, OpenSSL and Docker with the Compose plugin on the server. Clone the release and create its local settings file.

   ```bash
   git clone --branch v0.2.6 https://github.com/frontitle/Topbase.git
   cd Topbase
   cp -n .env.example .env
   chmod 600 .env
   openssl rand -base64 32
   ```

2. Open `.env` in a text editor. Set `TOPBASE_APP_DB_PASSWORD` to a strong password for the bundled PostgreSQL database. Paste the entire OpenSSL output into `TOPBASE_MASTER_KEY`. This key encrypts saved connections. Keep a separate protected copy for recovery. Save the file.
3. Start the services and wait for the readiness check to succeed.

   ```bash
   docker compose -f docker-compose.release.yml up -d
   curl --fail http://127.0.0.1:8101/api/ready
   ```

4. Open `http://YOUR_SERVER:8101`. The default mapping is **8101 on the server → 8080 inside the container**. To use server port 9000, set `TOPBASE_HTTP_PORT=9000` in `.env`, run the same Compose command again and use port 9000 in the browser and checks below.

The release image supports Linux amd64 and arm64. PostgreSQL application data and the Topbase data directory use named volumes. Keep these volumes when replacing a container. For a public HTTPS address, configure the reverse proxy, set `TOPBASE_SECURE_COOKIES=true` in `.env`, apply the Compose configuration again and set the [workspace public address](workspace.html).

## Run directly on a server

Install Git and the Go version listed in [go.mod](https://github.com/frontitle/Topbase/blob/v0.2.6/go.mod). Run the following commands in a terminal.

```bash
git clone --branch v0.2.6 https://github.com/frontitle/Topbase.git
cd Topbase
go build -o topbase ./cmd/topbase
TOPBASE_PORT=8080 TOPBASE_DATA_DIR=./data ./topbase
```

Open `http://YOUR_SERVER:8080`. Keep the process running. Without an external application database this setup stores accounts, analyses and dashboards in SQLite in `./data`. Preserve that directory across restarts. A managed server can use the [systemd service template](https://github.com/frontitle/Topbase/blob/v0.2.6/deploy/systemd/topbase.service).

## Check settings and query limits

| Setting | Where it applies |
| --- | --- |
| `TOPBASE_PORT`, `TOPBASE_ADDR` | Direct server listener. Its default port is 80. An explicit address and port such as `127.0.0.1:8080` in `TOPBASE_ADDR` takes precedence. |
| `TOPBASE_HTTP_PORT` | Server port published by Compose. The application inside the container listens on 8080. |
| `TOPBASE_APP_DB_ENGINE` | Application store for accounts and settings. The release Compose file supplies PostgreSQL. The basic development Compose file uses SQLite. |
| `TOPBASE_MAX_CONCURRENT_QUERIES` | A node runs up to eight interactive queries at once by default. |
| `TOPBASE_QUERY_RESULT_MAX_BYTES`, `TOPBASE_QUERY_CELL_MAX_BYTES` | Defaults are 16 MiB per result and 1 MiB per cell. Narrow the date range or aggregate rows when a result reaches a limit. |

For more settings, see the [environment template](https://github.com/frontitle/Topbase/blob/v0.2.6/.env.example). `/api/ready` checks application-database readiness. `/api/version` reports the running version.

## Back up a Docker installation

From the Compose project directory, export PostgreSQL to a new private directory under your home directory. The command uses the database user and name configured inside the container.

```bash
backup_dir=$(mktemp -d "$HOME/topbase-backup.XXXXXXXX") &&
docker compose -f docker-compose.release.yml exec -T appdb \
  sh -c 'exec pg_dump -Fc -U "$POSTGRES_USER" -d "$POSTGRES_DB"' \
  > "$backup_dir/appdb.dump"
```

After the command succeeds, keep this dump in protected backup storage. Also preserve the Topbase data volume and a separate protected copy of the master key. Back up connected business databases with their own backup tools.

## Change the Docker release

Back up the installation first. In the project directory, fetch the release tag you want and check it out. For example, for version 0.2.6:

```bash
git fetch --tags
git checkout v0.2.6
```

Edit the existing `.env` and set `TOPBASE_VERSION=0.2.6`. Keep its password and master key. Verify the resolved image:

```bash
docker compose -f docker-compose.release.yml config --images
```

For this example, look for `ghcr.io/frontitle/topbase:0.2.6`. If another tag appears, also check whether the shell has an exported `TOPBASE_VERSION` overriding `.env`. Once the image matches:

```bash
docker compose -f docker-compose.release.yml pull
docker compose -f docker-compose.release.yml up -d
curl --fail http://127.0.0.1:8101/api/ready
curl --fail http://127.0.0.1:8101/api/version
```

Check the returned `version` and open a saved analysis or dashboard.
