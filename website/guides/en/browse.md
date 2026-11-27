# Browse tables and export results

Use the table browser when you want to see the records behind a number. You can inspect a source, narrow the result to a few orders and save the query for a repeatable CSV export.

## Open a table

1. In the workspace, open **Source data**. Search for the database by its connection name and select its card. If you need a sample, use the database in [Getting started](getting-started.html).
2. Use **Search tables** above the table browser. You can search by table name, alias or column name. Tables are grouped under their database schema. Select `orders` and check the schema and physical table name under **Data**.
3. Wait for the row count beneath the query editor. Opening a table runs a read-only preview with a default limit of 1,000 rows. A count of 1,000 in a large table tells you how many rows came back. For the table's full row count, run **Summarize → Row count** with no filters or grouping.

**Source data** queries a connected database when you run the query. An administrator can preview an uploaded Excel worksheet separately under **Administration → Data → Local warehouse**. To try these query steps with the same spreadsheet records, connect the sample SQLite database. See [Upload and preview an Excel worksheet](excel.html).

<figure><a href="../assets/guide-images/browse-orders-en.jpg"><img src="../assets/guide-images/browse-orders-en.jpg" alt="The source browser shows the six orders in main.orders. Click Expand query editor above the rows to add a filter." loading="lazy"></a><figcaption>The source browser shows the six orders in main.orders. Click Expand query editor above the rows to add a filter.</figcaption></figure>

## Inspect the records for one channel

1. Open **Visual query → Data** and expand the field list. Select `order_id`, `channel`, `amount` and `status`. Clear the fields you do not need.
2. Under **Filter**, add `channel` equal to `Web` and `status` equal to `paid`. Keep **Calculate** on **No aggregation**. Click **Run and preview**.
3. Read the order IDs and amounts in the result. In the sample, the visible amounts should add up to **300**. To verify that in the database, change **Calculate** to **Sum**, select `amount`, leave **Group** empty and run again.
4. To export individual orders, restore **No aggregation** and run again before saving. Use **Sort** if the CSV needs a particular ordering. For a larger dataset, check the **Row limit** before interpreting the result as all matching orders.

## Export a saved result to CSV

1. Click **Save analysis**, name it **Paid Web orders**, select a writable group and confirm. You can reopen the saved item from **Analyses**.
2. On its detail page, click **Export CSV**. Topbase reruns the saved query. If you changed a filter temporarily on this page, the export confirmation tells you it will use the saved filters. To permanently change which rows qualify, use **Edit**, change the query, run it and save.
3. Open the file and check its header and order IDs. For this example, every record should have `channel = Web` and `status = paid`. SQL analyses with parameters use the supplied parameter values during export. See [Run a SQL analysis](sql.html).

## When a table or file is missing

- **The table is absent.** Check the selected database and schema, clear the search and ask an administrator to verify table visibility and your group's data access.
- **You can see table names but the preview fails.** A cached schema can still be listed while the remote database is unavailable. An administrator can test the connection and sync its schema after a table or column changes.
- **The analysis opens but the CSV is denied.** Export reruns the query. Ask an administrator to check data access and, for SQL analyses, native SQL permission. If the message says the session expired, sign in again and retry.
