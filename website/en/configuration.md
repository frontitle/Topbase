# Configuration

| Variable | Purpose |
| --- | --- |
| TOPBASE_PORT | Direct server port, default 80 |
| TOPBASE_ADDR | Listen address including port, overrides TOPBASE_PORT |
| TOPBASE_HTTP_PORT | Compose host port, default 8101 |
| TOPBASE_DATA_DIR | Persistent application files, default ./data |
| TOPBASE_MASTER_KEY | Stable 32-byte key encoded as base64 or hex |
| TOPBASE_APP_DB_ENGINE | Application database engine |
| TOPBASE_APP_DB_PASSWORD | Application database password |
| TOPBASE_SECURE_COOKIES | Set to true for production HTTPS |

See the [configuration template](https://github.com/frontitle/Topbase/blob/main/.env.example) for additional settings.
