# Query Topbase from an AI client or the CLI

Connect your AI client through MCP, or run the CLI yourself to inspect each step. Both use the Topbase HTTP API with the permissions of the person who created the key. This example uses the SQLite connection from [the first dashboard tutorial](getting-started.html).

## Enable access and create a personal key

1. An administrator opens **Administration → Settings → Developer → Developer mode**, enables developer mode and allows personal keys if ordinary members will create them. Set the public Topbase URL, default lifetime for new keys and maximum query rows (default 200). Enable AI analysis creation if users should be able to save their results.
2. Open **My account → AI and API**, choose **Create key**, and name it for the client. Store the full value when it appears. Topbase shows it only once. Revocation on this page immediately stops that client's access.
3. On the computer running the client, supply `TOPBASE_URL` (the base address of your instance) and `TOPBASE_API_KEY` through its environment or secret store. A source build with `make build` produces `bin/topbase-cli` and `bin/topbase-mcp`. The Docker image includes both under `/app/`.

<figure><a href="../assets/guide-images/ai-settings-en.jpg"><img src="../assets/guide-images/ai-settings-en.jpg" alt="Developer settings before activation. Choose the query row limit and key lifetime, then save the settings." width="980" height="595" loading="lazy"></a><figcaption>Developer settings before activation. Choose the query row limit and key lifetime, then save the settings.</figcaption></figure>

In Bash or Zsh, replace the example URL with your instance address, then paste the key when prompted. The key stays out of your command history.

```sh
export TOPBASE_URL="https://analytics.example.com"
printf 'Topbase API key: '
read -rs TOPBASE_API_KEY
printf '\n'
export TOPBASE_API_KEY
```

## Run and verify a CLI query

Run the first two commands below from a source build. In the database list, find **Sales orders**, then set the shell variable `ORDERS_DB_ID` to its returned ID before running the last two commands. Confirm there is a `main.orders` table with fields `channel`, `amount` and `status`.

```sh
./bin/topbase-cli status
./bin/topbase-cli databases
./bin/topbase-cli tables "$ORDERS_DB_ID"
./bin/topbase-cli describe "$ORDERS_DB_ID" main orders
```

Save the following as `query.json`, replacing the database ID placeholder with the same ID. The filter keeps paid orders, then the aggregation adds their amounts for each channel.

```json
{
  "version": 1,
  "source": {
    "database_id": "REPLACE_WITH_SALES_ORDERS_ID",
    "table": { "schema": "main", "name": "orders" }
  },
  "filters": [{ "field": "status", "op": "eq", "value": "paid" }],
  "group_by": [{ "field": "channel" }],
  "aggregations": [{ "fn": "sum", "field": "amount", "alias": "revenue" }],
  "limit": 10
}
```

Run `./bin/topbase-cli query --file query.json`. Check for Web 300, Partner 200 and Store 180. To save the verified result, run `./bin/topbase-cli create-analysis --name "Sales by channel" --file query.json --confirm`. By default it goes in your personal group. To choose a different destination, run `./bin/topbase-cli groups` and pass the writable group's ID with `--group`.

## Use the same workflow in an MCP client

For clients that read a JSON configuration and inherit the environment set above, use this server definition. Replace the command with the actual executable path. For a desktop client opened outside that shell, set the URL and key in its environment or secrets fields.

```json
{
  "mcpServers": {
    "topbase": {
      "command": "/absolute/path/to/bin/topbase-mcp"
    }
  }
}
```

Add a **stdio** server in your client's MCP settings. Set its command to the absolute path of the `topbase-mcp` executable on the machine running the client. Supply both environment variables to that server and reconnect. Ask “List the databases I can access, inspect the orders table, then show paid sales by channel.” Check the result before asking to save it.

If the client cannot connect, run `status` against the same URL and key. Check developer mode, expiry and revocation. If discovery succeeds but a table is missing, check the key owner's permissions. If the query works but saving fails, check AI creation permission and write access to the destination group.
