# 从 AI 客户端或命令行查询 Topbase

支持 MCP 的 AI 客户端可以调用 Topbase，CLI 则可以逐步检查每次操作。两种方式都通过 HTTP API 使用密钥创建者的权限。下面沿用[首个仪表盘教程](getting-started.html)中的 SQLite 连接。

## 开通访问并创建个人密钥

1. 管理员打开 **管理后台 → 设置 → 开发者 → 开发者模式** 并启用开发者模式。如果普通成员也要创建密钥，允许个人 Key。设置 Topbase 对外地址、新密钥的默认有效期和单次查询行数上限（默认 200 行）。需要保存分析时，启用允许 AI 新增分析的选项。
2. 进入 **个人中心 → AI 与 API**，点击 **创建密钥**，按客户端命名。完整密钥仅显示一次，此时保存到客户端的密钥存储。日后在同一页面撤销，该客户端会立即失去访问权限。
3. 在运行客户端的电脑上，通过环境变量或密钥存储提供 `TOPBASE_URL`（实例访问地址）和 `TOPBASE_API_KEY`。源码目录执行 `make build` 后，工具位于 `bin/topbase-cli` 和 `bin/topbase-mcp`。Docker 镜像中的工具位于 `/app/`。

<figure><a href="../assets/guide-images/ai-settings-zh.jpg"><img src="../assets/guide-images/ai-settings-zh.jpg" alt="打开开发者模式后，设置查询行数上限和新密钥的默认有效期，再点击「保存开发者设置」。" loading="lazy"></a><figcaption>打开开发者模式后，设置查询行数上限和新密钥的默认有效期，再点击「保存开发者设置」。</figcaption></figure>

在 Bash 或 Zsh 中，将示例网址改为自己的实例地址，执行以下命令，在提示时粘贴密钥。密钥不会出现在命令历史中。

```sh
export TOPBASE_URL="https://analytics.example.com"
printf 'Topbase API key: '
read -rs TOPBASE_API_KEY
printf '\n'
export TOPBASE_API_KEY
```

## 用 CLI 查询并核对渠道金额

在源码构建目录先运行前两条命令。从返回的数据库列表找到 **Sales orders**，将 shell 变量 `ORDERS_DB_ID` 设为其实际 ID，再运行后两条命令。确认存在 `main.orders` 表，并核对 `channel`、`amount` 和 `status` 字段。

```sh
./bin/topbase-cli status
./bin/topbase-cli databases
./bin/topbase-cli tables "$ORDERS_DB_ID"
./bin/topbase-cli describe "$ORDERS_DB_ID" main orders
```

将下面内容保存为 `query.json`，把数据库 ID 占位值替换为刚才查到的 ID。筛选器保留已支付订单，再按渠道对金额求和。

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

运行 `./bin/topbase-cli query --file query.json`，核对结果为 Web 300、Partner 200、Store 180。确认后运行 `./bin/topbase-cli create-analysis --name "已支付渠道销售额" --file query.json --confirm` 即可保存，默认存入个人分组。如需更换位置，先运行 `./bin/topbase-cli groups`，再用 `--group` 指定可写分组的 ID。

## 在 MCP 客户端中操作

支持 JSON 配置且继承上述环境变量的客户端，可以使用下面的服务定义。把启动命令换为实际可执行文件的绝对路径。如果桌面客户端不是从该终端启动，在其环境变量或密钥设置中单独提供网址与密钥。

```json
{
  "mcpServers": {
    "topbase": {
      "command": "/absolute/path/to/bin/topbase-mcp"
    }
  }
}
```

在 AI 客户端的 MCP 设置中新增一个 **stdio** 服务，启动命令填客户端所在电脑上 `topbase-mcp` 的绝对路径，为它提供上述两个环境变量，然后重新连接。可以提问「列出我能访问的数据源，检查订单表，再按渠道统计已支付金额」。核对三个渠道数值后，再明确要求保存分析。

连接不上时，用同一地址和密钥运行 `status`，核对开发者模式、有效期和撤销状态。能连接但看不到表时，检查密钥持有者的读取权限。查询成功、保存失败时，检查 AI 新增分析开关和目标数据组的写入权限。
