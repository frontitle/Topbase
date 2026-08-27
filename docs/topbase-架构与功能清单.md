# Topbase 架构与功能规格

> 本文是 Topbase 的**产品功能规格 + Go 实现指引**。
> 原则：**先把 Metabase 开源核心能力按操作级搬齐**（独立实现，不复制 AGPL/商业代码），再叠中国场景（飞书）、AI Chat、新可视化，以及「查询结果升级数仓」主路径。
> 对照清单：[`metabase-产品与功能结构拆解.md`](./metabase-产品与功能结构拆解.md)。本文不重复「Metabase 源码在哪」，只写 **Topbase 做什么、做到多细、落在哪个包/表/API、第几波交付**。

---

## 0. 怎么用这份文档

| 读者 | 用法 |
| --- | --- |
| 产品 | 按第 6 章功能域勾验收；每个功能点有交互、规则、边界 |
| 开发 | 按第 3 章包地图 + 第 4 章实体 + 第 7 章 QueryIR + 第 14 章 API 开工 |
| 排期 | 每个功能点有阶段 `W0–W5`；W0–W3 是 Metabase OSS 核心搬迁，W4 是 Topbase 增量，W5 是 OSS 长尾 |

标记约定：

- **OSS搬**：Metabase 开源就有，Topbase 必须做，行为对齐、实现自研
- **TB增量**：Topbase 相对 Metabase 多出来的（飞书、数仓升级按钮、AI 管调度、新 UI）
- **EE不做**：商业版能力，本文只留端口，不进 W0–W4
- **阶段**：`现有` 已有切片；`W0` 地基；`W1` 能问能存；`W2` 能分享；`W3` 语义与下钻；`W4` 数仓/飞书/AI 管理；`W5` OSS 长尾

中文产品名（对用户）：分析、仪表盘、数据组、模型、指标、分段、数据浏览、数仓表、计划任务。代码里仍用 `Question` `Dashboard` `Collection`。用户口中的「问题」已改为「分析」，避免理解成故障单；「集合」改为「数据组」。

---

## 1. 产品定义

Topbase 是面向中国团队的 **自助 BI + 可演进轻量数仓**。

先做到和 Metabase 开源版同等的日常分析能力：连库、同步元数据、可视化查询、SQL、保存分析、数据组、仪表盘、权限、告警、导出。在此之上：

1. **可视化层自研**：不复刻 Metabase UI，ChartSpec + 动效组件。
2. **飞书进主路径**：登录、部门组、卡片通知、分享。
3. **AI Chat 作为第三分析入口**：始终产出可审查 QueryIR/SQL，并可提案「写入数仓」。
4. **结果升级数仓**：任意已保存分析一键变成周期物化表，表回到目录再被查询。

### 1.1 明确不做（W0–W4）

| 类别 | 内容 |
| --- | --- |
| 许可 | 不复制 Metabase Clojure/TS；不进 `enterprise/` |
| EE 权限 | Block 表、行级安全、Impersonation、Database Routing、Tenants、下载行数档 |
| EE 身份 | SAML / JWT / OIDC / SCIM / MFA / 关密码登录 |
| EE 嵌入 | Modular SSO Embed、Full App、React SDK、白标 |
| EE 治理 | Official Collection、内容验证、Library、Snippet 文件夹权限、Serialization/Git |
| EE 分析 | Usage Analytics 仓、Security Center、MCP/Agent API 完整态 |
| 其它 | Python Transform 独立 runner、CDC、把 Topbase 做成通用 ETL |

W5 之后若做「类 EE」能力，必须独立设计。

### 1.2 设计原则

1. **查询是一等公民**。问题、仪表盘卡、AI 回复、计划任务、数仓表共用 QueryIR。
2. **前端不拼 SQL**。只提交 QueryIR 或 Native 模板；编译在 Go。
3. **写回与分析通道分离**。Explorer 只读；物化/Action 走 warehouse/action 服务。
4. **无权限即失败**。隐藏列不是安全措施。
5. **AI 只提案**。执行、保存、调度、物化需人确认（低风险改图可自动跑）。
6. **依赖向内**：`platform → app → core`，adapters 由 `cmd` 注入。

---

## 2. 现状差距

| 已有切片 | 相对 OSS 核心仍缺 |
| --- | --- |
| PG 连接、SSH、测连、连接池 | 应用库实体、多引擎、Sync/Scan/Fingerprint |
| 表发现、字段标注 | 完整语义类型、FK、格式、Model/Metric/Segment |
| 简易可视化查询、只读 SQL | QueryIR 全步骤、Notebook、参数、Join、表达式 |
| 内存 Schedule 字段 | 真调度、物化、告警、订阅 |
| demo AI、飞书入口 | 会话闭环、工具调用、飞书消息 |
| 管理向导 + 中文壳 | 集合/问题/仪表盘/权限/搜索全套页面 |

---

## 3. 架构（Go）

### 3.1 分层

```
工作台 UI / 飞书
    → HTTP + SSE
        → internal/app   用例、事务、鉴权编排
            → internal/core   实体、QueryIR、不变量、端口
                → adapters（源库 / 应用库 / 密钥 / 飞书 / LLM / 队列 / 数仓写入）
```

进程：W0–W3 单二进制 `cmd/topbase`（HTTP + 进程内 cron）。W4 起可 `-role=api|worker`。

### 3.2 包地图（开发必须按此放代码）

```
cmd/topbase/
internal/
  core/
    identity/          User Group Session ApiKey
    catalog/           Database Schema Table Field Feature
    queryir/           AST、校验、宏展开（Metric/Segment/Snippet）
    chartspec/         图类型与设置，不含像素
    content/           Question Dashboard Collection Model Metric …
    permission/        权限图求值（纯函数）
    warehouse/         Schedule Run MaterializedTable Lineage
    notify/            Channel Message 语义
  app/
    setup/ identity/ catalog/ sync/
    query/             编译执行取消缓存导出
    content/           问题仪表盘集合修订搜索
    viz/               ChartSpec 推断
    warehouse/ action/ ai/ notify/ settings/
  adapters/
    appdb/             pgx + sqlc
    secrets/
    source/postgres mysql clickhouse sqlite
    warehousepg/       物化 DDL/DML
    feishu/ llm/ mail/ cron/
  platform/
    httpapi/  jobs/  web/
migrations/
```

禁止：`core` import 驱动/HTTP；在 `httpapi` 里写权限或 SQL 拼接。

### 3.3 三种库角色

