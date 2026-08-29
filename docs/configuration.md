# 配置参考

Topbase 使用环境变量配置进程。生产环境不要把真实值写入镜像、Compose 文件或 Git 仓库。

| 环境变量 | 默认值 | 用途 |
| --- | --- | --- |
| `TOPBASE_ADDR` | `:8080` | HTTP 监听地址 |
| `TOPBASE_DATA_DIR` | `./data` | 应用库、目录与开发密钥存储目录 |
| `TOPBASE_APP_DB_ENGINE` | 兼容模式为 `sqlite` | 新部署使用 `postgres` 或 `mysql` |
| `TOPBASE_APP_DB_DSN` | 空 | 完整应用库连接串；填写后优先于分项配置 |
| `TOPBASE_APP_DB_HOST` | 空 | 应用数据库内网或 RDS 地址 |
| `TOPBASE_APP_DB_PORT` | PostgreSQL `5432` / MySQL `3306` | 应用数据库端口 |
| `TOPBASE_APP_DB_NAME` | 空 | 专用于 Topbase 的数据库 |
| `TOPBASE_APP_DB_SCHEMA` | PostgreSQL 为 `public` | PostgreSQL 专用 Schema；MySQL 不使用 |
| `TOPBASE_APP_DB_USER` | 空 | 应用数据库账号 |
| `TOPBASE_APP_DB_PASSWORD` | 空 | 应用数据库密码 |
| `TOPBASE_APP_DB_TLS_MODE` | `prefer` / `preferred` | 生产 RDS 建议 `verify-full` |
| `TOPBASE_APP_DB_CA_FILE` | 空 | RDS CA PEM 证书路径 |
| `TOPBASE_APP_DB_MAX_OPEN_CONNS` | `20` | 单个 Topbase 节点最大应用库连接数 |
| `TOPBASE_APP_DB_MAX_IDLE_CONNS` | `5` | 单个节点空闲连接数 |
| `TOPBASE_MASTER_KEY` | SQLite 兼容模式自动生成 | 32 字节 base64/hex 主密钥；多节点必须完全一致 |
| `TOPBASE_MASTER_KEY_FILE` | 空 | 从只读 Secret 文件加载主密钥 |
| `TOPBASE_INSTANCE_ID` | 自动生成 | 多节点中可读且唯一的节点标识 |
| `TOPBASE_CRON` | 开启 | 设置为 `off` 时停用进程内调度，仅建议测试使用 |
| `TOPBASE_SECURE_COOKIES` | `false` | HTTPS 生产环境必须设为 `true`，为会话和 CSRF Cookie 启用 Secure |
| `TOPBASE_VERSION` | 开发版本 | Docker 构建时注入的发行版本 |
| `TOPBASE_COMMIT` | `unknown` | Docker 构建时注入的 Git commit |
| `TOPBASE_BUILD_TIME` | `unknown` | Docker 构建时注入的 UTC 构建时间 |
| `FEISHU_APP_ID` | 空 | 飞书应用 ID |
| `FEISHU_APP_SECRET` | 空 | 飞书应用密钥 |
| `FEISHU_WEBHOOK_URL` | 空 | 飞书通知 Webhook |

## 应用数据库初始化

Topbase 连接到指定数据库或 PostgreSQL Schema 后会执行以下判断：

- 空命名空间：创建当前完整结构和 migration 记录；
- 存在 `schema_migrations`：校验历史 migration 校验和并执行缺失版本；
- 存在其他表但没有 Topbase migration 记录：拒绝启动，防止修改业务数据库。

建议提前创建专用数据库和账号。首次初始化账号需要在目标数据库内创建、修改表和索引以及读写数据的权限，不要求创建整个 RDS 实例。严格权限环境可以在发布时使用 migration 账号，运行期再切换为只读写 Topbase 表的账号。

数据源连接密码、DSN、SSH 密钥会使用 AES-GCM 加密后保存在应用数据库。丢失 `TOPBASE_MASTER_KEY` 将无法恢复这些连接，备份应用数据库时必须同时以独立安全方式备份主密钥。

## 数据目录

数据目录当前包含：

| 文件 | 内容 | 备份要求 |
| --- | --- | --- |
| `app.db` | 旧版或开发兼容模式的 SQLite 应用库 | SQLite 部署必须备份 |
| `app.db-wal`、`app.db-shm` | SQLite 运行时文件 | 使用一致性备份，不要单独复制 |
| `master.key` | SQLite 兼容模式自动生成的连接加密主密钥 | 必须单独安全备份 |
| `connection-secrets.json` | 旧版本连接配置导入文件；新版本只导入缺失项 | 升级完成前保留并限制权限 |
| `catalog.json` | 旧版本目录导入文件 | 升级期保留 |
| `table-metadata.json` | 旧版本字段标注导入文件 | 升级期保留 |

RDS 和多节点部署不自动生成本地主密钥，必须通过部署平台的 Secret、KMS 或 Vault 注入同一个主密钥。

## 开发体验模式

未设置 `TOPBASE_APP_DB_ENGINE` 时，Topbase 使用 `TOPBASE_DATA_DIR/app.db` 的 SQLite 兼容模式。该模式无需安装数据库，只适合本地体验、开发和单节点临时验证。Docker 必须持久化挂载 `/data`；否则容器重建或升级会连同用户、分析、仪表盘和加密连接一起丢失。

管理员可在后台先导出逻辑备份，再一次性迁移到 PostgreSQL 或 MySQL 空库。迁移只复制数据，不修改当前进程配置；切换时保持 `TOPBASE_MASTER_KEY` 不变，并将目标连接参数写入部署环境后重启。

## 开发者模式

开发者模式与 API Key 不通过环境变量直接开启。完成首次安装后，由管理员在“管理后台 → 设置 → 开发者模式”中显式启用，并配置 Key 默认有效期、查询行数上限、是否允许新增分析以及客户端使用的对外访问地址。关闭开发者模式会立即阻断所有 MCP、CLI 和 API Key 请求，但不会删除已有密钥。

## 反向代理

生产环境应在 Topbase 前部署支持 HTTPS 的反向代理，设置 `TOPBASE_SECURE_COOKIES=true`，并透传客户端地址与请求 ID。只对可信来源开放管理入口，数据库账号使用最小权限，SSH 私钥使用独立受限密钥。
