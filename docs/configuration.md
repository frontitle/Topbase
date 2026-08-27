# 配置参考

Topbase 使用环境变量配置进程。生产环境不要把真实值写入镜像、Compose 文件或 Git 仓库。

| 环境变量 | 默认值 | 用途 |
| --- | --- | --- |
| `TOPBASE_ADDR` | `:8080` | HTTP 监听地址 |
| `TOPBASE_DATA_DIR` | `./data` | 应用库、目录与开发密钥存储目录 |
| `TOPBASE_CRON` | 开启 | 设置为 `off` 时停用进程内调度，仅建议测试使用 |
| `TOPBASE_SECURE_COOKIES` | `false` | HTTPS 生产环境必须设为 `true`，为会话和 CSRF Cookie 启用 Secure |
| `TOPBASE_VERSION` | 开发版本 | Docker 构建时注入的发行版本 |
| `TOPBASE_COMMIT` | `unknown` | Docker 构建时注入的 Git commit |
| `TOPBASE_BUILD_TIME` | `unknown` | Docker 构建时注入的 UTC 构建时间 |
| `FEISHU_APP_ID` | 空 | 飞书应用 ID |
| `FEISHU_APP_SECRET` | 空 | 飞书应用密钥 |
| `FEISHU_WEBHOOK_URL` | 空 | 飞书通知 Webhook |

## 数据目录

数据目录当前包含：

| 文件 | 内容 | 备份要求 |
| --- | --- | --- |
| `app.db` | 用户、会话、数据源目录、分析、数据组、仪表盘、调度 | 必须备份 |
| `app.db-wal`、`app.db-shm` | SQLite 运行时文件 | 使用一致性备份，不要单独复制 |
| `connection-secrets.json` | 数据库密码、DSN、SSH 配置 | 必须加密备份并限制权限 |
| `catalog.json` | 旧版本目录导入文件 | 升级期保留 |
| `table-metadata.json` | 旧版本字段标注导入文件 | 升级期保留 |

开发环境的连接密钥文件权限为 `0600`。生产版本仍需接入 KMS 或 Vault 后，才能把文件秘密存储视为完成的生产安全能力。

## 反向代理

生产环境应在 Topbase 前部署支持 HTTPS 的反向代理，设置 `TOPBASE_SECURE_COOKIES=true`，并透传客户端地址与请求 ID。只对可信来源开放管理入口，数据库账号使用最小权限，SSH 私钥使用独立受限密钥。
