# Import an Excel worksheet

Use an import when your source is a file, such as a monthly sales export. The imported table reflects the worksheet at upload time. Changes in the original workbook require a new import.

## Prepare and upload

Use one header row and one record per row. Put totals outside the data range to avoid counting them twice. In Admin, open Data, select Local warehouse at the top and choose Upload Excel. Select the workbook and the sheet containing the records.

## Check the result

Find the dataset in Local warehouse. Check the file and sheet names, row count and field count against the workbook. Preview sample rows, especially identifiers with leading zeroes and numeric amounts. If columns are misaligned, correct the headers or merged cells in the workbook before importing again.
