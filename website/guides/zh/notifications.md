# 订阅每日简报或设置结果提醒

仪表盘订阅按计划把简报发到 Webhook 通道。结果提醒检查已保存分析是否返回行，使用 Topbase 站内收件箱。每天早上收渠道简报可以用订阅，想检查「有没有待支付订单」可以用结果提醒。

## 接通消息通道

1. 管理员进入 **管理后台 → 通知与订阅**（`/admin/integrations/`），创建 Webhook。在目标群聊或服务中获取接收消息的 Webhook URL。
2. 选择对应平台，通道命名为「销售晨报」，填写完整 URL，选择该平台支持的消息样式并保存通道。
3. 初次配置可保留平台默认样式。自定义标题或正文时，用 `{{title}}` 和 `{{body}}` 引用实际通知内容。自定义 JSON 需要填写目标平台接受的合法 JSON 对象。消息卡片要能打开仪表盘，还需在 **设置 → 访问与安全 → 域名与 SSL** 配置访问域名。
4. 启用通道并发送测试。在实际群聊确认收到消息，再创建订阅。

<figure><a href="../assets/guide-images/webhook-channel-zh.jpg"><img src="../assets/guide-images/webhook-channel-zh.jpg" alt="新建 Slack 通道时，填写目标聊天的 Webhook 地址，选择消息格式后保存。" loading="lazy"></a><figcaption>新建 Slack 通道时，填写目标聊天的 Webhook 地址，选择消息格式后保存。</figcaption></figure>

## 选择目标平台并获取地址

| 消息目的地 | 应填入 Topbase 的地址 |
| --- | --- |
| 飞书、钉钉 | 在目标群添加自定义机器人，复制该机器人的 Webhook 地址。 |
| 企业微信 | 在目标会话添加群机器人，复制 Webhook 地址。 |
| Slack | 在 Slack 应用中启用 Incoming Webhooks，添加工作区 Webhook，授权目标频道后复制地址。[Slack 官方设置](https://docs.slack.dev/messaging/sending-messages-using-incoming-webhooks/) |
| Microsoft Teams | 在 Workflows 建立接收 Webhook 后向频道或聊天发消息的流程。触发器允许「任何人」，保存并复制 HTTP POST URL。[Teams 官方设置](https://support.microsoft.com/en-us/workflows/send-messages-in-teams-using-incoming-webhooks) |
| Google Chat | 在目标聊天室的「应用和集成」中添加 Webhook，复制含 key 和 token 的完整地址。[Google Chat 官方设置](https://developers.google.com/workspace/chat/quickstart/webhooks) |
| 通用 Webhook | 填写能够接收 HTTP POST 的服务地址，并选择接收方支持的消息模板。 |

Slack、Teams、Google Chat 的快捷预设默认使用消息卡片，其他新通道默认使用文本消息。消息样式还提供富文本和自定义 JSON。编辑已有通道时，平台相同且地址留空可保留原地址。切换平台后需要填写新地址并重新测试。

## 添加每日仪表盘订阅

打开订单仪表盘，点击 **更多 → 订阅仪表盘**，选择「销售晨报」并创建订阅。需要当前仪表盘的编辑权限和数据浏览权限。这个入口的计划是 **Asia/Shanghai（UTC+8）每天 09:00**。没有可选通道时，先让管理员创建并启用 Webhook。

回到 **管理后台 → 通知与订阅**，按仪表盘 ID 或通道名称搜索。列表会显示计划、时区、最近发送时间和状态。点击 **验证**，到目标群聊查看实发消息，内容包括成功卡数、失败卡数和返回行数。教程中的渠道图和每日趋势各三行，预期为成功 2 张卡、失败 0 张、共 6 行。可以在同一行暂停或启用订阅。计划消息未收到时，先核对启用状态，再查看发送记录。

## 让分析有结果时提醒

在可视化查询编辑器打开入门示例的订单表，把 `status` 筛选为 `refunded`，保留订单明细并保存分析。新建仪表盘，将该可视化分析作为第一张分析卡加入。点击 **更多 → 设置结果提醒**，Topbase 会立即检查第一张分析卡，有结果就写入站内收件箱。示例应命中一条退款订单，编号为 **1003**。提醒执行器需要带结构化查询的可视化分析。

打开来源分析核对行数。待支付订单为空时，明细表应该返回零行；如果用计数汇总，即使计数是零，结果仍有一行。若要检查「金额超过 200 的订单」，先在明细分析中加入金额筛选，保存后再建立提醒。
