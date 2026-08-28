const id = location.pathname.split('/').filter(Boolean)[1];
let question = null, collections = [], filterBar = null, lastResult = null, spec = null, saveTimer = 0;

async function run(queryir) {
  if (queryir) return api('/api/dataset', 'POST', queryir);
  if (question.query_type === 'native' && question.native_sql) {
    return api('/api/queries/run', 'POST', { database_id: question.database_id, sql: question.native_sql });
  }
  if (question.queryir) return api('/api/dataset', 'POST', question.queryir);
  throw Error('这条分析没有可运行的查询');
}

function inferColumns(d) {
  return (d.columns || []).map((name, i) => {
    const sample = (d.rows || []).map(r => r[i]).find(v => v !== null && v !== undefined && v !== '');
    const type = typeof sample === 'number' ? 'numeric' : (/^\d{4}-\d{2}-\d{2}/.test(String(sample || '')) ? 'date' : 'text');
    return { name, data_type: type, display_name: name };
  });
}

function queryPreview() {
  if (question.native_sql) return question.native_sql;
  if (lastResult && lastResult.sql) return lastResult.sql;
  if (question.queryir) return JSON.stringify(question.queryir, null, 2);
  return '这条分析还没有查询定义。';
}

function showError(err) {
  const box = $('#error');
  if (!box) return;
  box.hidden = !err;
  box.innerHTML = err ? '<b>无法显示分析结果</b><p>' + esc(err) + '</p>' : '';
}

function saveView(state) {
  const columns = Object.assign({}, spec.columns || {});
  ((lastResult && lastResult.columns) || []).forEach(name => {
    const prev = Object.assign({}, columns[name] || {});
    if (state.hidden[name]) prev.visible = false;
    else delete prev.visible;
    if (state.filters[name]) prev.filter = state.filters[name];
    else delete prev.filter;
    if (prev.visible === false || prev.filter || prev.title) columns[name] = prev;
    else delete columns[name];
  });
  spec = Object.assign({}, spec, {
    columns,
    search: state.search || '',
    sort: state.sort || '',
    sort_dir: state.dir === 'desc' ? 'desc' : ''
  });
  question.chartspec = spec;
  scheduleSave();
}

function paint() {
  const d = lastResult || { columns: [], rows: [] };
  spec = TopbaseViz.infer(d.columns || [], d.rows || [], question.queryir, spec || question.chartspec || d.chartspec);
  const view = spec.type === 'table'
    ? { columns: d.columns || [], rows: d.rows || [] }
    : TopbaseViz.project(d.columns || [], d.rows || [], spec);
  TopbaseViz.renderTypes($('#viz-types'), spec, type => applySpec(Object.assign({}, spec, { type })));
  TopbaseViz.render($('#viz-stage'), {
    columns: d.columns || [],
    rows: d.rows || [],
    spec,
    queryir: question.queryir,
    tableFilters: false,
    onViewChange: spec.type === 'table' ? saveView : null
  });
  TopbaseViz.renderSettings($('#viz-settings'), view.columns, view.rows, spec, applySpec);
}

function applySpec(next) {
  spec = TopbaseViz.merge(next, spec, (lastResult && lastResult.columns) || [], (lastResult && lastResult.rows) || []);
  question.chartspec = spec;
  paint();
  scheduleSave();
}

function scheduleSave() {
  clearTimeout(saveTimer);
  saveTimer = setTimeout(saveSpec, 450);
}

async function saveSpec() {
  if (!question || !spec) return;
  try {
    question = await api('/api/questions/' + id, 'PUT', {
      name: question.name,
      description: question.description || '',
      collection_id: question.collection_id || '',
      chartspec: spec
    });
  } catch (e) {
    toast(e.message);
  }
}

function mountFilter(d) {
  if (filterBar) { filterBar.destroy(); filterBar = null; }
  if (!question.queryir) return;
  filterBar = TopbaseFilter('#filter-bar', {
    columns: inferColumns(d),
    filters: question.queryir.filters || [],
    fetchValues: async field => {
      const others = (question.queryir.filters || []).filter(f => f.field !== field);
      const res = await api('/api/dataset', 'POST', {
        version: 1,
        source: question.queryir.source,
        aggregations: [{ fn: 'count' }],
        group_by: [{ field }],
        order_by: [{ field: 'count', dir: 'desc' }],
        filters: others,
        limit: 80
      });
      return (res.rows || []).map(r => r[0]).filter(v => v !== null && v !== undefined);
    },
    onChange: async filters => {
      question.queryir = Object.assign({}, question.queryir, { filters, limit: question.queryir.limit || 2000 });
      await loadResult();
    }
  });
}

