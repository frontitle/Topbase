# Deployment and backups

Use [Docker Compose or a direct server installation](getting-started.html). For production, configure PostgreSQL or MySQL as the application database. Use TLS at your reverse proxy, configure the public URL for links and OAuth callbacks, and keep the master key outside the image.

## Back up the Compose installation

```bash
docker compose -f docker-compose.release.yml exec -T appdb \
  pg_dump -U topbase -d topbase -Fc > topbase-backup.dump
```

Keep the dump, persistent application files and master key together in protected backup storage. Check readiness at the published server port after restarting.
