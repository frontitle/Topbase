# Run a SQL analysis with parameters

Write a read-only SQL query when you want to control the calculation or let a reader choose a value at run time. This example starts with a fixed channel report and adds an optional channel selector to the saved analysis.

## Run the query in the editor

1. Open **Source data → Sales orders → orders** using the SQLite connection from [Getting started](getting-started.html). Choose **SQL mode** above the query steps. The current connection determines which tables and SQL dialect are available.
2. Paste the query below. You can also run a [visual query](analysis.html) first, switch modes and click **Use SQL from the visual query** to start with the statement Topbase generated.
3. Click **Run SQL**. Check that the two columns are called `channel` and `revenue`. The result should show Web **300**, Partner **200** and Store **180**, in that order.

```sql
SELECT channel, SUM(amount) AS revenue
FROM orders
WHERE status = 'paid'
GROUP BY channel
ORDER BY revenue DESC
```

To investigate one order, replace the query temporarily with `SELECT * FROM orders WHERE order_id = 1003`. Its status is `refunded`, so its amount of 50 is excluded from the paid total. Restore the grouped query before continuing.

## Give the reader an optional channel filter

1. In the SQL editor, insert the following line **between** the existing `WHERE` and `GROUP BY` lines. Keep the `AND` inside the square brackets so the statement is valid when the whole condition is skipped.

   ```sql
   [[AND channel = {{channel}}]]
   ```

2. Click **Save analysis**, enter **Paid sales by channel (SQL)**, choose a writable group and confirm. Use the saved analysis to supply parameters. The table editor's **Run SQL** executes SQL without a parameter form.
3. In the saved analysis, choose **Query parameters**, enter `Web` without quotes and run. You should see one row with **300**. Reopen **Query parameters**, clear the value and run again. All three channels should reappear.
4. To require a channel on every run, use `AND channel = {{channel}}` without the optional brackets. A newly opened analysis with this required parameter waits for input. Use names such as `channel` or `start_date`. Leave the placeholder itself outside SQL string quotes so Topbase can bind the value.

<figure><a href="../assets/guide-images/sql-parameters-en.jpg"><img src="../assets/guide-images/sql-parameters-en.jpg" alt="In the saved SQL analysis, the channel parameter takes Web as its value." loading="lazy"></a><figcaption>In the saved SQL analysis, the channel parameter takes Web as its value.</figcaption></figure>

<figure><a href="../assets/guide-images/sql-parameter-result-en.jpg"><img src="../assets/guide-images/sql-parameter-result-en.jpg" alt="Running the saved analysis with Web returns just that channel and revenue of 300." loading="lazy"></a><figcaption>Running the saved analysis with Web returns just that channel and revenue of 300.</figcaption></figure>

## Add a daily trend to the dashboard

To make the second chart in the [dashboard guide](dashboards.html), start a **new** SQL analysis of the same orders and run this query. The date is the grouping field. Sorting it in ascending order puts the points in chronological order.

```sql
SELECT order_date, SUM(amount) AS revenue
FROM orders
WHERE status = 'paid'
GROUP BY order_date
ORDER BY order_date
```

The result has three dates in October 2026. October 1 totals **200**, October 3 totals **300** and October 4 totals **180**. October 2 had only a refunded order, so there is no row for that day. Save as **Paid sales by day**. Set the chart to **Line chart**, with `order_date` on X and `revenue` on Y.

<figure><a href="../assets/guide-images/sql-editor-en.jpg"><img src="../assets/guide-images/sql-editor-en.jpg" alt="The daily query returns three dates. Compare revenue in the preview before choosing a line chart." loading="lazy"></a><figcaption>The daily query returns three dates. Compare revenue in the preview before choosing a line chart.</figcaption></figure>

## Reuse and troubleshoot a saved query

- **Inspect or change the query.** Open the analysis from **Analyses**. **View query** shows the saved SQL. **Edit** opens the editor if you have write access. Keep column aliases stable when a chart already refers to them.
- **Download the selected channel.** Set the parameter to `Web`, run, then click **Export CSV**. Check the file for the header `channel,revenue` and the Web row. Clearing the optional parameter restores the unfiltered channel report.
- **Table or column not found.** Check the physical names in the table browser and the database selected in the editor. If the table lives in another schema, qualify it using the source database's SQL syntax.
- **The query will not run.** If the message asks for a parameter, supply its value in the saved analysis. For an access error, ask an administrator to check data access and native SQL execution for this database. A shared analysis may be visible even when the reader lacks permission to execute it.
