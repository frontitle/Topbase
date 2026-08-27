# 部署 Topbase

## Docker Compose

仓库提供多阶段 `Dockerfile`。运行容器使用固定 UID `10001` 的非 root 用户、只读根文件系统、移除 Linux capabilities，并将应用状态和备份分别放在 `/data`、`/backups` 命名卷中。

```bash
git clone git@github.com:frontitle/Topbase.git
cd Topbase
cp .env.example .env
docker compose up --build -d
```

验证：

```bash
curl --fail http://localhost:8080/api/ready
curl --fail http://localhost:8080/api/version
docker compose ps
```

生产部署时应把 `8080` 只暴露给反向代理，由代理提供 HTTPS，并设置 `TOPBASE_SECURE_COOKIES=true`。飞书等秘密通过部署平台的 Secret 功能注入，不要直接写进 `.env` 后提交。

容器会响应 `SIGTERM`，停止接收请求、等待在途请求结束，并关闭源数据库连接和 SSH 隧道。`/api/health` 表示进程存活；`/api/ready` 还会验证应用库可用并返回当前 schema 版本。

## 二进制部署

```bash
make build
TOPBASE_ADDR=:8080 TOPBASE_DATA_DIR=/var/lib/topbase ./bin/topbase
```

建议创建独立系统用户，并让 `/var/lib/topbase` 只对该用户可读写。使用 systemd、容器编排平台或其他进程管理器保证异常退出后重启。

## 持久化与高可用边界

当前应用库是单机 SQLite，适合单实例部署。不要让多个 Topbase 实例同时读写同一个网络文件系统上的 `app.db`。正式支持横向扩展前，需要完成 PostgreSQL 应用库适配、分布式调度锁、共享 SecretStore 和任务队列。
