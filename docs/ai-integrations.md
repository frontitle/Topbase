# AI、MCP 与 CLI

Topbase 提供一套共享的 AI 数据能力，并通过两个入口使用：

- `topbase-mcp`：供支持 MCP 的 AI 客户端自动发现和调用；
- `topbase-cli`：供本地调试、脚本和受控自动化使用。

两个入口都只调用 Topbase HTTP API，不直接连接业务数据库。API Key 会映射到创建它的用户，数据浏览、数据组和分析写入仍由服务端权限控制。

## 启用开发者模式

开发者模式默认关闭。管理员进入“管理后台 → 设置 → 开发者模式”后，可以统一配置：

- 是否允许 MCP、CLI 和 API Key 访问；
- 是否允许普通成员创建个人 Key；
- AI 是否可以新增可视化分析；
- 新 Key 的默认有效期；
- 单次查询最大返回行数；
- 提供给客户端使用的 Topbase 对外访问地址。

关闭开发者模式会在服务端立即阻断所有现有 API Key，但不会删除密钥。重新启用后，尚未过期且未撤销的密钥可以继续使用。API Key 只能访问 AI 数据工作流所需的白名单接口，不能调用成员管理、系统设置、数据库连接配置和删除内容等管理接口。

## 创建 API Key

进入“个人中心 → AI 与 API”，为每个客户端创建一个独立 API Key。完整密钥只显示一次，请立即保存到客户端的 Secret 或环境变量中。不要把密钥写入 Git、截图、聊天记录或命令历史。

密钥有效期由管理员在创建前统一设置。改变默认有效期不会修改已有密钥；到期密钥会在认证阶段被拒绝。

撤销 API Key 后，使用它的 MCP 和 CLI 会立即失去访问权限，但已经创建的分析不会被删除。

## 构建

```bash
make build
```

生成的相关程序为：

- `bin/topbase-mcp`
- `bin/topbase-cli`

Docker 镜像内也包含 `/app/topbase-mcp` 与 `/app/topbase-cli`。

## 配置 MCP

在 AI 客户端的 MCP 配置中添加一个 stdio 服务。不同客户端的配置文件位置不同，服务定义通常类似：

```json
{
  "mcpServers": {
    "topbase": {
      "command": "/absolute/path/to/topbase-mcp",
      "env": {
        "TOPBASE_URL": "https://topbase.example.com",
        "TOPBASE_API_KEY": "保存在本机 Secret 中的密钥"
      }
    }
  }
}
```

服务使用官方 Go SDK，同时兼容 MCP `2026-07-28` 的无状态协议和旧版初始化握手。stdio 只在标准输出写协议消息，运行日志写入标准错误。协议说明见 [MCP 规范](https://modelcontextprotocol.io/specification/2026-07-28) 与 [stdio transport](https://modelcontextprotocol.io/specification/2026-07-28/basic/transports/stdio)。

## AI 可用工具

| 工具 | 用途 | 是否修改 Topbase |
| --- | --- | --- |
| `topbase_status` | 验证地址、密钥和服务端查询限制 | 否 |
| `topbase_list_databases` | 列出当前用户可见的数据源 | 否 |
| `topbase_list_tables` | 实时列出表、数据库注释和字段 | 否 |
| `topbase_describe_table` | 读取字段注释、显示名、语义类型和关系 | 否 |
| `topbase_list_data_groups` | 选择可写入的分析分组 | 否 |
| `topbase_list_analyses` | 查找已有分析 | 否 |
| `topbase_query_data` | 使用 QueryIR 执行受限实时查询并返回可视化建议 | 否 |
| `topbase_run_analysis` | 运行已有可视化分析 | 否 |
| `topbase_create_analysis` | 在用户明确要求并确认后保存分析 | 是，新增内容 |

推荐的问答过程是：先确认数据源，再读取表和字段含义，然后生成 QueryIR、执行查询并解释结果。模型不得猜测字段；存在歧义时应向用户说明假设。创建分析与问答查询是两个独立动作，普通问答不会自动保存内容。

AI 查询不接受原生 SQL，并限制单次返回行数，默认最多 200 行。可以在启动 MCP 时使用 `--max-rows` 调低限制：

```bash
TOPBASE_URL=https://topbase.example.com \
TOPBASE_API_KEY="$TOPBASE_API_KEY" \
./bin/topbase-mcp --max-rows 100
```

## CLI

```bash
export TOPBASE_URL=https://topbase.example.com
export TOPBASE_API_KEY='只保存在当前安全环境中的密钥'

./bin/topbase-cli status
./bin/topbase-cli databases
./bin/topbase-cli tables pg_example
./bin/topbase-cli describe pg_example public orders
./bin/topbase-cli query --file query.json
./bin/topbase-cli create-analysis --name '每日订单趋势' --file query.json --confirm
```

`query.json` 使用 Topbase QueryIR，例如：

```json
{
  "version": 1,
  "source": {
    "database_id": "pg_example",
    "table": { "schema": "public", "name": "orders" }
  },
  "aggregations": [{ "fn": "sum", "field": "amount", "alias": "revenue" }],
  "group_by": [{ "field": "created_at", "temporal": "day" }],
  "order_by": [{ "field": "created_at", "dir": "asc" }],
  "limit": 100
}
```

CLI 的 `create-analysis` 必须显式传入 `--confirm`，用于区分只读查询和内容写入。

## 安全边界

- MCP/CLI 不读取或返回数据源连接密码、DSN、SSH 私钥和 Topbase 会话 Cookie；
- QueryIR 由 Topbase 再次校验并编译为只读查询；
- 单次返回行数同时受客户端参数和管理员设置的服务端硬限制保护，仍应避免向外部模型发送敏感字段；
- API Key 只开放 MCP/CLI 所需接口；关闭开发者模式、密钥到期或撤销都会立即停止访问；
- 建议为 AI 用户配置独立权限组，并按客户端分别创建、定期轮换 API Key；
- 生产环境必须使用 HTTPS，密钥只通过环境变量或 Secret 注入。
