# Metabase 产品与功能结构拆解

> 依据 `metabase-master/` 源码（后端 `src/metabase/`、前端 `frontend/src/metabase/`、企业版 `enterprise/`、官方文档 `docs/`）整理。
> 目的：为 Topbase（Go 独立实现）提供产品级功能清单，拆到可落地的功能点。
> 合规：本文只描述产品能力与模块边界，不复制 Metabase 商业版实现。标 `EE` 的能力属于 Pro/Enterprise，Topbase 需独立设计，不得对照商业代码实现。

---

## 0. 怎么读这份文档

- **产品结构**：用户能看见、能进入的产品面（导航、页面、对象、角色）。
- **功能结构**：每个产品面里「能做什么」，拆到操作级功能点。
- **能力依赖**：某功能依赖哪些底层能力（驱动 feature、权限、元数据、调度）。
- **OSS / EE**：开源版默认有的能力 vs 商业版门控能力。Topbase 当前以开源业务能力为兼容方向。

---

## 1. 产品定位

Metabase 是面向全公司的 **自助商业智能（Self-service BI）+ 嵌入式分析（Embedded Analytics）** 平台。

一句话：让不会 SQL 的人能分析据，让会 SQL 的人能写复杂查询，并把结果做成仪表盘、文档、告警、嵌入组件，再按权限分发给内部团队或外部客户。

### 1.1 核心价值主张

| 主张 | 对应产品能力 |
| --- | --- |
| 5 分钟接入数据 | 安装 / Setup 向导 / 数据源连接 / Sample Database |
| 不会 SQL 也能分析 | Notebook Query Builder + 表达式 + Drill-through |
| 会 SQL 也能用 | Native/SQL Editor + 参数 + Snippet + 引用已保存问题 |
| 结果要好看、可交互 | 20+ 可视化 + 仪表盘筛选/点击行为 + 下钻 |
| 结果要推出去 | 告警 / 仪表盘订阅 / 公开链接 / 嵌入 / MCP / Agent API |
| 数据要可信 | 语义类型 / Model / Metric / Segment / Measure / Library / 内容验证 |
| 数据要可治理 | 权限矩阵 / 行级安全 / 模拟登录 / 多租户 / Data Studio / 血缘 |
| 分析要可嵌入 | Guest Embed / SSO Embed / Full App Embed / React SDK / Web Component |

### 1.2 产品形态

| 形态 | 说明 |
| --- | --- |
| 完整 Web App | 登录后的主应用：分析、仪表盘、集合、管理、Data Studio |
| 公开页 | 无需登录的公开问题 / 仪表盘 / 文档 / Action 表单 |
| 嵌入运行时 | iframe / Web Component / React SDK / 全应用嵌入 |
| 静态可视化 | 邮件/Slack/PDF/PNG 渲染用的无交互图表 |
| CLI / API | REST API、序列化、远程同步、JAR 命令 |
| AI 接口 | Metabot、Agent API、MCP Server、Slackbot |

### 1.3 发行版与许可（重构时必须分清）

| 发行 | 许可 | 含义 |
| --- | --- | --- |
| Open Source | AGPL | 主仓 `src/` + `frontend/` 中的开源能力 |
| Starter / Pro / Enterprise | 商业许可 | `enterprise/` 下的门控能力，靠 token feature 开关 |
| Cloud | 托管 | 在商业能力之上再加 SMTP、备份、自定义域名、计量计费 |

Topbase 明确：**以开源业务能力为兼容方向，独立领域模型，不得混入商业版代码。**

---

## 2. 用户角色与使用场景

### 2.1 角色（产品视角，不是数据库角色）

| 角色 | 典型目标 | 主要入口 |
| --- | --- | --- |
| 匿名访客 | 看公开/嵌入图表 | Public / Guest Embed |
| 嵌入终端用户 | 在宿主 App 里看自己的数据 | Modular / Full App Embed |
| 业务分析用户 | 用 Query Builder 分析、看仪表盘 | Home / Collection / Browse / Search |
| SQL 分析师 | 写 SQL、Snippet、Model | Native Editor / Data Studio |
| 数据建模者 / Data Analyst | 管元数据、Metric、Transform、血缘 | Data Studio |
| 内容策展人 | 整理集合、置顶、官方标记、验证 | Collection |
| 管理员 | 连库、权限、认证、外观、性能 | Admin |
| 运维 / Monitor | 看失败问题、任务、日志 | Monitor |
| AI / 外部 Agent | 用自然语言或工具协议分析 | Metabot / MCP / Agent API |

### 2.2 内置用户组

- **All Users**：所有人自动加入。权限设计惯例是先 Block 该组，再给具体组授权。
- **Administrators**：超级管理。
- **Data Analysts**（EE）：可进 Data Studio、跑 Transform、看依赖诊断。
- 自定义组：任意数量，权限按组叠加（更宽松者优先）。
- **Tenants**（EE）：面向嵌入多租户的客户隔离。

### 2.3 个人账户能力

- 资料：姓名、邮箱、语言
- 主题：浅色 / 深色（仅个人 UI，不影响图表色板与嵌入）
- 密码修改（SSO/LDAP 时隐藏）
- 双因素认证（EE）：Authenticator、恢复码、邮箱一次性码
- 登录历史
- 通知管理：自己订阅的 Alert / Dashboard Subscription，退订页 `/unsubscribe`
- 个人集合（Personal Collection）：草稿空间，默认仅自己和 Admin 可见
- 书签（Bookmark）：仅自己可见的快捷入口

---

## 3. 产品信息架构（页面与导航）

### 3.1 全局壳层

- 左侧导航：Home、Collections、Browse、Bookmarks、个人集合、回收站
- 顶部：`+ New`、搜索、命令面板（Cmd/Ctrl+K）、Metabot、网格菜单（Admin / Data Studio / Account）
- 命令面板：跳转页面、切主题、唤起 Metabot、搜内容
- 键盘快捷键：新建问题、保存、运行查询、打开 Metabot（Cmd/Ctrl+E）等

### 3.2 主路由地图（前端 `routes.tsx`）

| 路径 | 产品面 |
| --- | --- |
| `/setup` | 首次安装向导 |
| `/auth/login` `/auth/login/:provider` | 登录 / SSO |
| `/auth/forgot_password` `/auth/reset_password/:token` | 找回/重置密码 |
| `/auth/logout` `/auth/sso` | 登出 / SSO 回调 |
| `/` | 落地页（默认 Home，或指定仪表盘 / 自定义 URL） |
| `/getting-started` | 新手引导 Onboarding |
| `/search` | 全局搜索 |
| `/trash` | 回收站 |
| `/collection/:slug` | 集合：移动 / 归档 / 权限 / 时间线 / 清理 |
| `/dashboard/:slug` | 仪表盘：移动 / 复制 / 归档 |
| `/question` `/question/:slug` `/question/notebook` `/question/ask` | 问题 / Notebook / Metabot 分析 |
| `/model` `/model/new` `/model/:slug/{query,columns,metadata}` | 模型 |
| `/browse/{models,metrics,databases}` | 浏览模型 / 指标 / 数据库 |
| `/table/:slug` `/table/:id/detail/:rowId` | 表查询 / 行详情 |
| `/explore` | Metrics Viewer |
| `/auto/dashboard/*` | X-ray 自动仪表盘 |
| `/document/:entityId` | 文档 + 评论侧栏 |
| `/data-studio/*` | Data Studio |
| `/admin/*` | 管理后台 |
| `/unsubscribe` | 退订通知 |
| `/unauthorized` | 无权限页 |

另有公开/嵌入独立入口：`app-public`、`app-embed`、`app-embed-sdk`、`app-embed-mcp`、`app-static-viz`。

### 3.3 `+ New` 创建入口

- Question（Query Builder）
- SQL query（Native Editor）
- Dashboard
- Document
- Collection
- Model
- Metric（部分版本/权限下）

### 3.4 管理后台 IA（`/admin`）

| 分区 | 功能 |
| --- | --- |
| Databases | 数据源列表 / 新建 / 编辑 / 可写连接 / Database Routing 目标库 |
| Datamodel | 表元数据（库→schema→表→字段）/ Segment 列表与修订 |
| People | 用户 CRUD、激活/停用、重置密码、组、Tenants（EE） |
| Embedding | 嵌入总开关、Setup Guide、Guest、Security、Themes |
| Settings | 通用 / 邮件 / Slack / Webhook / 认证 / 本地化 / 外观 / 地图 / 上传 / 更新 / 远程同步 / 自定义可视化 |
| Permissions | 数据 / 集合 / 应用级权限 |
| Performance | 库级缓存策略 / 问题与仪表盘缓存 / Model Persistence |
| Metabot | AI 设置 / MCP / OAuth 授权 / AI Controls（EE） |
| Security Center（EE） | 安全态势 |
| Help | 帮助与 Support Access Grant（EE） |

