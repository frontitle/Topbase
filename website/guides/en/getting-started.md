# Build your first sales dashboard

Which channel brought in the most paid sales? Start with six simulated orders, answer the question in the visual query editor, then put the chart on a dashboard. The finished chart has three bars. Web leads with 300. The steps and screenshots were checked in Topbase 0.2.6.

## Open the sample orders

1. Download [the orders database](../assets/samples/orders.db). Ask a Topbase administrator to [connect the SQLite file](database-drivers.html) and give you access. If you are the administrator, name the connection **Sales orders**. The file must be readable by the Topbase server.
2. Open **Source data**, choose **Sales orders**, then select **orders**. Each row is one order. The table contains five fields. `order_id` identifies the order, `order_date` gives the date, `channel` is the sales channel, `amount` is the order amount and `status` distinguishes paid from refunded orders.
3. Check order **1003**. Its amount is **50** and its status is **refunded**. The question asks about paid sales, so this row should drop out before you sum the amounts.

The [Excel upload exercise](excel.html) uses the same six orders. Use the SQLite connection for this dashboard so that the query editor can read the records and sum a numeric column.

## Ask the question in the visual editor

1. With **orders** open, click **Expand query editor** if the editor is collapsed, then choose **Visual query**.
2. In **Filter**, choose `status`, select the equality operator and enter `paid`. In **Summarize**, choose **Sum**, select `amount` as the field and choose `channel` under **Group**.
3. Open **Sort**, select the aggregate result and choose descending order. Click **Run and preview**. Compare the result with the table below before you choose a chart.

| Channel | Sum of amount |
| --- | ---: |
| Web | 300 |
| Partner | 200 |
| Store | 180 |

The three groups add up to **680** across five paid orders. If Web shows **350**, check the status filter. If it shows **2**, the calculation is counting Web orders. Change the calculation to **Sum** and run the query again.

<figure><a href="../assets/guide-images/visual-query-en.jpg"><img src="../assets/guide-images/visual-query-en.jpg" alt="Compare your filter, calculation and grouping with these settings. The result contains Web, Partner and Store." loading="lazy"></a><figcaption>Compare your filter, calculation and grouping with these settings. The result contains Web, Partner and Store.</figcaption></figure>

## Save a chart

1. Choose **Save analysis**, enter **Sales by channel** and choose a writable group. Your personal group works for this exercise. Choose a group your colleagues can open if they will use the saved analysis.
2. On the saved analysis, choose **Column chart**. Use `channel` as the category and the summed amount as the value. Hover over the Web bar to check that it shows **300**. See [chart settings](charts.html) for axis and formatting options.

<figure><a href="../assets/guide-images/chart-settings-en.jpg"><img src="../assets/guide-images/chart-settings-en.jpg" alt="The saved analysis shows Web at 300. Choose Column chart above the result and set channel and sum in the right-hand panel." loading="lazy"></a><figcaption>The saved analysis shows Web at 300. Choose Column chart above the result and set channel and sum in the right-hand panel.</figcaption></figure>

## Put the answer on a dashboard

1. Open **Dashboards → New dashboard**. Enter edit mode if the canvas opens in view mode.
2. Choose **Insert → Add analysis**, search for **Sales by channel**, and select the result. Drag the card by its handle and resize it until the three channel names fit.
3. Choose **Preview** to see the dashboard as a reader. Finish editing and reopen the dashboard from the list to check the saved layout. To give another person access, follow [dashboard sharing](sharing.html).

If the analysis is missing from the insert menu, first open it from your analyses and check its name and group. Clear the group filter in the insert menu, then search again.
