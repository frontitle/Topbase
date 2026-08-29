# 部署 Topbase

## Docker Compose 正式镜像

正式版本发布到 GitHub Container Registry，同时提供 `linux/amd64` 和 `linux/arm64` 镜像。推荐使用版本标签，不要让生产环境长期跟随 `latest`：

```bash
git clone https://github.com/frontitle/Topbase.git
cd Topbase
git checkout v0.2.1
cp .env.example .env
# 编辑 .env，设置随机的 TOPBASE_APP_DB_PASSWORD 和 TOPBASE_MASTER_KEY
docker compose -f docker-compose.release.yml up -d
```

升级时修改 `TOPBASE_VERSION` 或切换目标 Git tag，再执行 `pull` 和 `up -d`。发布镜像使用非 root 用户、只读根文件系统、健康检查和独立数据卷。

## Docker Compose 源码构建

仓库提供多阶段 `Dockerfile`。运行容器使用固定 UID `10001` 的非 root 用户、只读根文件系统并移除 Linux capabilities。新安装推荐使用 `docker-compose.postgres.yml`，应用状态保存在独立 PostgreSQL 数据卷。

```bash
git clone git@github.com:frontitle/Topbase.git
cd Topbase
cp .env.example .env
# 编辑 .env，设置随机的 TOPBASE_APP_DB_PASSWORD 和 TOPBASE_MASTER_KEY
docker compose -f docker-compose.postgres.yml up --build -d
```

验证：

```bash
curl --fail http://localhost:8080/api/ready
curl --fail http://localhost:8080/api/version
docker compose -f docker-compose.postgres.yml ps
```

需要 MySQL 时，另在 `.env` 设置随机 `TOPBASE_MYSQL_ROOT_PASSWORD`，并使用 `docker-compose.mysql.yml`。两个 Compose 方案都只把数据库开放给容器内部网络。

生产部署时应把 `8080` 只暴露给反向代理，由代理提供 HTTPS，并设置 `TOPBASE_SECURE_COOKIES=true`。飞书等秘密通过部署平台的 Secret 功能注入，不要直接写进 `.env` 后提交。

容器会响应 `SIGTERM`，停止接收请求、等待在途请求结束，并关闭源数据库连接和 SSH 隧道。`/api/health` 表示进程存活；`/api/ready` 还会验证应用库可用并返回当前 schema 版本。

## SQLite 开发体验与生产迁移

`docker-compose.yml` 和未配置应用数据库环境变量的源码运行保留 SQLite 兼容模式。它适合首次体验和本地开发，但必须把 `/data` 持久化到命名卷或宿主机目录。容器重建不会保留可写层中的文件；`docker compose down -v` 还会主动删除命名卷。

迁移到正式环境：

1. 在“管理后台 → 设置 → 应用数据库与备份”下载逻辑 ZIP 备份；
2. 准备专用于 Topbase 的空 PostgreSQL 或 MySQL 数据库/RDS；
3. 在后台填写目标连接并确认迁移；
4. 等待系统在单个目标事务内复制并核对每张表的行数；
5. 将相同参数写入 `TOPBASE_APP_DB_*`，保持原 `TOPBASE_MASTER_KEY` 不变；
6. 停止旧进程并重启，检查 `/api/ready`、登录、连接恢复和分析结果。

成功迁移是一次性操作，当前 SQLite 文件不会自动删除，可作为切换前的回退副本。不要让 SQLite 进程和生产数据库进程同时对外写入；切换后应将旧实例下线。

## 阿里云与腾讯云 RDS

1. 创建独立的 PostgreSQL 数据库/Schema 或 MySQL Database 和账号。
2. 让 Topbase 服务器与 RDS 位于同地域、同 VPC，并只放行 Topbase 节点的安全组或白名单。
3. 从云控制台下载 CA 证书并保存为 `./secrets/rds-ca.pem`。
4. 在 `.env` 中填写 RDS 内网地址、账号、数据库、`TOPBASE_APP_DB_TLS_MODE=verify-full` 和随机主密钥。
5. 使用 RDS Compose 文件启动：

```bash
docker compose -f docker-compose.rds.yml up --build -d
curl --fail http://localhost:8080/api/ready
```

总连接预算约为“Topbase 节点数 × `TOPBASE_APP_DB_MAX_OPEN_CONNS`”，必须低于 RDS 账号和实例连接上限，并预留管理与迁移连接。

## 多节点负载均衡

多个 Topbase 节点必须连接同一个应用数据库，并共享相同的 `TOPBASE_MASTER_KEY`、Cookie 安全配置和发布版本。每个节点设置唯一 `TOPBASE_INSTANCE_ID`，负载均衡健康检查使用 `/api/ready`，不需要粘性会话。

应用启动迁移通过数据库全局锁串行执行；自动数仓调度和单项任务通过数据库租约互斥，节点退出后租约过期可由其他节点接管。数据源连接池仍属于各节点进程，节点启动时会从共享加密存储恢复连接。

滚动发布应先执行数据库备份，再逐台替换节点。当前 migration 只向前执行，不能在数据库已升级后直接把全部节点回退到旧二进制。

## 二进制部署

```bash
git fetch --tags
git checkout v0.2.1
make check
make build
TOPBASE_ADDR=:8080 \
TOPBASE_APP_DB_ENGINE=postgres \
TOPBASE_APP_DB_DSN='postgres://topbase:password@rds.internal:5432/topbase?sslmode=verify-full' \
TOPBASE_MASTER_KEY_FILE=/run/secrets/topbase-master-key \
./bin/topbase
```

建议创建独立系统用户，并让 `/var/lib/topbase` 只对该用户可读写。仓库提供 [systemd 服务模板](../deploy/systemd/topbase.service)，默认从 `/etc/topbase/topbase.env` 读取配置并执行 `/opt/topbase/topbase`。安装时可以：

```bash
sudo useradd --system --home-dir /var/lib/topbase --shell /usr/sbin/nologin topbase
sudo install -d -o topbase -g topbase /opt/topbase /var/lib/topbase /var/backups/topbase /etc/topbase
sudo install -o topbase -g topbase -m 0755 bin/topbase /opt/topbase/topbase
sudo install -m 0644 deploy/systemd/topbase.service /etc/systemd/system/topbase.service
sudo systemctl daemon-reload
sudo systemctl enable --now topbase
```

`/etc/topbase/topbase.env` 至少需要应用数据库连接、`TOPBASE_DATA_DIR=/var/lib/topbase` 和固定的 `TOPBASE_MASTER_KEY`。不要把环境文件提交到 Git。

`docker-compose.yml` 继续保留 SQLite 开发体验与兼容启动方式。SQLite 文件不能放在共享网络文件系统上供多个实例同时读写，也不支持横向扩展。
