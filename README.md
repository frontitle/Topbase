# Topbase

[![CI](https://github.com/frontitle/Topbase/actions/workflows/ci.yml/badge.svg)](https://github.com/frontitle/Topbase/actions/workflows/ci.yml)

Topbase 是面向中国团队的 Go 数据智能与轻量数据仓库产品。它以 Metabase 开源版的业务能力为兼容方向，但采用独立领域模型和实现；不得复制或混入 Metabase 的商业版代码。

Metabase 官方文档 `v0.63` 是当前产品行为基线。功能不能以“已有页面或 API”作为完成标准，必须按逐项 UX 与工程验收矩阵确认。

- [`docs/metabase-parity-matrix.md`](docs/metabase-parity-matrix.md)：官方能力基线、当前差距与完成条件
- [`docs/architecture.md`](docs/architecture.md)：依赖方向、扩展点、安全与 Definition of Done
- [`docs/topbase-架构与功能清单.md`](docs/topbase-架构与功能清单.md)：完整产品和领域规格
- [`docs/README.md`](docs/README.md)：快速开始、部署、配置、升级与数据库支持文档
- [`CONTRIBUTING.md`](CONTRIBUTING.md)：开发流程与 PR 质量门禁

## 当前可运行切片

- 应用库（开发默认 SQLite `data/app.db`）：设置、用户、会话、数据组、分析、数据源目录
- 首次访问引导 Setup，邮箱密码登录，HttpOnly 会话 Cookie
- 主流 SQL / OLAP 数据库真实连接：PostgreSQL、MySQL / MariaDB、ClickHouse、SQL Server、Oracle、SQLite；支持连接校验、SSL、网络数据库 SSH 跳板机与进程内连接池
- 工作台与管理后台菜单分离：管理员在前台看到「管理后台」入口
- 工作台「数据浏览」：选数据库 → 选表 → 立即以可筛选表格拉取真实数据（排序、列筛选、搜索）
- 分析列表 `/questions/`、分析详情 `/questions/:id/`；数据组列表 `/collections/`、数据组详情 `/collections/:id/`
- 表/字段别称、说明、语义类型与外键：仅管理后台 `/admin/datamodel/`
- 数据源管理：添加后可编辑连接，点修改会回填已保存的主机/库名/账号/密码/SSL/SSH
- 人员/权限/设置：`/admin/people/` `/admin/permissions/` `/admin/settings/`；模型浏览：`/browse/models/`
- QueryIR：Join、表达式、HAVING、分箱、模型/指标/分段、FK 隐式 Join、两类下钻、Native 字段筛选
- 数仓升级：已保存分析 / 模型 → 调度（replace 或增量 watermark）→ 立即运行 / Cron，写入 `warehouse.wh_*`，血缘与目录徽标
- AI 提案调度（需确认，描述含「增量」时用 `created_at` watermark）与飞书 Webhook 卡片（`FEISHU_WEBHOOK_URL`）
- 飞书部门同步为 `feishu_dept` 用户组（`FEISHU_APP_ID` / `FEISHU_APP_SECRET`）
- 仪表盘订阅：站内通知或飞书卡片，进程内 30s cron
- 仪表盘：网格、Tab、分析/标题/文本卡、日期筛选映射、点击更新筛选
- 搜索、书签、修订、回收站、CSV 导出、站内告警、API Key、权限图存储
- 数据组对象级授权与数据浏览 / 原生 SQL 能力授权均由服务端执行；浏览器写请求启用 CSRF 防护
- 应用库使用带校验和的顺序 migration；提供版本、就绪探针、优雅停止和在线一致性备份
- 可视化查询构建器走 QueryIR，可保存为分析
- schema / table / column 元数据发现与业务标注
- 只读事务、30 秒超时、1,000 行上限的 Native SQL API
- AI Chat 到可审查 SQL 的提供方端口（当前 demo provider）
- 带动效的中文数据工作台

## 运行

Docker Compose：

```bash
cp .env.example .env
docker compose up --build -d
curl --fail http://localhost:8080/api/ready
```

或从源代码运行：

```bash
go run ./cmd/topbase
```

打开 http://localhost:8080。未初始化时会跳转到 `/setup/`。

提交前运行完整质量门禁：

```bash
make check
```

数据目录默认 `data/`，可用 `TOPBASE_DATA_DIR` 覆盖（测试会写入临时目录）。

运行中的 Docker 实例可以在线备份：

```bash
docker compose exec topbase /app/topbase-backup /backups/topbase-manual
docker compose cp topbase:/backups/topbase-manual ./backups/
```

部署、升级和恢复细节见 [`docs/deployment.md`](docs/deployment.md) 与 [`docs/upgrading.md`](docs/upgrading.md)。

数据库接入能力、兼容产品和边界见 [`docs/database-drivers.md`](docs/database-drivers.md)。当前查询层支持多数据库，但数仓物化目标仍限定为 PostgreSQL，避免在尚未验证各引擎 DDL、事务和增量语义前给出错误承诺。

| 路径 | 用途 |
| --- | --- |
| `data/app.db` | 应用库（用户、会话、分析、数据组、目录） |
| `data/connection-secrets.json` | 数据源连接凭据，权限 `0600` |
| `data/catalog.json` | 旧版目录，仅在应用库为空时导入一次 |

当前应用库支持单实例 SQLite 部署；横向扩展前需要完成 PostgreSQL 应用库适配。生产环境还应把文件密钥存储替换为 KMS 或 Vault。

## 关键 API

- `GET /api/setup/status` · `POST /api/setup`
- `GET /api/health` · `GET /api/ready` · `GET /api/version`
- `GET /api/database-engines`（数据库驱动与能力声明）
- `POST /api/session` · `DELETE /api/session` · `GET /api/user/current`
- `POST /api/dataset` 提交 QueryIR
- `POST /api/dataset/drill` 下钻
- `POST /api/dataset/export` CSV
- `GET/PUT /api/databases/:id/tables/:schema/:table/fields` 语义与 FK
- `GET /api/databases/:id` · `GET /api/databases/:id/connection` · `PUT /api/databases/:id` · `POST /api/databases/:id/test` · `POST /api/databases/:id/sync` · `POST /api/databases/:id/tables/:schema/:table/rescan`
- `GET /api/user/current`（含 `is_admin`）· `GET/PUT /api/settings`
- `CRUD /api/models` `/api/metrics` `/api/segments` `/api/glossary`
- `POST /api/ai/propose-schedule` · `POST /api/schedules` · `POST /api/schedules/:id/run`
- `GET /api/warehouse/tables` · `GET /api/lineage/:type/:id`
- `GET /api/groups` · `POST /api/feishu/departments/sync`
- `GET/POST /api/dashboards/:id/subscriptions` · `POST /api/subscriptions/:id/run`
- `POST /api/questions` · `GET /api/questions` · `PUT /api/questions/:id`
- `GET/POST /api/collections` · `GET/PUT/DELETE /api/collections/:id`
- `GET/POST /api/dashboards` · `PUT /api/dashboards/:id` · `POST /api/dashboards/:id/cards/:cardId/dataset`
- `GET /api/search` · 书签 / 回收站 / 告警 / 通知
- `POST /api/databases/{id}/visual-query` 兼容入口，内部编译为 QueryIR

## 目标架构

`HTTP/API → application services → core ports → adapters`

当前能力大多仍处于“部分实现”。短期按 `P0` 收口连接、查询构建器、可视化、分析和仪表盘的完整体验，再进入语义层、权限和分发；Topbase 的飞书、AI 与数仓能力复用同一领域和权限基础。

## 合规

Metabase 开源版采用 AGPL；若直接复制、修改或分发其代码，必须遵守该许可证。Topbase 应保持独立实现，并在导入任何上游代码前完成许可证审查。

Topbase 自身的最终开源许可证尚待项目所有者确认；确认前不导入第三方源码。
