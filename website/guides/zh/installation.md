# 安装与运行 Topbase

可以使用 Docker Compose 启动正式镜像和 PostgreSQL，也可以编译 Go 程序并直接运行。首次启动后，在浏览器中打开 Topbase 并创建管理员账号。随后可用六笔模拟订单[创建第一张仪表盘](getting-started.html)。

## 使用 Docker Compose 安装

1. 在服务器上安装 Git、OpenSSL、Docker 和 Compose 插件。下载发布版本并创建本地配置文件。

   ```bash
   git clone --branch v0.2.6 https://github.com/frontitle/Topbase.git
   cd Topbase
   cp -n .env.example .env
   chmod 600 .env
   openssl rand -base64 32
   ```

2. 用文本编辑器打开 `.env`。为内置 PostgreSQL 数据库设置强密码 `TOPBASE_APP_DB_PASSWORD`，将 OpenSSL 输出的完整一行填入 `TOPBASE_MASTER_KEY`。主密钥用于加密已保存的连接信息，请单独安全备份以便恢复。保存文件。
3. 启动服务，等待就绪检查通过。

   ```bash
   docker compose -f docker-compose.release.yml up -d
   curl --fail http://127.0.0.1:8101/api/ready
   ```

4. 在浏览器打开 `http://服务器IP:8101`。默认映射为 **服务器 8101 → 容器 8080**。如果希望使用服务器的 9000 端口，在 `.env` 中设置 `TOPBASE_HTTP_PORT=9000`，重新执行启动命令，并将浏览器地址和下方检查命令中的端口改为 9000。

发布镜像支持 Linux amd64 和 arm64。PostgreSQL 应用数据和 Topbase 数据目录保存在命名卷中，更换容器时请保留这些卷。需要使用公网 HTTPS 时，配置反向代理，在 `.env` 中设置 `TOPBASE_SECURE_COOKIES=true` 并重新执行启动命令，再设置[工作区对外地址](workspace.html)。

## 在服务器上直接运行

安装 Git 和 [go.mod](https://github.com/frontitle/Topbase/blob/v0.2.6/go.mod) 中指定的 Go 版本。在终端执行以下命令。

```bash
git clone --branch v0.2.6 https://github.com/frontitle/Topbase.git
cd Topbase
go build -o topbase ./cmd/topbase
TOPBASE_PORT=8080 TOPBASE_DATA_DIR=./data ./topbase
```

打开 `http://服务器IP:8080`，保持程序运行。未接入外部应用数据库时，账号、分析和仪表盘保存在 `./data` 下的 SQLite 中，重启时请保留此目录。长期托管可使用 [systemd 服务模板](https://github.com/frontitle/Topbase/blob/v0.2.6/deploy/systemd/topbase.service)。

## 核对配置与查询限制

| 配置项 | 用途 |
| --- | --- |
| `TOPBASE_PORT`、`TOPBASE_ADDR` | 直接运行的服务默认监听 80 端口。`TOPBASE_ADDR` 可设置完整地址，例如 `127.0.0.1:8080`，优先于 `TOPBASE_PORT`。 |
| `TOPBASE_HTTP_PORT` | Compose 对外发布的服务器端口。容器内应用监听 8080。 |
| `TOPBASE_APP_DB_ENGINE` | 保存账号和配置的应用数据库。正式镜像 Compose 使用 PostgreSQL，基础开发版 Compose 使用 SQLite。 |
| `TOPBASE_MAX_CONCURRENT_QUERIES` | 每个节点默认允许 8 个交互查询并发执行。 |
| `TOPBASE_QUERY_RESULT_MAX_BYTES`、`TOPBASE_QUERY_CELL_MAX_BYTES` | 单次结果默认最多 16 MiB，单元格默认最多 1 MiB。结果触及限制时，可缩小日期范围或先聚合数据。 |

更多配置见[环境变量模板](https://github.com/frontitle/Topbase/blob/v0.2.6/.env.example)。访问 `/api/ready` 可检查应用数据库是否就绪，`/api/version` 返回运行中的版本号。

## 备份 Docker 实例

在 Compose 项目目录中执行以下命令，将 PostgreSQL 导出到用户主目录下新建的私有目录。导出使用容器内实际配置的数据库账号和库名。

```bash
backup_dir=$(mktemp -d "$HOME/topbase-backup.XXXXXXXX") &&
docker compose -f docker-compose.release.yml exec -T appdb \
  sh -c 'exec pg_dump -Fc -U "$POSTGRES_USER" -d "$POSTGRES_DB"' \
  > "$backup_dir/appdb.dump"
```

命令成功后，将导出文件保存在受保护的备份存储中。同时保留 Topbase 数据卷，并单独安全备份主密钥。已连接的业务数据库需要通过其自身的工具备份。

## 切换 Docker 版本

先备份实例，再到项目目录拉取标签并切换到目标版本。以 0.2.6 为例：

```bash
git fetch --tags
git checkout v0.2.6
```

编辑**已有的** `.env`，设置 `TOPBASE_VERSION=0.2.6`，保留原有密码和主密钥。检查 Compose 最终选择的镜像：

```bash
docker compose -f docker-compose.release.yml config --images
```

本例应出现 `ghcr.io/frontitle/topbase:0.2.6`。如果仍显示其他版本，检查终端中是否导出了覆盖 `.env` 的 `TOPBASE_VERSION`。确认镜像标签后执行：

```bash
docker compose -f docker-compose.release.yml pull
docker compose -f docker-compose.release.yml up -d
curl --fail http://127.0.0.1:8101/api/ready
curl --fail http://127.0.0.1:8101/api/version
```

核对返回值中的 `version`，再打开一份已保存的分析或仪表盘。