### 3.5 Data Studio IA（`/data-studio`）

| 分区 | 功能 |
| --- | --- |
| Guide | 首次引导 |
| Data | 表元数据、Segment、Measure |
| Transforms | 转换定义、Job、Run、Inspector |
| Glossary | 业务术语表 |
| Library（EE） | 可信内容策展 |
| Dependencies（EE） | 血缘图、替换数据源 |
| Schema Viewer（EE） | ERD |
| Git Sync | Remote Sync 工作区 |
| Settings | Data Studio 相关设置 |

### 3.6 Monitor IA

- 依赖诊断（从 Data Studio 迁来）
- 出错的问题（Erroring questions）
- 告警管理
- 后台任务 / 定时 Job
- 应用日志
- Model Persistence 日志
- CLI Analytics

---

## 4. 核心对象模型

重构时这些是领域实体，不是 UI 名词。

### 4.1 内容对象

| 对象 | 含义 | 关键字段/行为 |
| --- | --- | --- |
| **Card / Question** | 一条已保存查询 + 可视化设置 | MBQL 或 Native；可存到集合或仪表盘；有修订历史 |
| **Dashboard** | 卡片网格 + Tab + 筛选器 + 点击行为 | 订阅、自动刷新、全屏、PDF |
| **DashboardCard** | 仪表盘上的一张卡 | 问题卡 / 文本 / 标题 / 链接 / iframe / Action 按钮 |
| **DashboardTab** | 仪表盘分页 | 可复制 Tab、跨 Tab 移卡 |
| **Collection** | 文件夹 | 嵌套、个人集合、官方集合（EE）、Library（EE）、置顶 |
| **Document** | 长文分析 | Markdown + 内嵌图表 + 评论 + @提及 |
| **Model** | 语义化可复用数据集 | 基于 QB 或 SQL；可持久化；可挂 Action |
| **Metric** | 规范指标 | 可被 QB 引用；Browse Metrics / Metrics Viewer |
| **Segment** | 规范过滤器 | 出现在 QB 过滤下拉顶部 |
| **Measure** | 规范聚合（Data Studio） | 表级可复用聚合 |
| **Snippet** | SQL 片段 | 可嵌套；EE 有 Snippet 文件夹权限 |
| **Timeline / Event** | 时间线上的业务事件 | 可叠在时序图上 |
| **Action** | 回写表单/按钮 | Basic（增删改）或 Custom SQL；挂在 Model / 仪表盘 |
| **Comment** | 文档评论 | 通知邮件 |
| **Bookmark** | 个人书签 | |
| **Revision** | 内容版本 | 问题/仪表盘/Segment 等可回看 |
| **Pulse / Alert / Notification** | 推送任务 | 问题告警、仪表盘订阅 |

### 4.2 数据与元数据对象

| 对象 | 含义 |
| --- | --- |
| **Database** | 一条数据源连接（含 Sample Database） |
| **Schema** | 命名空间（部分引擎无 schema） |
| **Table** | 同步来的表/视图 |
| **Field** | 列：物理类型、语义类型、格式、指纹、缓存枚举值 |
| **FieldValues** | 扫描得到的字段取值，供筛选下拉 |
| **FK / Entity Key** | 用于隐式 Join、关联下钻、联动筛选 |
| **PersistedModel** | Model 物化表 |
| **Transform** | 写出新表的转换定义 |
| **TransformJob / Run / Tag** | 转换调度与执行 |
| **Upload** | CSV 上传生成的表 |
| **IndexedEntity** | Model 行级搜索索引 |

### 4.3 身份与权限对象

| 对象 | 含义 |
| --- | --- |
| **User** | 人：激活、邀请、语言、主题 |
| **Group** | 权限主体 |
| **Session** | 登录会话 |
| **ApiKey** | 静态 API 密钥 |
| **PermissionGraph** | 数据/集合/应用权限图及修订 |
| **Sandbox / Impersonation**（EE） | 行级安全 / 连接角色模拟 |
| **Tenant**（EE） | 嵌入多租户 |
| **LoginHistory** | 登录审计 |
| **OAuth Client / Authorization** | 作为 OAuth Server（MCP 等） |

### 4.4 系统对象

| 对象 | 含义 |
| --- | --- |
| **Setting** | 实例级键值配置（可环境变量/配置文件覆盖） |
| **Task / TaskHistory** | Quartz 后台任务 |
| **CacheConfig** | 查询缓存策略 |
| **Channel** | Email / Slack / HTTP Webhook |
| **Secret** | 连接密码、证书、SSH 私钥 |
| **RemoteSync / Serialization** | Git/文件导入导出 |
| **Usage Analytics**（EE） | 内部审计库 |

---

## 5. 功能域详解

以下每一节都拆到操作级功能点。`[OSS]` 开源即有，`[EE]` 商业门控，`[Cloud]` 仅托管。

---

### 5.1 安装、启动与 Setup

#### 5.1.1 安装形态

- JAR 运行
- Docker
- 作为系统服务
- AWS Elastic Beanstalk / RDS
- Metabase Cloud 托管
- 开发实例模式

#### 5.1.2 应用库（Application DB）

- 默认内嵌 H2（仅开发/试用，生产需外置）
- 生产支持：PostgreSQL、MySQL/MariaDB
- 从 H2 迁移到生产库
- 备份 / 升级 / 版本兼容

#### 5.1.3 Setup 向导（`/setup`）

1. 语言
2. 管理员账号
3. 站点名称
4. 用量匿名统计开关
5. 连接第一个数据库（可跳过，使用 Sample Database）
6. 完成进入应用

#### 5.1.4 首次体验

- Sample Database（可删除、可一键恢复）
- 示例集合 / 示例仪表盘（E-commerce insights）
- Onboarding（`/getting-started`）
- Data Studio Guide（首次进入）
- 落地页可配：默认 Home / 指定仪表盘 / 自定义 URL（后两项部分 EE）

---

### 5.2 认证与会话 `[OSS 基础 / EE 增强]`

#### 5.2.1 本地账号 `[OSS]`

- 邮箱 + 密码登录
- 忘记密码 / 邮件重置
- 邀请用户（邮件邀请链接）
- 停用 / 重新激活
- 密码复杂度策略
- 会话过期
- 新设备登录邮件提醒
- 登录历史

#### 5.2.2 SSO / 目录

| 方式 | 级别 | 要点 |
| --- | --- | --- |
| Google Sign-In | OSS 可用，部分增强 EE | 域名限制、组映射 |
| LDAP | OSS 可用，增强 EE | 组映射、同步 |
| SAML | EE | Okta / Entra ID / Google / Auth0 / Keycloak |
| JWT | EE | 嵌入 SSO 常用 |
| OIDC | EE | Keycloak 等 |
| SCIM | EE | 用户自动开通/关闭 |
| 关闭密码登录 | EE | 强制走 SSO |
| 会话超时可配置 | EE | |
| MFA / 2FA | EE | TOTP + 恢复码 + 邮箱码 |

#### 5.2.3 机器身份 `[OSS]`

- API Key：创建、前缀展示、权限组绑定、轮换、删除
- Session Cookie
- Static API Key（部分内部 notify 接口）
- 作为 OAuth Server：给 MCP / 外部客户端发授权

#### 5.2.4 会话安全细节

- Session cleanup 定时任务
- Challenge（登录风控相关）
- Support Access Grant（EE）：临时授权官方支持登录

---

### 5.3 数据源连接 `[OSS]`

#### 5.3.1 官方驱动

| 引擎 | 模块位置 | 备注 |
| --- | --- | --- |
| PostgreSQL | 内置 `driver/postgres` | 含 JSON 展开、Actions |
| MySQL / MariaDB | 内置 `driver/mysql` | Actions |
| SQLite | 内置 | |
| H2 | 仅应用库，不再支持作为业务库连接 | |
| Athena | `modules/drivers/athena` | 一连接多库 |
| BigQuery | `bigquery-cloud-sdk` | |
| ClickHouse | `clickhouse` | Transform 仅 Cloud |
| Databricks | `databricks` | 多级 schema |
| Druid | `druid-jdbc` | |
| MongoDB | `mongo` | 嵌套字段、需指定 collection |
| Oracle | `oracle` | |
| Presto | `presto-jdbc` | |
| Redshift | `redshift` | |
| Snowflake | `snowflake` | |
| SparkSQL | `sparksql` | |
| SQL Server | `sqlserver` | |
| Starburst | `starburst` | |
| Vertica | `vertica` | |
| 社区驱动 | 插件机制 | 不在官方支持列表 |

