# Contributing to Topbase

感谢参与 Topbase。项目坚持独立设计、清晰边界和可验证的用户体验；所有贡献都应遵守仓库许可证，并保持实现、测试与公开文档同步。

## 开发环境

要求 Go 版本以 `go.mod` 为准，并安装 Node.js（仅用于静态检查浏览器 JavaScript）。

```bash
go mod download
make check
go run ./cmd/topbase
```

默认数据写到 `data/`。开发隔离实例请使用 `TOPBASE_DATA_DIR` 指向临时目录；不要提交 `app.db`、WAL、连接密钥、私钥或真实数据快照。

## 提交一个功能

1. 关联一个公开 Issue 或在 Pull Request 中写清问题、目标和可验证的验收标准。
2. 先写领域/应用服务测试，再实现 API 和 UI。
3. 保持 `platform → app → core`，具体数据库、飞书、AI、密钥实现放在 `adapters`。
4. UI 使用全局 token 和公共组件，避免页面级复制按钮、弹层、Toast、表格行为。
5. 更新 API、配置、migration 和用户文档。
6. 运行 `make check`。

## Pull Request 清单

- [ ] 标题清楚描述改动，正文关联 Issue 或列出验收标准。
- [ ] 正常、加载、空、错误、无权限和重试场景都有明确行为。
- [ ] 所有写 API 有服务端校验和授权；所有读 API 按对象权限过滤。
- [ ] 没有日志或响应泄露 DSN、密码、SSH 私钥、Cookie、API Key。
- [ ] 新数据库能力通过端口/注册表扩展，没有在 handler 或页面硬编码分支。
- [ ] migration 只追加，没有修改已发布文件。
- [ ] 新的用户文案清晰一致，并可以进入统一的国际化词条。
- [ ] `make check` 通过；关键 UI 已在真实浏览器验证。

## 测试分层

| 层级 | 目的 | 位置 |
| --- | --- | --- |
| 领域单元测试 | QueryIR、权限、调度等纯规则 | `internal/core/**/*_test.go` |
| 应用服务测试 | 用例、事务、端口协作 | `internal/app/**/*_test.go` |
| 适配器集成测试 | PostgreSQL、SSH、应用库、飞书协议 | `internal/adapters/**/*_test.go` |
| HTTP 契约测试 | 状态码、JSON、认证和权限 | `internal/platform/httpapi/**/*_test.go` |
| 浏览器验收 | 关键业务路径和 Topbase 交互规范 | 测试说明或 E2E 套件 |

测试应确定、可并行，不依赖开发者个人数据库。需要真实服务的集成测试使用显式环境变量并在缺失时跳过。

## 兼容与安全

- QueryIR、持久化对象和公开 API 的破坏性变化必须提供版本迁移。
- 发现安全问题不要提交包含利用细节的公开 Issue，请按 `SECURITY.md` 私下报告。
- 项目采用 Apache License 2.0。引入依赖或第三方资产前必须确认许可证兼容性，并保留要求的版权和许可证声明。

## 公开文档边界

`docs/`、`README.md`、`CONTRIBUTING.md` 和 `SECURITY.md` 都会公开发布。不要提交竞品研究、内部排期、未公开产品策略、客户信息、真实业务数据或工作底稿；这类材料应保存在仓库之外。
