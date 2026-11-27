# Connect and inspect a database

An administrator needs the database type, network address, database name and credentials. Grant the connection account read access to the tables analysts need. SQLite requires a file accessible to the Topbase process.

## Test the connection

Open Admin → Data → Remote data sources → Add database. Select the driver and enter the connection details. Configure SSL if required. For an SSH tunnel, enter the database’s internal address in the connection tab and the jump host in the SSH tab. Test, then save.

## Locate a missing table

Open the saved source and synchronize its structure. Select a table to inspect its fields or rescan that table after a schema change. If a connection works but a table is missing, verify the database name and the connection account’s access to that schema.
