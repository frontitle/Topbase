const id = location.pathname.split('/').filter(Boolean)[1];
let question = null, collections = [], filterBar = null, lastResult = null, spec = null, saveTimer = 0;
let nameSaving = false;
let temporaryFilters = false;
let canEdit = false, canRun = false, nativeParameters = null;

async function run(queryir) {
  if (queryir) return api('/api/dataset', 'POST', queryir);
  if (question.query_type === 'native' || (!question.queryir && question.native_sql)) {
    return api('/api/questions/' + encodeURIComponent(id) + '/native-preview', 'POST', {parameters: nativeParameters || {}});
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
  return TopbaseLocale.t('这条分析还没有查询定义。');
}

function showError(err) {
  const box = $('#error');
  if (!box) return;
  box.hidden = !err;
  box.innerHTML = err ? '<b>无法显示分析结果</b><p>' + esc(err) + '</p>' : '';
}

function saveView(state) {
  if (!canEdit) return;
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
  if (canEdit) TopbaseViz.renderTypes($('#viz-types'), spec, type => applySpec(Object.assign({}, spec, { type })));
  TopbaseViz.render($('#viz-stage'), {
    columns: d.columns || [],
    rows: d.rows || [],
    spec,
    queryir: question.queryir,
    semanticTypes: d.meta && d.meta.semantic_types || {},
    tableFilters: false,
    onViewChange: canEdit && spec.type === 'table' ? saveView : null
  });
  if (canEdit) TopbaseViz.renderSettings($('#viz-settings'), view.columns, view.rows, spec, applySpec);
}

function applySpec(next) {
  if (!canEdit) return;
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
  if (!canEdit || !question || !spec) return;
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
      temporaryFilters = true;
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
  const updated = resultMeta.executed_at ? new Date(resultMeta.executed_at).toLocaleTimeString(TopbaseLocale.locale()) : '';
  const limited = queryLimitMessage(resultMeta);
  const groupCrumb = $('#group-crumb'), groupSeparator = $('#group-crumb-separator');
  if (groupCrumb && groupSeparator) {
    groupCrumb.hidden = groupSeparator.hidden = !col;
    if (col) { groupCrumb.textContent = col.name; groupCrumb.href = '/collections/' + encodeURIComponent(col.id) + '/'; }
  }
  $('#title').translate = false;
  $('#title').title = TopbaseLocale.t('点击修改名称');
  $('#read-title').translate = false;
  $('#heading').translate = false;
  $('#title').textContent = question.name;
  $('#read-title').textContent = question.name;
  $('#heading').textContent = question.name;
  $('#meta').translate = false;
  $('#meta').textContent = [TopbaseLocale.t(kind), col ? col.name : TopbaseLocale.t('我的分析'), canEdit ? '' : TopbaseLocale.t('只读'), lastResult ? TopbaseLocale.t(freshness) + ' · ' + TopbaseLocale.t('{n} 行',{n}) : '', limited, updated ? TopbaseLocale.t('更新于 {time}',{time:updated}) : ''].filter(Boolean).join(' · ');
  const isSQL = !!(question.native_sql || (lastResult && lastResult.sql));
  TopbaseCode.setCode('#query-json', queryPreview(), {
    language: isSQL ? 'sql' : 'json',
    label: isSQL ? '查询 SQL' : '查询定义'
  });
}

async function saveName(name) {
  if (!canEdit) return;
  const nextName = String(name || '').trim();
  if (!nextName) {
    toast('分析名称不能为空');
    setMeta();
    return;
  }
  if (nextName === question.name) { setMeta(); return; }
  if (nameSaving) return;
  nameSaving = true;
  try {
    question = await api('/api/questions/' + id, 'PUT', {
      name: nextName,
      description: question.description || '',
      collection_id: question.collection_id || '',
      chartspec: spec || question.chartspec
    });
    setMeta();
    toast('名称已保存');
  } catch (e) {
    setMeta();
    toast(e.message);
  } finally {
    nameSaving = false;
  }
}

function editName() {
  if (!canEdit || !question || nameSaving || $('#heading input')) return;
  const heading = $('#heading');
  const input = document.createElement('input');
  input.className = 'analysis-name-input';
  input.value = question.name;
  input.setAttribute('aria-label', TopbaseLocale.t('分析名称'));
  heading.textContent = '';
  heading.append(input);
  input.focus();
  input.select();
  let submitted = false;
  const submit = () => { if (!submitted) { submitted = true; saveName(input.value); } };
  input.onkeydown = event => {
    if (event.key === 'Enter') { event.preventDefault(); submit(); }
    if (event.key === 'Escape') { submitted = true; setMeta(); }
  };
  input.onblur = submit;
}

async function loadResult() {
  showError('');
  if (!canRun) {
    $('#viz-stage').innerHTML = '<div class="viz-empty"><b>当前账号无法运行此查询</b><p>请联系管理员检查数据查看权限和 SQL 执行权限。</p></div>';
    return;
  }
  if (nativeParameters === null && nativeParameterFields(question).some(field => field.required)) {
    $('#viz-stage').innerHTML = '<div class="viz-empty"><b>等待查询参数</b><p>点击「查询参数」填写必填项后运行。</p></div>';
    return;
  }
  $('#viz-stage').innerHTML = '<div class="viz-empty"><b>正在运行</b><p>正在查询数据。</p></div>';
  try {
    lastResult = await run();
    spec = TopbaseViz.merge(question.chartspec || lastResult.chartspec, TopbaseViz.infer((lastResult.columns || []), (lastResult.rows || []), question.queryir), lastResult.columns || [], lastResult.rows || []);
    question.chartspec = spec;
    mountFilter(lastResult);
    setMeta();
    try {
      paint();
    } catch (renderError) {
      showError(TopbaseLocale.t('图表渲染失败：{error}',{error:renderError.message}));
      $('#viz-stage').innerHTML = '<div class="viz-empty"><b>数据已查询成功</b><p>图表暂时无法渲染，请切换为表格或重新运行。</p></div>';
      if (canEdit) TopbaseViz.renderTypes($('#viz-types'), spec || { type: 'table' }, type => applySpec(Object.assign({}, spec, { type })));
      $('#viz-settings').innerHTML = '<h3>设置</h3><p class="viz-muted">可切换到表格查看查询结果。</p>';
    }
  } catch (e) {
    lastResult = null;
    setMeta();
    showError(e.message);
    $('#viz-stage').innerHTML = '<div class="viz-empty"><b>结果不可见</b><p>' + esc(e.message) + '</p></div>';
    if (canEdit) TopbaseViz.renderTypes($('#viz-types'), spec || { type: 'table' }, () => {});
    $('#viz-settings').innerHTML = '<h3>设置</h3><p class="viz-muted">查询成功后，可以在这里更改图表类型、横轴和纵轴。</p>';
  }
}

async function boot() {
  question = await api('/api/questions/' + id + '?view=detail');
  canEdit = question.can_edit === true;
  canRun = question.can_run === true;
  document.body.toggleAttribute('data-can-edit', canEdit);
  $('#read-title').hidden = canEdit;
  if (canEdit) {
    const heading = $('#heading');
    heading.classList.add('editable-analysis-name');
    heading.setAttribute('role', 'button');
    heading.tabIndex = 0;
    heading.title = TopbaseLocale.t('点击修改名称');
  }
  $('#export-action').hidden = !canRun || !question.id || !!question.archived_at || !(question.queryir || question.native_sql);
  $('#warehouse-action').hidden = !question.can_materialize;
  const fields = nativeParameterFields(question);
  $('#native-controls').hidden = !canRun || !fields.length;
  if (!fields.some(field => field.required)) nativeParameters = {};
  collections = question.collection_name ? [{id:question.collection_id, name:question.collection_name}] : [];
  $('#materialize').href = '/warehouse/?question=' + encodeURIComponent(question.id) + '#create-materialization';
  spec = question.chartspec || { type: 'table' };
  setMeta();
  await loadResult();
}

boot().catch(e => {
  showError(e.message);
  if ($('#title')) $('#title').textContent = TopbaseLocale.t('无法打开');
  if ($('#read-title')) $('#read-title').textContent = TopbaseLocale.t('无法打开');
  if ($('#heading')) $('#heading').textContent = TopbaseLocale.t('无法打开分析');
  if ($('#meta')) $('#meta').textContent = e.message;
});

function on(id, fn) {
  const el = document.getElementById(id);
  if (el) el.onclick = fn;
}
function nativeParameterFields(saved) {
  const sql = saved.query_type === 'native' || !saved.queryir ? saved.native_sql || '' : '';
  const token = /\{\{\s*([A-Za-z_][A-Za-z0-9_]*)\s*\}\}/g;
  const names = [...new Set(Array.from(sql.matchAll(token), match => match[1]))];
  const requiredNames = new Set(Array.from(sql.replace(/\[\[[\s\S]*?\]\]/g, '').matchAll(token), match => match[1]));
  return names.map((name, i) => {
    const parameter = (saved.parameters || []).find(item => item.name === name) || {};
    const required = requiredNames.has(name) && !parameter.field && parameter.type !== 'field';
    return {name: 'p' + i, label: name, translateLabel: false, required, type: parameter.type === 'date' ? 'date' : 'text', value: nativeParameters?.[name] ?? ''};
  });
}

async function chooseNativeParameters(confirmText) {
  const fields = nativeParameterFields(question);
  if (!fields.length) return {};
  const input = await formDialog({title: '查询参数', description: '预览和导出使用相同的参数。可选项留空时跳过对应条件。', fields, confirmText});
  if (!input) return null;
  return Object.fromEntries(fields.flatMap(field => {
    const value = input[field.name];
    return value === '' ? [] : [[field.label, value]];
  }));
}

on('native-parameters', async () => {
  const button = $('#native-parameters');
  if (!canRun || button.disabled) return;
  button.disabled = true;
  try {
    const values = await chooseNativeParameters('运行分析');
    if (values === null) return;
    nativeParameters = values;
    await loadResult();
  } finally {
    button.disabled = false;
  }
});
async function exportSavedCSV() {
  const button = $('#export-csv');
  if (!canRun || !question?.id || button.disabled) return;
  button.disabled = true;
  let denied = false;
  try {
    if (temporaryFilters && !await confirmDialog({
      title: '导出已保存的查询结果？', description: '导出使用已保存的筛选条件。临时筛选仅用于预览。', confirmText: '导出已保存结果'
    })) return;
    const parameters = nativeParameters ?? await chooseNativeParameters('导出');
    if (parameters === null) return;
    nativeParameters = parameters;
    button.textContent = '正在导出…';
    const headers = {'Content-Type': 'application/json'};
    const csrf = document.cookie.match(/(?:^|; )topbase_csrf=([^;]+)/);
    if (csrf) headers['X-Topbase-CSRF'] = decodeURIComponent(csrf[1]);
    const response = await fetch('/api/questions/' + encodeURIComponent(id) + '/export', {
      method: 'POST', credentials: 'same-origin', headers, body: JSON.stringify({parameters})
    });
    if (!response.ok) {
      const body = await response.text();
      let message = body;
      try { message = JSON.parse(body).error || body; } catch {}
      denied = response.status === 403 && !/csrf|origin/i.test(message);
      if (response.status === 401) message = '登录已过期，请重新登录后导出。';
      if (denied) message = '当前账号缺少导出这条分析所需的权限。';
      throw Error(message || '导出失败，请重试。');
    }
    const blob = await response.blob();
    if (!blob.type.startsWith('text/csv')) throw Error('登录状态已变化，请刷新页面后重试。');
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = (question.name || id).replace(/[\\/:*?"<>|]/g, '_') + '.csv';
    document.body.append(link);
    link.click();
    link.remove();
    setTimeout(() => URL.revokeObjectURL(url), 1000);
    toast('CSV 已开始下载');
  } catch (error) {
    toast(error.message);
  } finally {
    button.disabled = denied;
    button.textContent = denied ? '无导出权限' : '导出 CSV';
    if (denied) button.title = '需要数据查看权限，原生 SQL 分析还需要 SQL 执行权限。';
  }
}
on('export-csv', exportSavedCSV);
on('edit-analysis', () => { if (canEdit) location.href = '/data/?edit=' + encodeURIComponent(id); });
on('title', editName);
on('heading', editName);
$('#heading').onkeydown = event => { if (event.key === 'Enter' || event.key === ' ') { event.preventDefault(); editName(); } };
on('move', async () => {
  if (!canEdit) return;
  try {
    collections = await api('/api/collections?access=edit');
  } catch (error) {
    toast(error.message);
    return;
  }
  const collection_id = await choiceDialog({
    kicker: '分析分组',
    title: '移动到分组',
    description: '选择目标分组。仪表盘可继续引用此分析。',
    label: '目标位置',
    value: question.collection_id || '',
    confirmText: '移动分析',
    options: [{ value: '', label: '我的分析', description: '保存到你的默认个人分组' }].concat(collections.filter(c=>!c.read_only).map(c => ({ value: c.id, label: c.name, translate:false, description: TopbaseLocale.t(c.kind === 'personal_project' ? '个人分组' : '企业项目') })))
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
  if (!canEdit) return;
  if (!await confirmDialog({ kicker: '删除分析', title: '删除这条分析？', description: '删除后可在回收站恢复。', confirmText: '删除', tone: 'danger' })) return;
  await api('/api/questions/' + id, 'DELETE');
  location.href = '/questions/';
});
