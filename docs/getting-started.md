# 快速开始

## 前置条件

- Go 版本以仓库 `go.mod` 为准
- Node.js 22 或更高版本，仅用于运行前端静态检查与测试
- 或安装 Docker 与 Docker Compose

## 使用 Docker Compose

```bash
cp .env.example .env
docker compose up --build -d
```

访问 <http://localhost:8080>，按照首次初始化页面创建管理员。运行状态和日志：

```bash
docker compose ps
docker compose logs -f topbase
curl --fail http://localhost:8080/api/ready
```

应用状态保存在命名卷 `topbase_data`。删除容器不会删除数据；执行 `docker compose down -v` 会删除数据卷，不应在生产环境使用。

需要备份时使用镜像内置工具，不要直接复制运行中的 SQLite 文件：

```bash
docker compose exec topbase /app/topbase-backup /backups/topbase-first-backup
docker compose cp topbase:/backups/topbase-first-backup ./backups/
```

## 从源代码运行

```bash
go mod download
make check
go run ./cmd/topbase
```

默认监听 `:8080`，运行数据写入 `./data`。需要并行启动测试实例时，使用独立地址与数据目录：

```bash
TOPBASE_ADDR=:18080 TOPBASE_DATA_DIR=/tmp/topbase-dev go run ./cmd/topbase
```

## 接入第一个数据库

1. 完成首次初始化并登录管理员账号。
2. 进入「管理后台 → 数据库」。
3. 选择数据库类型，填写基础连接信息；需要内网访问时切换到「SSH 跳板机」。
4. 点击「测试连接」。测试成功后才能保存。
5. 保存后等待结构同步，再从左侧「源数据」选择表。
6. 使用字段、筛选、汇总、Join 和排序构建分析，并选择表格或图表呈现。

测试连接会真实访问源数据库。建议创建只读账号，并仅授予需要分析的 schema 和表权限。支持范围见[数据库驱动矩阵](database-drivers.md)。
