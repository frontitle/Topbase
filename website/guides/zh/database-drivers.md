# 连接数据库并检查表结构

管理员先连接保存业务明细的数据库，分析人员就能在「源数据」中打开它。第一次练习可使用示例 SQLite 文件。连接已有业务库时，请准备对目标表有读取权限的数据库账号。

## 连接示例 SQLite 文件

1. 下载包含六笔模拟订单的 SQLite 数据库 [orders.db](../assets/samples/orders.db)。直接在服务器运行 Topbase 时，将文件放在服务进程能读取的目录，例如 `/srv/topbase-samples/orders.db`。
2. 使用 Docker Compose 时，将文件放到 Compose 文件旁的 `samples` 目录。在已有的 `services.topbase.volumes` 列表中追加下面这一行，保留原有挂载。使用发行版 Compose 文件时，执行 `docker compose -f docker-compose.release.yml up -d topbase` 应用修改。容器内的用户需要能读取这个目录。

   ```yaml
   - ./samples:/samples:ro
   ```

3. 进入「管理后台 → 数据 → 远程数据源」，点击「添加数据库」。数据库类型选择 **SQLite**，显示名称填 **Sales orders**。「数据库文件」填服务进程可见的绝对路径。按上例挂载时填 `/samples/orders.db`。SQLite 使用文件连接。
4. 点击「测试连接」，通过后点击「保存数据库」。打开已保存的数据源，找到 **orders** 表。再进入「源数据 → Sales orders → orders」，确认能看到 **1001–1006** 六笔订单。接下来可以[制作第一张销售仪表盘](getting-started.html)。

<figure><a href="../assets/guide-images/database-connect-zh.jpg"><img src="../assets/guide-images/database-connect-zh.jpg" alt="SQLite 测试通过后即可点击「保存数据库」。文件路径请填自己服务器上实际可读的位置。" loading="lazy"></a><figcaption>SQLite 测试通过后即可点击「保存数据库」。文件路径请填自己服务器上实际可读的位置。</figcaption></figure>

## 连接网络数据库

同样从「添加数据库」进入。例如连接 PostgreSQL 时，类型选择 **PostgreSQL**，显示名称填「销售报表库」，服务器地址填数据库管理员提供的主机名，端口填 **5432**，数据库名称填实际库名（例如 `sales`），再填写报表账号及密码。如果数据库使用了自定义监听端口，以实际端口为准。

| 驱动 | 表单默认端口 | 需要填写的内容 |
| --- | ---: | --- |
| PostgreSQL | 5432 | 数据库名称，以及有权读取相应 schema 和表的账号。 |
| MySQL / MariaDB | 3306 | 数据库名称，以及允许从 Topbase 所在服务器连接的账号。 |
| ClickHouse | 9000 | 原生 TCP 监听地址和库名，默认用户名是 `default`。 |
| SQL Server | 1433 | 服务器地址、要查询的数据库名称和登录账号。 |
| Oracle Database | 1521 | 监听地址，在「数据库名称」处填写 **Service Name**。 |
| SQLite | — | 服务可见的绝对路径，容器内示例为 `/samples/orders.db`。 |

「显示名称」是团队在 Topbase 中看到的名字。「服务器地址」和「端口」指 Topbase 进程能访问的数据库地址和监听端口。在 Docker 中，`localhost` 指当前容器。两个服务在同一 Compose 网络时，可以填写数据库服务名和它的内部监听端口。

## 配置加密、连接字符串和跳板机

- 「SSL 与高级」用于设置网络数据库的传输方式。按数据库管理员要求选择自动／优先加密、必须加密、加密但不校验证书或不加密。哪些选项可用取决于驱动及服务器配置。
- 「连接字符串」可以填写相应驱动的 DSN。填写后，它会覆盖单独输入的主机、端口、库名和账号。如果改了主机后仍然连到旧地址，检查是否设置了 DSN。含密码的连接串应避免出现在截图或消息中。
- 「SSH 跳板机」适用于通过堡垒机访问网络数据库。连接页仍填数据库内网地址和监听端口。SSH 页填跳板机地址、SSH 端口（默认 **22**）和 SSH 用户名，选择密码或私钥认证。加密私钥还需提供私钥密码。使用经确认的服务器 SHA256 主机密钥指纹可以校验跳板机身份。
- 「测试连接」通过后才能保存。修改连接信息后需要重测；编辑已有连接时，测试成功后点击「保存并重新扫描」。

## 核对结构并排查缺表

在「管理后台 → 数据 → 远程数据源」打开已保存的连接。点击「同步全部结构」刷新表清单，点选一张表可查看字段与类型。只改动了一张表的列时，可选「重新扫描此表」。再从[源数据](browse.html)打开表核对实际记录。

| 遇到的问题 | 先检查什么 |
| --- | --- |
| 连接超时 | 从 Topbase 所在服务器或容器检查数据库地址和监听端口。通过跳板机连接时，还要检查 SSH 入口及跳板机能否到达数据库内网地址。 |
| 测试成功但没有目标表 | 确认库名和 schema，核对连接账号是否获准读取该表，再同步结构。 |
| SQLite 看起来是空库，或文件打不开 | 核对文件全路径、名称和读取权限，使用 Docker 时检查挂载。下载文件应位于这次练习指定的样本目录。 |

<figure><a href="../assets/guide-images/database-schema-zh.jpg"><img src="../assets/guide-images/database-schema-zh.jpg" alt="选择 main.orders 后可看到五个字段。「同步全部结构」刷新整个数据源的目录，「重新扫描此表」仅刷新所选表。" loading="lazy"></a><figcaption>选择 main.orders 后可看到五个字段。「同步全部结构」刷新整个数据源的目录，「重新扫描此表」仅刷新所选表。</figcaption></figure>