#### 5.3.2 连接配置功能点

- 主机 / 端口 / 库名 / 用户 / 密码
- 连接串 / 高级 JDBC 参数
- SSL / 客户端证书 / 服务端证书
- SSH 隧道：密码或私钥、主机密钥
- 连接前测试（Can connect）
- 连接加密存储（应用级 secret）
- 只读 vs 可写账号（Actions / Upload / Transform / 表编辑）
- 云厂商托管库向导（如 AWS RDS）
- 删除库（级联删除依赖问题/卡片，不可逆）
- 恢复 Sample Database
- Database Auth Provider（EE）：用外部身份换临时凭证
- Writable Connection（EE）：专用于回写的第二连接
- Database Routing（EE）：按登录者路由到不同物理库

#### 5.3.3 同步 / 扫描 / 指纹 `[OSS]`

三种后台元数据作业，互不相同：

| 作业 | 做什么 | 默认节奏 | 用途 |
| --- | --- | --- | --- |
| **Sync schema** | 拉库/schema/表/列/PK/FK，停用已删表 | 每小时 | 浏览、QB 选表、隐式 Join |
| **Scan field values** | 抽样列值，缓存给筛选下拉 | 每天；仅最近 14 天用过的字段 | Filter widget 下拉/搜索 |
| **Fingerprint** | 前 1 万行统计：空值率、基数、min/max/分位 | 随同步 | 自动可视化、X-ray、分箱 |

可配置：

- 按库选择：小时/天同步；扫描「定期 / 仅新增筛选器时 / 从不」
- 手动：Sync database schema
- 手动：Re-scan table / field
- 清空某表/字段的缓存取值
- 外部触发：`/api/notify`（static API key）通知 Metabase 去同步
- 任务进度：Admin > Tools/Monitor > Tasks

#### 5.3.4 驱动能力开关（`driver/features`）

每个引擎声明自己支持哪些能力，Query Builder / 表达式 / Transform 会按此裁剪 UI：

**查询与表达式**

- `basic-aggregations` / `standard-deviation-aggregations` / `percentile-aggregations` / `distinct-where`
- `expressions` / `expression-aggregations` / `expression-literals`
- `advanced-math-expressions`
- `expressions/{integer,text,date,datetime,float,today}`
- `nested-queries` / `native-parameters` / `native-parameter-card-reference` / `native-temporal-units`
- `binning`
- `left-join` / `right-join` / `inner-join` / `full-join`
- `regex` / `regex/lookaheads-and-lookbehinds`
- `temporal-extract` / `date-arithmetics` / `now` / `convert-timezone` / `datetime-diff`
- `window-functions/cumulative` / `window-functions/offset`
- `split-part` / `collate` / `uuid-type`
- `case-sensitivity-string-filter-options`
- `parameterized-sql`

**元数据与结构**

- `metadata/key-constraints`
- `nested-fields` / `nested-field-columns`（Mongo 全嵌套 vs PG JSON 列）
- `schemas` / `multi-level-schema`
- `describe-fields` / `describe-indexes` / `index-info` / `fingerprint`
- `connection/multiple-databases`
- `identifiers-with-spaces`
- `set-timezone`

**回写与工程**

- `actions` / `actions/custom` / `actions/data-editing`
- `uploads` / `upload-with-auto-pk`
- `persist-models` / `persist-models-enabled`
- `transforms/table` / `transforms/python`
- `rename` / `atomic-renames` / `create-or-replace-table`
- `index/standalone-create` / `index/inline-create`
- `table-privileges`
- `connection-impersonation` / `connection-impersonation-requires-role`
- `saved-question-sandboxing`
- `native-requires-specified-collection`（Mongo）

---

### 5.4 元数据与语义层 `[OSS 基础 / Data Studio 增强]`

#### 5.4.1 物理类型（Data type）

Metabase 内部类型层级，屏蔽各引擎差异：

- Numeric
- Temporal
- Text
- Text-like（BSON ID、Enum）
- Boolean
- Collection（JSON / RECORD / Object）
- 数组：基本不支持，只能「为空 / 非空」过滤

可对部分列做 **Cast**（例如 Text → Date），作用于整个实例；查询内也可用 `date()` / `integer()` 等表达式临时转换。

#### 5.4.2 语义类型（Semantic / Field type）

**任意字段**

- Entity Key
- Foreign Key（必须配置才能：隐式 Join、仪表盘联动筛选、关联行下钻）

**数值**

- Quantity / Score / Percentage
- Currency / Discount / Income
- Latitude / Longitude
- Category

**时间**

- Creation date/time/timestamp
- Joined date/time/timestamp
- Birthday

**文本**

- Entity name / Name / Title / Description / Product / Source / Category
- Email / URL / Image URL / Avatar URL
- City / State / Country / ZipCode

**集合**

- Field containing JSON → JSON unfolding

语义类型影响：显示格式、自动选图、地图类型、字段过滤器、X-ray、从列提取 host/domain/日期部件。

#### 5.4.3 表级元数据编辑

- 显示名、描述、隐藏表、排序
- 字段显示名、描述、语义类型、FK 目标
- 字段可见性：普通 / 详情才显示 / 敏感隐藏
- 过滤控件类型：下拉 / 搜索 / 输入框
- 默认格式：小数位、分隔符、货币、日期格式、百分号
- JSON unfolding 开/关
- 字段币种、单位
- 友好化表名（把 `some_name` 变成 `Some Name`，可关）

#### 5.4.4 Model（模型）`[OSS]`

- 从 Query Builder 或 SQL 创建
- 选列、改名、设语义类型（尤其 SQL 模型，不设则无法完整下钻）
- 列描述、格式
- 可被其他问题当数据源
- 可被 SQL `{{#card}}` 引用
- Model Persistence：把结果物化回仓库，加速查询
- 行级搜索：用 Entity Key + 文本列做 Indexed Entity
- 作为 Action 容器
- Browse Models
- 详情页：编辑查询 / 列 / 元数据

#### 5.4.5 Metric `[OSS]`

- 规范指标定义（聚合 + 可选过滤）
- Query Builder 可直接选 Metric 当数据源
- Browse Metrics / `/explore` Metrics Viewer
- 可被 Metabot / Library 优先推荐

#### 5.4.6 Segment `[OSS]`

- 表上的命名过滤器（如「活跃用户」）
- 出现在 QB Filter 下拉顶部（紫色星标）
- 有修订历史
- Admin Datamodel 与 Data Studio 两处可管

#### 5.4.7 Measure `[Data Studio]`

- 表级可复用聚合，和 Metric 类似但挂在表/Data Studio 语义层

#### 5.4.8 Glossary `[OSS/Data Studio]`

- 业务术语定义
- 给人看，也给 Metabot / Agent 当上下文

#### 5.4.9 Library `[EE]`

- 特殊集合：只放「官方推荐」的表、Metric、Snippet
- Query Builder 数据选择器默认先只显示 Library，再「Browse all」

#### 5.4.10 Schema Viewer `[EE]`

- 表关系 ERD

#### 5.4.11 内容验证与官方集合 `[EE]`

- Verified 标记：问题/仪表盘可信
- Official Collection：黄标，搜索加权靠前
- Collection Cleanup：清理长期不用内容

---

### 5.5 浏览与探索 `[OSS]`

#### 5.5.1 Home

- 最近查看
- 热门/精选
- 书签
- X-ray 入口
- 可被管理员改成某张仪表盘

#### 5.5.2 Browse

- Browse Databases → Schema → Tables
- Browse Models
- Browse Metrics
- 表永久链接（按库名/schema/表名）
- 表行详情 `/table/:id/detail/:rowId`
- 可编辑表（EE `table-data-editing`）：在 Browse 里直接改行

#### 5.5.3 Data Reference（数据参考）

- 按库/表/字段看描述、字段类型、样例
- 与 Glossary 打通
- 供分析师理解数仓

#### 5.5.4 X-rays（自动洞察）`[OSS，可关闭]`

- 对表 / 字段 / 模型 / 实体一键生成自动仪表盘
- 模板在 `resources/automagic_dashboards/`：
  - 通用表 / 用户表 / 事件表 / 交易表
  - 按国家/州/来源/产品/季节性对比
  - 通用指标、通用字段、FK、Country、State
