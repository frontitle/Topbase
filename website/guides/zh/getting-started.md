# 创建第一张销售仪表盘

哪个渠道的已支付销售额最高？用六笔订单在可视化查询编辑器里算出答案，再把图表放进仪表盘。完成后应看到三根柱子，其中 Web 最高，销售额为 300。

## 打开示例订单表

1. 下载[订单示例数据库](../assets/samples/orders.db)，请管理员[连接这个 SQLite 文件](database-drivers.html)并开放访问权限。如果你就是管理员，可以把连接命名为 **Sales orders**。文件要放在 Topbase 服务器可以读取的位置。
2. 从侧栏进入「源数据」，选择 **Sales orders → orders**。每一行是一笔订单，共五个字段。`order_id` 是订单编号，`order_date` 是日期，`channel` 是渠道，`amount` 是订单金额，`status` 记录支付或退款状态。
3. 找到订单 **1003**，确认金额为 **50**、状态为 **refunded**（已退款）。这笔订单应在汇总已支付销售额之前被筛掉。

[Excel 上传练习](excel.html)使用相同的六笔订单。这里选择 SQLite 数据源，查询编辑器就可以直接读取记录，并对数值类型的金额求和。

## 在可视化查询中算出答案

1. 打开 **orders** 表，选择「可视化查询」。如果编辑器已收起，先在表格上方展开。
2. 在「筛选」选择字段 `status`，运算符选「等于」，值填 `paid`。在「汇总 → 计算」选择「求和」，字段选 `amount`，「分组」选 `channel`。
3. 打开「排序」，排序字段选「汇总结果」，方向选「降序」。点击「运行并预览」，先将结果与下表核对。

| 渠道 | 金额之和 |
| --- | ---: |
| Web | 300 |
| Partner | 200 |
| Store | 180 |

三个渠道合计 **680**，对应五笔已支付订单。如果 Web 为 **350**，检查状态筛选是否生效；如果为 **2**，说明算出了 Web 的订单数。将计算改为「求和」后再次运行。

<figure><a href="../assets/guide-images/visual-query-zh.jpg"><img src="../assets/guide-images/visual-query-zh.jpg" alt="对照筛选、计算与分组设置，再检查结果中的 Web、Partner 和 Store。" width="754" height="876" loading="lazy"></a><figcaption>对照筛选、计算与分组设置，再检查结果中的 Web、Partner 和 Store。</figcaption></figure>

## 保存为渠道销售图

1. 点击「保存分析」，名称填「Sales by channel」，选择可写入的分组。自己练习可保存在个人分组，需要同事打开这份分析时，选择对方有权访问的分组。
2. 在保存后的分析页面选择柱状图，类别选 `channel`，数值选金额的汇总结果。将鼠标移到 Web 的柱子上，应看到 **300**。坐标轴和数值格式见[图表配置](charts.html)。

<figure><a href="../assets/guide-images/chart-settings-zh.jpg"><img src="../assets/guide-images/chart-settings-zh.jpg" alt="保存后的分析中 Web 为 300。图表上方选「柱状图」，在右侧设置 channel 和 sum。" width="1360" height="850" loading="lazy"></a><figcaption>保存后的分析中 Web 为 300。图表上方选「柱状图」，在右侧设置 channel 和 sum。</figcaption></figure>

## 把图表放入仪表盘

1. 进入「仪表盘 → 新建仪表盘」。如果画布打开时处于查看状态，点击「编辑」。
2. 在「插入 → 添加分析」中搜索「Sales by channel」，点击搜索结果。拖动卡片手柄调整位置，再拉宽卡片，直到三个渠道名都能完整显示。
3. 点击「预览」查看读者看到的效果。完成编辑后，从仪表盘列表再次打开，核对布局和数值。让同事访问时，按[仪表盘分享](sharing.html)设置权限。

如果插入面板找不到分析，先在分析列表打开它，确认保存的名称与分组，再清除插入面板的分组筛选并重新搜索。
