# Connect a database and inspect its tables

Connect the database that holds the records you want to analyze. An administrator sets up the connection. Analysts then open it under **Source data**. You can practice with the downloadable SQLite file or use an existing database account with read access to the required tables.

## Connect the sample SQLite file

1. Download [orders.db](../assets/samples/orders.db). For a direct server installation, place the file on the server in a directory readable by the Topbase service, for example `/srv/topbase-samples/orders.db`.
2. For Docker Compose, put the file in a `samples` directory next to the Compose file. Add the following entry to the **existing** `services.topbase.volumes` list. Apply the updated service configuration (for the release Compose file, run `docker compose -f docker-compose.release.yml up -d topbase`). The directory must be readable by the container.

   ```yaml
   - ./samples:/samples:ro
   ```

3. In **Admin → Data → Connections**, choose **Add connection**. Set the database type to **SQLite** and the display name to **Sales orders**. In **Database file**, enter the absolute path visible to the running service. For the mount above, enter `/samples/orders.db`. SQLite only needs a file path.
4. Choose **Test connection**, then **Save connection**. Open the saved source and look for **orders**. Under **Source data → Sales orders → orders**, verify that you can see orders **1001–1006**. Continue with [the dashboard tutorial](getting-started.html).

<figure><a href="../assets/guide-images/database-connect-en.jpg"><img src="../assets/guide-images/database-connect-en.jpg" alt="After the SQLite connection passes its test, Save connection becomes available. Use the file path on your own server." loading="lazy"></a><figcaption>After the SQLite connection passes its test, Save connection becomes available. Use the file path on your own server.</figcaption></figure>

## Set up a network database

Use the same **Add connection** dialog. For example, to connect a PostgreSQL database, select **PostgreSQL**, give it the display name **Sales reporting**, enter the hostname your administrator provided, port **5432**, the database name (such as `sales`), and the credentials of the reporting account. Use the database's actual listener port if it differs from the default.

| Driver | Default port in the form | What to enter |
| --- | ---: | --- |
| PostgreSQL | 5432 | Database name and a user with access to the target schema and tables. |
| MySQL / MariaDB | 3306 | Database name and a user allowed to connect from the Topbase server. |
| ClickHouse | 9000 | Native TCP endpoint and database. The default username is `default`. |
| SQL Server | 1433 | Server address, the specific database to query and the database login. |
| Oracle Database | 1521 | Listener address and **service name** in the database name field. |
| SQLite | — | Absolute file path, for example `/samples/orders.db` inside a container. |

The **display name** is what your team sees in Topbase. **Server address** and **port** refer to the database endpoint reachable from the Topbase process. In Docker, `localhost` refers to that container. For two services on the same Compose network, use the database service name and its internal listener port.

## Encryption, connection strings and SSH

- **SSL and advanced** controls the transport setting for a network driver. Choose the mode required by the database administrator. The form offers automatic/preferred encryption, required encryption, encryption without certificate verification, and disabled encryption. Driver support and server configuration determine which connection succeeds.
- **Connection string** accepts a driver-specific DSN. When supplied, it overrides the individual host, port, database and account fields. If you change a field and the test still reaches the old endpoint, check for an existing DSN. Keep DSNs containing passwords out of screenshots and messages.
- **SSH jump host** is for network databases reached through a bastion. Keep the database's internal hostname and listener port in the connection tab. In the SSH tab enter the bastion hostname, its SSH port (default **22**) and the SSH username. Select password or private key authentication. An encrypted key also needs its passphrase. Enter the bastion's trusted SHA256 host-key fingerprint to verify the server.
- **Test connection** enables **Save connection** on success. Editing connection settings requires a fresh test. When modifying a saved source, test and choose **Save and rescan**.

## Inspect the catalog or diagnose missing tables

Open the connection under **Admin → Data → Connections**. Use **Sync schema** to refresh the table list, select a table to inspect fields and types, or choose **Rescan table** after a column change. Then reopen it under [Source data](browse.html) to check actual rows.

| Symptom | First check |
| --- | --- |
| Connection timed out | Verify the host and listener port from the Topbase server or container. For a tunnel, check the SSH endpoint and the database address reachable from the bastion. |
| Test succeeds but the table is missing | Confirm the selected database and schema, grant the connection account access to the table, then refresh the schema. |
| SQLite opens an empty source or cannot open the file | Check the full path, filename, file permissions and Docker mount. The download belongs in the sample/data directory you created for this exercise. |

<figure><a href="../assets/guide-images/database-schema-en.jpg"><img src="../assets/guide-images/database-schema-en.jpg" alt="Selecting main.orders reveals its five fields. Sync schema refreshes the source catalog, while Rescan table refreshes the selected table." loading="lazy"></a><figcaption>Selecting main.orders reveals its five fields. Sync schema refreshes the source catalog, while Rescan table refreshes the selected table.</figcaption></figure>