- 路由：`/auto/dashboard/*`
- 下钻菜单里的「Automatic insights」
- 可在 Settings 关闭（贵查询场景）

#### 5.5.5 搜索 `[OSS / EE 语义搜索]`

- 搜 Question / Dashboard / Collection / Model / Metric / Table / Database / Document
- Official / Verified 加权
- Model 行级实体搜索（Indexed Entity）
- EE：Semantic Search
- 命令面板复用同一套搜索

#### 5.5.6 集合组织 `[OSS]`

- 嵌套文件夹
- 移动、复制、归档、从回收站恢复
- 置顶（大卡片，全员可见）
- 个人集合
- 把集合内问题批量挪进仪表盘
- 集合权限弹窗
- 集合时间线（Events）
- 回收站 `/trash`（旧 `/archive` 重定向）
- 书签
- EE：Official、Cleanup、Tenant 集合、Snippet 集合

#### 5.5.7 历史与修订 `[OSS]`

- 问题 / 仪表盘 / Segment 等 Revision
- 谁在何时改了什么
- 可回看（部分可还原）

#### 5.5.8 时间线 Events `[OSS]`

- 在集合上建 Timeline
- 事件：时间点 + 标题 + 描述 + 图标
- 叠在折线/面积等时序图上，解释「为什么这天突增」

---

### 5.6 Question：查询构建器（Notebook）`[OSS]`

这是 Metabase 的产品心脏。所有 GUI 问题最终编译为 MBQL，再由 Query Processor 译成各引擎 SQL/原生语言。

#### 5.6.1 数据源选择（Pick data）

- 表 / Model / Metric / 已保存 Question
- 搜索或浏览库与集合
- Library 优先（EE）
- 预览数据；Cmd/Ctrl+Click 新开表
- 勾选要输出的列（不勾仍可用于过滤，只是结果不展示）
- **注意**：可视化里隐藏列 ≠ 安全，真正排除列必须在 Data 步骤取消勾选

#### 5.6.2 Join

- 同库多表
- Join 类型：Inner / Left / Right / Full（按驱动 feature）
- 多条件 Join
- 隐式 Join：配置了 FK 后，可像用同一张表一样取关联表字段
- 可 Join 已保存问题 / Model（嵌套查询）

#### 5.6.3 自定义列（Custom column）

- 四则运算 `+ - * /` 与括号
- 完整表达式函数（见 5.6.8）
- 可出现在 Summarize 之前或之后
- 只存在于该问题，不写回数据库

#### 5.6.4 Filter

按列类型给出不同控件：

| 列类型 | 过滤能力 |
| --- | --- |
| 数值 | `=` `≠` `>` `≥` `<` `≤` `between` 空/非空 |
| 文本/分类 | 是/不是、包含/不包含、开头/结尾、空/非空；下拉/搜索/输入 |
| 日期 | 具体日期、日期范围、相对日期（含 Starting from 偏移）、年月/季年 |
| 经纬度 | 数值过滤 + **Inside** 地理框选 |
| JSON/结构化 | 仅空/非空（除非 unfolding） |
| Boolean | 真/假/空 |
| Segment | 一键套用命名过滤 |

- 可在 Summarize **之后**再过滤（等价 SQL `HAVING`）
- 可用自定义表达式写 `AND/OR` 复合条件
- 顶部紫色药丸可编辑/删除单个过滤

#### 5.6.5 Summarize & Group（聚合与分组）

内置聚合：

- Count / CountIf
- Sum / SumIf
- Average
- Distinct / DistinctIf
- Min / Max
- Median / Percentile（部分引擎不支持）
- Share
- StandardDeviation / Variance
- CumulativeSum / CumulativeCount

分组（Breakout / Group by）：

- 任意维度列
- 时间分桶：分钟/小时/日/周/月/季/年，以及更多 temporal unit
- 数值分箱（Binning）：自动/固定宽度/固定箱数
- 多维度组合
- 自定义聚合表达式，如 `Average(sqrt([X])) + Sum([Y])`

自动选图规则（可改）：

| Group by | 默认图 |
| --- | --- |
| 文本/分类 | 柱状图 |
| 时间 | 折线图 |
| 数值已分箱 | 柱状图 |
| 数值未分箱 | 表 |
| 布尔 | 柱状图 |
| 无聚合 | 表 |
| 经纬度已分箱 | 网格地图 |
| 经纬度未分箱 | 钉图 |
| Country | 世界区域图 |
| State | 美国区域图 |

#### 5.6.6 Sort / Limit

- 多列排序，每列升/降序
- 累计聚合的排序会影响计算顺序
- Limit 只能放在最后；先算完再截断
- 未聚合结果浏览器最多展示约 2,000 行，聚合约 10,000 行（避免撑爆浏览器）

#### 5.6.7 Notebook 交互细节

- 每一步右侧 Preview（前 10 行）
- 多阶段流水线：Filter → Summarize → Filter → Summarize…
- 「View SQL / View query」预览原生语句（需 QB+Native 权限）
- **转为 SQL**：单向，不可转回 Notebook
- Visualize 进入结果/图表
- 保存到集合或直接保存到某张仪表盘
- 另存为、复制、移动、归档
- 可被 Metabot 生成/改写

#### 5.6.8 表达式函数清单（QB Custom Expression）

**聚合**

- Average, Count, CountIf, Distinct, DistinctIf
- Max, Median, Min, Percentile
- Share, StandardDeviation, Sum, SumIf, Variance
- CumulativeSum, CumulativeCount

**逻辑**

- between, case, if, coalesce
- in, notIn, isNull, notNull

**数学**

- abs, ceil, exp, floor, integer, log, power, round, sqrt

**字符串**

- concat, contains, doesNotContain, startsWith, endsWith
- domain, host, path, subdomain
- isEmpty, notEmpty
- lTrim, rTrim, trim, lower, upper, length
- regexExtract, replace, splitPart, substring
- text, float, integer, date, datetime

**日期**

- convertTimezone, date, datetime
- datetimeAdd, datetimeSubtract, datetimeDiff
- day, dayName, hour, minute, second
- month, monthName, quarter, quarterName
- week, weekday, year
- now, today, interval, relativeDateTime, timeSpan

**窗口**

- Offset（lag/lead）
- CumulativeCount / CumulativeSum

引擎不支持的函数会在 UI 隐藏或运行时报错（见 5.3.4）。

---

### 5.7 Question：Native / SQL 编辑器 `[OSS]`

#### 5.7.1 编辑器能力

- 选数据库（Mongo 还需选 collection）
- 语法高亮、运行、只运行选中片段
- 快捷键 Ctrl/⌘ + Enter
- SQL 格式化（SQLite / SQL Server 无）
- Postgres `?` JSON 运算符需写成 `??`（JDBC 占位符冲突）
- 保存、下载、转 Model、加入仪表盘
- Metabot：自然语言生成 SQL、行内改 SQL、修报错
- 引用 Model / 已保存问题：`{{#card-id}}` 或 UI 选择
- Snippet：`{{snippet: name}}`，可嵌套；EE 有文件夹权限

#### 5.7.2 SQL 参数类型

| 类型 | 作用 | 典型控件 |
| --- | --- | --- |
| Field Filter | 绑定真实字段，生成智能控件，可接仪表盘筛选 | 日期选择、分类下拉 |
| Basic variable | text / number / date 简单占位 | 输入框 |
| Optional variable | `[[ ... {{x}} ]]` 无值则整段省略 | |
| Time grouping | 改时间粒度，不删数据 | 日/周/月… |
| Table variable | 选要查的表，常配合 Snippet | 表选择器 |
| Card reference | 把另一问题当 CTE/子查询 | |

参数配置：

- Widget 类型、标签
- 下拉 / 搜索 / 输入
- 单值 / 多值
- 默认值
- 别名表需声明 table/field alias
- URL 传参 `?var=value`
- 多参数用 `&` 连接
- 必须有变量才能被仪表盘筛选器连接

#### 5.7.3 Native 的能力边界（产品关键）

Metabase **不解析 SQL 文本**，因此：

- Native 问题只有「基于结果」的下钻（过滤/排序/分布），没有「改写查询」的下钻（See these records / Break out / Zoom in）
- 只要库里任何一张表被 Block 或行级安全，整库 Native 都被禁用
- 下载权限若只开了部分表，Native 结果也不能下
- 必须先保存，才能在结果上做有限下钻

---

### 5.8 查询处理管道（Query Processor）

