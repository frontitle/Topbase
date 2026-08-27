# Metabase 能力对齐矩阵

本文是 Topbase 的产品验收基线，不是愿望清单。每个能力只有同时满足领域逻辑、API、权限、交互反馈、失败态和自动化测试，才可以标记为“完成”。

## 基线与边界

- 上游基线：Metabase 官方文档 `v0.63`，记录日期 `2026-08-26`。
- 对齐目标：开源版核心业务行为与用户心智；Topbase 使用 Go 独立实现，不复制 Metabase 源码或视觉资产。
- UI 基线：全站 Vercel 风格；数据表操作参考飞书多维表格；连接、保存、运行等核心操作必须有 loading、成功和可理解的失败反馈。
- 中国场景：飞书登录/组织/通知、中文默认文案、`Asia/Shanghai` 调度，以及“查询升级数仓”属于 Topbase 增量。
- 商业版能力：单独标记为 Extended，不把付费特性混入 OSS 完成率。

状态定义：

| 状态 | 定义 |
| --- | --- |
| 已完成 | 主路径、权限、异常、持久化和自动化测试均通过 |
| 部分 | 有可运行切片，但尚未满足完整验收条件 |
| 未开始 | 还没有用户可用的纵向能力 |

## P0：连接、查询、可视化、分析、仪表盘

