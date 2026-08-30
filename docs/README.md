# Topbase 文档

## 使用与运维

- [快速开始](getting-started.md)：本地运行、首次初始化和接入第一个数据库
- [配置参考](configuration.md)：环境变量、数据目录和秘密配置
- [部署](deployment.md)：Docker Compose 与二进制部署
- [升级与备份](upgrading.md)：版本策略、migration、备份和回滚
- [0.2.2 升级说明](releases/0.2.2.md)：网络、公开地址与工作台体验更新
- [数据库驱动](database-drivers.md)：支持矩阵与能力边界
- [AI、MCP 与 CLI](ai-integrations.md)：让 AI 理解数据、问答并创建分析

## 开源贡献者文档

- [工程架构](architecture.md)：依赖方向、稳定扩展点和安全基线
- [信息架构与数据命名](information-architecture.md)：一级导航、分析分组与数据生命周期术语
- [前端功能组件](frontend-components.md)：公共组件清单、接口与准入规则
- [贡献指南](../CONTRIBUTING.md)：开发流程与质量门禁
- [安全策略](../SECURITY.md)：安全问题报告和敏感数据要求

文档描述必须区分“已经可以运行”和“规划中”。功能是否完成以验收条件和自动化测试为准，不能只以页面、按钮或 API 是否存在判断。

`docs/` 只存放可以公开给用户和开源贡献者阅读的内容。竞品研究、内部排期、未公开产品策略和工作底稿不得提交到本目录或 Git 仓库。
