# Give a team access to its dashboards

A permission group contains people. A team data group contains content such as analyses and dashboards. To give sales staff access to a dashboard, add the staff to a permission group, set its capabilities, then give it a role in the data group that holds the dashboard.

## Add members and choose a project role

1. As an administrator, open **Administration → People and groups** (`/admin/people/`). For a new local user, choose **Add member** and enter their name and email.
2. Choose **New group**, name it **Sales readers**, and click **Create group**. Open the **Permission groups** tab, find its card and select **Manage group**.
3. In **Group members**, select an existing non-admin user. Set the four workspace capabilities as shown below. System and synchronized groups take their membership from their source.
4. In the same dialog, find **Data group ·** followed by the name of the project containing the dashboard. Select **View**. The choices are **No grant**, **View**, **Edit** and **Manage**. Each existing team data group has a separate dropdown.
5. Click **Save group settings**. Reopen **Manage group** and confirm the member, capabilities and project role were saved. The built-in administrators group has fixed permissions.

<figure><a href="../assets/guide-images/permissions-en.jpg"><img src="../assets/guide-images/permissions-en.jpg" alt="The group editor lists members and workspace capabilities. The example shows the Sales analysts group before any grants are assigned." width="620" height="640" loading="lazy"></a><figcaption>The group editor lists members and workspace capabilities. The example shows the Sales analysts group before any grants are assigned.</figcaption></figure>

## Set the four capabilities

| Setting | Available levels | Sales readers example |
| --- | --- | --- |
| Browse data | No access, Browse, Manage | Browse |
| SQL queries | Deny, Query, Run native SQL | Query for members who use visual queries |
| Content and data groups | No access, View, Edit | View |
| Admin access | None, Allow admin console access | None |

For a separate author group that creates charts, grant **Edit** under **Content and data groups** and **Edit** in the destination project's dropdown. Choose **Run native SQL** for groups that need the SQL editor. Use **Save group settings** after changing the group's capabilities or project roles.

## Connect a directory or sign-in provider

1. In **Administration → People and groups → Identity providers**, choose **Connect organization**. Select Feishu and name the connection. Continue setup, enter the application's App ID and App Secret, then choose **Verify and connect**. Use an application whose directory access covers the departments you want to synchronize.
2. Choose **Sync organization**. Check the returned department count, then open the permission groups list. The built-in sync creates a group for each returned Feishu department and matches existing Topbase users by Feishu Open ID. If a group appears without members, check the linked users and the app's directory scope before running sync again.
3. For example, open the synchronized Sales department group and give it **View** for the sales project. Its membership comes from Feishu. The connection also has a department-to-group mappings field that accepts comma-separated `external_department_id:topbase_group_id` pairs. This sync writes memberships to the generated department groups. Check those actual groups when granting roles.
4. DingTalk and WeCom also appear in the provider selector. Their organization-sync adapters are unavailable in this distribution. Their incoming message webhooks are configured separately in [Notifications and subscriptions](notifications.html).

For individual sign-in, choose **Add sign-in provider**, select **Google** or **WeChat Open Platform**, give it a name and enter the client credentials. Verify and connect, then check **Allowed sign-in methods**. At least one method must remain enabled. Configure your instance's external domain for OAuth. The callback path on that domain is `/auth/oauth/PROVIDER_ID/callback`, where the ID is the Topbase provider identifier in the sign-in URL. Bind the identity to a member through their account settings or the member's actions, then test sign-in in a separate session. Confirm both login and project access before changing the other enabled methods.

## Verify as the reader

Use a separate browser session and sign in as the member you added. Open the sales data group, open the dashboard and verify its charts load. Test that the member can perform the actions you intended. If their access is broader than expected, inspect all their permission groups, including the Everyone group, and their project roles. If they cannot see the dashboard, check its actual data group and the role granted there. If a visible card fails to load, check access to the source analysis and data.
