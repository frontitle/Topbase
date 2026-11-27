# Schedule a warehouse table

Use a materialized table when several analyses reuse the same expensive result or a team needs a scheduled snapshot. Run and save the source analysis first. Its output fields become the inputs for later analysis.

## Choose an update strategy

In the warehouse, choose New materialization, select the saved analysis, name the target and configure its schedule. A full refresh is appropriate when older records change. For incremental updates, choose a reliable watermark field. Rows older than the watermark can be missed when late corrections arrive.

## Inspect a run

Inspect the recent run, status and target table. Compare a known total with the source analysis for the same time range. If the result is stale, check the last successful run and schedule before changing a downstream chart.
