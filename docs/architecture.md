# Topbase 工程架构

## 目标

Topbase 采用模块化单体起步。业务规模尚未要求微服务时，保持单二进制、清晰边界和可替换端口，比提前拆分网络服务更容易维护。后台调度可以通过相同应用服务以 `api` / `worker` 角色拆出，但领域模型不依赖部署方式。

## 依赖方向

```text
cmd/topbase（组合根）
  ├─ platform/httpapi（协议、鉴权入口、DTO）
  ├─ app（用例、事务、权限编排）
  ├─ adapters（数据库、密钥、飞书、LLM、时钟）
  └─ core（实体、QueryIR、不变量、端口）

platform ──> app ──> core
adapters ──────────> core
cmd 负责把实现注入端口
```

强制规则：

- `internal/core` 不得依赖 `app`、`adapters`、`platform` 或具体数据库驱动。
- `internal/app` 不得依赖 `adapters` 或 `platform`。
- HTTP handler 只负责解析/校验 DTO、调用用例、映射错误，不拼 SQL、不直接写应用库。
- 数据权限在应用服务入口统一求值；前端隐藏按钮不能替代服务端授权。
- 业务时间通过可注入时钟获得；ID 通过统一生成器获得，测试不得依赖真实时间碰撞。

组合根负责创建并注入具体适配器，协议层只通过稳定端口调用应用服务。新增基础设施时应保持这一依赖方向，避免把实现细节扩散到领域代码。

## 核心模块

| 模块 | 职责 | 不负责 |
| --- | --- | --- |
| `core/queryir` | 引擎无关查询 AST、校验、宏语义、下钻变换 | SQL 方言、网络执行、HTTP JSON |
| `core/catalog` | 数据源、schema/table/field 快照与连接端口 | 具体数据库系统表细节 |
| `app/query` | 展开模型/指标/分段、编译选择、执行、结果元数据 | 浏览器渲染 |
| `app/content` | 分析、仪表盘、数据组、修订、搜索等用例 | SQLite SQL、HTTP 状态码 |
| `app/warehouse` | 调度、物化策略、运行、watermark、血缘 | Cron 进程形态、飞书 HTTP |
| `adapters/appdb` | 应用库仓储与 migrations | 产品规则 |
| `platform/httpapi` | REST、Cookie/API Key 鉴权、静态 UI | 查询语义和权限策略本身 |

## 必须稳定的扩展点

### 数据库驱动

新增驱动不能修改 QueryIR 或 HTTP 表单硬编码。驱动注册项应声明：

- `Engine`、显示名、连接字段 schema、默认端口和敏感字段；
- 能力：schema、Join 类型、表达式函数、参数、取消、SSH、SSL、可写连接；
- `Connector`：测连、打开/关闭、健康；
- `MetadataProvider`：表、字段、主外键、字段值与指纹；
- `QueryCompiler` / `QueryExecutor`；
- 可选 `WarehouseWriter`。

连接设置持久化时保存 `engine + config version`。驱动升级必须提供配置迁移，禁止让 UI 猜旧字段。

当前第一阶段由 `adapters.EngineDefinitions` 声明引擎身份、默认端口、网络/SSH/账号能力和兼容产品，`SQLConnector` 统一承载连接生命周期，方言与元数据扫描按引擎分派。管理界面已有对应动态字段，但连接字段 schema 与全部 capability 仍需收敛到注册表，新增驱动时不得继续扩散条件分支。

### 可视化

每种图表通过注册表提供 `type`、字段角色约束、默认推断分数、设置 schema 和渲染器。分析只保存稳定的 `ChartSpec`，不保存 DOM/ECharts 临时对象。未知设置必须向前兼容保留。

### 前端功能组件

页面层按“业务编排 + 公共组件”拆分。查询编辑器、代码展示、筛选、表格、可视化、对话框和应用外壳不得在业务页面中复制实现。组件通过参数和回调连接业务，不直接持有 API 路由或领域持久化逻辑；页面负责把 QueryIR、SQL、ChartSpec 和权限结果传入组件。

公共组件采用 `tb-` 样式命名空间，并覆盖加载、空、成功、失败、无权限和窄屏状态。完整清单、接口和准入规则见 [前端功能组件](frontend-components.md)。

### 身份、通知、密钥与 AI

- `IdentityProvider`：统一不同身份来源，并把认证结果转换为内部 User/Session。
- `NotificationChannel`：站内、飞书、邮件、Webhook；统一测试、投递、重试和审计模型。
- `SecretStore`：开发文件、生产 KMS/Vault；API 只返回“已配置”，不回传密钥。
- `AIProvider`：只产生提案；授权、执行、保存和调度仍走正常应用服务。

## API 约定

- API 前缀保留 `/api`；破坏性修改必须升版本或提供迁移期。
- JSON 字段使用 `snake_case`；时间使用 RFC 3339 UTC，显示层按用户时区转换。
- 错误逐步统一为 `{ "error": { "code", "message", "field", "request_id", "details" } }`。
- 创建返回 `201`，删除成功返回 `204`；校验错误 `400`，未登录 `401`，无权限 `403`，不存在 `404`，冲突 `409`。
- 列表必须预留分页、排序和过滤，不得依赖无限数组。
- 每个写请求产生 request ID；日志不得记录密码、DSN、私钥、Cookie、API Key 或查询结果中的敏感值。

## 数据与迁移

- migration 只向前执行；已发布 migration 不修改，修复必须追加新 migration。
- 应用库事务边界位于应用服务，不跨越用户等待或长查询。
- 分析持久化 QueryIR 版本；读取旧版本先迁移到内存新版本，保存时再升级。
- 删除默认归档；永久删除前检查依赖并要求显式确认。
- 源库查询默认只读、可取消、有超时和行数上限；数仓写入使用独立能力与权限。

## 安全基线

- 拒绝默认；目录发现、运行、下载、公开链接和嵌入都做相同权限判断。
- Native SQL 的词法拦截只是保护层，不是权限模型；生产还需只读数据库账号、只读事务、超时和数据库侧资源限制。
- SSH 主机指纹可选；填写后必须严格匹配。生产建议引导管理员填写。
- 密码使用现代 KDF；会话 Cookie 使用 `HttpOnly`、`SameSite`，生产 HTTPS 下 `Secure`；状态变更请求实施 CSRF 防护。
- 公开令牌可撤销、不可枚举、有限流；嵌入 CSP 按明确来源配置，不长期使用通配符。

## 可观测性和可靠性

- 统一结构化日志字段：`request_id`、`user_id`、`database_id`、`query_id`、`duration_ms`、`status`，敏感信息脱敏。
- `/api/health` 只表示进程存活；另提供 readiness 检查应用库与关键依赖。
- 查询、同步、订阅和物化都使用运行记录，支持取消、超时、幂等键和失败重试。
- Prometheus 指标至少覆盖 HTTP、查询、连接池、同步、调度、通知和错误码。

## Definition of Done

一个功能只有在以下条件全部满足时才可合并：

1. 对应公开 Issue 或需求中的验收场景，行为边界清晰且可复现。
2. 领域规则位于 `core/app`，协议和具体实现位于外层。
3. 单元测试覆盖规则；HTTP 契约测试覆盖权限和错误；关键连接/查询有集成测试。
4. UI 覆盖 loading、empty、success、error、permission denied 和窄屏；核心动作有即时反馈。
5. migrations、配置、API 与用户文档同步更新。
6. `make check` 通过；没有新秘密、生成物或本地数据库进入版本控制。
