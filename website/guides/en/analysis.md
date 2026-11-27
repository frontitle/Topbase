# Build a visual analysis

How much did each channel earn from paid orders? Use the visual query editor to answer that question, check the total, and save a reusable analysis. Follow along with the six-row orders table from [Getting started](getting-started.html).

## Calculate paid sales by channel

1. Open **Source data**, select the sample database, then select `orders`. Select **Show query editor** if it is collapsed, then choose **Visual query**. The first preview shows individual orders.
2. Under **Filter**, add a condition on `status`, choose equality and select or enter `paid`. Under **Summarize → Calculate**, choose **Sum** and select `amount`. Leave **Group** empty and click **Run query**. The single value should be **680**.
3. Set **Group** to `channel`. Open **Sort**, select the aggregated result `sum` and choose **Descending**. Run the query again. Each row now represents one channel.

| Channel | Sum of paid order amounts |
| --- | ---: |
| Web | 300 |
| Partner | 200 |
| Store | 180 |

4. Click **Save analysis**, enter **Sales by channel**, pick a group you can write to, and confirm. The saved analysis opens. Use **Edit** to change its query later. Run the query after changing the visual steps and before saving so the saved definition includes those changes.

<figure><a href="../assets/guide-images/visual-query-en.jpg"><img src="../assets/guide-images/visual-query-en.jpg" alt="The paid filter and sum of amount grouped by channel return three rows. Compare the totals before saving." loading="lazy"></a><figcaption>The paid filter and sum of amount grouped by channel return three rows. Compare the totals before saving.</figcaption></figure>

## Choose what one row represents

| Question | Calculate | Group |
| --- | --- | --- |
| Which orders were paid? | No aggregation. Select the desired columns under **Data**. | Leave empty |
| How many orders were paid? | Row count, with the paid filter. The example returns 5. | Leave empty |
| How many channels had a paid order? | Distinct count of `channel`. The example returns 3. | Leave empty |
| How did sales change over time? | Sum of `amount` with the paid filter | Choose a date-typed `order_date`, then select Day, Week, Month, Quarter or Year |

**Average**, **Minimum** and **Maximum** also require a field. **Row limit** controls how many result rows come back, from 1 to 10,000, with a default of 1,000. For the top two channels, keep the descending sort and set the limit to 2. Remove that limit before reconciling the grand total.

## Join a lookup table

Suppose your database also contains a `channels` table with one row per channel and an `owner` column. Open **Join data**, choose `channels`, and match `orders.channel` to `channels.channel`. Select **Keep all rows from the current table** to retain orders with a missing owner, or **Keep matching rows only** to omit unmatched orders. Choose `owner` under **Include fields from the joined table** and run with **No aggregation** to inspect the pairs. A known foreign key can preselect the match.

If order IDs repeat after the join, look for duplicate keys in the lookup table. Two matching owners create two copies of an order and can inflate a sum.

## Calculate a value for each record

With **No aggregation** selected, open **Custom field**. Name the field `amount_cents`, set the left operand to `amount`, select **Multiply**, and enter `100` as the right operand. Run the query to see the extra column beside each order. The editor also supports addition, subtraction, division and concatenation with a field or a constant. To summarize the result of a calculation in the same query, use the [SQL editor](sql.html).

To turn the saved channel totals into a chart, follow [Choose and configure a chart](charts.html).
