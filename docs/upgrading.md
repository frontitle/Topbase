# 版本、升级与备份

## 版本策略

Topbase 使用语义化版本：

- `0.x`：快速研发期，允许经过文档说明和 migration 的不兼容调整。
- `1.x`：核心连接、分析、可视化、仪表盘、权限和升级链路达到稳定验收后发布。
- 补丁版本只包含兼容修复；次版本增加向后兼容能力；主版本用于明确的不兼容变化。

每次发布使用 Git tag，例如 `v0.1.0`。发布说明必须列出新增能力、修复、安全变化、配置变化、migration 和已知限制。当前仓库尚未发布首个版本标签。

## 数据迁移原则

- `migrations/` 中的 migration 只追加，不修改已发布文件；应用会校验已执行 migration 的 SHA-256，发现历史文件被改写时拒绝启动。
- 应用启动时在事务中按版本顺序升级应用库，并把版本、校验和、应用版本写入 `schema_migrations`；失败时不会以半升级状态继续提供服务。
- QueryIR、图表设置和连接配置分别保存版本号；读取旧数据时迁移到内存新结构，确认成功后再持久化。
- 源数据库不属于 Topbase 升级范围，默认只读；数仓物化表需要独立备份策略。

## 升级前备份

1. 使用内置备份命令生成 SQLite 一致性快照；运行中的服务无需停机。
2. 将备份导出到宿主机的加密存储。
3. 记录当前镜像 tag 或 Git commit。
4. 在备份副本或预发布环境运行新版本，验证登录、数据库重连、分析、仪表盘和调度。
5. 再升级生产实例并检查 `/api/health` 和运行日志。

Docker Compose 示例（备份名不能与已有目录重复）：

```bash
docker compose exec topbase /app/topbase-backup /backups/topbase-before-upgrade
mkdir -p backups
docker compose cp topbase:/backups/topbase-before-upgrade ./backups/
```

备份包含一致性的 `app.db`、连接 Secret、兼容目录文件和 `manifest.json`，文件权限为 `0600`。备份中含数据库密码和 SSH 私钥，必须加密并限制访问。不要只复制 `app.db-wal` 或 `app.db-shm`。

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
docker compose build --pull
docker compose up -d
curl --fail http://localhost:8080/api/ready
curl --fail http://localhost:8080/api/version
```

升级时 `/data` 和 `/backups` 命名卷不会随容器替换而删除。不要执行 `docker compose down -v`，该命令会删除命名卷。

## 回滚

如果 migration 已经执行，不要仅回退二进制并继续使用被升级的应用库。停止服务，恢复升级前的完整数据目录，再启动原版本。迁移脚本需要在发布测试中验证“从每个受支持旧版本升级”，而不是只验证全新安装。
