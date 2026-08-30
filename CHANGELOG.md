# 更新日志

Topbase 遵循语义化版本。每个正式版本对应 Git tag、升级说明和可复现构建元数据。

## [0.2.2] - 2026-08-30

### 新增

- 引入 Topbase 正式产品 Logo，并应用到工作台、管理后台和公开仓库首页。
- 支持以 `TOPBASE_PORT` 配置直接运行端口，并可选使用 `TOPBASE_TLS_CERT_FILE` 与 `TOPBASE_TLS_KEY_FILE` 直接提供 HTTPS。
- 管理后台可保存公开协议、域名和端口，用于生成分享、嵌入和 OAuth 的外部地址。

### 调整

- Docker 默认将宿主机 `8101` 映射至容器内的 Topbase 服务端口；部署和升级文档同步更新。
- 精简工作台与管理后台的交互与可视化编辑体验，并修复既有安装的 migration 010 校验兼容性。

### 升级

完整步骤和验收清单见 [0.2.2 升级说明](docs/releases/0.2.2.md)。

## [0.2.1] - 2026-08-30

### 新增

- 完成从数据源连接、实时浏览、可视化分析到仪表盘的核心工作流。
- 新增 PostgreSQL、MySQL 和 SQLite 应用数据库模式，以及 SQLite 到生产数据库的一次性迁移。
- 新增应用数据库逻辑备份、共享应用数据库、多节点迁移锁和调度租约。
- 新增开发者模式、API Key、MCP 与 CLI，支持受权限约束的数据问答和分析创建。
- 新增个人中心、身份绑定、权限、订阅、公开链接与嵌入管理。
- 新增源码部署、Docker Compose、RDS 和多架构容器发布方案。

### 调整

- 应用数据库 Schema 升级到版本 14。
- 数据源连接秘密迁移到 AES-GCM 加密存储，生产环境必须长期保留 `TOPBASE_MASTER_KEY`。
- 开发者模式默认关闭，升级后需要管理员显式启用既有 API Key。

### 升级

完整步骤和兼容性说明见 [0.2.1 升级说明](docs/releases/0.2.1.md)。

[0.2.1]: https://github.com/frontitle/Topbase/releases/tag/v0.2.1
[0.2.2]: https://github.com/frontitle/Topbase/releases/tag/v0.2.2
