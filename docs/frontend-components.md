# Topbase 前端功能组件

本文登记跨页面复用的交互组件及其稳定边界。业务页面负责准备数据和调用 API；组件负责一致的结构、状态、可访问性与交互反馈。组件清单与代码必须同步更新。

## 组件清单

| 组件 | 源码 | 全局入口 | 主要使用页面 | 状态 |
| --- | --- | --- | --- | --- |
| 应用外壳 | `web/shell.js` | 页面上的 `data-shell` | 全部登录后页面 | 稳定 |
| UI 基础设施 | `web/ui.js` | `api`、`toast`、`promptDialog`、`choiceDialog`、`confirmDialog` | 全站 | 稳定 |
| 查询编辑器 | `web/components/query-editor/` | `TopbaseQueryEditor.mount` | 数据浏览、创建分析 | 稳定 |
| 代码展示与编辑 | `web/components/code/` | `TopbaseCode.mountBlock`、`mountEditor`、`setCode` | 查询编辑器、分析详情 | 稳定 |
| 筛选构建器 | `web/filter.js`、`web/filter.css` | `TopbaseFilter` | 数据浏览、分析详情 | 稳定 |
| 数据表格 | `web/grid.js`、`web/grid.css` | `TopbaseGrid` | 数据浏览、分析详情、仪表盘卡片 | 稳定 |
| 可视化渲染器 | `web/viz.js`、`web/viz.css` | `TopbaseViz` | 分析详情、仪表盘卡片 | 稳定 |

这里的“全部组件”指已经形成跨页面公共接口的前端功能组件。只服务单一页面、仍与页面业务状态强耦合的局部模块不计入公共组件；当它在第二个页面复用时，必须先抽出接口再登记。

## 公共接口

### 查询编辑器

查询编辑器将“可视化查询”和“SQL 模式”组织成同一组件，但不直接依赖任何后端地址。

```js
const editor = TopbaseQueryEditor.mount('#ask-panel', {
  onModeChange(mode) {},
  onSQLChange(sql) {},
  onRun(mode) {}
});

editor.mode();
editor.setMode('visual');
editor.sql();
editor.setSQL('SELECT 1', { dirty: true });
editor.setGeneratedSQL('SELECT * FROM orders LIMIT 1000');
editor.setSummary('1 条筛选 · 按日期汇总');
```

- 页面负责把可视化 QueryIR 交给 `/api/dataset`，把 SQL 交给 `/api/queries/run`。
- 首次切换到 SQL 模式时，组件可载入最近一次可视化查询生成的 SQL；用户编辑后不再自动覆盖。
- 运行按钮始终位于组件固定底栏，组件统一处理 loading、禁用和恢复。
- SQL 模式保存为原生查询；可视化模式保存版本化 QueryIR。页面不得混淆两种持久化格式。
- 可视化专属动作（例如下钻）在 SQL 模式中必须隐藏或给出明确不可用反馈。

### 代码展示与编辑

```js
TopbaseCode.mountBlock('#query-code', {
  language: 'sql',
  label: '查询 SQL',
  code: 'SELECT 1'
});

const editor = TopbaseCode.mountEditor('#sql-editor', {
  language: 'sql',
  label: 'SQL 查询',
  onChange(value) {}
});
```

- 产品中出现完整 SQL、JSON 查询定义或可复制代码时，必须使用本组件，不直接渲染裸 `<pre>` / `<code>`。
- 展示块和编辑器都带语言标签与复制按钮；复制成功、失败必须即时反馈。
- 编辑器支持 Tab 缩进、等宽字体和窄屏布局，不使用浏览器原生弹窗。

### 筛选构建器

- 输入字段模型、已有筛选和可选的离散值加载函数，输出结构化筛选条件。
- 页面中同一结果只能存在一个主筛选入口，数据表格内部筛选需要显式关闭，避免重复功能。
- 新增操作符时，同时补齐字段类型约束、中文文案、序列化和后端校验测试。

### 数据表格

- 输入列、行、别名、类型、字段说明以及显示状态；输出隐藏列、搜索、排序等视图状态。
- 组件不持久化业务对象，也不直接请求数据。
- 大数据量必须由查询端限制或分页，不能依赖浏览器一次渲染无限行。

### 可视化渲染器

- 只消费查询结果和稳定 `ChartSpec`，负责图表推断、类型切换、投影、渲染和设置面板。
- 图表实例和 DOM 临时状态不得写入分析对象。
- 新图表类型必须登记字段角色约束、默认推断、空态、错误态和响应式行为。

### 应用外壳与 UI 基础设施

- 应用外壳统一生成侧栏、用户入口、活动菜单与管理权限入口。
- 所有 HTTP 请求通过 `api`，核心动作通过 `toast` 反馈。
- 需要输入、选择或确认时使用 Topbase 对话框组件，禁止 `alert`、`confirm`、`prompt`。

## 新组件准入规则

1. 同一种交互在两个页面出现时，抽成公共组件，页面不得复制实现。
2. 组件使用 `tb-` CSS 命名空间，样式不得依赖具体页面层级。
3. 组件通过参数、事件或回调与业务连接，不硬编码 API、数据库或路由。
4. 必须覆盖 loading、empty、success、error、permission denied 和窄屏状态。
5. 交互元素提供键盘操作、焦点样式和必要的 ARIA 语义。
6. 新增或修改组件时，同步更新本文、JS 自动化测试和至少一个真实页面验收场景。

## 验收清单

- `node --test tests/js/*.test.js`
- `go test ./...`
- `git diff --check`
- 在浏览器中验证模式切换、核心按钮 loading、成功/失败反馈、复制按钮和窄屏布局。
