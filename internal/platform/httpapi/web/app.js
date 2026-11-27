function optional(path, fallback) {
  return api(path).catch(() => fallback);
}

function dateLabel(value) {
  if (!value) return '最近更新';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '最近更新';
  const diff = Date.now() - date.getTime();
  if (diff >= 0 && diff < 60 * 60 * 1000) return TopbaseLocale.t('{n} 分钟前', {n: Math.max(1, Math.floor(diff / 60000))});
  if (diff >= 0 && diff < 24 * 60 * 60 * 1000) return TopbaseLocale.t('{n} 小时前', {n: Math.floor(diff / 3600000)});
  return new Intl.DateTimeFormat(TopbaseLocale.locale(), { month: 'numeric', day: 'numeric' }).format(date);
}

function recentRow(item) {
  return `<a href="${esc(item.href)}"><i>${esc(item.icon)}</i><span><strong translate="no">${esc(item.name)}</strong><small>${esc(TopbaseLocale.t(item.type))} · ${esc(TopbaseLocale.t(dateLabel(item.created_at)))}</small></span><b>→</b></a>`;
}

function renderRecent(questions, dashboards) {
  const items = [
    ...questions.map(item => ({ ...item, type: '分析', icon: '◇', href: `/questions/${item.id}/` })),
    ...dashboards.map(item => ({ ...item, type: '仪表盘', icon: '☷', href: `/dashboard/${item.id}/` }))
  ].sort((a, b) => String(b.created_at || '').localeCompare(String(a.created_at || ''))).slice(0, 8);
  $('#recent-content').innerHTML = items.length
    ? items.map(recentRow).join('')
    : emptyHTML({ icon: '◇', title: '还没有分析或仪表盘', body: '从一张数据表开始创建第一个分析。', href: '/questions/new/', cta: '创建分析' });
}

function renderDatabaseHealth(databases, allowed) {
  if (!allowed) {
    $('#database-health').innerHTML = '<div class="home-permission-note"><i>◌</i><span><strong>数据浏览尚未授权</strong><small>联系管理员授予数据浏览权限。</small></span></div>';
    $('#stat-databases').textContent = '—';
    $('#stat-databases-meta').textContent = '当前账号无数据浏览权限';
    return;
  }
  const connected = databases.filter(item => item.connected || item.status === 'connected').length;
  $('#stat-databases').textContent = databases.length;
  $('#stat-databases-meta').textContent = databases.length ? TopbaseLocale.t('{n} 个连接正常', {n:connected}) : '等待添加第一个数据源';
  $('#database-health').innerHTML = databases.length
    ? databases.slice(0, 6).map(item => {
      const healthy = item.connected || item.status === 'connected';
      return `<a href="/data/?db=${encodeURIComponent(item.id)}"><span class="health-dot ${healthy ? 'healthy' : ''}"></span><span><strong translate="no">${esc(item.name)}</strong><small>${esc(item.engine || TopbaseLocale.t('数据库'))} · ${TopbaseLocale.t(healthy ? '连接正常' : '需要检查')}</small></span><b>→</b></a>`;
    }).join('')
    : '<div class="home-permission-note"><i>＋</i><span><strong>还没有数据源</strong><small>管理员连接数据库后即可开始分析。</small></span></div>';
}

