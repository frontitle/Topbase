# Getting started

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

Install the Go version specified in [go.mod](https://github.com/frontitle/Topbase/blob/main/go.mod), then run:

```bash
git clone https://github.com/frontitle/Topbase.git
cd Topbase
git checkout v0.2.5
go build -o topbase ./cmd/topbase
TOPBASE_PORT=8080 TOPBASE_DATA_DIR=./data ./topbase
```

Open `http://YOUR_SERVER:8080`. This command selects port 8080 explicitly. The binary defaults to port 80. Keep the process running and preserve the data directory.
