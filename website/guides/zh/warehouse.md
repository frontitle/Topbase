# 定时把分析保存为数仓表

在 **数据沉淀**（`/warehouse/`）中，管理员可以把已保存分析的结果写为一张表，再按计划更新。比如，多张图都引用渠道汇总时，可以一起使用这张只有三行的结果表。

## 创建每日渠道汇总

当前数据沉淀使用 **PostgreSQL** 连接，在来源连接的 `warehouse` 模式中建表。连接账号需要能在该模式下创建与刷新表，模式尚未创建时还需要建模式权限。入门教程下载的 SQLite 文件用于制作仪表盘。本节使用一个 PostgreSQL 连接，五条示例数据直接写在查询中。

1. 从「源数据」选择 PostgreSQL 连接，打开任意可浏览的表并切换到「SQL 模式」。运行下方查询，保存为 **Paid channel summary**。先检查结果有三行。

   ```sql
   SELECT channel, SUM(amount) AS revenue
   FROM (VALUES ('Web', 120), ('Web', 180), ('Partner', 200), ('Store', 100), ('Store', 80)) AS paid(channel, amount)
   GROUP BY channel
   ```

2. 打开 **数据沉淀 → 新建数据沉淀**，在 **来源分析** 中选择渠道汇总。列表支持已保存的 SQL 分析和可视化分析。在 **沉淀名称** 中填 **Daily paid sales by channel**，供团队识别。
3. **保存表名** 填 `paid_sales_by_channel`。表名允许 1 至 60 个字母、数字或下划线。Topbase 会转为小写，在 `warehouse.wh_*` 下创建托管表。
4. **更新频率** 选 **每天 09:00**，**保存方式** 选 **全量刷新**。从此入口建立的计划使用 **Asia/Shanghai（UTC+8）**，创建后可在计划列表查看时区。也可选择每小时或每周一 09:00。全量刷新时水位字段留空。

<figure><a href="../assets/guide-images/warehouse-schedule-zh.jpg"><img src="../assets/guide-images/warehouse-schedule-zh.jpg" alt="PostgreSQL 示例选择每天全量刷新 paid_sales_by_channel，水位字段留空。" width="620" height="700" loading="lazy"></a><figcaption>PostgreSQL 示例选择每天全量刷新 paid_sales_by_channel，水位字段留空。</figcaption></figure>

5. 点击 **创建并立即沉淀**。成功后列表显示实际表名、已保存行数、来源、计划及最后运行时间。点击该行的 **创建分析** 检查结果，应有三个渠道，金额分别为 **300、200、180**，合计 **680**。

<figure><a href="../assets/guide-images/warehouse-results-zh.jpg"><img src="../assets/guide-images/warehouse-results-zh.jpg" alt="沉淀成功后的列表显示已保存三行及物理目标表。点击「创建分析」可核对各渠道金额。" width="1360" height="850" loading="lazy"></a><figcaption>沉淀成功后的列表显示已保存三行及物理目标表。点击「创建分析」可核对各渠道金额。</figcaption></figure>

## 选择后续写入方式

| 保存方式 | 适用数据 | 下次更新时的行为 |
| --- | --- | --- |
| 全量刷新 | 旧订单会更正、汇总金额会变化，或来源已做分组。 | 按来源分析的当前结果重新生成表。 |
| 仅追加变化数据 | 明细持续新增，且有可靠递增字段。 | 只追加水位字段严格大于上次已保存水位的行。 |

例如订单事件明细包含递增事件编号时，在来源分析的输出中保留该列，把准确列名填入 **水位字段**。首次运行保存当次的全部结果。后来补录的事件如果与旧水位相等或更小，下次刷新就不会读到。渠道汇总中同一渠道的金额会变化，应选全量刷新。

## 排查刷新失败

如果提示计划已创建、首次更新失败，在原计划那一行查看错误。先确认来源分析仍能运行，修正原因后点 **重试更新**。状态为 **待管理员核对** 时，由管理员核对目标表的行、水位与运行记录，再决定是否再次刷新。成功后重新打开结果表，与来源分析核对。
