# Build an orders dashboard

Choose **Dashboards → New dashboard**. Topbase creates a dashboard in your personal data group, gives it a timestamped name and opens the empty canvas in editing mode. On an existing dashboard, choose **Edit**. You need permission to edit the dashboard and access to the saved analyses you plan to add.

## Add two analyses

1. Prepare two saved analyses from the tutorial orders table. For **Sales by channel**, filter `status` to `paid`, sum `amount`, group by `channel` and choose **Column chart**. For **Paid sales by day**, keep the same filter, sum `amount` by `order_date`, sort the dates ascending and choose a line chart. The [SQL editor walkthrough](sql.html) includes the daily query.
2. On the dashboard, choose **Insert → Add analysis**. Search for the channel analysis by name or select its data group. Click the result to add it, or drag it to a position on the canvas. Repeat for the daily trend. An analysis already on the canvas appears as added.
3. Choose **Insert → Add widget** if you want a heading or an explanation. Use a short text such as “Paid orders only” to tell readers which orders count. Select the widget to edit its text and appearance.

<figure><a href="../assets/guide-images/dashboard-editor-en.jpg"><img src="../assets/guide-images/dashboard-editor-en.jpg" alt="In Insert → Add analysis, search for Sales by channel. Select the visual analysis to add it beside Paid sales by day." loading="lazy"></a><figcaption>In Insert → Add analysis, search for Sales by channel. Select the visual analysis to add it beside Paid sales by day.</figcaption></figure>

## Arrange and check the canvas

Arrange the channel breakdown and daily trend side by side. Drag the chart's resize handle until you can read all three channel names. Press **Escape** during a drag to cancel that move. Use **Components** to find and select an item when several items overlap. The properties panel changes with the selected item.

Open **Canvas settings** to choose a color scheme and background. **Show grid** makes alignment lines visible while editing. **Snap to grid** aligns items as you move them. You can change these independently.

Choose **Preview**. The bars should show **Web 300**, **Partner 200** and **Store 180**. The line should show **October 1, 2026 · 200**, **October 3 · 300** and **October 4 · 180**. October 2 only has a refund. For an optional third card, save a single-number analysis summing paid amounts with no grouping. It should show **680**. Press **Escape** to leave preview, then choose **Finish editing**. Reopen the dashboard to check the saved layout.

<figure><a href="../assets/guide-images/dashboard-en.jpg"><img src="../assets/guide-images/dashboard-en.jpg" alt="The finished dashboard pairs Sales by channel with Paid sales by day. The daily chart peaks at 300 on October 3." loading="lazy"></a><figcaption>The finished dashboard pairs Sales by channel with Paid sales by day. The daily chart peaks at 300 on October 3.</figcaption></figure>

## Configure cards and widgets

Select the channel chart and open **Appearance**. Use its card title to label the result “Sales by channel”. The **Visualization** selector controls how this card draws the saved result. Card style controls the surface, border, padding and whether to show the title. Use **Position and size** to set the dimensions precisely when a label is clipped.

| Widget or card | Where to use it | Settings to try |
| --- | --- | --- |
| Heading or text | Explain that refunded orders are excluded. | Choose font, size and alignment. A text widget can also scroll horizontally or vertically. |
| Table analysis | Show the five individual paid orders below the chart. | Under **Table playback**, choose static, continuous scrolling or automatic paging. |
| Image | Display a product image or a set of slides. | Select one or several images. For multiple images, enable the carousel and choose a 3, 5, 8 or 12 second interval. |
| Time | Add a clock to a display screen. | Choose date and time, date only, time only or weekday and date. |
| Link or web page | Open a related report or display an embedded page. | Enter a name and URL. An embedded page must allow framing by the Topbase site. |

Open **More** on the finished dashboard to bookmark it or make a copy. A copy gives you a separate dashboard for trying a different layout. To send it to another reader, follow [Share or embed a dashboard](sharing.html).

## Read existing tabs and filters

On dashboards with multiple tabs, click a tab above the canvas to load its cards. Cards shared across tabs remain visible. If the board already has filter mappings, its toolbar shows a text box or start and end dates. For example, enter `Web` in a channel field, then switch tabs to load the selected tab with that value. Only cards mapped to that field are filtered. Typing alone does not refresh the cards on the current tab. Switching tabs returns the current input values to the card queries.

The canvas editor has no controls for adding tabs or defining new filter mappings. For a new board built entirely in the UI, put shared conditions such as `status = paid` into each source analysis before adding it.

## When a card looks wrong

If an analysis is missing from Insert, clear the search and group filter, then confirm the analysis was saved and is visible to your account. If the card shows an unexpected value, open the underlying analysis and check its aggregation and paid-status filter. On a dashboard that already has filter controls, clear them to compare the unfiltered dashboard with its source analysis.