| 角色 | 存什么 | 落地 |
| --- | --- | --- |
| 应用库 | 用户权限内容 QueryIR 调度血缘 | Postgres + migrations；禁止 H2 |
| 源库 | 业务数据，分析默认只读 | 用户接入；W1 先 PG，W3 加 MySQL/CH |
| 数仓区 | 物化表 | W4：应用库 schema `warehouse`；之后可独立 PG/CH |

密钥：`secret_ref` 进应用库，明文进 secrets adapter（开发文件 0600，生产 KMS）。

---

## 4. 领域模型（表级，指引建表）

ID 一律 `text` 主键（`qst_…` 这类前缀），另有 `entity_id` 稳定对外 ID。时间全部 `timestamptz`。软删 `archived_at`。

### 4.1 身份

`users`：id, email, name, password_hash nullable, feishu_open_id unique, locale, theme, is_active, invited_at, last_login_at
`groups`：id, name, kind(`all_users`|`admins`|`custom`)
`group_members`：group_id, user_id
`sessions`：id, user_id, expires_at, user_agent, ip
`api_keys`：id, name, prefix, hash, group_id, created_by
`login_histories`：user_id, at, ip, user_agent, success
`settings`：key, value jsonb, 环境变量可覆盖

内置组：所有人、管理员。权限按组并集，更宽松优先。

### 4.2 数据源与元数据

`databases`：id, name, engine, secret_ref, ssh_secret_ref, timezone, sync_schedule, scan_mode, is_sample, is_warehouse, created_by
`schemas` / `tables` / `fields`：database_id, schema_name, name, display_name, description, visibility(`normal`|`detail`|`sensitive`), position, active
`fields` 另：data_type, semantic_type, fk_target_field_id, format jsonb, fingerprint jsonb, has_field_values, filter_widget(`list`|`search`|`input`), cast_to
`field_values`：field_id, values jsonb, updated_at
`table_annotations` 可并入 tables/fields，不必单独 json 文件

### 4.3 内容

`collections`：id, parent_id, name, personal_owner_user_id nullable, location_path
`questions`：id, collection_id nullable, dashboard_id nullable（只属于某仪表盘）, name, description, queryir jsonb, native_sql text, chartspec jsonb, query_type(`queryir`|`native`), database_id, archived_at, created_by
`dashboards`：id, collection_id, name, description, auto_refresh_seconds, archived_at
`dashboard_tabs`：dashboard_id, name, position
`dashboard_cards`：dashboard_id, tab_id, type(`question`|`heading`|`text`|`link`|`iframe`|`action`), question_id, config jsonb, layout jsonb（x,y,w,h）
`dashboard_filters`：dashboard_id, type, config jsonb, mappings jsonb
`models`：基于 question，额外 column_meta jsonb
`metrics`：database_id, table_id, aggregation jsonb, filters jsonb
`segments`：table_id, filters jsonb
`snippets`：name, content, database_id nullable
`timelines` / `timeline_events`：collection_id, timestamp, title, description, icon
`bookmarks`：user_id, target_type, target_id
`revisions`：target_type, target_id, actor_id, diff jsonb, created_at
`comments`：document_id, parent_id, body, author_id
`documents`：collection_id, body jsonb（块列表）

规则：一个问题同时只能在一个集合，或只属于一张仪表盘。个人集合问题不能挂到公共仪表盘。

### 4.4 权限

`permission_graph`：revision, data_graph jsonb, collection_graph jsonb
求值纯函数：`permission.Evaluate(user, db/table, action) → allow`

OSS 对齐动作：

- 数据：`view`；`create_query` = `builder` | `builder_and_native` | `none`（可到表）
- 集合：`none` | `view` | `curate`
- 数仓物化：`materialize`（W4，按库）
- 管理元数据：W3 起 Admin 或「可管元数据」组（OSS 里管元数据在 EE，Topbase 给 Admin + 显式授权，避免卡住建模）

Native 规则（对齐 OSS 产品语义）：若用户对库内任表没有 `view`，禁止对该库跑 Native（Topbase 同样不解析 SQL 文本）。

### 4.5 调度 / 数仓 / 通知

`schedules`：question_id, cron, timezone 默认 Asia/Shanghai, materialize_to, strategy(`replace`|`truncate_insert`|`incremental`|`none`), watermark_field, notify_channel_ids, enabled, pinned_revision
`runs`：schedule_id, started_at, finished_at, status, row_count, error, sql_compiled
`materialized_tables`：database_id, schema, name, schedule_id, question_id, last_run_at, watermark
`lineage_edges`：from_type, from_id, to_type, to_id
`channels`：kind(`feishu_user`|`feishu_chat`|`email`|`webhook`), config jsonb
`alerts`：question_id, kind(`results`|`goal`|`progress`), schedule, channel_ids, once
`subscriptions`：dashboard_id, cron, channel_ids, format
`conversations` / `messages` / `tool_calls`：AI 审计

---

## 5. 产品信息架构（页面，开发按路由实现）

视觉自研，**信息架构对齐 OSS 核心**，另加数仓与飞书。

### 5.1 全局壳

- 左：首页、数据组、分析、数据浏览、仪表盘、数仓、模型、搜索、回收站
- 顶：+新建、搜索、命令面板、AI Chat、网格（管理 / 账户）
- +新建：分析、SQL、仪表盘、模型、指标、数据组、文档（W5）
- 命令面板：跳转、搜内容、切主题、打开 AI

### 5.2 路由

