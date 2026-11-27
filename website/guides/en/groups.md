# Organize, find and recover analyses

Use a personal group to organize reports you own, or a team project when several people need different roles. This example places the channel sales analysis into a monthly reporting group and shows how a colleague can find it.

## Create a group and move an analysis

1. Open **Analyses → New group**, name it **Sales reporting** and create it. Members create personal groups. Administrators can also select **Team project** and choose the initial managing user group.
2. In the analysis groups area, open **Sales reporting**. Choose **New subgroup**, enter **Monthly** and create it. The group page lists its subgroups, saved analyses and dashboards.
3. Open the saved **Sales by channel** analysis. Choose **Move to group**, select **Monthly** under the group you created and confirm. Check the group name above the analysis, then reopen **Monthly** and find the analysis there. Existing dashboard cards can continue to reference it.

<figure><a href="../assets/guide-images/analysis-library-en.jpg"><img src="../assets/guide-images/analysis-library-en.jpg" alt="The Analyses page shows the Sales reporting group above the search field and the two saved analyses below it." width="1360" height="850" loading="lazy"></a><figcaption>The Analyses page shows the Sales reporting group above the search field and the two saved analyses below it.</figcaption></figure>

## Give someone access

Open a personal group you own and choose **Share group**. Select the member and save. Ask the member to open **Analyses**, locate the shared group and open the analysis. A member with shared view access can read the group. To change that member's access, reopen the member picker, update the selection and save.

For an enterprise project, an administrator assigns roles to permission groups. Use the [permissions guide](permissions.html) to distinguish viewing a project, editing its contents and querying the underlying database.

## Find a report again

- **Home → Recent content** opens a recently used item. From **Analyses**, search by analysis name or narrow the list to a group. Clear the group filter if the report was moved.
- The sidebar search filters *navigation entries*. To search across analyses, dashboards, groups and connections, open the workspace search page at `/search/` on your Topbase instance. Enter a distinctive name and check the type shown beside the result.
- To rename a writable group, open its detail page, choose **Rename**, enter the new name and save. Before deleting an empty group, confirm that its analyses, dashboards and child groups have been moved out.

## Recover an archived report

Open an analysis you can edit and choose **Delete**, or open a dashboard and choose **More → Move to trash**. Confirm the item name in the dialog. To undo this, open **Trash** from the workspace sidebar, find the item by name and type, and choose **Restore**. Reopen it from **Analyses** or **Dashboards** and check that it contains the expected chart.
