let databases = [], analyses = [], activeDatabase = null, activeTab = 'tables', tables = [];

function normalize(value) { return String(value || '').toLowerCase(); }
function matches(value, query) { return !query || normalize(value).includes(query); }
function row(icon, title, meta, attrs) {
  return `<button class="source-row" type="button" ${attrs}><span class="source-symbol">${icon}</span><span class="source-copy"><b>${esc(title)}</b><small>${esc(meta)}</small></span><span class="source-arrow">›</span></button>`;
}
function empty(title, body, action) {
  return `<div class="source-empty"><b>${esc(title)}</b><p>${esc(body)}</p>${action || ''}</div>`;
}
function setStatus(text, hidden) {
  $('#source-status').textContent = text || '';
  $('#source-status').hidden = !!hidden;
}
function renderDatabases() {
  const query = normalize($('#source-search').value);
  const items = databases.filter(db => matches(`${db.name} ${db.host || ''} ${db.engine || ''}`, query));
  $('#picker-path').hidden = true;
  setStatus(`${items.length} 个可用数据源`);
  $('#source-list').innerHTML = items.map(db => row('▣', db.name, `${db.engine || 'PostgreSQL'} · ${db.table_count || 0} 张数据表`, `data-database="${esc(db.id)}"`)).join('') ||
    empty(databases.length ? '没有匹配的数据源' : '还没有可用数据源', databases.length ? '换一个名称、主机或数据库类型搜索。' : '管理员连接数据库并同步结构后，数据表会显示在这里。', '<a class="secondary" href="/admin/">前往数据源管理</a>');
  $$('#source-list [data-database]').forEach(button => button.onclick = () => openDatabase(button.dataset.database));
}
function renderTables() {
  const query = normalize($('#source-search').value);
  const items = tables.filter(table => !table.hidden && matches(`${table.schema} ${table.display_name || ''} ${table.name} ${table.description || ''} ${(table.columns || []).map(column => `${column.name} ${column.description || ''}`).join(' ')}`, query));
  $('#picker-path').hidden = false;
  $('#picker-current').textContent = activeDatabase.name;
  setStatus(`${items.length} 张可分析的数据表`);
  const groups = {};
  items.forEach(table => (groups[table.schema] || (groups[table.schema] = [])).push(table));
  $('#source-list').innerHTML = Object.keys(groups).sort().map(schema => `<div class="source-group-title">${esc(schema)}</div>` + groups[schema].map(table => row('▦', table.display_name || table.name, table.display_name ? `${schema}.${table.name}${table.description ? ' · '+table.description : ''}` : (table.description || `${(table.columns || []).length} 个字段 · ${schema}`), `data-schema="${esc(table.schema)}" data-table="${esc(table.name)}"`)).join('')).join('') || empty('没有匹配的数据表', '可以搜索表别称、表名、Schema、字段名或数据库注释。');
  $$('#source-list [data-table]').forEach(button => button.onclick = () => {
    const params = new URLSearchParams({ db: activeDatabase.id, schema: button.dataset.schema, table: button.dataset.table, from: 'new-analysis' });
    location.href = '/data/?' + params.toString();
  });
}
function renderAnalyses() {
  const query = normalize($('#source-search').value);
  const items = analyses.filter(item => matches(`${item.name} ${item.description || ''} ${item.query_type || ''}`, query));
  $('#picker-path').hidden = true;
  setStatus(`${items.length} 条已保存分析`);
  $('#source-list').innerHTML = items.map(item => row('◇', item.name, item.query_type === 'native' ? 'SQL 分析' : '可视化分析', `data-analysis="${esc(item.id)}"`)).join('') || empty(analyses.length ? '没有匹配的分析' : '还没有已保存分析', analyses.length ? '换一个名称或描述搜索。' : '从数据表开始创建并保存后，会显示在这里。');
  $$('#source-list [data-analysis]').forEach(button => button.onclick = () => location.href = '/questions/' + button.dataset.analysis + '/');
}
function render() {
  if (activeTab === 'analyses') return renderAnalyses();
  if (activeDatabase) return renderTables();
  renderDatabases();
}
async function openDatabase(id) {
  activeDatabase = databases.find(db => db.id === id);
  if (!activeDatabase) return;
  $('#source-search').value = '';
  setStatus('正在读取数据表…');
  $('#source-list').innerHTML = '';
  try {
    tables = await api('/api/databases/' + encodeURIComponent(id) + '/tables');
    renderTables();
  } catch (error) {
    tables = [];
    setStatus('数据表读取失败');
    $('#source-list').innerHTML = empty('暂时无法读取数据表', error.message, '<a class="secondary" href="/admin/?id=' + encodeURIComponent(id) + '">检查数据源连接</a>');
  }
}
async function boot() {
  try {
    [databases, analyses] = await Promise.all([api('/api/databases'), api('/api/questions')]);
    render();
  } catch (error) {
    setStatus('加载失败');
    $('#source-list').innerHTML = empty('无法读取数据', error.message);
  }
}
$$('[data-picker-tab]').forEach(tab => tab.onclick = () => {
  activeTab = tab.dataset.pickerTab;
  activeDatabase = null;
  $('#source-search').value = '';
  $$('[data-picker-tab]').forEach(item => { const active = item === tab; item.classList.toggle('active', active); item.setAttribute('aria-selected', active ? 'true' : 'false'); });
  render();
});
$('#picker-back').onclick = () => { activeDatabase = null; tables = []; $('#source-search').value = ''; renderDatabases(); };
$('#source-search').oninput = render;
document.addEventListener('keydown', event => { if (event.key === '/' && !/INPUT|TEXTAREA|SELECT/.test(document.activeElement.tagName)) { event.preventDefault(); $('#source-search').focus(); } });
boot();