| 路径 | 产品面 | 阶段 |
| --- | --- | --- |
| `/setup` | 语言、管理员、站点名、连第一个库或跳过 | W0 |
| `/auth/login` `/auth/feishu/*` | 密码 / 飞书 | W0 |
| `/auth/forgot_password` `/auth/reset_password/:token` | 找回密码 | W1 |
| `/` | 首页：最近、书签；可配默认仪表盘 | W1 |
| `/search` | 全局搜索 | W2 |
| `/browse/databases` `…/:db/schema/:s` | 浏览库表 | 现有→W1 |
| `/browse/models` `/browse/metrics` | 浏览模型/指标 | W3 |
| `/table/:id` `/table/:id/detail/:rowId` | 表查询 / 行详情 | W3 |
| `/questions/` `/questions/:id/` | 分析列表 / 详情（可筛选结果表、重命名、移入数据组、归档） | W1 |
| `/collections/` `/collections/:id/` | 数据组列表 / 详情（子数据组、分析、重命名、空数据组删除） | W1 |
| `/question` `/question/:id` `/question/:id/notebook` | 构建器入口（代码实体仍为 Question） | W1 |
| `/question/ask` | AI 分析（同构建器结果） | W1 |
| `/model/new` `/model/:id/{query,columns,metadata}` | 模型 | W3 |
| `/collection/:id` | 数据组别名；当前实现走 `/collections/:id/` | W1 |
| `/dashboard/:id` | 仪表盘 | W2 |
| `/warehouse` `/warehouse/:tableId` | 数仓表目录与详情 | W4 |
| `/document/:id` | 文档 | W5 |
| `/auto/dashboard/*` | X-ray | W5 |
| `/admin/databases` | 数据源 | 现有 |
| `/admin/datamodel` | 表元数据、分段 | W3 |
| `/admin/people` `/admin/permissions` | 人与权限 | W1–W2 |
| `/admin/settings` | 站点、邮件、飞书、AI、时区、外观、缓存、公开分享、地图、上传 | W2 起分批 |
| `/admin/tasks` `/admin/alerts` | 任务与告警 | W2 |
| `/trash` `/unauthorized` `/unsubscribe` | 回收站 / 无权限 / 退订 | W2 |
| `/public/question/:token` `/public/dashboard/:token` | 公开 | W5 |

### 5.3 管理分区

Databases、数据模型、人员、权限、设置、性能（缓存/模型持久化）、任务、告警、帮助。不做 Embedding Hub / Security Center。

---

## 6. 功能域详解（产品 + 开发）

每个功能点格式：**做什么 / 规则 / 落点 / 阶段**。落点写包名。

---

### 6.1 安装与 Setup  `OSS搬`

- **单二进制运行**：`go run ./cmd/topbase`，默认 `:8080`。Docker 随后。落点：`cmd/topbase`。`现有/W0`
- **应用库**：Postgres 必选；首次启动跑 migrations。无库则报错并给出连接环境变量 `TOPBASE_APP_DB`。落点：`adapters/appdb`。`W0`
- **Setup 向导**：①语言 ②管理员 ③站点名 ④是否匿名统计（默认关）⑤连库或跳过。未 setup 访问其它页重定向 `/setup`。落点：`app/setup`。`W0`
- **示例库**：可内置 SQLite 或只读演示 PG 快照；可删、可「恢复示例库」。`W1`
- **示例集合 + 示例仪表盘**（电商洞察一类）。`W2`
- **Onboarding 页** `/getting-started`。`W2`
- **落地页设置**：默认首页 / 指定仪表盘。`W2`

---

### 6.2 认证与会话  `OSS搬` + `TB增量`

**本地账号 OSS搬**

- 邮箱密码登录；密码哈希（argon2id）；复杂度可配。`app/identity` `W0`
- 忘记密码邮件重置。`W1`（无 SMTP 时管理后台直接重置）
- 邀请链接；停用/重新激活；会话过期；过期清理任务。`W1`
- 登录历史；新设备通知（有邮件渠道时）。`W2`
- 个人：姓名、邮箱、语言、深浅色、改密码。`W1`
- 个人集合自动创建。`W1`
- 书签仅自己可见。`W2`

**飞书 TB增量**

- OAuth 授权码；`open_id` 绑定 User；拉姓名头像。未配置时 `/auth/feishu/login` 返回明确错误（保持现状）。`W0`
- 部门 → Group 只读同步，权限仍在 Topbase 授。`W4`
- 飞书未登录用户可用密码（私有化兜底）。

**机器身份 OSS搬**

- API Key：名称、前缀展示、绑定组、轮换、删除；请求头鉴权。`W2`
- Session HttpOnly Cookie。`W0`

**EE不做**：SAML/JWT/OIDC/SCIM/MFA/强制关密码。

---

### 6.3 数据源  `OSS搬`

#### 引擎优先级

| 引擎 | 阶段 | 备注 |
| --- | --- | --- |
| PostgreSQL | 现有 | JSON 展开 W3；Action W5 |
| MySQL / MariaDB | W3 | |
| ClickHouse | W3 | |
| SQLite | W1 | 示例库 |
| 其余（SQL Server 等） | W5+ | 同一 `source.Driver` 端口 |

连接字段：name, engine, host, port, database, user, password, ssl_mode, dsn, SSH（密码/私钥/指纹）。测连不写 catalog。删库前检查引用（问题/仪表盘/调度），确认后级联归档。只读账号默认；物化另配数仓目标。

落点：`app/catalog` + `adapters/source/*` + `adapters/ssh`（现有）。

#### Sync / Scan / Fingerprint  `OSS搬` 必须做细

| 作业 | 行为 | 默认 | 用途 |
| --- | --- | --- | --- |
| Sync | 拉 schema/表/列/PK/FK，停用已删表 | 每小时 | 浏览、构建器、隐式 Join |
| Scan | 抽样列值写入 `field_values` | 每天；仅 14 天内用过的字段 | 筛选下拉 |
| Fingerprint | 前 1 万行：空值率、基数、min/max/分位 | 随 Sync | 自动选图、分箱、X-ray |

可配：按库小时/天；扫描「定期 / 仅新增筛选器 / 从不」。手动：Sync 库、Re-scan 表/字段、清空缓存值。任务进 `/admin/tasks`。
落点：`app/sync`。`W1` Sync 手动+首次连接；`W3` 定时 Scan/Fingerprint。

#### 驱动 Feature  `OSS搬`

每个 Driver 声明能力，构建器按此裁剪（与拆解文档 5.3.4 同名语义，Go 用字符串常量）：

查询：`basic_aggregations` `stddev` `percentile` `expressions` `expression_aggregations` `nested_queries` `native_parameters` `binning` `left_join` `right_join` `inner_join` `full_join` `regex` `temporal_extract` `date_arithmetics` `now` `convert_timezone` `datetime_diff` `window_cumulative` `window_offset` `advanced_math` `case_sensitive_string_filter`

结构：`schemas` `nested_field_columns` `metadata_key_constraints` `set_timezone` `uuid` `split_part`

回写：`actions` `uploads` `persist_models` `transforms_table` `create_or_replace_table`

编译前 `queryir.CheckFeatures`；不支持则 API 返回 `feature_unsupported`，UI 隐藏入口。

---

### 6.4 元数据与语义层  `OSS搬`

#### 物理类型

Numeric / Temporal / Text / TextLike / Boolean / Collection(JSON)。数组仅空/非空过滤。表级 Cast（Text→Date）写在 field.cast_to；查询内再用表达式。

