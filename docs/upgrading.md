# 版本、升级与备份

## 版本策略

Topbase 使用语义化版本：

- `0.x`：快速研发期，允许经过文档说明和 migration 的不兼容调整。
- `1.x`：核心连接、分析、可视化、仪表盘、权限和升级链路达到稳定验收后发布。
- 补丁版本只包含兼容修复；次版本增加向后兼容能力；主版本用于明确的不兼容变化。

每次整体功能升级由维护者确认版本号，版本源统一保存在仓库根目录 `VERSION`。发布提交必须同步更新 `CHANGELOG.md`、对应的 `docs/releases/<版本>.md`、README 和部署示例，然后创建同版本 Git tag，例如 `v0.2.1`。Tag 必须与 `VERSION` 完全一致。

从 `0.2.1` 开始，补丁版本用于兼容修复，次版本用于向后兼容的整体能力升级。Git tag 会触发发布门禁、多架构镜像构建、GHCR 推送和 GitHub Release。发布记录见 [更新日志](../CHANGELOG.md)。

## 数据迁移原则

- `migrations/` 中的 migration 只追加，不修改已发布文件；应用会校验已执行 migration 的 SHA-256，发现历史文件被改写时拒绝启动。
- 应用启动时通过数据库全局锁串行升级应用库，并把版本、校验和、应用版本写入 `schema_migrations`；失败时不会以半升级状态继续提供服务。
- QueryIR、图表设置和连接配置分别保存版本号；读取旧数据时迁移到内存新结构，确认成功后再持久化。
- 源数据库不属于 Topbase 升级范围，默认只读；数仓物化表需要独立备份策略。

## 升级前备份

1. PostgreSQL/MySQL 使用云数据库快照、PITR 或对应逻辑导出；SQLite 兼容部署使用内置一致性备份。
2. 将备份导出到宿主机的加密存储。
3. 记录当前镜像 tag 或 Git commit。
4. 在备份副本或预发布环境运行新版本，验证登录、数据库重连、分析、仪表盘和调度。
5. 再升级生产实例并检查 `/api/health` 和运行日志。

内置 PostgreSQL Compose 示例：

```bash
docker compose -f docker-compose.postgres.yml exec -T appdb \
  pg_dump -U topbase -d topbase -Fc > topbase-before-upgrade.dump
```

应用数据库包含加密后的数据库密码和 SSH 私钥，备份仍必须加密并限制访问。`TOPBASE_MASTER_KEY` 不应和数据库备份存放在同一位置，但二者必须配套保留。

管理员也可以在“管理后台 → 设置 → 应用数据库与备份”下载引擎无关的逻辑 ZIP。它包含 manifest、Schema 版本和每张应用表的 JSONL 数据，可用于审计与迁移辅助。该文件包含密码哈希和加密后的连接秘密，安全等级应与数据库完整备份一致。生产灾难恢复仍优先使用 RDS 快照、PITR 或数据库原生逻辑备份。

非 Docker 部署可以执行：

```bash
make backup
# 或指定不可已存在的目标目录
TOPBASE_DATA_DIR=/var/lib/topbase ./bin/topbase-backup /secure/backups/topbase-before-upgrade
```

## Docker 升级

```bash
git fetch --tags
git checkout <目标版本标签>
docker compose -f docker-compose.postgres.yml build --pull
docker compose -f docker-compose.postgres.yml up -d
curl --fail http://localhost:8080/api/ready
curl --fail http://localhost:8080/api/version
```

使用正式镜像时执行：

```bash
export TOPBASE_VERSION=<目标版本号>
docker compose -f docker-compose.release.yml pull
docker compose -f docker-compose.release.yml up -d
```

升级时 `/data` 和 `/backups` 命名卷不会随容器替换而删除。不要执行 `docker compose down -v`，该命令会删除命名卷。

## 回滚

如果 migration 已经执行，不要仅回退二进制并继续使用被升级的应用库。停止服务，恢复升级前的完整数据目录，再启动原版本。迁移脚本需要在发布测试中验证“从每个受支持旧版本升级”，而不是只验证全新安装。