前端不直接拼 SQL。统一走 QP。重构时这是引擎内核。

```
请求
  → 鉴权 / 数据权限 / 行级安全(EE) / Impersonation(EE)
  → 规范化 MBQL
  → 展开 Macro（Metric/Segment/Snippet/Card 引用）
  → 解析 Source Table / 字段 / Join / 隐式 Join
  → 参数代入（MBQL 参数 + Native 模板标签）
  → 时间分桶 / 分箱 / 累计聚合 / Pivot 嵌套
  → 检查驱动 feature
  → 编译为原生查询
  → 缓存查找（未命中则执行）
  → 驱动执行（可取消；连接关闭即 cancel）
  → 后处理：时区、格式化、大整数、行截断、结果元数据
  → 流式输出：JSON / CSV / XLSX / 地图瓦片
```

关键中间件（`query_processor/middleware/`）：

- `permissions` / `constraints` / `cache` / `persistence`
- `parameters` / `expand_macros` / `metrics` / `measures`
- `resolve_source_table` / `resolve_fields` / `resolve_joins` / `add_implicit_joins`
- `binning` / `auto_bucket_datetimes` / `cumulative_aggregations`
- `limit` / `format_rows` / `add_rows_truncated` / `large_int`
- `visualization_settings` / `results_metadata` / `update_used_cards`
- `catch_exceptions` / `process_userland_query`

执行约束（产品可见）：

- 查询超时
- 最大行数（下载另有 1 万 / 100 万档，EE）
- 只读事务（对支持的引擎）
- 报表时区 `report-timezone`

---

### 5.9 可视化 `[OSS + 自定义 EE/插件]`

#### 5.9.1 注册图表类型（`visualizations/register.ts`）

| 标识 | 产品名 | 适用数据 |
| --- | --- | --- |
| Table | 表格 | 任意；默认可视化 |
| List | 列表 | 记录列表 |
| Scalar | 数字 | 单值 |
| SmartScalar | 趋势数字 | 单值 + 对比期 |
| Progress | 进度条 | 当前值 vs 目标 |
| Gauge | 仪表盘 | 单值 + 区间 |
| Line / Area / Bar / Row | 折/面/柱/条 | 1+ 指标，1+ 维度 |
| Combo | 组合图 | 柱+线 |
| Waterfall | 瀑布图 | 增减构成 |
| Scatter | 散点/气泡 | X/Y/(Z 大小) |
| BoxPlot | 箱线图 | 分布 |
| Pie | 饼/环 | 分类占比 |
| Funnel | 漏斗 | 阶段转化 |
| Map | 地图 | 经纬度 / Country / State |
| PivotTable | 透视表 | 行列维度 + 指标 |
| Sankey | 桑基 | 流向 |
| Treemap | 矩形树图 | 层级占比 |
| ObjectDetail | 行详情 | 主键行 |
| Custom | 自定义可视化 | 插件/EE |

#### 5.9.2 图表设置控件（跨图复用）

- 文本/数字输入、单选、开关、分段控件
- 字段选择、多字段、字段分区（透视行列）
- 颜色、色带、系列顺序、系列显隐
- 目标线 Goal、最大分类数
- 表格列显隐/排序、条件格式
- 饼图维度/扇区名、SmartScalar 对比期、Treemap 分组
- 链接 URL 模板

#### 5.9.3 表格可视化细节

- 列重排、隐藏、固定
- 条件格式（色阶、单色规则）
- Display as：文本 / 链接 / email / 图片 / 头像 / 进度
- 列头下钻：过滤、排序、分布、求和/平均、去重、按时间求和、提取日期/URL 部件、合并列
- 单元格下钻：过滤、查看详情、跳关联记录、看构成行
- 透视（轻量 pivot，不同于 Pivot Table 图）
- 行详情 Object Detail
- 提取 Domain/Host/Path/日期部件

#### 5.9.4 地图

- Pin map（未分箱经纬度）
- Grid map（分箱经纬度）
- Region map：世界（Country）、美国（State）
- 自定义 GeoJSON 地图（Admin > Maps）
- 国家/州编码表
- 地图瓦片 API `/api/tiles`

#### 5.9.5 其他可视化细节

- 多系列、堆叠、双轴
- 趋势线
- 目标线（给 Goal Alert 用）
- Tooltip 自定义
- 时间轴叠加 Timeline Events
- 图例点击：显隐系列 / 对整系列下钻
- 时序图拖拽框选时间范围
- 静态渲染：邮件/Slack/PDF 走 `static-viz` + Graal JS
- 自定义可视化：管理员上传/白名单（嵌入时 Guest 会回退默认图）

#### 5.9.6 Drill-through（下钻）

两类：

1. **改结果**：Filter / Sort / Distribution —— QB 与已保存 SQL 都可用
2. **改查询**：See these records / Break out by time|location|category / Zoom in / 更细时间粒度 —— **仅 QB**

权限：需要该数据源的 Create queries。下钻生成新问题，不改原问题。

仪表盘上可覆盖默认下钻，改为自定义点击行为（见 5.10.6）。

---

### 5.10 Dashboard `[OSS]`

#### 5.10.1 卡片类型

- **Question 卡**：来自集合，或「只属于该仪表盘」的问题
- **Heading**：全宽标题，可挂筛选器
- **Text**：Markdown（表格、代码、图片、可选变量 `{{x}}`、可选段落 `[[ ]]`）
- **Link**：搜内部对象或外链；外链 URL 可含变量
- **Iframe**：嵌视频/问卷/外部图；域名白名单；src 可含变量
- **Action 按钮**：触发回写（见 5.12）

规则：

- 已挂在其他仪表盘上的问题不能再挂到第二张仪表盘；要复用必须存到集合
- 个人集合里的问题只能挂到个人集合里的仪表盘

#### 5.10.2 布局

- 网格拖拽、缩放
- 多 Tab；复制 Tab；跨 Tab 移卡
- 可视化选项（卡级）
- 同一卡叠加额外系列（Add series）
- 自动刷新间隔
- 全屏播放
- 夜间/全屏演示
- 导出 PDF（整页仪表盘）
- 复制整张仪表盘
- 移动到其他集合、归档

#### 5.10.3 筛选器与参数

**过滤器（改「看什么」）**

- Date picker：月年 / 季年 / 单日 / 范围 / 相对 / All Options
- Location：City / State / ZIP / Country；运算符 Is/Is not/Contains/…
- ID：下拉/搜索/输入；单选/多选
- Number：= ≠ between ≥ ≤
- Text or category：Is / Is not / Contains / …
- Boolean

**参数（改「怎么看」）**

- Time grouping：日/周/月…（不删点，只改粒度）

**挂载位置**

- Dashboard 级：可跨所有 Tab（只连到当前 Tab 的卡时才显示）
- Header 级：仅当前 Tab
- Card 级：仅该卡

**配置点**

- 连接到哪些卡的哪些字段/变量
- 默认值、是否必填
- 可选值来源：字段扫描 / 自定义列表
- 联动筛选（Linked filters）：A 的取值约束 B，依赖 FK
- 订阅时可再套一层订阅筛选（EE）

#### 5.10.4 点击行为（Click behavior）

QB 卡三种：

1. 打开默认下钻菜单
2. 跳转自定义目的地：另一仪表盘 / 问题 / 外部 URL（可传列值到目标筛选器）
3. 更新本仪表盘某个筛选器（用图表当筛选器）

SQL 卡没有默认下钻菜单，只有 2 和 3。

内部目的地同页打开，外部 URL 新开标签。

#### 5.10.5 订阅（Dashboard subscriptions）

- 频道：Email / Slack / Webhook
- 日程：分/时/日/周/月 或 cron
- 附件格式、是否含 PDF
- 收件人：用户 / 邮箱 / Slack channel
- 退订链接
- 损坏订阅通知（broken subscription）
- EE：订阅级筛选器

#### 5.10.6 其他

- 仪表盘级缓存策略（EE 更细）
- 公开分享 / 嵌入
- 权限走集合权限
- iframe 域名白名单（Settings）

---

### 5.11 Documents `[OSS]`

- 富文本 + Markdown
- `/` 命令：Ask Metabot、插入 Chart、Link
- `@` 提及内部对象
- 插入已有问题/模型（**拷贝**一份进文档，互不影响）
- 在文档内新建 QB / SQL 问题（只属于文档；删除不可进回收站）
- 图表：改可视化、改查询、替换、下载、旁注文字
- 一行最多 3 个块，可横向拖宽
- 评论侧栏、通知邮件
- 公开文档（嵌入文档暂不支持）
- 保存到集合，走集合权限

