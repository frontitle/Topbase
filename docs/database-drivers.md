# 数据库驱动支持矩阵

Topbase 的数据库接入以真实驱动为准：测试连接会实际打开源库并执行 ping，保存后同步真实 schema、表、字段、注释和主外键，数据浏览与可视化分析直接查询该连接。页面上出现一个数据库名称不代表完成接入。

## 当前支持

| 引擎 | 兼容产品 | 测连与查询 | 元数据与注释 | SSL | SSH 跳板机 | 可视化 QueryIR |
| --- | --- | --- | --- | --- | --- | --- |
| PostgreSQL | PostgreSQL 兼容服务 | 支持 | 支持 | 支持 | 支持 | 支持 |
| MySQL | MySQL、MariaDB、TiDB、OceanBase MySQL、Apache Doris、StarRocks | 支持 | 支持 | 支持 | 支持 | 支持 |
| ClickHouse | ClickHouse Native 协议 | 支持 | 支持 | 支持 | 支持 | 支持 |
| SQL Server | SQL Server、Azure SQL 的 SQL Server 连接方式 | 支持 | 支持 | 支持 | 支持 | 支持 |
| Oracle | Oracle Database Service Name 连接 | 支持 | 支持 | 支持 | 支持 | 支持 |
| SQLite | 本地 SQLite 文件 | 支持 | 支持 | 不适用 | 不适用 | 支持 |

连接入口统一位于「管理后台 → 数据库」。网络数据库可填写基础字段或 DSN，并在独立 Tab 配置 SSL 与 SSH；SQLite 只要求数据库名称和文件路径。测试连接不保存配置，保存成功后自动同步目录。

## 能力边界

- 目前覆盖主流关系型和 SQL OLAP 数据库，不把 MongoDB、Redis、Elasticsearch 等非 SQL 系统伪装成 SQL 驱动；它们需要独立查询模型、权限和元数据连接器。
- BigQuery、Snowflake、Databricks、Trino 等云数仓需要各自的 OAuth、Service Account、Catalog 或计费配置，后续按同一驱动接口接入。
- 多数据库已可用于查询、图形化筛选、汇总、Join 和可视化。跨两个不同数据库实例的联邦 Join 尚不支持；当前 Join 在同一数据源内执行。
- 数仓计划任务当前只允许物化到 PostgreSQL 目标。扩展写入目标前必须分别验证 DDL、事务、幂等替换、增量 watermark 和失败恢复。
- 兼容产品按其兼容协议接入，个别厂商扩展类型和函数需要专门的方言回归测试。

## 新增驱动验收

新增数据库不能只增加下拉选项。至少要同时交付：驱动注册与连接字段、密钥脱敏、真实测连、连接池与关闭、参数绑定、只读查询、QueryIR 方言、表/字段/注释/主外键同步、错误分类、管理界面反馈、单元测试和容器集成测试。