#### 语义类型（必须全做，影响格式/选图/过滤/下钻）

- 任意：EntityKey, ForeignKey
- 数值：Quantity, Score, Percentage, Currency, Discount, Income, Latitude, Longitude, Category
- 时间：CreationDate/Time/Timestamp, JoinedDate/Time/Timestamp, Birthday
- 文本：EntityName, Email, URL, ImageURL, AvatarURL, Category, Name, Title, Description, Product, Source, City, State, Country, ZipCode
- JSON：FieldContainingJSON → unfolding（PG W3）

FK 未配置：禁止隐式 Join、仪表盘联动筛选、关联行下钻。

#### 表/字段编辑（Admin 数据模型）

显示名、描述、隐藏表、排序；字段显隐（普通/仅详情/敏感）；过滤控件类型；格式（小数、分隔符、货币、日期、百分号）；友好表名开关（`some_name`→`Some Name`）。
落点：`app/catalog`。`W1` 显示名/语义子集；`W3` 全集+格式+FK。

#### 模型 Model  `OSS搬` `W3`

- 从 QueryIR 或 Native 创建；列改名/语义/格式/描述
- Native 模型不设语义则禁止「改查询」类下钻
- 可作其它问题数据源；Native 用卡片引用
- 详情：查询 / 列 / 元数据三页
- 模型持久化：定时写成表（与数仓物化共用 warehouse 执行器，入口在模型设置）`W4`

#### 指标 Metric  `OSS搬` `W3`

聚合+可选过滤；构建器可选为数据源；浏览指标。

#### 分段 Segment  `OSS搬` `W3`

表级命名过滤；出现在过滤下拉顶部；有修订。

#### 术语 Glossary  `OSS搬` `W3`

给人看，也注入 AI 上下文。

#### Measure  `W5`（可先用 Metric 代替）

EE Library / ERD / Official：不做。

---

### 6.5 浏览与组织  `OSS搬`

**首页 W1**：最近查看、书签、新建入口。W2 可设默认仪表盘。

**浏览 W1**：库→schema→表。W3：模型、指标、表行详情、永久链接。

**数据参考 W3**：库/表/字段描述、类型、样例，链到 Glossary。

**搜索 W2**：分析、仪表盘、数据组、表、库；W3 加模型/指标；命令面板复用。

**数据组 W1**：列表页、嵌套、移动分析、新建子数据组、空数据组删除、个人数据组「我的数据组」。W2：批量把分析挪进仪表盘、归档/置顶/权限。W5：数据组时间线。

**修订 W2**：问题/仪表盘/分段；谁何时改；可回看。

**时间线 Events W5**：叠到折线/面积图。

**X-ray W5**：对表/字段/模型生成自动仪表盘；设置可关；模板自研（交易/用户/事件），不搬 Metabase yaml。

**回收站 W2**。隐藏列 ≠ 安全（产品文案必须写在构建器帮助里）。

---

### 6.6 可视化查询构建器  `OSS搬` 核心心脏

UI 不复刻 Notebook 外观，**步骤与能力必须对齐**。前端只编辑 QueryIR。落点：`core/queryir` `app/query` `web` 构建器。`W1` 最小步骤；`W3` Join/表达式/分箱/多阶段。

#### 步骤

1. **选数据**：表 / 模型 / 指标 / 已保存问题 / 数仓表；搜索；预览；勾选输出列（不勾仍可过滤）。
2. **Join**：同库；inner/left/right/full 按 feature；多条件；FK 隐式 Join；Join 模型/问题（嵌套查询）。`W3`
3. **自定义列**：`+ - * /` 与函数；Summarize 前后都可。`W3`
4. **过滤**：见下表；Summarize 后过滤 = HAVING；表达式 `AND/OR`；分段快捷。药丸编辑/删除。
5. **汇总/分组**：内置聚合 + 表达式聚合；时间分桶；数值分箱（自动/固定宽/固定箱数）；多维度。
6. **排序**：多列升降序；累计聚合排序影响计算。
7. **Limit**：只能最后；先算完再截。浏览器展示上限：未聚合 2000、聚合 10000。

每步预览 10 行。可查看编译 SQL（需 builder_and_native）。**转 SQL 单向**。保存到集合或仪表盘。另存/复制/移动/归档。AI 可生成/改写同一 QueryIR。

#### 过滤按类型

| 列类型 | 运算符 |
| --- | --- |
| 数值 | eq neq gt gte lt lte between is_null not_null |
| 文本/分类 | is is_not contains not_contains starts_with ends_with is_empty not_empty；控件 list/search/input |
| 日期 | 具体日、范围、相对（含 starting_from）、月年、季年 |
| 经纬度 | 数值 + inside 框 |
| JSON | 仅空/非空（除非 unfolding） |
| 布尔 | true/false/empty |

#### 聚合（W1 子集加粗）

**Count, Sum, Average, Min, Max, Distinct**；W3：CountIf, SumIf, DistinctIf, Median, Percentile, Share, StdDev, Variance, CumulativeSum, CumulativeCount。

#### 自动选图（ChartSpec 推断，可改）

| 分组 | 默认图 |
| --- | --- |
| 文本/分类 | bar |
| 时间 | line |
| 数值已分箱 | bar |
| 数值未分箱 / 无聚合 | table |
| 布尔 | bar |
| 经纬度已分箱 | map_grid |
| 经纬度未分箱 | map_pin |
| Country | map_region_world |
| 中国省/市（语义 State/City 且 locale=zh） | map_region_cn `TB增量 W4` |

#### 表达式函数（W3 按 driver feature 裁剪）

与拆解 5.6.8 对齐，Go 实现同名：

- 逻辑：between, case, if, coalesce, in, notIn, isNull, notNull
- 数学：abs, ceil, exp, floor, integer, log, power, round, sqrt
- 字符串：concat, contains, doesNotContain, startsWith, endsWith, domain, host, path, subdomain, isEmpty, notEmpty, trim/lTrim/rTrim, lower, upper, length, regexExtract, replace, splitPart, substring, text, float, date, datetime
- 日期：convertTimezone, datetimeAdd/Subtract/Diff, day, dayName, hour, minute, second, month, monthName, quarter, quarterName, week, weekday, year, now, today, interval, relativeDateTime, timeSpan
- 窗口：offset, cumulativeCount, cumulativeSum

---

### 6.7 Native SQL  `OSS搬`

