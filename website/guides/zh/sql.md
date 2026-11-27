# 编写带参数的 SQL 分析

需要自行控制计算方式，或想让读者运行时选择渠道，可以编写一条只读 SQL。以下示例先核对一份固定报表，再给保存后的分析增加可选渠道条件。

## 在编辑器中运行查询

1. 使用[入门指南](getting-started.html)里的 SQLite 连接，打开 **源数据 → Sales orders → orders**，再点击查询步骤上方的 **SQL 模式**。当前连接决定可查询的表和可使用的 SQL 方言。
2. 粘贴下方查询。也可以先运行一次[可视化查询](analysis.html)，切换模式后点击 **使用可视化生成的 SQL**，在系统生成的语句上继续修改。
3. 点击 **运行 SQL 并预览**。检查结果列名为 `channel` 和 `revenue`，并按顺序显示 Web **300**、Partner **200**、Store **180**。

```sql
SELECT channel, SUM(amount) AS revenue
FROM orders
WHERE status = 'paid'
GROUP BY channel
ORDER BY revenue DESC
```

要核实单笔订单，可暂时运行 `SELECT * FROM orders WHERE order_id = 1003`。这笔订单金额为 50，状态为 `refunded`，因此没有计入已支付总额。核对完成后，恢复上面的分组查询。

## 增加可选的渠道参数

1. 在编辑器内，将下方一行插入原查询的 `WHERE` 和 `GROUP BY` 之间。把 `AND` 一起放在方括号内，这样参数留空时，移除整段条件后的 SQL 仍能执行。

   ```sql
   [[AND channel = {{channel}}]]
   ```

2. 点击 **保存分析**，命名为 **Paid sales by channel (SQL)**，选择可写入的分组并确认。参数在保存后的分析中填写。数据表编辑器内的 **运行 SQL 并预览** 直接执行 SQL，没有填写参数的表单。
3. 在分析详情页点 **查询参数**，输入 `Web`，无需加引号，点击运行。应只返回一行，金额为 **300**。再次打开 **查询参数**，清空该值并运行，应恢复全部三个渠道。
4. 如果每次都必须选定渠道，去掉外层方括号，改为 `AND channel = {{channel}}`。打开这条分析时，页面会等待读者填写必填项。参数名可以用 `channel`、`start_date` 等。SQL 内的占位符本身无需加引号，Topbase 会绑定输入值。

<figure><a href="../assets/guide-images/sql-parameters-zh.jpg"><img src="../assets/guide-images/sql-parameters-zh.jpg" alt="打开已保存 SQL 分析的「查询参数」，在 channel 中填入 Web。" loading="lazy"></a><figcaption>打开已保存 SQL 分析的「查询参数」，在 channel 中填入 Web。</figcaption></figure>

<figure><a href="../assets/guide-images/sql-parameter-result-zh.jpg"><img src="../assets/guide-images/sql-parameter-result-zh.jpg" alt="使用 Web 运行后，结果中仅有该渠道的一行记录，revenue 为 300。" loading="lazy"></a><figcaption>使用 Web 运行后，结果中仅有该渠道的一行记录，revenue 为 300。</figcaption></figure>

## 为仪表盘增加按日趋势

要制作[仪表盘指南](dashboards.html)中的第二张图，基于同一张订单表**新建**一条 SQL 分析，运行以下查询。按日期分组，并按日期升序排列，折线图的点就会按时间顺序展示。

```sql
SELECT order_date, SUM(amount) AS revenue
FROM orders
WHERE status = 'paid'
GROUP BY order_date
ORDER BY order_date
```

结果包含 2026 年 10 月的三个日期。1 日合计 **200**，3 日合计 **300**，4 日合计 **180**。2 日只有退款单，因此当天没有结果行。将分析保存为 **Paid sales by day**，图表选「折线图」，横轴选 `order_date`，纵轴选 `revenue`。

<figure><a href="../assets/guide-images/sql-editor-zh.jpg"><img src="../assets/guide-images/sql-editor-zh.jpg" alt="按日查询返回三个日期。先在预览中核对 revenue，再选择折线图。" loading="lazy"></a><figcaption>按日查询返回三个日期。先在预览中核对 revenue，再选择折线图。</figcaption></figure>

## 复用、导出和排查

- **查看或更改 SQL。** 在 **分析** 列表中打开已保存的分析，点击 **查看查询** 读取 SQL。具备写入权限时可点击 **编辑**。若已有图表使用某个结果列，修改 SQL 时宜保持该列的别名一致。
- **下载指定渠道。** 将参数设为 `Web` 并运行，然后点击 **导出 CSV**。文件表头应为 `channel,revenue`，且仅包含 Web 一行。清空可选参数后可重新查看全渠道结果。
- **找不到表或字段。** 在数据表浏览器核对物理名称和当前数据库。如果表在其他 schema 下，按所连接数据库的语法补全表名。
- **查询无法运行。** 若提示缺少参数，在已保存分析的 **查询参数** 中填写对应值。若提示权限不足，请管理员检查账号的数据查看权限以及该数据库的原生 SQL 执行权限。获得分析分享的读者仍需具备执行此查询的权限。
