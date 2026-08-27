# Security Policy

## Supported versions

Topbase 目前处于早期开发阶段，只维护主开发版本。生产部署前应完成威胁建模、外部安全审计，并把应用库和秘密存储迁移到生产级实现。

## Reporting a vulnerability

请不要在公开 Issue 中披露可利用细节。请通过 GitHub 仓库的 **Security → Report a vulnerability** 私密报告入口联系维护者；在仓库启用 Private vulnerability reporting 前，不要公开提交利用细节。

报告建议包含受影响版本、复现步骤、影响范围和可行缓解措施。维护者确认后应协调修复、测试、公告与披露时间。

## Sensitive data

不得提交或记录数据库密码、完整 DSN、SSH 私钥、Cookie、API Key、飞书密钥、用户查询结果或真实业务数据。开发数据目录、连接密钥和本地数据库必须保持在版本控制之外。
