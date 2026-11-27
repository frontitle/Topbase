# Performance and capacity

By default a node runs up to eight interactive queries concurrently. A result is limited to 16 MiB and a single cell to 1 MiB. Narrow the date range or aggregate rows before increasing limits. Inspect memory usage in the admin monitor and check the database connection pool when queries queue.