---

### 5.12 Actions 与表编辑

#### 5.12.1 Actions `[OSS，目前仅 PG/MySQL]`

前置：

- 连接开启 Model actions
- 库账号有写权限
- 至少有一个 Model

类型：

- **Basic**：基于单表 Model 的 Insert / Update / Delete 自动表单
- **Custom**：参数化 SQL 回写，可与模型数据无关（不推荐但允许）

运行入口：

- Model 详情页 Run
- 仪表盘按钮（可把仪表盘筛选/行值传入参数）
- 公开 Action 表单（收集外部数据）

权限：

- 创建/编辑：需要 Native 查询权限
- 运行：有 Model/仪表盘查看权或公开链接即可

限制：

- 不写 Model 定义，只写底层表
- 无撤销
- 缓存可能导致界面暂时看不到变更
- 无自增 PK 时需手填可用 ID
- 公开仪表盘与 Guest Embed **不能**跑 Action；SSO Embed / Full App 可以

#### 5.12.2 Editable tables `[EE]`

- 在 Browse 里直接编辑表数据
- 与 Actions 不同：面向管理员/分析师改数，不是做业务表单

---

### 5.13 导出 `[OSS / EE 限额]`

问题导出：

- CSV / XLSX / JSON / PNG（图表）
- Formatted / Unformatted
- 透视表：导出已透视扁平表，或未透视聚合结果；**不是** Excel 原生 PivotTable

仪表盘：

- PDF
- 订阅附件

文档卡：单独下载

权限档位（EE）：

- 禁止
- 1 万行
- 100 万行
- 按表/schema 细分
- Native 下载要求整库都有下载权

嵌入里默认能下，只有 Pro/EE 能关掉下载。

---

### 5.14 告警 Alerts `[OSS]`

对象：单个 Question（仪表盘用订阅）。

触发类型：

1. **Results**：查询「有结果」就发（适合「出现差评」这类稀疏事件）
2. **Goal line**：折/柱/面积图越过目标线（上穿或下穿）
3. **Progress bar**：进度条达到或低于目标

日程：分钟/小时/日/周/月/cron
频道：Email / Slack / Webhook（Webhook 需 Settings 权限）
一次性告警：发送后自删
Send now 测试
Monitor 里集中管理（仅 Admin）
退订页

---

### 5.15 通知渠道 `[OSS]`

#### Email

- SMTP 配置、发件人、Reply-To、测试邮件
- Cloud 自定义 SMTP（EE/Cloud）
- 模板：邀请、重置密码、新设备、告警、订阅、评论、MFA、Transform 失败、持久化失败、Slack token 错误等
- 附件：CSV/XLSX + 图表 PNG/PDF
- 收件人限制 / 发件域名白名单（EE）

#### Slack

- Bot Token
- 频道/用户缓存刷新任务
- 告警与订阅投递
- Metabot in Slack（EE/AI）
- Token 失效通知

#### Webhook（HTTP Channel）

- 自定义 URL、Header
- 告警/订阅 JSON 回调
- 仅 Admin 或 Settings 权限可配

渲染栈：Handlebars 模板 + 服务端静态可视化（Graal JS）+ PDF 排版。

---

### 5.16 权限体系

权限是 **组 × 资源** 的图，存在 Permission Graph，带修订号。

#### 5.16.1 数据权限（库 / schema / 表）

| 维度 | 取值 | 级别 | 备注 |
| --- | --- | --- | --- |
| View data | Can view / Granular / Row&Column security / Impersonated / Blocked | 库/schema/表各不同 | OSS 实际只有 Can view；其余 EE |
| Create queries | QB+Native / 仅 QB / Granular / 无 | 库/schema/表 | Block 或沙箱任意表 ⇒ 整库禁 Native |
| Download results | 无 / 1万 / 100万 / Granular | EE | Native 需整库下载权 |
| Manage table metadata | Yes / No / Granular | EE | 改表字段元数据 |
| Manage database | Yes / No | EE | 改连接、手动 sync/scan；不能删库 |
| Transform | 是否可在该库执行转换 | EE | 另需 Data Analyst/Admin |

**Blocked 的产品语义**：即使集合可见，只要问题查了被 Block 的数据，也看不到。Block 一张表 ⇒ 同库所有 SQL 问题都不可见（因为不解析 SQL）。

多组取并集，更宽松优先。

#### 5.16.2 集合权限 `[OSS]`

- View
- Curate（改/移/删/置顶/新建子集合）
- No access
- 仪表盘可见但其中问题在无权限集合 ⇒ 该卡显示无权限占位
- 只有 Admin 能改集合权限
- EE：Snippet 文件夹权限

#### 5.16.3 应用权限 `[EE]`

- Settings：通用/邮件/Slack/Webhook/地图/本地化/外观/公开分享/嵌入/缓存
- Monitoring：Monitor 页、Help、只读日志（不含依赖诊断与告警管理）
- Subscriptions and alerts：能否创建订阅/告警

#### 5.16.4 行级 / 列级安全 `[EE]`

- 按用户属性过滤行
- 用保存的问题当沙箱视图
- 列级限制
- 仅表级可配

#### 5.16.5 Impersonation `[EE]`

- 查询时切换到数据库角色
- 权限完全交给数仓
- 仅库级

#### 5.16.6 Database Routing `[EE]`

- 同一逻辑库，按用户/租户连到不同物理库
- 与 Transform 互斥
- 嵌入多客户「每客户一个库」的主方案

#### 5.16.7 Tenants `[EE]`

- 嵌入 SaaS 多租户
- Tenant 集合、Tenant 用户个人集合
- 与 SSO / 权限一起做客户隔离

#### 5.16.8 嵌入权限

- Guest：靠 locked parameters 隔离
- SSO Embed：走完整数据权限 + Tenant
- 可为嵌入单独规划组与权限

---

### 5.17 嵌入与公开分享

#### 5.17.1 Public `[OSS]`

- 公开问题链接
- 公开仪表盘链接
- 公开文档
- 公开 Action 表单
- 公开 iframe 片段
- 无鉴权，有链接即可见
- Admin 可关公开分享总开关

#### 5.17.2 Guest Embed `[OSS]`

- 签名 JWT/token 的图表、仪表盘
- 筛选控件
- Locked parameters（服务端锁定，前端改不了）
- 基础外观
- **无**下钻、QB、SQL、AI、集合浏览器、高级主题、自定义可视化（回退默认图）

#### 5.17.3 Modular Embed + SSO `[EE 为主]`

组件：

- Question / Dashboard
- Query Builder
- SQL Editor
- AI Chat
- Collection Browser
- 数据浏览器

接入方式：

- Web Components：`<metabase-question>` 等，无构建步骤
- React SDK：可组合、可插件化改行为
- 应用内向导生成代码

能力：完整权限、下钻、主题、自定义可视化白名单、Usage Analytics、Tenant。

#### 5.17.4 Full App Embed `[EE]`

- 整个 Metabase 放进 iframe
- 与宿主 SSO 打通
- 可隐藏顶栏等 chrome

#### 5.17.5 嵌入管理

- Embedding Hub / Setup Guide：权限、SSO 向导
- 主题列表与编辑器
- Security：来源白名单、SDK origins
- 隐藏嵌入品牌（EE）
- 文档目前不能 modular embed
- Usage Analytics 记 embedding context / hostname / auth method

---

### 5.18 Data Studio 与 Transforms

#### 5.18.1 Transforms `[OSS 基础查询转换 / EE Python 与 Inspector]`

流程：在 Metabase 写查询或 Python → 在目标库 **创建/替换表** → 同步回 Metabase → 作为新数据源。

支持写出的引擎：BigQuery、ClickHouse Cloud、MySQL/MariaDB、PostgreSQL、Redshift、Snowflake、SQL Server。
不支持：Database Routing 的库、Sample Database。

对象：

- Transform 定义（QB 或 SQL 或 Python）
- Tag（如 daily/hourly）
- Job：按 cron 跑某 Tag 下所有 Transform
- Run：一次执行，替换目标表；成功/失败历史
- Inspector（EE）：数据流、Join、列分布
- DAG run：依赖顺序
- Metabot 可生成转换代码

权限：

- OSS：仅 Admin
- EE：Admin 或 Data Analyst + 该库 Transform 权限

Python Transform（EE）：

- 独立 runner
- S3 兼容存储（AWS S3 / MinIO）
- 与 SQL Transform 可混用