- 选库；高亮；运行；跑选中片段；Ctrl/⌘+Enter；格式化。`W1`
- 保存、下载、转模型、入仪表盘。
- 引用问题/模型：`{{#question_id}}`。`W3`
- Snippet：`{{snippet:name}}`，可嵌套。`W5`
- AI：生成、行内改、修报错。`W1` 生成；`W3` 修错。
- Postgres JSON `?` 在文档中说明与占位符冲突（Go 用 `$n` 绑定，产品层仍避免用户单 `?`）。

#### 参数类型 `W3`

| 类型 | 行为 |
| --- | --- |
| Field filter | 绑真实字段，智能控件，可接仪表盘 |
| Basic | text/number/date |
| Optional | `[[ … {{x}} ]]` 无值省略 |
| Time grouping | 改粒度不删点 |
| Table variable | 选表，配 Snippet |
| Card reference | 另一问题当 CTE |

配置：控件类型、标签、list/search/input、单/多值、默认值、别名表。URL `?var=`。无变量则不能接仪表盘筛选。

#### Native 边界（必须写进实现与帮助）

不解析 SQL ⇒ 无「改查询」下钻；库内任表无 view ⇒ 禁整库 Native；必须先保存才有基于结果的下钻。

---

### 6.8 查询管道  `OSS搬` 引擎内核

```
鉴权 + 数据权限
  → QueryIR 规范化 / Native 模板校验
  → 展开 Metric / Segment / Snippet / 卡片引用
  → 解析表、字段、Join、隐式 Join
  → 代入参数
  → 时间分桶 / 分箱 / 累计 / Pivot 嵌套
  → CheckFeatures
  → 编译方言 SQL
  → 缓存
  → 只读事务执行（可 cancel；客户端断开即 cancel）
  → 时区、格式化、大整数、行截断、结果元数据
  → JSON 或流式 CSV/XLSX
```

中间件拆文件，对标拆解 5.8，放 `app/query/pipeline/`。约束：超时（默认 30s 可配）、展示行上限、下载上限（OSS 先统一大上限，不做 EE 档位）、报表时区。

物化 **不走** 此只读管道，走 `app/warehouse`。

---

### 6.9 可视化组件  `OSS搬` + `TB增量` 新渲染

后端只出 ChartSpec。前端自研动效。图表清单对齐 OSS 注册表：

| type | 名称 | 阶段 |
| --- | --- | --- |
| table | 表格 | W1 |
| scalar / smart_scalar | 数字 / 带对比 | W1 / W3 |
| line area bar row | 折面柱条 | W1 |
| pie | 饼/环 | W1 |
| progress gauge | 进度/仪表 | W2 |
| combo waterfall | 组合/瀑布 | W3 |
| scatter boxplot | 散点/箱线 | W3 |
| funnel sankey treemap | 漏斗/桑基/树图 | W5 |
| pivot | 透视 | W3 |
| map_pin map_grid map_region | 地图 | W4（中国行政区优先） |
| list object_detail | 列表/行详情 | W3 |
| custom | 不做 W0–W4 | |

**表格细节必须做**：列重排隐藏固定、条件格式、Display as（文本/链接/email/图片/头像/进度）、列头动作（过滤排序分布求和平均去重按时间求和提取日期/URL）、单元格下钻、轻量透视、行详情。`W1` 基础表；`W3` 条件格式与列头动作。

**图设置**：轴、系列显隐顺序、颜色、堆叠、双轴、趋势线、目标线、最大分类数、tooltip、图例开关系列。时序拖拽框选 = 过滤。Timeline 叠加 W5。

**动效 TB增量**：查询骨架、结果切入、筛选 morph、仪表盘入场、问题升级数仓徽标。禁止无意义循环动画。静态 PNG 供飞书/邮件：无头渲染适配器，不绑 Graal。

#### 下钻  `OSS搬` `W3`

1. 改结果：Filter / Sort / Distribution（QueryIR 与已保存 Native 均可）
2. 改查询：看这些记录、按时间/地理/分类拆、Zoom in、更细时间（仅 QueryIR）

需 `create_query`。生成新问题不改原问题。仪表盘可覆盖为点击行为。

---

### 6.10 仪表盘  `OSS搬` `W2` 起必须按细项做齐

**卡片**：问题卡（集合或仅本仪表盘）；Heading；Text Markdown（`{{var}}`、`[[可选]]`）；Link（内搜或外链变量）；Iframe（域名白名单）；Action 按钮 W5。

规则：已挂其它仪表盘的问题不能再挂；要复用先存集合。个人集合问题只能挂个人仪表盘。

**布局**：网格拖拽缩放；多 Tab；复制 Tab；跨 Tab 移卡；卡级可视化；加系列；自动刷新；全屏；导出 PDF（W5 可先 PNG 拼接）；复制整板；移动；归档。

**筛选器**

- 过滤：日期（月年/季年/单日/范围/相对/全选项）、地点（市/省/邮编/国家 + is/contains…）、ID、数值、文本分类、布尔
- 参数：时间分组
- 挂载：仪表盘级 / 标题级 / 卡级（语义同拆解 5.10.3）
- 映射到卡字段或 Native 变量；默认值；必填；可选值来自 scan 或自定义；联动筛选依赖 FK `W3`

**点击行为**：①默认下钻 ②跳转仪表盘/问题/URL（传列值）③更新本板筛选。Native 卡无①。

**订阅**：飞书/邮件/Webhook；cron；附件；退订；损坏通知。`W4` 飞书；`W5` 邮件完整模板。

---

### 6.11 文档  `OSS搬` `W5`

富文本；`/` 插入图、链接、问 AI；`@` 提及；插入问题为拷贝；文档内新建问题删除不可进回收站；一行最多 3 块；评论。

---

### 6.12 Actions  `OSS搬` `W5`

仅 PG/MySQL；连接开启 actions；库账号可写；挂在模型上。Basic 增删改；Custom 参数 SQL。入口：模型页、仪表盘按钮、公开表单。创建需 Native 权限；运行需能看模型/仪表盘。无撤销；不改模型定义只改底层表；公开仪表盘不可跑 Action。

表编辑 EE：不做。

---

### 6.13 导出  `OSS搬`

问题：CSV / XLSX / JSON / PNG；Formatted / Unformatted；透视导出扁平表而非 Excel PivotTable。仪表盘 PDF W5。文档卡单独下载 W5。OSS 不做下载档位。流式写出，禁止全表进内存。落点：`app/query` streaming。`W2` CSV；`W3` XLSX/PNG。

---

### 6.14 告警  `OSS搬` `W2`

