# Choose and configure a chart

The same query can answer different questions depending on how you display it. A column chart shows which channel leads. A donut shows its share of the total. Start with the three rows in [Sales by channel](analysis.html).

## Turn channel totals into a column chart

1. Open **Analyses** and select your saved **Sales by channel**. Wait for the result. You need edit access to change its visualization. The chart types appear above the result and **Settings** appears beside it.
2. Select **Column chart**. In **Data**, set **X-axis** to `channel` and **Y-axis** to the numeric total. The visual query labels it `sum`. If you used the [SQL example](sql.html), choose `revenue`.
3. Under **Display**, enable **Show values**. For a target of 200 per channel, turn on **Target line**, enter **200** as the target value and label it **Channel target**. Web should be above the line at 300, Partner on it at 200 and Store below at 180.
4. Under **Axes**, give the Y-axis a title such as **Paid sales**. Set the unit in the title if your amounts have a known currency. Chart changes save automatically. Reopen the analysis to check the chart before adding it to a [dashboard](dashboards.html).

When several series share an axis, **Add grouping** splits a single metric by another dimension present in the result. For a query that returns separate numeric columns, use **Add metric** with no additional grouping. Use percentage stacking when the question is about each series' share within a group.

<figure><a href="../assets/guide-images/chart-settings-en.jpg"><img src="../assets/guide-images/chart-settings-en.jpg" alt="For the channel comparison, select Column chart and map channel to X and sum to Y." width="1360" height="850" loading="lazy"></a><figcaption>For the channel comparison, select Column chart and map channel to X and sum to Y.</figcaption></figure>

## Show the share of each channel

Select **Pie chart** on the saved analysis. In **Data**, choose `channel` as **Dimension** and the sum as **Metric**. In **Display**, turn on **Donut chart** and **Show total**. Set **Show percentages** to **Legend** and **Percentage decimal places** to **1**.

The total should be **680**. Web accounts for **44.1%**, Partner **29.4%** and Store **26.5%**. **Maximum categories** and **Minimum slice share (%)** control which small categories are collected under **Other**. Leave the category limit at 8 and the threshold at 0 to keep these three channels visible. Changing the chart type updates this saved analysis. To keep both charts, save a second analysis from the query editor.

## Match another question to a result

| What the reader needs | Result to prepare | Chart and settings |
| --- | --- | --- |
| A single paid-sales total | One row with a numeric total of 680 | **Number**. Choose the total. **Compare with** uses another numeric column returned by the query. |
| A trend over several weeks | One row per date or week, with a numeric measure | **Line chart** or **Area chart**. Map time to X and the measure to Y. Check the date order. |
| Ranked categories with long labels | One row per category, sorted by total | **Bar chart** draws horizontal bars. Keep the category labels visible. |
| Individual points across dates or categories | One dimension and one numeric measure | Choose **Scatter plot** to display the observations as points. Hover over a point to check its value against the table. |
| Individual values or order IDs | Detail rows with the desired fields | **Table**. Use the display settings to select visible columns and change their titles. |
| A geographic distribution | A region column and one numeric total per region | **Map**. Select a map package, region field and value field. The China provinces package accepts values such as `广东`, `广东省` or `440000`. |

## Check an unexpected chart

Switch to **Table** to see the rows and names actually returned. If an axis lists order IDs, group the query by channel or date first. If a numeric field is missing, inspect the source values for currency symbols or text and return a numeric column. After fixing a query, check both axis selections again, especially if you renamed a SQL alias.
