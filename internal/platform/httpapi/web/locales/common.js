// Shared navigation, dialogs, query controls and first-party server feedback.
window.TopbaseLocale.add({
  '首页':'Home', '工作台':'Workspace', '数据工作台':'Workspace', '管理后台':'Administration',
  '进入管理后台':'Administration', '返回工作台':'Back to workspace', '源数据':'Source data',
  '仪表盘':'Dashboards', '分析':'Analyses', '数据沉淀':'Materializations', '回收站':'Trash',
  '数据':'Data', '数据模型':'Data model', '人员与组':'People and groups', '通知与订阅':'Notifications and subscriptions',
  '嵌入与公开链接':'Embedding and public links', '设置':'Settings', '版本':'Version', '当前版本':'Current version',
  '搜索功能':'Find a page', '折叠侧边栏':'Collapse sidebar', '展开侧边栏':'Expand sidebar',
  '账户菜单':'Account menu', '个人中心':'My account', '退出登录':'Sign out', '未登录':'Signed out',
  '界面语言':'Language', '请确认':'Confirm', '取消':'Cancel', '确认':'Confirm', '关闭':'Close',
  '保存':'Save', '删除':'Delete', '编辑':'Edit', '名称':'Name', '说明':'Description',
  '无':'None', '去看看':'Open', '暂无内容':'Nothing here yet', '请求失败':'Request failed',
  '代码':'Code', '复制代码':'Copy code', '复制 {label}':'Copy {label}', '已复制':'Copied', '代码已复制':'Code copied',
  '复制失败：{reason}':'Copy failed: {reason}', '浏览器拒绝了复制操作':'Your browser blocked clipboard access',
  'SQL 编辑器':'SQL editor', 'SQL 查询':'SQL query', '运行 SQL 并预览':'Run SQL', '运行并预览':'Run query',
  '正在运行 SQL…':'Running SQL…', '正在运行…':'Running…', '已载入可视化查询生成的 SQL':'Generated SQL loaded',
  '最多显示 {count} 行':'Showing up to {count} rows', '结果较大，仅显示部分数据':'Showing a partial result due to its size',
  '部分字段过长，已截断':'Long values have been truncated', '仅显示部分结果':'Showing partial results',
  '分享截止时间必须晚于当前时间':'Choose a future expiration time',
  '访问密码需为 8 至 128 个字符，且不能全为空格':'Use 8–128 characters, including a non-space character',
  'Excel 文件不能超过 20 MB':'Choose an Excel file under 20 MB', '请选择 Excel 文件':'Choose an Excel file',
  '目前支持 .xlsx 或 .xlsm 文件':'Choose an .xlsx or .xlsm file', '读取上传文件失败':'Could not read the uploaded file',
  'Excel 中没有可导入的工作表':'This workbook has no sheets to import',
  '请选择至少包含表头和一行数据的工作表':'Choose a sheet with headers and at least one data row',
  '上传的数据集不存在':'Uploaded dataset not found', '读取本地数据预览失败':'Could not load the dataset preview',
  '读取分享设置失败':'Could not load sharing settings', '工作区尚未同时允许公开链接与外部嵌入':'Enable both public links and embedding in workspace settings',
  '请确认目标为空库并已先下载备份':'Confirm the target database is empty and you have downloaded a backup',
  '将相同连接参数写入部署环境变量，保持 TOPBASE_MASTER_KEY 不变，然后重启所有 Topbase 节点。当前进程仍使用 SQLite。':'Set the same connection details in the deployment environment and restart every Topbase node. Keep TOPBASE_MASTER_KEY unchanged. This process still uses SQLite.',
  '请先由管理员启用开发者模式':'Ask an administrator to enable developer mode',
  '管理员未允许成员创建个人 API Key':'Personal API keys are disabled in this workspace',
  '个人数据组不能移动':'The personal group cannot be moved', '个人数据组不能删除':'The personal group cannot be deleted',
  '请先把数据组里的分析移走或归档':'Move or archive the analyses in this group first',
  '请先移走或删除子数据组':'Move or delete the subgroups first',
  'PostgreSQL 的目标 schema 和表名最长各为 63 字节':'PostgreSQL schema and table names must each fit within 63 bytes',
  '沉淀计划缺少目标数据库连接':'Select a target database for this schedule',
  '沉淀运行记录与目标表登记服务未配置':'Run history or target registration is unavailable',
  '工作区尚未允许公开分享':'Public sharing is disabled in this workspace', '分享设置无效':'Invalid sharing settings',
  '请输入有效的到期时间':'Enter a valid expiration time', '访问密码格式无效':'Invalid access password format',
  '分享链接已失效或已关闭':'This share link has expired or was disabled',
  '工作区已暂停公开分享':'Public sharing is paused', '请输入访问密码':'Enter the access password',
  '验证过于频繁，请稍后再试':'Too many attempts. Try again later.', '访问密码不正确':'Incorrect access password',
  '更新说明不能超过 12000 个字符':'Release notes cannot exceed 12,000 characters',
  '请输入可公开访问的 http 或 https 地址':'Enter a public HTTP or HTTPS URL',
  '无法连接目标网站，请确认地址可访问':'Could not reach the site. Check its URL.',
  '目标网站设置了 X-Frame-Options，禁止被嵌入':'The site blocks embedding with X-Frame-Options',
  '目标网站的内容安全策略禁止在此画布中嵌入':'The site’s content security policy blocks embedding here',
  '未发现禁止嵌入的响应头':'The site’s response headers allow embedding',
  '所选第三方登录尚未完成接入或已被停用':'This sign-in provider is unavailable',
  '必须至少保留一种登录方式':'Keep at least one sign-in method enabled',
  '账号密码登录关闭时，不能删除仍在使用的第三方登录':'Enable password sign-in or another provider first',
  '请先填写应用凭据、启用接入并保存配置':'Enter credentials, enable the integration and save it first',
  '当前部署尚未安装该平台的组织同步适配器':'Organization sync for this provider is unavailable on this deployment',
  '该账号平台不提供组织同步':'This provider does not support organization sync',
  '请完整填写接入名称和 App ID / Client ID':'Enter an integration name and an app or client ID',
  '请填写 App Secret / Client Secret':'Enter the app or client secret',
  '暂不支持该账号平台':'Unsupported identity provider', '请先验证完整的应用凭据':'Verify the app credentials first',
  '账号接入不存在':'Identity provider not found', '请先在通知与订阅中创建并启用 Webhook 通道':'Create and enable a webhook in Notifications and subscriptions',
  '非空增量目标表的水位为空，请检查变化字段':'The incremental target contains rows but has no watermark. Check its tracking field.',
  '此计划正在运行，请稍后查看结果':'This schedule is running. Check the result shortly.',
  '已有沉淀计划或数据表使用此目标，请更换保存表名':'This target is already in use. Choose another table name.',
  '目标库提交结果未确认':'The target database has not confirmed the commit',
  '无法解析 Excel 文件':'Could not parse the workbook', '保存本地数据集失败':'Could not save the dataset',
  '读取目标表水位失败':'Could not read the target watermark', '检查上次写入状态失败':'Could not check the previous write',
  '登记目标库写入意图失败，尚未写入目标表':'Could not register the write. The target table has not been updated.',
  '登记失败运行状态':'Could not record the failed run', '数仓物化成功':'Materialization completed',
  '数仓物化失败':'Materialization failed', '数仓物化已恢复':'Materialization recovered', '仪表盘订阅':'Dashboard subscription',
  '恢复运行 {run} 的登记失败':'Could not restore registration for run {run}',
  '目标库已提交 {rows} 行，水位 {mark}，应用库登记未确认':'The target committed {rows} rows at watermark {mark}. Application registration is unconfirmed.',
  '运行 {run} 需核对 {target}，已暂停重写目标表':'Check run {run} and target {target}. Further writes are paused.',
  '写入回执未能登记，请管理员核对运行 {run} 和目标 {target}':'Could not save the write receipt. Check run {run} and target {target}.',
  '运行 {run} 的写入屏障与历史记录不一致，请核对数据':'The write state and history disagree for run {run}. Check the data.',
  '运行 {run} 的写入尚未核对，请管理员核对 {target} 的数据、水位与运行记录后恢复计划。':'Check the data, watermark and run history for {target} before resuming run {run}.',
  '重试将使用已确认的回执恢复登记':'Retry to restore registration from the confirmed receipt',
  '我已把问题转为可预览的只读查询。确认后可运行，并可直接创建周期任务。':'Preview the read-only query, then run or schedule it.'
});