对象=问题。类型：有结果；目标线上穿/下穿（需图开 goal）；进度条达/低于目标。日程 cron。渠道飞书/邮件/Webhook。一次性发完自删。Send now。Admin 集中管理。退订页。

---

### 6.15 通知渠道  `OSS搬` + `TB增量`

- **飞书**（主渠道）：用户/群卡片，缩略图+链接。`W4`
- **邮件**：SMTP、发件人、测试、邀请/重置/告警/订阅模板。`W2` 能发；`W5` 全模板
- **Webhook**：URL+Header，Admin 可配。`W5`

---

### 6.16 权限  `OSS搬`（不要用三角色糊弄过去）

组 × 资源图，带 revision。

**数据（库/schema/表）**

- View：能看基于该数据的问题结果（OSS 无 Block）
- Create queries：无 / 仅构建器 / 构建器+Native；可 Granular 到表
- 任表无 view ⇒ 整库禁 Native

**集合**：无 / 查看 / 策展（改移删置顶建子集合）。仪表盘可见但问题在无权限集合 ⇒ 该卡占位。仅 Admin 改集合权限。`W1` 库级；`W2` 表级 granular + 集合。

**谁能建告警/订阅**：集合 view 或 curate。Admin 可关某组该能力（设置项，不必做 EE 应用权限矩阵）。

**管元数据/管连接**：Admin；W3 可把「编辑数据模型」授给指定组（避免全员找 Admin）。删库仅 Admin。

EE 行级/模拟/路由/租户：不做。

---

### 6.17 公开分享  `OSS搬` `W5`

公开问题/仪表盘链接与 iframe；总开关；无鉴权。Guest 签名 embed（锁定参数、无下钻无构建器）可 W5。SSO 嵌入 EE：不做。

---

### 6.18 数仓升级  `TB增量` `W4`（OSS Transform 的产品化主路径）

Metabase Transform 是旁路；Topbase **结果页主按钮「写入数仓」**。

路径：保存问题 → 周期、目标表、策略、通知 → Run → 注册 MaterializedTable → 浏览里带「数仓」徽标 → 可被新 QueryIR 引用。

策略：replace / truncate_insert / incremental / create_only。默认写 warehouse schema，前缀 `wh_`。禁止 AI 直接 DDL。同一 schedule 同时只跑一个。失败重试+飞书。手动立即运行。血缘自动写。容量与保留策略。W5：按血缘建议依赖顺序，不强制画 DAG。

模型持久化复用同一执行器。

---

### 6.19 AI Chat  `TB增量`（OSS Metabot 多为 EE，这里当核心做）

与构建器、SQL 并列的第三入口。中文。先搜已有问题再生成 QueryIR。

| 工具 | 确认 |
| --- | --- |
| describe_catalog | 否 |
| draft_query → 展示 QueryIR/SQL → run_query | 本轮可「允许自动跑只读」 |
| patch_query / patch_chart | 低风险自动 |
| save_question / add_to_dashboard | 是 |
| propose_schedule（写入数仓） | 是，展示 cron/表/策略 |
| list_runs | 否 |
| notify_feishu | 是 |
| fix_native_sql | 展示 diff 后确认 |

国内模型适配器。默认不发送样例行。审计 conversation/tool_call。禁止任意 SQL 字符串绕过闸门。`W1` 分析；`W4` 调度/飞书工具。

不做完整 MCP/Agent API（W5 以后独立设计）。

---

### 6.20 性能  `OSS搬`

实例级查询缓存；按库策略。`W3`。抢先缓存 EE：不做。大结果流式。报表时区。

---

### 6.21 CSV 上传  `OSS搬` `W5`

指定目标库/schema；建表；可选自增 `_tb_row_id`；追加或替换。

---

### 6.22 本地化与外观  `OSS搬` + `TB增量`

默认 zh-CN、Asia/Shanghai、中国数字日期。个人深浅色。站点名、Logo、favicon。友好表名开关。自定义地图 GeoJSON W4。白标/内容翻译 EE：不做。

---

### 6.23 监控  `OSS搬`

出错的问题、告警管理、后台任务、日志、模型/数仓刷新日志、健康检查。依赖诊断随血缘 W4。Usage Analytics EE：不做；W4 只做审计日志表（登录/查询/物化/AI）。

---

## 7. QueryIR 规格（开发契约）

`version: 1`。Go 类型在 `internal/core/queryir`。JSON 存 `questions.queryir`。

### 7.1 顶层

```
Query {
  version
  source: TableRef | QuestionRef | ModelRef | MetricRef | WarehouseTableRef
  joins?: Join[]
  expressions?: { alias: Expr }[]
  fields?: FieldRef[]          // 输出列，空=全部
  filters?: Filter             // 可嵌套 and/or
  aggregations?: Aggregation[]
  group_by?: Breakout[]
  having?: Filter
  order_by?: Order[]
  limit?: int
  parameters?: Parameter[]     // 仪表盘/控件
  stages?: Query[]             // 多阶段，后一阶段 source=前一阶段
}
```

W1 实现：source=table、fields、filters、aggregations、group_by、order_by、limit。
W3：joins、expressions、having、stages、parameters、嵌套 source。

### 7.2 与 Native 的关系

`questions.query_type=native` 时 `native_sql` + `parameters`，`queryir` 为空。转 SQL 后不可逆。

### 7.3 编译接口

```go
type Compiler interface {
    Compile(ctx context.Context, q Query, dialect Dialect) (SQL, Args, error)
}
type Executor interface {
    Execute(ctx context.Context, databaseID string, sql SQL, args Args) (ResultSet, error)
}
```

禁止字符串拼接用户输入。

---

## 8. ChartSpec 规格

```
ChartSpec { type, encodings: { x, y[], series, size, group }, settings, animation }
```

`settings` 按类型：table.columns、goal.value、stack、axis、legend、tooltip、conditional_formats、display_as。
推断函数 `viz.Infer(resultMeta, query) ChartSpec`；用户覆盖深合并。前端禁止直接写 ECharts option。

---

## 9. 权限求值伪代码

```
allow(user, table, view) =
  any group in user.groups:
    data_graph[group][db][schema][table].view == true
    or 上级 schema/db 为 can_view 且未在子级关掉

allow_native(user, db) =
  allow(user, every table in db, view) AND create_query == builder_and_native

allow_collection(user, coll, curate) =
  any group: collection_graph[group][coll] >= curate
  个人集合: owner or admin
```

