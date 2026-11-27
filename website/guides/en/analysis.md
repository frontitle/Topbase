# Filter, summarize and join data

Start in Analyses → New analysis. Choose a table or a saved analysis. Inspect a few rows before selecting fields. Keep an identifier in the result while checking the level of detail.

## Filter and summarize

For paid sales by channel, first filter the status to paid and limit the order date. Choose sum of amount and group by channel. Sort the total descending. A row count measures volume, while a sum measures revenue. Run once without grouping to check that the channel totals add up.

## Join or calculate a field

Join orders to a customer table using customer ID to break down sales by customer segment. Check that each customer ID occurs once in the customer table. Compare totals before and after joining. For line items, calculate quantity times unit price in a custom column before summing the result.

## Save and reuse

Run the analysis, give it a descriptive name and select its group. For a SQL analysis, select the database and use the SQL editor with native query permission. Check the returned field names before adding the result to a dashboard.