#### 5.18.2 血缘 Dependencies `[EE]`

- 内容依赖图
- 改表/模型前看影响面
- 一键替换数据源（所有引用一起换）
- 坏依赖进 Monitor > Dependency diagnostics

#### 5.18.3 Remote Sync / Serialization `[EE 为主]`

- 把集合/权限/问题等序列化成文件
- Git 同步（Data Studio > Git Sync）
- CLI 导入导出
- 多环境（开发→生产）搬运
- `resources/serialization-order.edn` 定义导出顺序

---

### 5.19 AI 能力 `[绝大多数 EE]`

#### 5.19.1 Metabot

入口：侧栏、Cmd/Ctrl+E、问题页图标、文档 `/Ask Metabot`、SQL 编辑器、Slack。

能做：

- 自然语言分析（先搜已有问题，没有再生成 QB）
- 生成/改写 SQL
- 修 SQL 报错
- 分析当前图表（表格则走 X-ray）
- 生成 Transform
- 在文档里生成图
- 反馈：赞/踩、重跑
- 术语走 Glossary

限制：会话不持久；换话题需重置；英文效果最好；结果必须人工复核。

管理：

- 模型/密钥/托管 AI
- 用量控制与审计（EE）
- 系统 Prompt 定制
- 文件式开发（file-based development）
- Privacy 说明

#### 5.19.2 Agent API

- 给外部 Agent 的稳定分析 API
- 与嵌入 AI Chat 配套

#### 5.19.3 MCP Server

- `/api/mcp` 与 `/api/metabase-mcp`
- 工具：读资源、跑查询、写/改 SQL、改图等
- OAuth 授权管理页
- Embed MCP 回调

#### 5.19.4 其他

- LLM 通用接口 `/api/llm`
- AI tracing / eval-trace
- OSI AI context
- Slackbot
- Entity analysis

---

### 5.20 性能、缓存与物化 `[OSS 基础 / EE 细粒度]`

- 实例级查询缓存
- 按数据库的缓存策略
- 按问题/仪表盘的策略（EE）
- 抢先缓存 preemptive caching（EE）
- Model Persistence：定时把 Model 写成表
- Persistence 日志与失败邮件
- 查询约束：超时、行数
- 报表时区
- 大结果流式下载，避免一次性进内存

---

### 5.21 上传 CSV `[OSS / EE 管理]`

- 指定上传目标库/schema
- 自动建表，默认可加 `_mb_row_id` 自增 PK
- 追加 / 替换
- EE：Upload management（治理上传表）

---

### 5.22 国际化、外观、本地化 `[OSS / EE 白标]`

- 界面语言（Crowdin）
- 用户个人语言
- 日期/数字/货币格式
- 报表时区 vs 数据库时区 vs 浏览器时区
- 实例外观：颜色、Logo、favicon、登录页（EE 白标更完整）
- 字体（含 PDF 字体）
- 深色模式（个人）
- 内容翻译（EE）：嵌入场景把字段值译成用户语言
- 自定义地图 GeoJSON
- 友好表名开关

---

### 5.23 监控、审计、反馈

#### Monitor `[OSS 部分 / EE Usage Analytics]`

- Erroring questions
- Alerts 管理
- Background tasks / Scheduled jobs（Quartz）
- Application logs
- Model persistence log
- Dependency diagnostics
- CLI analytics
- Health inspector
- Bug reporting
- Frontend error 上报

#### Usage Analytics `[EE]`

- 谁看了什么、跑了什么查询
- 嵌入用量
- Metabot 会话元数据（需显式打开收集）
- IP / UA / 路径 / 参数（默认关）
- 内置审计集合与参考仪表盘

#### 产品反馈 `[OSS]`

- `/api/product-feedback`
- 匿名 usage ping（可关）

---

### 5.24 开发者与扩展表面

- REST API（OpenAPI：`/api/docs`）
- 实体稳定 ID（entity id）与旧数字 ID 重定向
- Serialization / Remote Sync
- Driver 插件（`modules/drivers`、community drivers）
- 自定义可视化插件
- Embedding SDK / Web Components
- MCP / Agent API
- 配置文件 + 环境变量覆盖 Settings
- Jetty 线程/端口等 Web 服务器调优
- Data Apps（EE，较新）
- Metabot skills（`resources/metabot/skills/`）

---

## 6. 前端产品模块对照

便于在源码里找「这个功能在哪」。

| 前端目录 | 产品功能 |
| --- | --- |
| `query_builder` | Question / Notebook / Native / 结果 |
| `querying` | 查询状态与执行 |
| `visualizations` / `visualizer` / `static-viz` | 图表与静态渲染 |
| `dashboard` | 仪表盘 |
| `documents` / `rich_text_editing` / `comments` | 文档与评论 |
| `collections` | 集合 |
| `browse` | 浏览库/模型/指标 |
| `home` | 首页与 Onboarding |
| `search` | 搜索 |
| `reference` | Data Reference |
| `timelines` | 事件时间线 |
| `parameters` | 参数控件 |
| `actions` | Action 表单 |
| `models` / `metrics` / `metrics-viewer` | 模型与指标 |
| `metadata` | 表元数据编辑 |
| `data-studio` / `transforms` | Data Studio |
| `admin` | 管理后台 |
| `account` | 个人设置与通知 |
| `auth` / `setup` | 登录与安装 |
| `embedding` / `embedding-sdk` / `public` | 嵌入与公开 |
| `metabot` | AI 助手 |
| `notifications` / `pulse` | 告警遗留与通知 |
| `monitor` | 监控 |
| `detail-view` | 行详情 |
| `explorations` | 探索/自动分析 |
| `palette` | 命令面板 |
| `nav` | 导航 |
| `plugins` | EE 功能插槽（权限、租户、库、血缘等） |

后端几乎每个产品名词都有独立域模块：`queries`、`dashboards`、`collections`、`warehouses`、`warehouse_schema`、`permissions`、`notification`、`channel`、`search`、`sync`、`xrays`、`embedding`、`public_sharing`、`metabot`、`transforms`、`upload` 等，REST 再以 `*_rest` 暴露。

---

## 7. API 表面清单

前缀均为 `/api`。重构时可按资源做 Go handler。

| 路径 | 域 |
| --- | --- |
| `/session` `/setup` `/user` `/api-key` `/login-history` | 身份 |
| `/permissions` `/setting` | 权限与配置 |
| `/database` `/table` `/field` `/sync` `/notify` | 数据源与元数据 |
| `/dataset` `/card` `/cards` `/metric` `/measure` `/segment` | 查询与语义 |
| `/dashboard` `/collection` `/bookmark` `/search` `/activity` | 内容组织 |
| `/document` `/comment` `/timeline` `/timeline-event` | 文档与事件 |
| `/pulse` `/alert` `/notification` `/channel` `/email` `/slack` | 通知 |
| `/action` `/upload` `/persist` `/model-index` `/index` | 回写/物化/索引 |
| `/transform` `/transform-job` `/transform-tag` `/transform-dag-run` | 转换 |
| `/embed` `/embed-theme` `/preview_embed` `/public` | 嵌入与公开 |
| `/automagic-dashboards` `/tiles` `/geojson` | X-ray / 地图 |
| `/revision` `/cache` `/task` `/logger` `/bug-reporting` `/health-inspector` | 运维 |
| `/metabot` `/llm` `/agent` `/mcp` `/metabase-mcp` `/embed-mcp` | AI |
| `/native-query-snippet` `/glossary` `/data-studio` `/typed-schemas` | 语义与工作室 |
| `/google` `/ldap` `/oauth` `/premium-features` `/cloud-migration` | 认证/许可/迁移 |
| `/analytics` `/user-key-value` `/eid-translation` `/frontend-errors` `/util` `/docs` | 杂项 |

企业版路由优先于 OSS 路由注册。

---

## 8. 开源 vs 商业功能对照表

Topbase 默认应对齐左列；右列若要做，必须独立设计。

