# Send a daily report or check for results

A dashboard subscription delivers a scheduled report to a webhook channel. A result alert checks a saved analysis for returned rows and uses the Topbase inbox. Choose a subscription for a morning channel report. Choose a result alert for a question such as “Are there any unpaid orders?”

## Connect the delivery channel

1. An administrator opens **Administration → Notifications and subscriptions** (`/admin/integrations/`) and creates a webhook. In the destination chat or service, create or obtain its incoming webhook URL.
2. Choose the matching provider, give the channel a recognizable name such as **Sales morning report**, and enter the full webhook URL. Choose a message style supported by the provider and save the channel.
3. Start with the provider's default message. If you personalize the title or body, use `{{title}}` and `{{body}}` to insert the report values. Custom JSON requires a valid JSON object in the destination's format. For message cards that link back to the dashboard, configure the external domain under **Settings → Access and security → Domain and SSL**.
4. Enable the channel and send a test. Check the actual destination chat for the message before adding a subscription.

<figure><a href="../assets/guide-images/webhook-channel-en.jpg"><img src="../assets/guide-images/webhook-channel-en.jpg" alt="A new Slack channel needs the incoming webhook URL. Choose the message format before saving." width="480" height="620" loading="lazy"></a><figcaption>A new Slack channel needs the incoming webhook URL. Choose the message format before saving.</figcaption></figure>

## Choose the webhook provider

| Destination | Address to paste into Topbase |
| --- | --- |
| Feishu or DingTalk | Add a custom bot to the target group and copy its webhook address. |
| WeCom | Create a group robot in the target chat and copy its webhook address. |
| Slack | Enable incoming webhooks in a Slack app, add a webhook to the workspace and authorize the destination channel. [Slack setup](https://docs.slack.dev/messaging/sending-messages-using-incoming-webhooks/) |
| Microsoft Teams | Create a workflow that accepts a webhook and posts to a channel or chat. Use the trigger that allows anyone to call it, save the workflow and copy the HTTP POST URL. [Teams setup](https://support.microsoft.com/en-us/workflows/send-messages-in-teams-using-incoming-webhooks) |
| Google Chat | In the destination space's apps and integrations, add a webhook and copy its full URL, including its key and token. [Google Chat setup](https://developers.google.com/workspace/chat/quickstart/webhooks) |
| Generic webhook | Supply the HTTP POST endpoint of your receiver and choose a matching message template. |

Slack, Teams and Google Chat offer quick presets that start with a card. Other new channels start with text. The message style selector also offers rich text and custom JSON. For an existing channel on the same platform, leave the URL blank to retain its saved address. When you change the platform, supply its new URL and test the destination again.

## Subscribe a dashboard

Open the orders dashboard and choose **More → Subscribe to dashboard**. You need edit access to the dashboard and permission to browse data. Pick **Sales morning report** and create the subscription. This dialog schedules it for **09:00 every day in Asia/Shanghai (UTC+8)**. If the dropdown has no channels, an administrator needs to create and enable one first.

In **Administration → Notifications and subscriptions**, search for the dashboard ID or channel name. The row shows the schedule, time zone, latest attempt and status. Choose **Test** and check the actual message. It reports the number of successful cards, failed cards and result rows. With the channel and daily charts in the tutorial you should see two successful cards, zero failures and six rows. Pause or resume the subscription from the same row. Inspect delivery history if a scheduled message is missing.

## Check whether an analysis returned rows

In the visual query editor, open the tutorial orders, filter `status` to `refunded`, leave the result as detail rows and save the analysis. Create a dashboard and add this visual analysis as its first analysis card. Open **More → Set result alert**. Topbase checks the first analysis card immediately and records a notification in your inbox when it finds rows. In this example it should find one refunded order, **1003**. The alert runner needs a visual analysis with a structured query.

Run the source analysis to compare results. A table with zero matching rows is suitable for checking when nothing needs attention. A single count of zero still produces one row. If you need a condition such as an order amount over 200, put that filter in the saved detail analysis before creating its alert.
