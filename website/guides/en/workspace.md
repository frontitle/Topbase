# Configure your workspace and public address

The first administrator creates the workspace. Administrators can then set its name, time zone, external address and sharing policy under **Admin → Settings**. These settings govern the instance your team signs into.

## Create the workspace and set its clock

1. On a fresh installation, open the Topbase address in your browser. The setup screen shows the application database being used. Check that the storage will persist across restarts. For a container using local storage, make sure the deployment mounts its data directory.
2. Enter the workspace name, administrator name, sign-in email and a password of at least eight characters. Choose **Create workspace**. For an existing workspace, sign in with an account provided by its administrator.
3. Open **Admin → Settings → Workspace**. Set a descriptive name such as **Sales reporting**, choose the default time zone and click **Save changes**. Check the time zone on each existing schedule. The [materialization guide](warehouse.html) explains the time used for warehouse refreshes.

<figure><a href="../assets/guide-images/workspace-settings-en.jpg"><img src="../assets/guide-images/workspace-settings-en.jpg" alt="The Workspace tab puts the name and default time zone together. Save changes in the upper-right corner." loading="lazy"></a><figcaption>The Workspace tab puts the name and default time zone together. Save changes in the upper-right corner.</figcaption></figure>

## Generate a link that teammates can open

Suppose colleagues open Topbase at `https://bi.example.com:8443` through a reverse proxy. In **Access and security**, set these three values, check the generated address and save.

| Setting | Example value | Meaning |
| --- | --- | --- |
| Protocol | HTTPS | The scheme used in the colleague's browser. Automatic detection is also available. |
| Custom domain | `bi.example.com` | Hostname only. Enter the port separately. |
| External port | `8443` | Port for generated links. Leave it empty for the protocol default, such as 443 for HTTPS. |

Open an existing dashboard's **Share** dialog and check the generated hostname. DNS, certificates, reverse proxy routing, Docker port mapping and the server listener are configured in your deployment. If the address looks right but cannot be opened, verify those settings from the reader's network.

## Control public access

Under **Access and security**, **Allow public links** lets published dashboards be opened without signing in. **Allow embedding** allows an already published dashboard to appear in an iframe. Save changes, then open **Admin → Embedding and public links** to review the affected content. Turning off public links disables existing public access and embedding. Test intended reader access with the [sharing guide](sharing.html).

## Check or back up application data

Under **Application data**, check the engine and storage status. This database holds workspace accounts, connections, analyses, dashboards and schedules. Choose **Export backup** to download a logical backup and store the export securely. It includes password hashes and encrypted connection information. Back up business source databases separately.

Where **Migrate to production** appears for a SQLite application database, first export a backup and prepare an empty, dedicated PostgreSQL or MySQL database. Enter its host, port, database, credentials and TLS mode. PostgreSQL also takes a schema. Review the confirmation before starting the one-time copy. After success, check the reported row and table counts, update the deployment's application-database settings and restart Topbase. Reopen **Application data** and verify the active engine and a known dashboard.