function setMeta() {
  const col = collections.find(c => c.id === question.collection_id);
  const kind = question.query_type === 'native' ? 'SQL' : '可视化查询';
  const n = lastResult ? (lastResult.rows || []).length : 0;
  const resultMeta = (lastResult && lastResult.meta) || {};
  const freshness = resultMeta.execution === 'direct' && resultMeta.cache_hit === false ? '实时直连' : '查询结果';
  const updated = resultMeta.executed_at ? new Date(resultMeta.executed_at).toLocaleTimeString('zh-CN') : '';
  $('#title').textContent = question.name;
  $('#heading').textContent = question.name;
  $('#meta').textContent = kind + ' · ' + (col ? col.name : '我的分析') + ' · ' + freshness + ' · ' + n + ' 行' + (updated ? ' · 更新于 ' + updated : '');
  const isSQL = !!(question.native_sql || (lastResult && lastResult.sql));
  TopbaseCode.setCode('#query-json', queryPreview(), {
    language: isSQL ? 'sql' : 'json',
    label: isSQL ? '查询 SQL' : '查询定义'
  });
}

async function loadResult() {
  showError('');
  $('#viz-stage').innerHTML = '<div class="viz-empty"><b>正在运行</b><p>正在拉取这条分析的结果。</p></div>';
  try {
    lastResult = await run();
    spec = TopbaseViz.merge(question.chartspec || lastResult.chartspec, TopbaseViz.infer((lastResult.columns || []), (lastResult.rows || []), question.queryir), lastResult.columns || [], lastResult.rows || []);
    question.chartspec = spec;
    mountFilter(lastResult);
    setMeta();
    try {
      paint();
    } catch (renderError) {
      showError('数据查询成功，但可视化渲染失败：' + renderError.message);
      $('#viz-stage').innerHTML = '<div class="viz-empty"><b>数据已查询成功</b><p>图表暂时无法渲染，请切换为表格或重新运行。</p></div>';
      TopbaseViz.renderTypes($('#viz-types'), spec || { type: 'table' }, type => applySpec(Object.assign({}, spec, { type })));
      $('#viz-settings').innerHTML = '<h3>设置</h3><p class="viz-muted">可视化组件发生错误，原始查询结果没有丢失。</p>';
    }
  } catch (e) {
    lastResult = { columns: [], rows: [], sql: question.native_sql || '' };
    setMeta();
    showError(e.message);
    $('#viz-stage').innerHTML = '<div class="viz-empty"><b>结果不可见</b><p>' + esc(e.message) + '</p></div>';
    TopbaseViz.renderTypes($('#viz-types'), spec || { type: 'table' }, () => {});
    $('#viz-settings').innerHTML = '<h3>设置</h3><p class="viz-muted">查询成功后，可以在这里更改图表类型、横轴和纵轴。</p>';
  }
}

async function boot() {
  question = await api('/api/questions/' + id);
  collections = await api('/api/collections');
  spec = question.chartspec || { type: 'table' };
  setMeta();
  await loadResult();
}

boot().catch(e => {
  showError(e.message);
  if ($('#title')) $('#title').textContent = '无法打开';
  if ($('#heading')) $('#heading').textContent = '无法打开分析';
  if ($('#meta')) $('#meta').textContent = e.message;
});

function on(id, fn) {
  const el = document.getElementById(id);
  if (el) el.onclick = fn;
}
on('rerun', () => loadResult().catch(e => toast(e.message)));
on('rename', async () => {
  const name = await promptDialog({ kicker: '分析设置', title: '重命名分析', label: '分析名称', value: question.name, confirmText: '保存名称' });
  if (!name) return;
  question = await api('/api/questions/' + id, 'PUT', {
    name,
    description: question.description || '',
    collection_id: question.collection_id || '',
    chartspec: spec || question.chartspec
  });
  setMeta();
  toast('已重命名');
});
on('move', async () => {
  const collection_id = await choiceDialog({
    kicker: '分析分组',
    title: '移动到分组',
    description: '选择分析的新位置。移动不会影响查询或仪表盘中的引用。',
    label: '目标位置',
    value: question.collection_id || '',
    confirmText: '移动分析',
    options: [{ value: '', label: '未分组', description: '暂不放入任何分析分组' }].concat(collections.map(c => ({ value: c.id, label: c.name, description: c.kind === 'personal_project' ? '个人分组' : '团队分组' })))
  });
  if (collection_id === null) return;
  question = await api('/api/questions/' + id, 'PUT', {
    name: question.name,
    description: question.description || '',
    collection_id,
    chartspec: spec || question.chartspec
  });
  toast(collection_id ? '已移动到分组' : '已移出分组');
  setMeta();
});
on('archive', async () => {
  if (!await confirmDialog({ kicker: '归档分析', title: '将这条分析移到回收站？', description: '分析会从列表和数据组中隐藏，之后仍可在回收站恢复。', confirmText: '移到回收站', tone: 'danger' })) return;
  await api('/api/questions/' + id, 'DELETE');
  location.href = '/questions/';
});
