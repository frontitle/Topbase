# Topbase

[![CI](https://github.com/frontitle/Topbase/actions/workflows/ci.yml/badge.svg)](https://github.com/frontitle/Topbase/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/license-Apache--2.0-111111.svg)](LICENSE)
[![Version](https://img.shields.io/badge/version-0.2.1-111111.svg)](CHANGELOG.md)

> **看见数据，理解业务，沉淀增长。**

Topbase 是一站式数据分析与可视化数仓平台，让团队通过连接数据、可视化分析、仪表盘观测和周期物化，把业务问题转化为持续可用的数据资产。

![Topbase 模拟数据仪表盘](docs/assets/topbase-dashboard-demo.jpg)

_截图使用模拟零售数据，仅用于展示仪表盘的指标、趋势、结构和明细分析能力。_

## 核心功能

- **连接数据**：支持 PostgreSQL、MySQL / MariaDB、ClickHouse、SQL Server、Oracle 和 SQLite，提供连接测试、SSL 与 SSH 跳板机接入。
- **可视化分析**：无需编写 SQL，即可通过字段选择、筛选、汇总、排序、跨表关联和自定义列完成分析；熟悉 SQL 的用户也可以切换到代码模式。
- **指标可视化**：将分析结果快速转换为数字、折线图、面积图、柱状图、条形图、饼图、散点图和明细表格。
- **仪表盘观测**：自由组合分析与内容组件，配置筛选联动、主题、布局、分享、嵌入、订阅和告警，持续追踪核心指标。
- **数据沉淀**：把需要周期统计的分析配置为计划任务，将结果物化到数仓表，并通过目录和血缘持续管理数据资产。
- **分析协作**：使用数据组组织个人与团队内容，通过服务端权限控制分析、仪表盘、数据浏览和原生 SQL 能力。
- **AI 与自动化**：通过 MCP 或 CLI 让 AI 在既有权限内理解表和字段、实时问答数据，并在用户确认后创建可继续编辑的分析。

## 安装

推荐直接使用正式镜像部署。请先安装 [Docker](https://docs.docker.com/get-docker/) 和 Docker Compose，然后执行：

```bash
git clone https://github.com/frontitle/Topbase.git
cd Topbase
git checkout v0.2.1
cp .env.example .env
# 编辑 .env，设置随机的 TOPBASE_APP_DB_PASSWORD 和 TOPBASE_MASTER_KEY
docker compose -f docker-compose.release.yml up -d
```

确认服务已经就绪：

```bash
curl --fail http://localhost:8080/api/ready
```

打开 `http://localhost:8080`，首次访问时按照页面引导创建管理员和工作区。

应用数据默认持久化在 PostgreSQL 数据卷中。`TOPBASE_MASTER_KEY` 用于加密数据源连接，首次启动后必须保持不变。升级前建议先导出应用数据库，随后拉取新版本并重建服务：

```bash
docker compose -f docker-compose.release.yml exec -T appdb \
  pg_dump -U topbase -d topbase -Fc > topbase-before-upgrade.dump
git fetch --tags
git checkout <目标版本标签>
docker compose -f docker-compose.release.yml pull
docker compose -f docker-compose.release.yml up -d
```

生产部署、配置和升级说明：

- [部署与备份](docs/deployment.md)
- [配置说明](docs/configuration.md)
- [版本升级](docs/upgrading.md)
- [数据库支持](docs/database-drivers.md)
- [AI、MCP 与 CLI](docs/ai-integrations.md)
- [0.2.1 升级说明](docs/releases/0.2.1.md)
- [更新日志](CHANGELOG.md)

## 开源说明

Topbase 使用 [Apache License 2.0](LICENSE) 开源。你可以：

- 免费用于个人项目、企业内部系统或商业产品；
- 修改源代码并开发自己的功能；
- 分发原始版本或修改后的版本；
- 在许可证范围内使用贡献者授予的相关专利权利。

再分发时需要保留许可证和版权声明，并在修改过的文件中说明变更。Topbase 名称与标识不因代码许可证而自动获得商标授权，软件按“现状”提供且不附带担保。项目包含的第三方组件继续遵循各自的许可证。

## 参与项目

- [贡献指南](CONTRIBUTING.md)
- [安全策略](SECURITY.md)
- [架构说明](docs/architecture.md)
- [功能组件](docs/frontend-components.md)