| ID | Metabase 对应能力 | Topbase 当前状态 | 当前落点 | 完成验收条件 |
| --- | --- | --- | --- | --- |
| DB-01 | 管理后台添加/编辑/删除数据库 | 部分 | `/admin/`、`/api/databases`、`core.CatalogService` | 新增与编辑表单按驱动声明动态生成；必填/选填准确；测试连接不落库；保存成功后自动同步；错误定位到网络、SSH、TLS、认证或数据库；普通用户不可进入 |
| DB-02 | PostgreSQL 连接 | 部分 | `adapters.SQLConnector`、pgx | 直连、DSN、密码、超时、连接池、重连、删除清理均有集成测试；凭据不进入日志/API 明文响应 |
| DB-03 | SSL 与 SSH 隧道 | 部分 | `adapters/ssh_tunnel.go` | SSL/SSH Tab 可独立切换并保留输入；密码和私钥认证真实可用；主机指纹选填但填写后强校验；测试/保存按钮固定；成功/失败反馈可见 |
| DB-04 | 元数据同步与字段扫描 | 部分 | Schema snapshot、`sync`/`rescan` API | 后台任务化；记录开始/结束/失败；支持 schema 包含/排除、失效表字段、FK/PK、字段值扫描、指纹和可配置周期；浏览默认读快照而非阻塞源库 |
| DB-05 | 多数据库驱动 | 部分 | `adapters.EngineDefinitions`、`SQLConnector`、`CompileForEngine`；PostgreSQL、MySQL / MariaDB、ClickHouse、SQL Server、Oracle、SQLite | 当前六类引擎已支持真实测连、查询、可视化 QueryIR、元数据/注释/主外键同步；继续把连接字段 schema 与 capability 完全注册表化，补齐各引擎容器集成测试、取消查询和云数据库身份认证；非 SQL 数据源另建连接器 |
| QB-01 | 图形化查询步骤 | 部分 | `/data/`、`queryir.Query` | 可按顺序添加数据源、Join、表达式、筛选、汇总/分组、排序、Limit；每步可编辑/删除/重排；状态可序列化恢复 |
| QB-02 | 数据源选择 | 部分 | 表、模型、指标、已保存分析实体已有 | 选择器统一搜索数据库表、模型、指标、分析；显示位置与权限；支持新标签页预览；无权限对象不出现 |
| QB-03 | 字段选择与隐藏 | 部分 | QueryIR `fields`、表格列设置 | “从结果排除”和“仅在可视化隐藏”严格区分；被排除字段仍可用于筛选；隐藏列不作为安全控制；列设置持久化 |
| QB-04 | 筛选 | 部分 | QueryIR `Filter`、UI 筛选器 | 类型感知操作符、AND/OR 组、空值、相对日期、字段值列表/搜索/输入、Segment；无效组合不可提交；筛选摘要可读 |
| QB-05 | Join | 部分 | QueryIR `Join`、多引擎 compiler | FK 推荐、手工字段条件、多个条件、Left/Inner/Right/Full（按驱动能力）、别名、多次联同一表；字段冲突清晰；权限逐表校验 |
| QB-06 | 自定义列与表达式 | 部分 | QueryIR `Expression` | 表达式编辑器有字段/函数补全、类型检查、错误位置、数学/文本/日期/逻辑函数；查询级字段不写回源表 |
| QB-07 | 汇总、分组、排序、限制 | 部分 | Aggregation/GroupBy/Having/OrderBy/Limit | 多聚合、多维度、时间粒度、分箱、累计、聚合后筛选、多字段排序；浏览器显示上限与下载上限分离 |
| QB-08 | 每一步预览 | 未开始 | — | 任一步可运行到该步骤并预览前 10 行；取消旧请求；展示耗时与错误；不覆盖完整结果 |
| QB-09 | 查看/转换 SQL | 部分 | Dataset 返回编译 SQL；Native API | 按权限显示 SQL 侧栏；复制；Query Builder 单向转 Native 前二次确认；转换后保留名称/可视化；禁止逆向伪转换 |
| QB-10 | 下钻 | 部分 | `/api/dataset/drill` | 单元格值、列头、时序、聚合点提供上下文相关动作；生成的新 QueryIR 可继续编辑和另存；Native 只提供有限动作 |
| SQL-01 | Native SQL 编辑器 | 部分 | `/api/queries/run`、Native question | 编辑器、运行/取消、错误行列、历史、格式化、只读保护、参数、可保存；管理员可配置超时与最大结果 |
| SQL-02 | SQL 参数与字段筛选器 | 部分 | `queryir.ApplyNative` | 基础变量、可选块、字段筛选器、日期分组、表变量、默认值、URL 参数；绑定参数不字符串拼接 |
| VIZ-01 | 自动选图与图表切换 | 部分 | `app/viz`、ECharts | 根据列语义/基数/聚合推断；用户可切换且不丢设置；不兼容时说明原因而非静默失败 |
| VIZ-02 | 核心可视化 | 部分 | 表、部分图表 | 表格、数字、趋势、折线、柱、面积、组合、饼/环、散点、漏斗、进度、仪表、透视；每种有字段映射、格式、空态、tooltip、图例和响应式验收 |
| VIZ-03 | 友好表格 | 部分 | `/data/` grid | 冻结/拖拽/缩放/隐藏/排序/筛选、分页或虚拟滚动、类型化单元格、复制、行详情；大数据量保持流畅；键盘与无障碍可用 |
| Q-01 | 保存和管理分析 | 部分 | `/questions/`、Question store | 新建/另存/覆盖、命名/说明/数据组、移动/复制/归档/恢复、修订历史、权限、最近浏览和搜索均完整 |
| DASH-01 | 仪表盘列表与创建 | 部分 | `/dashboard/` | 页面直接列出所有可见仪表盘；一键新建并自动命名；空仪表盘直接进入编辑态；创建/跳转失败有反馈 |
| DASH-02 | 编辑仪表盘 | 部分 | `/dashboard/:id/` | 编辑态左侧列出可添加分析；添加后立即进入网格；标题/文本/链接/iframe、拖拽缩放、Tab、撤销、离开未保存确认；保存原子化 |
| DASH-03 | 查看态与交互 | 部分 | Dashboard cards/filter mapping | 非编辑态才显示分享/嵌入等；筛选映射、联动筛选、点击行为、多序列、自动刷新、全屏、卡片错误隔离完整 |
| DASH-04 | 订阅、导出和分享 | 部分 | subscriptions/public link/embed | 定时订阅到站内/飞书/邮件；公开链接总开关与撤销；导出 CSV/XLSX/JSON/PDF/PNG；分享行为受权限和管理员策略约束 |

## P1：语义层、组织、权限与治理

