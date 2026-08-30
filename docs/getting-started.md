# 快速开始

## 前置条件

- Go 版本以仓库 `go.mod` 为准
- Node.js 22 或更高版本，仅用于运行前端静态检查与测试
- 或安装 Docker 与 Docker Compose

## 使用 Docker Compose

安装时先选择应用数据的保存方式：

- **生产正式模式**：使用 `docker-compose.postgres.yml` 或 `docker-compose.mysql.yml`，适合持续使用、升级、RDS 和多节点；
- **开发体验模式**：使用仓库兼容的 `docker-compose.yml` 或直接 `go run`，SQLite 保存在 `TOPBASE_DATA_DIR/app.db`，无需额外数据库。

SQLite 模式只用于本地体验和单机开发。Docker 必须持久化挂载 `/data`；如果数据目录只存在于容器可写层，重建或升级容器会清空项目数据。首次初始化页面会显示当前模式并要求确认风险。

```bash
cp .env.example .env
# 编辑 .env，至少填写：
# TOPBASE_APP_DB_PASSWORD=<随机数据库密码>
# TOPBASE_MASTER_KEY=<openssl rand -base64 32 的输出>
docker compose -f docker-compose.postgres.yml up --build -d
```

使用 MySQL 作为应用数据库时，另外填写 `TOPBASE_MYSQL_ROOT_PASSWORD`，并将文件名替换为 `docker-compose.mysql.yml`。

访问 <http://localhost:8101>，按照首次初始化页面创建管理员。运行状态和日志：

```bash
docker compose -f docker-compose.postgres.yml ps
docker compose -f docker-compose.postgres.yml logs -f topbase
curl --fail http://localhost:8101/api/ready
```

应用状态保存在 `topbase_appdb` PostgreSQL 命名卷，兼容文件保存在 `topbase_data`。删除容器不会删除数据；执行 `docker compose down -v` 会删除数据卷，不应在生产环境使用。

需要备份时使用 PostgreSQL 一致性导出，不要直接复制运行中的数据库目录：

```bash
docker compose -f docker-compose.postgres.yml exec -T appdb \
  pg_dump -U topbase -d topbase -Fc > topbase-first-backup.dump
```

## 从源代码运行

```bash
go mod download
make check
go run ./cmd/topbase
```

默认监听 `:80`。可以通过 `TOPBASE_PORT=9000` 只修改端口，或使用 `TOPBASE_ADDR=127.0.0.1:9000` 同时限制监听地址。未设置应用数据库变量时，为兼容旧安装使用 `./data/app.db`；新部署应设置 PostgreSQL 或 MySQL 应用数据库。需要并行启动测试实例时，使用独立地址与数据目录：

```bash
TOPBASE_ADDR=:18080 TOPBASE_DATA_DIR=/tmp/topbase-dev go run ./cmd/topbase
```

开发验证完成后，管理员可以进入“管理后台 → 设置 → 应用数据库与备份”，先下载逻辑备份，再把 SQLite 一次性迁移到空的 PostgreSQL 或 MySQL 数据库。迁移不会在运行中切换数据库；完成后必须保持原 `TOPBASE_MASTER_KEY`，修改部署环境变量并重启。

## 接入第一个数据库

1. 完成首次初始化并登录管理员账号。
2. 进入「管理后台 → 数据库」。
3. 选择数据库类型，填写基础连接信息；需要内网访问时切换到「SSH 跳板机」。
4. 点击「测试连接」。测试成功后才能保存。
5. 保存后等待结构同步，再从左侧「源数据」选择表。
6. 使用字段、筛选、汇总、Join 和排序构建分析，并选择表格或图表呈现。

测试连接会真实访问源数据库。建议创建只读账号，并仅授予需要分析的 schema 和表权限。支持范围见[数据库驱动矩阵](database-drivers.md)。
