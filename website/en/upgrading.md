# Upgrade and restore

Back up the application database and files. Preserve the current master key. Read the target release notes, fetch tags and check out the target version. Update TOPBASE_VERSION in your existing environment file, pull the image and restart the Compose service.

Check /api/ready and /api/version, then open an existing dashboard. To roll back a schema change, stop the application, restore the matching database backup and start its original version.
