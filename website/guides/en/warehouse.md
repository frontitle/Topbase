# Refresh a saved analysis into a warehouse table

Use **Materializations** (`/warehouse/`) to store the output of an analysis and refresh it on a schedule. An administrator maintains these datasets. If other charts reuse the same channel summary, they can read the saved three-row table between refreshes.

## Create a daily channel summary

The current warehouse writer requires a **PostgreSQL** connection. It creates tables in the `warehouse` schema on the source connection. The connection account needs permission to create and refresh tables there, and to create the schema if it is absent. The downloadable SQLite file can be used for the dashboard tutorial. For this exercise, use a PostgreSQL connection and five example rows supplied by the query itself.

1. Open **Source data**, choose the PostgreSQL connection, select any browsable table and switch to **SQL mode**. Run the query below and save it as **Paid channel summary**. Check that the result has three rows.

   ```sql
   SELECT channel, SUM(amount) AS revenue
   FROM (VALUES ('Web', 120), ('Web', 180), ('Partner', 200), ('Store', 100), ('Store', 80)) AS paid(channel, amount)
   GROUP BY channel
   ```

2. Open **Materializations → New materialization**. Under **Source analysis**, select the saved channel summary. Saved SQL and visual analyses appear in the list. Under **Materialization name**, enter **Daily paid sales by channel**.
3. Under **Target table**, enter `paid_sales_by_channel`. The field accepts 1–60 letters, digits or underscores. Topbase converts the name to lowercase and creates a managed table under `warehouse.wh_*`.
4. Choose **Daily at 09:00** and **Full refresh**. This dialog schedules jobs in **Asia/Shanghai (UTC+8)**. The plan list shows the time zone after creation. Other presets run hourly or on Monday at 09:00. Leave the watermark empty for a full refresh.

<figure><a href="../assets/guide-images/warehouse-schedule-en.jpg"><img src="../assets/guide-images/warehouse-schedule-en.jpg" alt="The PostgreSQL example uses a daily full refresh of paid_sales_by_channel. The watermark field stays empty." width="620" height="698" loading="lazy"></a><figcaption>The PostgreSQL example uses a daily full refresh of paid_sales_by_channel. The watermark field stays empty.</figcaption></figure>

5. Select **Create and run**. When the run succeeds, the list shows the target `warehouse.wh_paid_sales_by_channel`, saved row count, source, schedule and latest run. Choose **Create analysis** on that row to inspect the result. Expect three channels with sums of **300, 200 and 180**, totaling **680**.

<figure><a href="../assets/guide-images/warehouse-results-en.jpg"><img src="../assets/guide-images/warehouse-results-en.jpg" alt="The completed materialization lists three saved rows and the physical target. Open Create analysis to verify the channel totals." width="1360" height="850" loading="lazy"></a><figcaption>The completed materialization lists three saved rows and the physical target. Open Create analysis to verify the channel totals.</figcaption></figure>

## Choose how subsequent runs write data

| Save method | Choose it when | What the next run does |
| --- | --- | --- |
| Full refresh | Totals may change, an order may be corrected, or the source is already grouped. | Rebuilds the current result from the saved analysis. |
| Append changes only | The source is an append-only detail result with a reliable increasing field. | Appends rows whose watermark is strictly greater than the previous saved watermark. |

For an append-only order event log, include an increasing event ID in the analysis output and enter its exact field name in **Watermark field**. The first run stores the available result. New rows with the same or an earlier watermark fall outside the next refresh. For a channel summary, updating one channel changes an existing aggregate, so use a full refresh.

## Recover a refresh

If the first run fails after the plan was created, locate its existing row and inspect the error. Confirm the source analysis still runs, correct the cause and use **Retry update**. If the status says **Needs admin review**, have an administrator compare the target table, watermark and run record before attempting another refresh. Reopen the target after a successful run and compare its result with the source.
