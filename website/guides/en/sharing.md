# Publish or embed a dashboard

Use **Share** at the top of an open dashboard to give guests a read-only view. Start by opening the dashboard as a member and checking which cards and data will be visible. For example, an external partner reviewing channel totals should see only the analyses you intend to publish.

## Publish a seven-day link

1. A workspace administrator opens **Administration → Settings → Access and security → Public access**, allows public dashboard links and saves. If guests will access Topbase from another network, configure the external site address under **Settings → Access and security → Domain and SSL**.
2. As someone with permission to manage this dashboard, open **Share** and switch on public sharing. Choose **7 days** for a weekly review. The other choices are permanent, 1 day, 30 days or a custom future deadline in local time.
3. To require a password, select **Require access password** and enter 8 to 128 characters. Click **Enable sharing**. Copy the public link that appears.
4. Open the link in a private browser window. Enter the password if prompted, then compare a chart with the signed-in dashboard. Send the guest the public URL. The editing URL belongs to the workspace.

<figure><a href="../assets/guide-images/dashboard-sharing-en.jpg"><img src="../assets/guide-images/dashboard-sharing-en.jpg" alt="Set a seven-day lifetime and select Require access password. The link is created after the password is entered and sharing is enabled." width="500" height="464" loading="lazy"></a><figcaption>Set a seven-day lifetime and select Require access password. The link is created after the password is entered and sharing is enabled.</figcaption></figure>

## Change access for an existing link

Open **Share** again, set a new expiry or password and save the sharing settings. An empty replacement password keeps the existing one. To remove the password requirement, clear its checkbox and save. To extend an expired link, choose a future deadline and save. Guests can use the same address again.

To close access, turn off public sharing and choose **Revoke link**. Reopen the old link in a private window and confirm the dashboard is unavailable. Revocation also disables its embed.

## Place the dashboard on an external page

A workspace administrator must allow embedding under **Settings → Access and security → Public access**. On the dashboard, publish a current link, save any outstanding changes, then have the administrator switch on **Allow iframe embedding**. Copy the iframe snippet into a page that supports HTML embeds. Leave its source URL intact and adjust its width or height to fit the host page.

Open the destination page as a guest. The embed uses the dashboard's password and expiry. If the embed switch is disabled, check the workspace settings, admin account, link expiry and any unsaved share settings. If the link works only from the server, check the configured external site address and guest network access.
