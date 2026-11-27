# Upload and preview an Excel worksheet

Use the local warehouse to keep a copy of a spreadsheet and inspect its records in Topbase. Try it with [orders.xlsx](../assets/samples/orders.xlsx). You will import one worksheet with six rows and five fields.

## Prepare the workbook

| What to check | How the upload handles it |
| --- | --- |
| Format and size | Use an `.xlsx` or `.xlsm` workbook. Keep the upload below 20 MB. Save older `.xls` files as `.xlsx` first. |
| Worksheet order | Move the sheet you want to the first tab and save. The upload button reads that sheet automatically. |
| Header row | Use one header per column in row 1. Remove report titles, merged headers and blank columns. An empty header inside the header range gets a name such as `字段2`. A repeated header gets a suffix such as `amount_2`. |
| Data rows | Place one record per row starting in row 2. Entirely empty rows are skipped. The importer keeps up to 100,000 nonempty rows. Split longer sheets before uploading. |
| Cell values | Values are stored as text. Put a header above every column you want to keep. Remove totals or footnotes that should not count as records. |

## Upload the first sheet

1. Sign in as an administrator. Open **Admin → Data**. At the top of the page, switch from **Connections** to **Local warehouse**.
2. Click **Upload Excel** and choose the downloaded **orders.xlsx** in the file picker. Selecting the file starts the import.
3. Find **orders** in the local warehouse. Topbase takes the dataset name from the filename. Check the source filename **orders.xlsx**, worksheet **orders**, **6 rows** and **5 fields**.
4. Click **Preview rows**. Find order **1003** and verify that its `amount` is **50** and its `status` is **refunded**. This checks that the columns and their values line up.

<figure><a href="../assets/guide-images/excel-library-en.jpg"><img src="../assets/guide-images/excel-library-en.jpg" alt="Under Local warehouse, the orders card shows the filename, sheet name and counts. Upload Excel is in the upper-right corner." loading="lazy"></a><figcaption>Under Local warehouse, the orders card shows the filename, sheet name and counts. Upload Excel is in the upper-right corner.</figcaption></figure>

## Read the preview and correct an import

The dataset card shows the imported row count. Preview displays up to the first 50 rows and 50 fields. Very long cell values can exhaust the 128 KiB preview limit sooner. Check the count on the card when verifying a large import.

| What you see | What to do |
| --- | --- |
| The wrong worksheet appeared | Move the intended sheet to the first tab, save the workbook and upload again. |
| A title became a field, or data moved into the wrong column | Put the real field names in row 1, remove merged cells and make sure every record follows the same column order. |
| Two entries have the same name | Each upload creates a separate dataset. Give the corrected file a distinct name, preview it and then remove the obsolete entry with **Delete**. Deleting an entry removes the imported copy in Topbase. |
| You want to chart or query these orders | The Excel import is available in the local warehouse preview. For the query editor and SQL, use a connected database. Download [the matching SQLite file](../assets/samples/orders.db) and follow [Build your first sales dashboard](getting-started.html). |

<figure><a href="../assets/guide-images/excel-preview-en.jpg"><img src="../assets/guide-images/excel-preview-en.jpg" alt="The worksheet preview includes the refunded Web order 1003 and its amount of 50." loading="lazy"></a><figcaption>The worksheet preview includes the refunded Web order 1003 and its amount of 50.</figcaption></figure>