改图必须带测试：多组并集、个人集合、Native 整库禁、仪表盘卡占位。

---

## 10. HTTP API（按资源，全部要有）

错误：`{"error":{"code","message"}}`。W0 兼容旧 `{"error":"string"}`。

鉴权：Cookie 或 `Authorization: Bearer <api_key>`。

### 10.1 系统与身份

| 方法 | 路径 | 阶段 |
| --- | --- | --- |
| GET | `/api/health` | 现有 |
| POST | `/api/setup` | W0 |
| POST | `/api/session` DELETE `/api/session` | W0 |
| GET | `/auth/feishu/login` `/auth/feishu/callback` | W0 |
| POST | `/api/session/forgot_password` `/api/session/reset_password` | W1 |
| CRUD | `/api/user` `/api/user/current` | W1 |
| CRUD | `/api/group` `/api/group/:id/members` | W1 |
| CRUD | `/api/api-key` | W2 |
| GET | `/api/login-history` | W2 |
| GET/PUT | `/api/setting` | W1 |

### 10.2 数据源与元数据

| 方法 | 路径 | 阶段 |
| --- | --- | --- |
| GET/POST | `/api/databases` | 现有 |
| POST | `/api/databases/test` | 现有 |
| PATCH/DELETE | `/api/databases/:id` | W1 |
| POST | `/api/databases/:id/sync` `:id/rescan` | W1 |
| GET | `/api/databases/:id/tables` | 现有 |
| GET/PUT | `/api/table/:id` `/api/field/:id` | W1 |
| GET/PUT | `.../annotation` | 现有兼容→迁到 table/field |
| GET | `/api/database/:id/idfields` | W3 FK |

### 10.3 查询与内容

| 方法 | 路径 | 阶段 |
| --- | --- | --- |
| POST | `/api/queries/run` Native | 现有 |
| POST | `/api/dataset` QueryIR 执行（主入口） | W1 |
| POST | `/api/queries/:id/cancel` | W1 |
| POST | `/api/dataset/export` | W2 |
| CRUD | `/api/question` `/api/card` 二选一，建议 `/api/questions` | W1 |
| POST | `/api/questions/:id/convert-to-sql` | W1 |
| CRUD | `/api/dashboard` `/api/dashboard/:id/cards` `:id/filters` | W2 |
| CRUD | `/api/collection` | W1 |
| GET | `/api/search` | W2 |
| CRUD | `/api/bookmark` | W2 |
| GET | `/api/revision` | W2 |
| CRUD | `/api/model` `/api/metric` `/api/segment` `/api/snippet` `/api/glossary` | W3 |
| CRUD | `/api/timeline` `/api/timeline-event` | W5 |
| CRUD | `/api/document` `/api/comment` | W5 |
| GET | `/api/automagic-dashboard` | W5 |

### 10.4 权限 / 通知 / 数仓 / AI

| 方法 | 路径 | 阶段 |
| --- | --- | --- |
| GET/PUT | `/api/permissions/graph` | W2 |
| CRUD | `/api/alert` `/api/subscription` `/api/channel` | W2–W4 |
| CRUD | `/api/schedules` POST `:id/run` GET `/api/runs` | 现有→W4 持久化执行 |
| GET | `/api/warehouse/tables` `/api/lineage/:type/:id` | W4 |
| POST | `/api/ai/chat` SSE | 现有→W1 |
| POST | `/api/action` | W5 |
| POST | `/api/upload` | W5 |
| GET | `/api/public/question/:token` `/api/public/dashboard/:token` | W5 |
| GET | `/api/task` | W1 |
| GET | `/api/docs` OpenAPI | W2 |

现有 `POST /api/databases/{id}/visual-query` W1 改为调用 `/api/dataset`，保留一层兼容。

---

## 11. 后台任务

| 任务 | 阶段 |
| --- | --- |
| Session cleanup | W1 |
| Sync / Scan / Fingerprint | W1/W3 |
| Field values 14 天过期 | W3 |
| Query cache 淘汰 | W3 |
| Alert / Subscription 投递 | W2/W4 |
| Schedule 物化 | W4 |
| 模型持久化刷新 | W4 |
| 飞书部门同步 | W4 |

---

## 12. 前端模块（自研，按产品域拆包）

不要按 Metabase 目录抄。建议：

`web/shell` `auth` `setup` `browse` `notebook` `sql-editor` `result` `charts` `dashboard` `collection` `admin` `ai-chat` `warehouse` `search`

`charts` 只消费 ChartSpec。Admin 与工作台共用 `design/tokens.css`。W1 可用 Vite 打进 `embed`，保持单二进制。

---

## 13. 阶段交付（核心先搬）

顺序固定：**先 OSS 核心，再 TB 增量**。不能先做数仓把仪表盘/权限留空。

### W0 地基  `现有补齐`

应用库 migrations；Setup；密码+飞书会话；Cookie；health；catalog 迁出 json；密钥分离。
验收：新库启动向导后能登录，数据源进 Postgres。

### W1 能问能存  **OSS 核心第一刀**

组（所有人/管理员/自定义）+ 库级 view/create_query；集合 CRUD+个人集合+权限；QueryIR W1 子集编译（PG）；构建器步骤 1/4/5/6/7；Native 无参数；问题保存；表/折/柱/面/饼/KPI；查看编译 SQL；转 SQL；手动 Sync；AI 分析出可审查 SQL/QueryIR；查询取消。
验收：飞书登录 → 连 PG → 构建器或 AI 出日聚合图 → 保存进集合 → 别人同组能打开。

### W2 能分享  **OSS 核心第二刀**

仪表盘网格+Tab+问题卡+Heading/Text；筛选日期/分类/ID/数值并映射 QueryIR；点击行为②③；搜索；书签；修订；回收站；告警 results；CSV 导出；API Key；人员邀请停用；权限图 API；邮件重置密码。
验收：一张板三个卡一个日期筛选，告警能发（邮件或先站内）。

### W3 语义与分析  **OSS 核心第三刀**

语义类型全集+FK+格式；Join+表达式+函数+分箱+多阶段；Native 参数；模型/指标/分段/Glossary；下钻两类；表条件格式与列头动作；更多图（combo/waterfall/pivot/smart_scalar）；Scan/Fingerprint 定时；MySQL/ClickHouse；缓存；XLSX/PNG；数据参考；浏览模型指标；行详情。
验收：FK 隐式 Join 下钻到明细；SQL 问题接仪表盘 field filter。

### W4 Topbase 增量