// Protocol messages are matched only on the error path. SQL, identifiers and
// free-form database diagnostics pass through unchanged.
(function (i18n) {
  const errors = {
    'not signed in':'请先登录', 'invalid credentials':'邮箱或密码错误', 'invalid email or password':'邮箱或密码错误', 'admin access required':'需要管理员权限',
    'forbidden':'无权执行此操作', 'access denied':'无权访问', 'permission denied':'权限不足',
    'not found':'未找到', 'analysis not found':'未找到分析', 'dashboard not found':'未找到仪表盘',
    'name and email are required':'请填写姓名和邮箱', 'unsupported language':'不支持此语言',
    'unsupported theme':'不支持此主题', 'email is already in use':'该邮箱已被使用',
    'current password is required to change email':'修改邮箱需要当前密码',
    'invalid current password':'当前密码错误', 'password must be at least 8 characters':'密码至少需要 8 个字符'
  };
  const formats = [
    [/^恢复运行 (\S+) 的登记失败$/, '恢复运行 {run} 的登记失败', ['run']],
    [/^目标库已提交 (\d+) 行，水位 (.*)，应用库登记未确认$/, '目标库已提交 {rows} 行，水位 {mark}，应用库登记未确认', ['rows','mark']],
    [/^运行 (\S+) 需核对 (.+)，已暂停重写目标表$/, '运行 {run} 需核对 {target}，已暂停重写目标表', ['run','target']],
    [/^写入回执未能登记，请管理员核对运行 (\S+) 和目标 (.+)$/, '写入回执未能登记，请管理员核对运行 {run} 和目标 {target}', ['run','target']],
    [/^运行 (\S+) 的写入屏障与历史记录不一致，请核对数据$/, '运行 {run} 的写入屏障与历史记录不一致，请核对数据', ['run']],
    [/^运行 (\S+) 的写入尚未核对，请管理员核对 (.+) 的数据、水位与运行记录后恢复计划。$/, '运行 {run} 的写入尚未核对，请管理员核对 {target} 的数据、水位与运行记录后恢复计划。', ['run','target']]
  ];
  function message(part) {
    if (i18n.locale() === 'zh-CN') return Object.hasOwn(errors, part) ? errors[part] : part;
    for (const [pattern, key, names] of formats) {
      const match = part.match(pattern);
      if (match) return i18n.t(key, Object.fromEntries(names.map((name, i) => [name, match[i + 1]])));
    }
    return i18n.t(part);
  }
  i18n.errorText = function (value) {
    if (!value || typeof value !== 'string') return value;
    const exact = message(value);
    if (exact !== value) return exact;
    return value.split(/(: |：|；|\n)/).map(part => part === '；' && i18n.locale() === 'en' ? '. ' :
      part === '：' && i18n.locale() === 'en' ? ': ' : message(part)).join('');
  };
})(window.TopbaseLocale);
