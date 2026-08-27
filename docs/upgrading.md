# 版本、升级与备份

## 版本策略

Topbase 使用语义化版本：

- `0.x`：快速研发期，允许经过文档说明和 migration 的不兼容调整。
- `1.x`：核心连接、分析、可视化、仪表盘、权限和升级链路达到稳定验收后发布。
- 补丁版本只包含兼容修复；次版本增加向后兼容能力；主版本用于明确的不兼容变化。

每次发布使用 Git tag，例如 `v0.1.0`。发布说明必须列出新增能力、修复、安全变化、配置变化、migration 和已知限制。当前仓库尚未发布首个版本标签。

## 数据迁移原则

- `migrations/` 中的 migration 只追加，不修改已发布文件。
- 应用启动时按顺序初始化和升级应用库；任何需要长时间运行的数据改造应拆成可恢复后台任务。
- QueryIR、图表设置和连接配置分别保存版本号；读取旧数据时迁移到内存新结构，确认成功后再持久化。
- 源数据库不属于 Topbase 升级范围，默认只读；数仓物化表需要独立备份策略。

## 升级前备份

1. 停止 Topbase，避免复制 SQLite 时仍有写入。
2. 备份整个 `TOPBASE_DATA_DIR`，并对备份加密。
3. 记录当前镜像 tag 或 Git commit。
4. 在备份副本或预发布环境运行新版本，验证登录、数据库重连、分析、仪表盘和调度。
5. 再升级生产实例并检查 `/api/health` 和运行日志。

Docker Compose 示例：

```bash
docker compose stop topbase
docker run --rm -v topbase_topbase_data:/source:ro -v "$PWD/backups:/backup" alpine tar -czf /backup/topbase-data.tgz -C /source .
docker compose up --build -d
```

命名卷前缀取决于 Compose 项目名；执行备份前先用 `docker volume ls` 确认精确名称。

## 回滚

如果 migration 已经执行，不要仅回退二进制并继续使用被升级的应用库。停止服务，恢复升级前的完整数据目录，再启动原版本。迁移脚本需要在发布测试中验证“从每个受支持旧版本升级”，而不是只验证全新安装。