| ID | 能力 | 状态 | 完成验收条件 |
| --- | --- | --- | --- |
| META-01 | 表/字段元数据 | 部分 | 展示名、说明、可见性、实体键/FK、语义类型、格式、字段值策略、Cast 均可编辑并影响 Query Builder、筛选器和可视化 |
| META-02 | Models | 部分 | 从 QueryIR/Native 创建；列元数据；预览、引用、修订、持久化、依赖追踪、权限与失效诊断完整 |
| META-03 | Metrics / Segments | 部分 | 可创建、编辑、归档、修订；在数据选择/汇总/筛选顶部可发现；引用展开稳定且有循环检测 |
| ORG-01 | Collections | 部分 | 嵌套、移动、复制、固定、个人数据组、权限继承、事件时间线、空数据组删除、归档/恢复与搜索完整 |
| ORG-02 | 搜索、书签、历史、回收站 | 部分 | 全局统一搜索；对象级权限过滤；最近浏览；修订对比/恢复；归档依赖提示；回收站永久删除需二次确认 |
| AUTH-01 | 本地认证和账户 | 部分 | Setup、登录、退出、邀请、激活/停用、忘记/重置密码、会话过期、个人偏好、登录历史和 CSRF 防护完整 |
| AUTH-02 | 飞书登录和组织 | 未开始 | OAuth 回调、账号绑定、防重放、部门与成员增量同步、离职停用策略、头像/姓名更新、审计和可配置映射完整 |
| PERM-01 | 数据权限 | 部分 | 后端在目录、QueryIR、Native、下载、仪表盘、公开/嵌入每条路径统一求值；支持 view、builder、native 到表级；拒绝默认 |
| PERM-02 | 数据组和应用权限 | 部分 | view/curate、继承、个人数据组、管理入口/订阅/下载等应用权限；前端隐藏与后端拒绝一致；权限变更有审计 |
| GOV-01 | 依赖和血缘 | 部分 | 分析、模型、指标、仪表盘、调度、数仓表形成有向图；删除前影响分析；循环/断裂诊断；可追到源表字段 |

## P2：分发、嵌入、运维

| ID | 能力 | 状态 | 完成验收条件 |
| --- | --- | --- | --- |
| DIST-01 | Alerts | 部分 | 结果出现、目标线、变化等条件；调度、时区、一次性、退订、失败重试、权限变化处理和运行历史 |
| DIST-02 | 通知渠道 | 部分 | 飞书、邮件、Webhook 统一端口；模板预览、测试发送、签名/密钥、重试退避、幂等、死信和审计 |
| EMBED-01 | Public/Guest embed | 部分 | 实例总开关、逐对象令牌、撤销、参数锁定、frame 策略、限流、权限边界、无登录体验与安全文档 |
| OPS-01 | 配置和运维 | 未开始 | 配置文件/环境变量优先级；结构化日志、健康/就绪、Prometheus 指标、任务页、备份恢复、升级迁移说明 |
| PERF-01 | 查询生命周期 | 未开始 | 取消、排队、并发限制、超时、缓存、去重、查询历史、慢查询、资源指标和可配置行数/下载限制 |

## P3：Topbase 差异化

| ID | 能力 | 状态 | 完成验收条件 |
| --- | --- | --- | --- |
| TB-AI-01 | AI Chat 分析 | 部分 | AI 只生成可审查 QueryIR/SQL；展示依据和风险；继承用户权限；执行/保存/分享/调度需明确确认；全链路审计和可插拔模型供应商 |
| TB-WH-01 | 查询升级数仓 | 部分 | 已保存分析/模型一键生成计划；replace/incremental；预览目标 DDL；幂等运行、锁、重试、watermark、血缘、运行历史、飞书通知 |
| TB-UX-01 | Vercel + 飞书表格体验 | 部分 | 全局 token 与组件统一；核心按钮固定；弹层无不必要内部长滚动；Tab 保持草稿；异步反馈一致；响应式、键盘和 WCAG AA 基线 |

## 完成度规则

1. PR 必须填写本矩阵 ID，例如 `QB-04`，并链接对应自动化测试。
2. 仅新增路由、表或按钮不能把状态改为“已完成”。
3. “已完成”至少包含：正常路径、空态、加载态、权限拒绝、服务端校验、可恢复错误、持久化重开、自动化测试。
4. 同一能力的 Web、API、公开页、嵌入页必须共用应用服务和权限策略，禁止各写一套业务逻辑。
5. 每次跟进 Metabase 新版本先更新本文基线，再评估迁移；不得静默改变既有 Topbase 行为。

## 官方参考

- [Metabase v0.63 文档目录](https://www.metabase.com/docs/latest/)
- [Query Builder](https://www.metabase.com/docs/latest/questions/query-builder/editor)
- [Dashboards](https://www.metabase.com/docs/latest/dashboards/introduction)
- [Data modeling](https://www.metabase.com/docs/latest/data-modeling/start)
- [Adding and managing databases](https://www.metabase.com/docs/latest/databases/connecting)
- [Collections](https://www.metabase.com/docs/latest/exploration-and-organization/collections)
- [Permissions](https://www.metabase.com/docs/latest/permissions/start)
