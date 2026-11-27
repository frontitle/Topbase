# Describe tables and fields in the data model

A reader seeing `amount` needs to know whether a refunded order is included. An administrator can add a business definition, a readable field name and a semantic type in Topbase. These settings help people interpret the source data when building a query.

## Open the metadata editor

1. Open **Administration → Data model**. For the sample database, select **Remote sources**, choose **Sales orders** under **Data source**, search for `orders` and click the table. The remote-sources tab includes configured database connections such as the sample SQLite file. For an uploaded Excel dataset, choose **Local warehouse** and select the imported dataset.
2. You can also open a connected table from **Source data** and click **Edit metadata**. This administrator shortcut opens the corresponding database, schema and table. Compare the physical name above the editor with the table you intend to describe.
3. Enter **Order details** under **Table alias**. Under **Business notes**, enter “One row per order. Amount includes all statuses. Filter status to paid to report paid sales.” Topbase shows this note in preference to the source table comment in the workspace.
4. Pause after typing to allow the automatic save to finish. Reopen the table to check the alias and note. Use **Full screen** above the fields for a wider editor and **Esc** to return.

<figure><a href="../assets/guide-images/data-model-en.jpg"><img src="../assets/guide-images/data-model-en.jpg" alt="The data model editor shows a table alias and business note above the field grid. The physical field names are listed on the left." width="1360" height="850" loading="lazy"></a><figcaption>The data model editor shows a table alias and business note above the field grid. The physical field names are listed on the left.</figcaption></figure>

## Give fields a business meaning

The **Physical name** and **Type** columns come from the source. Use the editable **Alias**, **Field description** and **Semantic type** columns to describe how each field should be used. Example settings for `orders` follow.

| Physical name | Alias | Semantic type | Field description |
| --- | --- | --- | --- |
| `order_id` | Order number | Primary key | Identifies one order. Use it to look up a record. |
| `order_date` | Order date | Creation date | Calendar date the order was placed. |
| `channel` | Sales channel | Category | Web, Partner or Store. Group by this field to compare channels. |
| `amount` | Order amount | Currency | Amount on the order, including records with refunded status. |
| `status` | Payment status | Category | Filter to paid for the paid-sales report. |

Field edits save automatically. Watch for the save notification before switching tables. A semantic type describes a field's meaning. If a source stores an amount as text, correct its input type or return a numeric value in SQL before summing it.

## Describe a relationship

If you also maintain a separate `channels` lookup table, select the `orders.channel` row. Choose **Foreign key** and fill in both **Referenced table** (`channels`) and **Referenced field** (`channel`). Use the target's physical names. Confirm that its channel key is unique and that the order values match. Known relationships can help **Join data** preselect fields the next time you [build a visual analysis](analysis.html).

## Set visibility and check the result

The eye beside the table title controls its workspace visibility. The eye on a field row controls that field's visibility. To undo a change, open the same object in **Data model** and click the eye again. For sensitive fields such as an email or phone number, the semantic type menu marks options that are masked by default. Use [permissions](permissions.html) to control who can read the underlying data.

Return to **Source data** and reopen the sample table. Look for **Order details** and its business note. In the result preview, check the display name **Order amount**. In SQL, continue to address the physical field as `amount`. If a newly added column is absent from the editor, sync the source schema in database administration, then reopen the table.