写入数仓+策略+Run+血缘+目录徽标；飞书通知/分享；部门同步；AI propose_schedule；中国地图；仪表盘订阅飞书；模型持久化复用执行器。
验收：日 GMV 问题每天 9:00 写入 `warehouse.wh_gmv_daily`，飞书成功卡，新问题能查该表。

### W5 OSS 长尾

文档、X-ray、公开链接/Guest embed、Snippet、时间线、Actions、上传、Webhook、PDF、漏斗/桑基/树图、依赖顺序。

---

## 14. 从当前代码迁移

1. 保留 `QueryService` 只读闸门，Native 与 QueryIR 都走它。
2. `CatalogService` 仓储换成 appdb；json 做 `cmd/topbase import-catalog`。
3. `visual-query` → QueryIR。
4. `[]Schedule` → `app/warehouse` + 表；W4 才真正执行。
5. `DemoAI` 留测试；生产 `adapters/llm`。
6. `/admin` 向导可留到 W2；工作台按新 token 重做，两套视觉不得超过 W2。

---

## 15. 技术选型

| 问题 | 选择 |
| --- | --- |
| HTTP | 标准库 ServeMux |
| 应用库 | pgx + sqlc |
| 调度 W1–W3 | robfig/cron 进程内（告警） |
| 调度 W4 | 同进程 cron；负载上来再用 asynq/river |
| 图表 | ChartSpec → AntV G2 或 ECharts 适配器 |
| 飞书 | 官方 OAuth + 消息 API |
| LLM | OpenAI 兼容 HTTP |
| 密码 | argon2id |

---

## 16. 测试要求（功能即测试名）

- `queryir`：每种过滤/聚合/join 的 SQL 快照（PG/MySQL/CH）
- `permission`：并集、Native 整库禁、集合占位、个人集合
- `pipeline`：取消、超时、多语句拒绝、写入拒绝
- `warehouse`：replace 事务换表、增量 watermark、禁止写源库除非二次确认
- `ai`：draft 不直接执行；run 走闸门
- HTTP golden：Setup、保存问题、仪表盘筛选映射

---

## 17. 功能点总表（排期勾选）

复制到 Issue。状态栏留空。

### 地基与身份

- [ ] W0 Setup 向导五步
- [ ] W0 应用库 migrations
- [ ] W0 密码登录与会话
- [ ] W0 飞书 OAuth 闭环
- [ ] W1 邀请/停用/重置密码
- [ ] W1 个人资料/主题/个人集合
- [ ] W1 内置组+自定义组
- [ ] W2 API Key
- [ ] W2 登录历史
- [ ] W4 飞书部门同步

### 数据源

- [x] PG 连接 SSL SSH 测连
- [ ] W0 catalog/密钥进应用库
- [ ] W1 手动 Sync
- [ ] W1 删库引用检查
- [ ] W1 SQLite 示例库
- [ ] W3 定时 Sync/Scan/Fingerprint
- [ ] W3 MySQL / ClickHouse
- [ ] W3 JSON unfolding
- [ ] Driver feature 矩阵（随引擎）

### 语义

- [x] 基础字段标注
- [ ] W1 显示名/描述/隐藏
- [ ] W3 语义类型全集
- [ ] W3 FK/PK、格式、Cast、控件类型
- [ ] W3 Model / Metric / Segment / Glossary
- [ ] W4 模型持久化

### 查询构建器

- [x] 简易可视化查询（需升 QueryIR）
- [ ] W1 选表、列、过滤、聚合、分组、排序、Limit、预览
- [ ] W1 查看编译 SQL、转 SQL
- [ ] W3 Join 四类 + 隐式 Join
- [ ] W3 自定义列与表达式函数全集
- [ ] W3 HAVING、多阶段、分箱、时间分桶
- [ ] W3 分段作为过滤

### SQL

- [x] 只读闸门
- [ ] W1 编辑器体验（高亮/格式化/选中运行）
- [ ] W3 六类参数 + URL
- [ ] W3 引用问题/模型
- [ ] W5 Snippet

### 可视化与下钻

- [x] 表 + 简易趋势
- [ ] W1 table/line/area/bar/pie/scalar + ChartSpec
- [ ] W1 动效骨架与切入
- [ ] W2 progress/gauge、目标线
- [ ] W3 combo/waterfall/scatter/boxplot/pivot/smart_scalar
- [ ] W3 表格条件格式、Display as、列头/单元格下钻
- [ ] W3 两类下钻
- [ ] W4 中国地图
- [ ] W5 funnel/sankey/treemap、list、object_detail

### 集合与问题

- [ ] W1 问题 CRUD、集合嵌套、置顶、归档
- [ ] W2 搜索、书签、修订、回收站
- [ ] W5 时间线、X-ray

### 仪表盘

- [ ] W2 网格、Tab、问题/标题/文本卡
- [ ] W2 筛选器类型与映射
- [ ] W2 点击行为
- [ ] W3 Link/Iframe、联动筛选、加系列
- [ ] W4 飞书订阅
- [ ] W5 PDF、自动刷新

### 告警导出权限

- [ ] W2 告警三种触发
- [ ] W2 CSV
- [ ] W3 XLSX/PNG
- [ ] W1 库级权限
- [ ] W2 表级 granular + 集合权限图

### 数仓 / AI / 飞书

- [x] Schedule API 字段（内存）
- [x] demo AI
- [x] 飞书入口
- [ ] W1 AI SSE + 可审查 QueryIR
- [ ] W4 物化策略与 Run
- [ ] W4 数仓目录与血缘
- [ ] W4 飞书卡片
- [ ] W4 AI 提案调度

### 长尾

- [ ] W5 文档评论
- [ ] W5 公开链接 / Guest embed
- [ ] W5 Actions / 上传
- [ ] W5 Webhook / 全量邮件模板

---

## 18. 与拆解文档的对照原则

`metabase-产品与功能结构拆解.md` 描述 **上游有什么**。
本文描述 **Topbase 做什么、细到哪、代码放哪、何时做完**。

若拆解里有而本文没有：视为遗漏，应补进本章（OSS）或写入「EE不做」。
若本文有而拆解没有：必为 TB增量（飞书、数仓主按钮、AI 工具、中国地图、新 UI）。

实现时禁止对照 `enterprise/` 与 Metabase 前端组件结构；只对照本文功能点与 QueryIR 契约。

---

*W0–W3 完成 = Metabase 开源核心可日常使用。W4 完成 = 中国团队可把分析沉淀成数仓。W5 = OSS 长尾。*