| 能力 | OSS | EE / Cloud |
| --- | --- | --- |
| 连接官方数据库、Sync/Scan、QB、SQL、可视化、仪表盘、文档 | ✓ | ✓ |
| 集合/个人集合/回收站/搜索/书签/修订 | ✓ | ✓ |
| Model / Metric / Segment / Snippet / Timeline / X-ray | ✓ | ✓ |
| Alert / Subscription / Email / Slack / Webhook | ✓ | ✓ |
| Public link / Guest embed | ✓ | ✓ |
| API Key、Google/LDAP 基础 SSO | ✓ | 增强 |
| 基础查询缓存、Model Persistence | ✓ | 更细策略 |
| CSV 上传、基础 Transforms | ✓ | Python Transform、Inspector、可写连接 |
| 数据权限 View=Can view，QB/Native | ✓ | Block / 行级安全 / Impersonation / Routing / 下载限额 / 元数据权 / 管库权 |
| 应用级权限、官方集合、内容验证、Snippet 文件夹 | | ✓ |
| SAML/JWT/OIDC/SCIM/MFA/关密码登录 | | ✓ |
| 白标、隐藏嵌入品牌、高级主题、内容翻译 | | ✓ |
| Modular SSO Embed、Full App、SDK 插件、Tenant | | ✓ |
| Library、血缘、Schema Viewer、Semantic Search | | ✓ |
| Metabot / MCP / Agent 完整能力、AI 用量审计 | | ✓ |
| Usage Analytics、Security Center、Serialization/Git Sync | | ✓ |
| 表编辑、Data Apps、Support Grant | | ✓ |

---

## 9. 后台任务地图（产品可感知的调度）

| 任务 | 作用 |
| --- | --- |
| Sync / Scan / Fingerprint | 元数据 |
| Session cleanup | 清过期会话 |
| Field values 过期 | 下拉缓存 14 天 |
| Query cache 刷新 / 抢先缓存 | 性能 |
| Model persistence refresh | 物化模型 |
| Alert / Subscription 投递 | 通知 |
| Slack 用户频道缓存 | Slack |
| Transform Job | 出数表 |
| Token cache / metering | EE 许可与计量 |
| Usage metadata 日批 | EE 审计 |
| Remote sync | Git |

---

## 10. 查询与内容的生命周期（端到端）

```
连库 → Sync 出 Table/Field
     → 管理员补语义类型 / FK / 格式
     →（可选）建 Model / Metric / Segment / Measure / Glossary
     → 用户用 QB 或 SQL 建 Question
     → 选可视化、设格式
     → 保存到 Collection 或 Dashboard
     → 仪表盘加筛选、点击行为、Tab、文本/iframe
     → 分享：集合权限 / 公开链 / 嵌入 / 订阅 / 告警
     → 他人下钻、评论、书签、搜索
     → 修订历史、回收站
     →（工程）Transform 写出新表 → 再 Sync → 被新问题使用
     →（治理）血缘、验证、Library、权限收紧
```

---

## 11. 对 Topbase Go 重构的功能优先级建议

结合 `topbase-master` 现状（PG 连接、SSH 隧道、元数据发现、只读 SQL、简单图表、数据浏览器、demo AI、飞书 OAuth 入口），建议按「开源主路径」分阶段对齐，而不是一次性铺 EE。

### 已有切片（保持并产品化）

- 数据源目录 + 连接校验 + 连接池
- SSH 跳板
- schema/table/column 发现
- 只读 SQL + 行数/超时限制
- 表结果 + 简易趋势图
- 管理向导 UI

### P0 — 能当内部 BI 用

1. **身份**：用户/组/会话/密码；飞书 OAuth 落地
2. **权限**：集合 View/Curate + 数据 View + Create queries（QB / Native）
3. **MBQL 子集 + Notebook**：选表、过滤、聚合、分组、排序、Limit、预览
4. **可视化 v1**：Table / Line / Bar / Area / Pie / Scalar / Number
5. **保存的 Question + Collection**
6. **Dashboard v1**：问题卡、网格、Tab、基础筛选（日期/分类/ID）
7. **导出 CSV**
8. **应用库**：用 Postgres 存实体，替代 `catalog.json`

### P1 — 语义与可信

9. 语义类型 + 格式 + FK 隐式 Join
10. Model / Metric / Segment
11. Native 参数（basic + field filter）
12. Drill-through（QB 改查询 + 表头过滤）
13. 修订历史、回收站、搜索、书签
14. 查询缓存、报表时区
15. 邮件告警 / 订阅（可先对接现有通知渠道）

### P2 — 分析师与嵌入

16. Join / 自定义表达式 / 分箱 / 时间分桶
17. 更多图表：Pivot、Map、Combo、Waterfall、Funnel
18. 仪表盘点击行为、Markdown/Heading 卡
19. Snippet、引用已保存问题
20. Public link + Guest embed
21. Sync/Scan/Fingerprint 调度
22. MySQL / ClickHouse 驱动

### P3 — 独立设计的「EE 级」能力（不要抄商业代码）

23. 行级安全、数据库路由、多租户
24. 血缘、Transform 调度器、Library
25. 真·Metabot / MCP（已有 provider 端口可接）
26. SSO 企业协议、SCIM、审计仓
27. 白标与 SDK 嵌入

---

## 12. 源码索引（按功能找模块）

| 想查 | 先看 |
| --- | --- |
| 产品文档目录 | `metabase-master/docs/README.md` |
| 前端路由 | `frontend/src/metabase/routes.tsx` |
| 管理路由 | `frontend/src/metabase/admin/routes.tsx` |
| Data Studio 路由 | `frontend/src/metabase/data-studio/routes.tsx` |
| API 路由表 | `src/metabase/api_routes/routes.clj` |
| 驱动能力 | `src/metabase/driver.clj` `def features` |
| QP 中间件 | `src/metabase/query_processor/middleware/` |
| 图表注册 | `frontend/src/metabase/visualizations/register.ts` |
| 商业功能开关 | `src/metabase/premium_features/settings.clj` |
| 企业模块列表 | `enterprise/backend/src/metabase_enterprise/` |
| X-ray 模板 | `resources/automagic_dashboards/` |
| Metabot skills | `resources/metabot/skills/` |
| 官方支持库 | `docs/databases/connecting.md` |

---

## 13. 附录：功能点检查清单（可当重构 backlog）

把每一行当成一个可验收的产品点。

### 接入与系统

- [ ] Setup 向导
- [ ] Sample Database
- [ ] 应用库迁移/备份
- [ ] Settings（站点名、URL、主页、时区、友好表名、X-ray 开关、iframe 白名单）
- [ ] 环境变量 / 配置文件覆盖
- [ ] 健康检查、日志、任务历史

### 数据源

- [ ] 多引擎连接表单
- [ ] SSL / SSH
- [ ] 测试连接
- [ ] Sync / Scan / Fingerprint 调度与手动
- [ ] 删库级联
- [ ] 上传 CSV
- [ ] 驱动 feature 矩阵

### 语义

- [ ] 物理类型映射与 Cast
- [ ] 语义类型全集
- [ ] 表/字段显示名、描述、显隐、格式
- [ ] FK / PK
- [ ] JSON unfolding
- [ ] Model / Persistence
- [ ] Metric / Segment / Measure
- [ ] Glossary

### 查询

- [ ] Notebook 全步骤
- [ ] 表达式函数按 feature 裁剪
- [ ] Native 编辑器 + 四种参数 + Snippet + Card 引用
- [ ] 转 SQL（单向）
- [ ] 预览原生 SQL
- [ ] 查询取消、超时、行限制
- [ ] 缓存
- [ ] Pivot 查询

### 可视化与下钻

- [ ] 19 种内置图 + List + ObjectDetail
- [ ] 表格条件格式与 Display as
- [ ] 地图三种模式 + 自定义 GeoJSON
- [ ] 下钻两类动作
- [ ] 静态图（邮件/PDF）

### 仪表盘与文档

- [ ] 六种卡
- [ ] Tab / 网格 / 自动刷新 / 全屏 / PDF
- [ ] 筛选器类型与三级挂载
- [ ] 联动筛选
- [ ] 点击行为三种
- [ ] 订阅
- [ ] 文档 `/` `@` 评论

### 组织与分发

- [ ] 集合嵌套、置顶、个人集合、回收站
- [ ] 搜索、书签、修订、时间线
- [ ] X-ray
- [ ] 公开链接
- [ ] Guest embed
- [ ] Alert 三种触发

### 权限与人

- [ ] 用户/组/邀请/停用
- [ ] 数据权限 + 集合权限
- [ ] API Key
- [ ] 登录历史
- [ ] （独立设计）行级安全 / 路由 / 租户

### 工程与 AI

- [ ] Transform + Job + Run
- [ ] （独立设计）血缘
- [ ] AI Chat → 可审查 SQL
- [ ] （独立设计）MCP / Agent

---

*文档生成自对 `metabase-master` 的只读拆解，供 Topbase 领域建模与排期使用。后续若要对某一域（例如 MBQL 子集或权限图）做「实现规格」，可按本章节再开专篇。*