// Existing notification records store the rendered message. Match the producer's
// format only, leaving user-provided titles, names, and diagnostics untouched.
const alertPatterns = [
  [/^查询返回 (\d+) 行$/, '查询返回 {rows} 行', ['rows']],
  [/^当前值 (-?[\d.]+) 已达到目标 (-?[\d.]+)$/, '当前值 {value} 已达到目标 {goal}', ['value','goal']],
  [/^进度 (-?[\d.]+) 已达到 (-?[\d.]+)$/, '进度 {value} 已达到 {goal}', ['value','goal']]
];
const systemNotifications = new Map([
  ['数仓物化成功', [[/^([\s\S]+) 已写入 (.+)，共 (\d+) 行$/, '{name} 已写入 {table}，共 {rows} 行', ['name','table','rows']]]],
  ['数仓物化已恢复', [[/^([\s\S]+) 的写入记录与水位已恢复$/, '{name} 的写入记录与水位已恢复', ['name']]]],
  ['仪表盘订阅', [[/^([\s\S]+)：成功 (\d+) 张卡，失败 (\d+)，共 (\d+) 行$/, '{name}：成功 {ok} 张卡，失败 {failed}，共 {rows} 行', ['name','ok','failed','rows']]]],
  ['数仓物化失败', []],
  ['仪表盘订阅投递失败', []]
]);
function notificationText(item) {
  const body = item.body || '';
  const patterns = item.alert_id ? alertPatterns : systemNotifications.get(item.title) || [];
  let message = body;
  for (const [pattern, key, names] of patterns) {
    const match = body.match(pattern);
    if (match) {
      message = TopbaseLocale.t(key, Object.fromEntries(names.map((name, i) => [name, match[i + 1]])));
      break;
    }
  }
  const title = !item.alert_id && systemNotifications.has(item.title) ? TopbaseLocale.t(item.title) : item.title;
  return {title: title || TopbaseLocale.t('任务动态'), body: message};
}

function renderNotifications(items) {
  $('#notifications').innerHTML = items.length
    ? items.slice(0, 6).map(notificationText).map(item => `<div><i>◴</i><span><strong translate="no">${esc(item.title)}</strong><small translate="no">${esc(item.body)}</small></span></div>`).join('')
    : '<div class="home-quiet"><i>✓</i><strong>暂无待处理动态</strong><small>计划任务、告警和订阅结果会显示在这里。</small></div>';
}

async function loadOverview() {
  const databaseRequest = api('/api/databases').then(items => ({ allowed: true, items })).catch(() => ({ allowed: false, items: [] }));
  const [questions, dashboards, collections, notifications, databaseResult, readiness] = await Promise.all([
    optional('/api/questions', []),
    optional('/api/dashboards', []),
    optional('/api/collections', []),
    optional('/api/notifications', []),
    databaseRequest,
    optional('/api/ready', null)
  ]);

  $('#stat-questions').textContent = questions.length;
  $('#stat-dashboards').textContent = dashboards.length;
  $('#stat-collections').textContent = collections.length;
  renderRecent(questions, dashboards);
  renderDatabaseHealth(databaseResult.items, databaseResult.allowed);
  renderNotifications(notifications);

  const ready = readiness && readiness.status === 'ready';
  $('#ready-badge').textContent = ready ? '运行正常' : '需要检查';
  $('#ready-badge').classList.toggle('healthy', ready);
}

async function boot() {
  try {
    const status = await api('/api/setup/status');
    if (!status.completed) {
      location.replace('/setup/');
      return;
    }
  } catch (error) {
    toast(error.message);
    return;
  }

  let user;
  try {
    user = await api('/api/user/current');
  } catch (_) {
    location.replace('/auth/login/');
    return;
  }
  const name = user.name || user.email || '';
  $('#warehouse-link').hidden = !user.is_admin;
  $('#admin-database-link').hidden = !user.is_admin;
  const welcome = $('#welcome');
  welcome.translate = false;
  welcome.textContent = name ? TopbaseLocale.t('欢迎回来，{name}', {name}) : TopbaseLocale.t('欢迎回来');
  await loadOverview();
}

TopbaseUI.popover($('#home-create'), $('#home-create-menu'));
const dashboardButtons = [$('#create-dashboard'), $('#menu-create-dashboard')];
async function createDashboard() {
  if (dashboardButtons.some(button => button.disabled)) return;
  dashboardButtons.forEach(button => { button.disabled = true; });
  try {
    const dashboard = await api('/api/dashboards', 'POST', {});
    location.href = `/dashboard/${dashboard.id}/`;
  } catch (error) {
    toast(error.message);
    dashboardButtons.forEach(button => { button.disabled = false; });
  }
}
dashboardButtons.forEach(button => { button.onclick = createDashboard; });

boot();
