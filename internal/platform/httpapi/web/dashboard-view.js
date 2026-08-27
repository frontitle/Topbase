const COLS = 12, ROW = 76, GAP = 12, MIN_W = 2, MIN_H = 2;
const pathParts = location.pathname.split('/').filter(Boolean);
const isPublicDashboard = ['public', 'embed'].includes(pathParts[0]) && pathParts[1] === 'dashboard';
const boardId = isPublicDashboard ? '' : (pathParts[1] || new URLSearchParams(location.search).get('id'));
const publicUUID = isPublicDashboard ? pathParts[2] : '';
let board = null, questions = [], questionMap = {}, activeTab = '', editing = false, saveTimer = 0, drag = null, cardData = {}, selectedCard = '';
const APPEARANCE_DEFAULTS = { theme: 'deep', background: '#07111f', grid: true, snap: true, glow: true };

function newID(prefix) {
  return prefix + '_' + Math.random().toString(16).slice(2) + Date.now().toString(16).slice(-4);
}
function tabCards() {
  return (board.cards || []).filter(c => !activeTab || !c.tab_id || c.tab_id === activeTab);
}
function cellW() {
  const width = $('#grid').clientWidth;
  return (width - GAP * (COLS - 1)) / COLS;
}
function rect(layout) {
  const cw = cellW();
  return {
    left: layout.x * (cw + GAP),
    top: layout.y * (ROW + GAP),
    width: layout.w * cw + (layout.w - 1) * GAP,
    height: layout.h * ROW + (layout.h - 1) * GAP
  };
}
function clamp(n, a, b) { return Math.max(a, Math.min(b, n)); }
function overlaps(a, b) {
  return a.x < b.x + b.w && a.x + a.w > b.x && a.y < b.y + b.h && a.y + a.h > b.y;
}
function nextY(w) {
  w = w || 4;
  const cards = tabCards();
  if (!cards.length) return 0;
  return Math.max(...cards.map(c => (c.layout.y || 0) + (c.layout.h || 1)));
}
function resolveOverlap(moved) {
  const others = tabCards().filter(c => c.id !== moved.id);
  let guard = 0;
  while (guard++ < 80) {
    const hit = others.find(c => overlaps(moved.layout, c.layout));
    if (!hit) break;
    hit.layout.y = moved.layout.y + moved.layout.h;
  }
}
function cellAt(clientX, clientY, w, h) {
  const box = $('#grid').getBoundingClientRect();
  const cw = cellW();
  let x = Math.floor((clientX - box.left) / (cw + GAP));
  let y = Math.floor((clientY - box.top) / (ROW + GAP));
  x = clamp(x, 0, COLS - (w || 1));
  y = Math.max(0, y);
  return { x, y, w: w || 4, h: h || 4 };
}
function usedQuestionIDs() {
  return new Set((board.cards || []).filter(c => c.question_id).map(c => c.question_id));
}
function filterValues() {
  const values = {};
  document.querySelectorAll('[data-filter]').forEach(input => {
    const id = input.dataset.filter, key = input.dataset.key;
    if (key === 'value') { values[id] = input.value; return; }
    values[id] = values[id] || {};
    values[id][key] = input.value;
  });
  return values;
}
function payload() {
  return {
    name: board.name,
    description: board.description || '',
    collection_id: board.collection_id || '',
    auto_refresh_seconds: board.auto_refresh_seconds || 0,
    appearance: board.appearance || APPEARANCE_DEFAULTS,
    tabs: board.tabs || [],
    cards: board.cards || [],
    filters: board.filters || []
  };
}
function appearance() {
  board.appearance = Object.assign({}, APPEARANCE_DEFAULTS, board.appearance || {});
  return board.appearance;
}
function applyAppearance() {
  const canvas = $('#board-canvas');
  if (!canvas || !board) return;
  const a = appearance();
  canvas.dataset.theme = a.theme;
  canvas.style.setProperty('--canvas-custom-bg', a.background);
  canvas.classList.toggle('canvas-grid-off', !a.grid);
  canvas.classList.toggle('canvas-glow-off', !a.glow);
}
function renderCanvasControls() {
  const host = $('#canvas-controls');
  if (!host) return;
  host.hidden = !editing;
  if (!editing) return;
  const a = appearance();
  host.innerHTML = `<div class="canvas-control-group"><span>画布</span><button data-theme="deep" class="${a.theme==='deep'?'active':''}" type="button">深空</button><button data-theme="aurora" class="${a.theme==='aurora'?'active':''}" type="button">极光</button><button data-theme="light" class="${a.theme==='light'?'active':''}" type="button">明亮</button></div><label class="canvas-color">背景 <input id="canvas-background" type="color" value="${esc(a.background)}"></label><label class="canvas-switch"><input id="canvas-grid-toggle" type="checkbox" ${a.grid?'checked':''}> 网格</label><label class="canvas-switch"><input id="canvas-snap-toggle" type="checkbox" ${a.snap?'checked':''}> 吸附</label><label class="canvas-switch"><input id="canvas-glow-toggle" type="checkbox" ${a.glow?'checked':''}> 微光</label>`;
  $$('[data-theme]', host).forEach(button => button.onclick = () => { appearance().theme = button.dataset.theme; applyAppearance(); renderCanvasControls(); scheduleSave(); });
  $('#canvas-background').oninput = event => { appearance().background = event.target.value; applyAppearance(); scheduleSave(); };
  $('#canvas-grid-toggle').onchange = event => { appearance().grid = event.target.checked; applyAppearance(); scheduleSave(); };
  $('#canvas-snap-toggle').onchange = event => { appearance().snap = event.target.checked; scheduleSave(); };
  $('#canvas-glow-toggle').onchange = event => { appearance().glow = event.target.checked; applyAppearance(); scheduleSave(); };
}
async function saveBoard() {
  board = await api('/api/dashboards/' + board.id, 'PUT', payload());
  if (activeTab && !(board.tabs || []).some(t => t.id === activeTab)) activeTab = board.tabs?.[0]?.id || '';
}

function scheduleSave() {
  clearTimeout(saveTimer);
  saveTimer = setTimeout(async () => {
    try { await saveBoard(); }
    catch (e) { toast(e.message); }
  }, 350);
}

function renderTabs() {
  $('#tabs').innerHTML = (board.tabs || []).map(t => `<button data-tab="${t.id}" class="${t.id === activeTab ? 'active' : ''}">${esc(t.name)}</button>`).join('');
  $$('#tabs button').forEach(b => b.onclick = () => { activeTab = b.dataset.tab; render(); loadCards(); });
}
function renderFilters() {
  $('#filters').innerHTML = (board.filters || []).map(f => {
    if (f.type === 'date') return `<label>${esc(f.name)} 起<input data-filter="${f.id}" data-key="start" type="date"></label><label>止<input data-filter="${f.id}" data-key="end" type="date"></label>`;
    return `<label>${esc(f.name)}<input data-filter="${f.id}" data-key="value"></label>`;
  }).join('');
}
function renderPalette() {
  if (!$('#palette-questions')) return;
  const q = ($('#qsearch').value || '').toLowerCase();
  const used = usedQuestionIDs();
  const items = questions.filter(item => ((item.name || '') + ' ' + (item.query_type || '')).toLowerCase().includes(q));
  $('#palette-questions').innerHTML = items.map(item => {
    const added=used.has(item.id);
    return `<button class="palette-item ${added?'added':''}" draggable="${added?'false':'true'}" data-qid="${item.id}" type="button" ${added?'disabled':''}><b>${esc(item.name)}</b><small>${item.query_type === 'native' ? 'SQL 分析' : '可视化分析'} · ${added?'已添加':'点击添加'}</small></button>`;
  }).join('') || '<div class="palette-empty"><b>还没有分析</b><p>选择起始数据，通过可视化构建器保存第一条分析。</p><a href="/questions/new/">创建分析</a></div>';
  $$('#palette-questions .palette-item:not([disabled])').forEach(btn => {
    btn.ondragstart = ev => {
      ev.dataTransfer.setData('text/plain', btn.dataset.qid);
      ev.dataTransfer.effectAllowed = 'copy';
    };
    btn.onclick = () => addQuestion(btn.dataset.qid);
  });
}

function cardTitle(card) {
  if (card.title) return card.title;
  if (card.question_id && questionMap[card.question_id]) return questionMap[card.question_id].name;
  if (card.type === 'heading') return '标题';
  if (card.type === 'text') return '文本';
  if (card.type === 'link') return card.title || '链接';
  if (card.type === 'iframe') return card.title || '网页';
  if (card.type === 'metric') return card.title || '核心指标';
  return '分析卡';
}

function cardHTML(card) {
  const tools = editing ? `<div class="card-tools">
      ${card.type === 'question' ? `<select data-viz="${card.id}">${TopbaseViz.types.map(t => `<option value="${t.id}">${t.label}</option>`).join('')}</select>` : ''}
      <button type="button" data-copy="${card.id}">复制</button><button type="button" data-del="${card.id}">删除</button>
    </div>` : (card.question_id ? `<div class="card-tools"><a class="secondary" href="/questions/${card.question_id}/">打开</a></div>` : '');
  const body = card.type === 'heading'
    ? `<h2 ${editing ? 'contenteditable="true" data-edit="title"' : ''}>${esc(card.title || '标题')}</h2>`
    : card.type === 'text'
      ? `<p ${editing ? 'contenteditable="true" data-edit="body"' : ''}>${esc(card.body || '')}</p>`
      : card.type === 'link'
        ? (editing ? `<div class="card-config"><label>链接名称<input data-card-field="title" value="${esc(card.title||'链接')}"></label><label>跳转地址<input data-card-field="body" value="${esc(card.body||'https://')}"></label></div>` : `<a class="dashboard-link" href="${esc(card.body||'#')}" target="_blank" rel="noopener">${esc(card.title||card.body||'打开链接')}</a>`)
        : card.type === 'iframe'
          ? (editing ? `<div class="card-config"><label>网页标题<input data-card-field="title" value="${esc(card.title||'嵌入网页')}"></label><label>网页地址<input data-card-field="iframe_url" value="${esc(card.config&&card.config.url||'https://')}"></label></div>` : `<iframe class="dashboard-frame" src="${esc(card.config&&card.config.url||'about:blank')}" title="${esc(card.title||'嵌入网页')}" loading="lazy"></iframe>`)
          : card.type === 'metric'
            ? `<div class="dashboard-metric"><small ${editing ? 'contenteditable="true" data-edit="title"' : ''}>${esc(card.title || '核心指标')}</small><b ${editing ? 'contenteditable="true" data-edit="body"' : ''}>${esc(card.body || '98,765')}</b><em>较上周期 <i>↑ 12.6%</i></em></div>`
            : card.type === 'divider'
              ? `<div class="dashboard-divider"><span>${esc(card.title || '')}</span></div>`
          : `<div class="viz-stage" data-vizhost="${card.id}"><div class="viz-empty"><b>加载中</b><p>正在运行分析。</p></div></div>`;
  const title = card.type === 'heading' ? '' : `<div class="card-head" data-drag="${card.id}"><h3>${card.question_id ? `<a href="/questions/${card.question_id}/">${esc(cardTitle(card))}</a>` : esc(cardTitle(card))}</h3>${tools}</div>`;
  return `<article class="board-card ${editing ? 'editing' : ''} ${selectedCard===card.id ? 'selected' : ''} ${card.type === 'divider' ? 'board-divider-card' : ''}" data-card="${card.id}">
    ${title || `<div class="card-head" data-drag="${card.id}">${tools}</div>`}
    <div class="card-body">${body}</div>
    <div class="resize-handle" data-resize="${card.id}"></div>
  </article>`;
}

function layoutEls() {
  const cards = tabCards();
  let maxY = 4;
  cards.forEach(card => {
    const el = document.querySelector(`[data-card="${card.id}"]`);
    if (!el) return;
    const r = rect(card.layout || { x: 0, y: 0, w: 4, h: 4 });
    el.style.left = r.left + 'px';
    el.style.top = r.top + 'px';
    el.style.width = r.width + 'px';
    el.style.height = r.height + 'px';
    maxY = Math.max(maxY, (card.layout.y || 0) + (card.layout.h || 1));
  });
  $('#grid').style.minHeight = Math.max(420, maxY * (ROW + GAP) + 40) + 'px';
}

function bindCardChrome() {
  $$('[data-copy]').forEach(btn => btn.onclick = async ev => {
    ev.stopPropagation();
    const original = board.cards.find(c => c.id === btn.dataset.copy);
    if (!original) return;
    const copy = JSON.parse(JSON.stringify(original));
    copy.id = newID('crd'); copy.layout.y += 1;
    board.cards.push(copy); selectedCard = copy.id; render();
    try { await saveBoard(); } catch (e) { toast(e.message); }
  });
  $$('[data-del]').forEach(btn => btn.onclick = async ev => {
    ev.stopPropagation();
    board.cards = board.cards.filter(c => c.id !== btn.dataset.del);
    render();
    try { await saveBoard(); render(); await loadCards(); }
    catch (e) { toast(e.message); }
  });
  $$('[data-viz]').forEach(sel => {
    const card = board.cards.find(c => c.id === sel.dataset.viz);
    const current = card && card.config && card.config.chartspec && card.config.chartspec.type;
    const qspec = card && questionMap[card.question_id] && questionMap[card.question_id].chartspec;
    sel.value = current || (qspec && qspec.type) || 'table';
    sel.onchange = () => {
      card.config = Object.assign({}, card.config || {}, { chartspec: Object.assign({}, (card.config && card.config.chartspec) || qspec || {}, { type: sel.value }) });
      if (cardData[card.id]) paintCard(card, cardData[card.id]);
      scheduleSave();
    };
  });
  $$('[data-edit]').forEach(el => {
    el.onblur = () => {
      const card = board.cards.find(c => c.id === el.closest('[data-card]').dataset.card);
      if (!card) return;
      if (el.dataset.edit === 'title') card.title = el.textContent.trim();
      else card.body = el.innerText.trim();
      scheduleSave();
    };
  });
  $$('[data-card-field]').forEach(input => {
    input.onchange = () => {
      const card = board.cards.find(item => item.id === input.closest('[data-card]').dataset.card);
      if (!card) return;
      if (input.dataset.cardField === 'iframe_url') card.config = Object.assign({}, card.config || {}, { url: input.value.trim() });
      else card[input.dataset.cardField] = input.value.trim();
      scheduleSave();
    };
  });
  $$('[data-drag]').forEach(el => el.onpointerdown = ev => startDrag(ev, el.dataset.drag, 'move'));
  $$('[data-resize]').forEach(el => el.onpointerdown = ev => startDrag(ev, el.dataset.resize, 'resize'));
  $$('[data-card]').forEach(el => el.onclick = ev => { if (!editing || ev.target.closest('a,button,select,input,[contenteditable]')) return; selectedCard = el.dataset.card; render(); });
}

function startDrag(ev, cardId, mode) {
  if (!editing) return;
  if (ev.button != null && ev.button !== 0) return;
  if (ev.target.closest('a,button,select,input,[contenteditable]')) return;
  const card = board.cards.find(c => c.id === cardId);
  if (!card) return;
  ev.preventDefault();
  const el = document.querySelector(`[data-card="${cardId}"]`);
  el.setPointerCapture(ev.pointerId);
  el.classList.add('dragging');
  drag = {
    mode, card, el, pointer: ev.pointerId,
    startX: ev.clientX, startY: ev.clientY,
    layout: Object.assign({}, card.layout)
  };
  window.addEventListener('pointermove', onDrag);
  window.addEventListener('pointerup', endDrag);
}

function onDrag(ev) {
  if (!drag) return;
  const cw = cellW() + GAP;
  const dx = Math.round((ev.clientX - drag.startX) / cw);
  const dy = Math.round((ev.clientY - drag.startY) / (ROW + GAP));
  if (drag.mode === 'move') {
    drag.card.layout.x = clamp(drag.layout.x + dx, 0, COLS - drag.card.layout.w);
    drag.card.layout.y = Math.max(0, drag.layout.y + dy);
  } else {
    drag.card.layout.w = clamp(drag.layout.w + dx, MIN_W, COLS - drag.card.layout.x);
    drag.card.layout.h = clamp(drag.layout.h + dy, MIN_H, 24);
  }
  layoutEls();
}

function endDrag() {
  window.removeEventListener('pointermove', onDrag);
  window.removeEventListener('pointerup', endDrag);
  if (!drag) return;
  drag.el.classList.remove('dragging');
  resolveOverlap(drag.card);
  drag = null;
  layoutEls();
  paintCached();
  scheduleSave();
}

function render() {
  applyAppearance();
  $('#layout')?.classList.toggle('viewing', !editing);
  $('#grid').classList.toggle('editing', editing);
  if ($('#view-actions')) $('#view-actions').hidden = editing;
  if ($('#edit')) $('#edit').textContent = editing ? '完成编辑' : '编辑';
  const cards = tabCards();
  $('#grid').innerHTML = cards.map(cardHTML).join('') + (!cards.length ? '<div class="board-empty"><b>从左侧开始搭建仪表盘</b><p>点击分析即可加入，也可以添加标题、文本、链接或网页。</p></div>' : '');
  layoutEls();
  bindCardChrome();
  bindGridDrop();
  renderPalette();
  renderCanvasControls();
}

function bindGridDrop() {
  const grid = $('#grid');
  grid.ondragover = ev => {
    if (!editing) return;
    ev.preventDefault();
    ev.dataTransfer.dropEffect = 'copy';
    let ghost = $('#ghost');
    if (!ghost) {
      ghost = document.createElement('div');
      ghost.id = 'ghost';
      ghost.className = 'board-ghost';
      grid.appendChild(ghost);
    }
    const cell = cellAt(ev.clientX, ev.clientY, 6, 5);
    const r = rect(cell);
    ghost.style.left = r.left + 'px';
    ghost.style.top = r.top + 'px';
    ghost.style.width = r.width + 'px';
    ghost.style.height = r.height + 'px';
  };
  grid.ondragleave = ev => {
    if (ev.target === grid) { const g = $('#ghost'); if (g) g.remove(); }
  };
  grid.ondrop = ev => {
    ev.preventDefault();
    const g = $('#ghost'); if (g) g.remove();
    const value = ev.dataTransfer.getData('text/plain');
    if (!value) return;
    const cell = cellAt(ev.clientX, ev.clientY, 6, 5);
    if (value.startsWith('component:')) addStatic(value.slice('component:'.length), cell);
    else addQuestion(value, cell);
  };
}

async function addQuestion(questionId, layout) {
  if (!questionId || usedQuestionIDs().has(questionId)) return toast('这条分析已经在看板上');
  const q = questionMap[questionId];
  const place = layout || { x: 0, y: nextY(6), w: 6, h: 5 };
  const card = {
    id: newID('crd'),
    type: 'question',
    question_id: questionId,
    title: q ? q.name : '',
    tab_id: activeTab,
    layout: place,
    config: { chartspec: { type: (q && q.chartspec && q.chartspec.type) || 'table' } }
  };
  resolveOverlap(card);
  board.cards = (board.cards || []).concat([card]);
  render();
  try {
    await saveBoard();
    render();
    await loadCards();
  } catch (e) { toast(e.message); }
}

async function addStatic(type, layout) {
  const defaults = {
    heading: { type:'heading', title:'标题', layout:{x:0,y:nextY(12),w:12,h:1} },
    text: { type:'text', body:'说明文字', layout:{x:0,y:nextY(12),w:12,h:2} },
    link: { type:'link', title:'链接', body:'https://', layout:{x:0,y:nextY(6),w:6,h:2} },
    iframe: { type:'iframe', title:'嵌入网页', config:{url:'https://'}, layout:{x:0,y:nextY(6),w:6,h:5} },
    metric: { type:'metric', title:'核心指标', body:'98,765', layout:{x:0,y:nextY(3),w:3,h:2} },
    divider: { type:'divider', title:'', layout:{x:0,y:nextY(12),w:12,h:1} }
  };
  const value=defaults[type];
  if(!value)return;
  const card = Object.assign({id:newID('crd'),tab_id:activeTab},value,{layout:layout||value.layout});
  resolveOverlap(card);
  board.cards = (board.cards || []).concat([card]);
  render();
  try { await saveBoard(); render(); }
  catch (e) { toast(e.message); }
}

function paintCard(card, d) {
  const host = document.querySelector(`[data-vizhost="${card.id}"]`);
  if (!host) return;
  const q = questionMap[card.question_id];
  const qspec = (q && q.chartspec) || {};
  const cardSpec = (card.config && card.config.chartspec) || {};
  const spec = TopbaseViz.merge(
    Object.assign({}, qspec, cardSpec),
    d.chartspec,
    d.columns || [],
    d.rows || []
  );
  spec.dashboard_theme = appearance().theme;
  TopbaseViz.render(host, { columns: d.columns || [], rows: d.rows || [], spec, queryir: q && q.queryir, compact: true });
  if (!editing) {
    const el = host.closest('[data-card]');
    el.onclick = ev => {
      if (ev.target.closest('a,button,select')) return;
      const click = card.click || {};
      if (click.type === 'link' && click.url) { location.href = click.url; return; }
      if ((click.type === 'update_filter' || !click.type) && board.filters?.[0] && d.rows[0]) {
        const first = document.querySelector(`[data-filter="${click.filter_id || board.filters[0].id}"]`);
        if (first) { first.value = d.rows[0][0]; loadCards(); }
      }
    };
  }
}

function closeQuestionPicker() {
  $('#modal').hidden = true;
  $('#modal').innerHTML = '';
}
function openQuestionPicker() {
  if (!editing) { editing = true; render(); }
  const modal = $('#modal');
  const draw = query => {
    const needle = (query || '').toLowerCase();
    const used = usedQuestionIDs();
    const items = questions.filter(question => ((question.name || '') + ' ' + (question.description || '')).toLowerCase().includes(needle));
    modal.innerHTML = `<section class="question-picker" role="dialog" aria-modal="true" aria-label="添加分析">
      <header><div><small>仪表盘</small><h2>添加分析</h2><p>选择一条分析，立即作为卡片加入当前仪表盘。</p></div><button class="secondary" id="close-picker" type="button">关闭</button></header>
      <input id="picker-search" type="search" value="${esc(query || '')}" placeholder="搜索全部分析">
      <div class="picker-list">${items.map(question => {
        const added = used.has(question.id);
        const kind = question.query_type === 'native' ? 'SQL 查询' : '可视化分析';
        return `<button class="picker-question" data-picker-question="${esc(question.id)}" type="button" ${added ? 'disabled' : ''}><span><b>${esc(question.name)}</b><small>${kind}${question.description ? ' · ' + esc(question.description) : ''}</small></span><em>${added ? '已添加' : '添加'}</em></button>`;
      }).join('') || '<p class="hint">没有找到匹配的分析。</p>'}</div>
    </section>`;
    $('#close-picker').onclick = closeQuestionPicker;
    $('#picker-search').oninput = event => draw(event.target.value);
    $$('.picker-question:not([disabled])').forEach(button => button.onclick = async () => {
      button.disabled = true;
      button.querySelector('em').textContent = '添加中…';
      try { await addQuestion(button.dataset.pickerQuestion); closeQuestionPicker(); toast('已添加到仪表盘'); }
      catch (error) { toast(error.message); draw(query); }
    });
  };
  modal.hidden = false;
  modal.onclick = event => { if (event.target === modal) closeQuestionPicker(); };
  draw('');
  setTimeout(() => $('#picker-search')?.focus(), 0);
}

function paintCached() {
  tabCards().forEach(card => {
    if (card.type === 'question' && cardData[card.id]) paintCard(card, cardData[card.id]);
  });
}

async function loadCards() {
  const filters = filterValues();
  for (const card of board.cards || []) {
    if (card.type !== 'question') continue;
    if (activeTab && card.tab_id && card.tab_id !== activeTab) continue;
    const host = document.querySelector(`[data-vizhost="${card.id}"]`);
    if (!host) continue;
    try {
      const path = isPublicDashboard
        ? `/api/public/dashboard/${publicUUID}/cards/${card.id}/dataset`
        : `/api/dashboards/${board.id}/cards/${card.id}/dataset`;
      const d = await api(path, 'POST', { filters });
      cardData[card.id] = d;
      paintCard(card, d);
    } catch (e) {
      host.innerHTML = `<div class="viz-error"><b>卡片无法显示</b><p>${esc(e.message)}</p></div>`;
    }
  }
}

async function boot() {
  if (isPublicDashboard) {
    const payload = await api('/api/public/dashboard/' + publicUUID);
    board = payload.dashboard;
    questions = payload.questions || [];
  } else {
    [board, questions] = await Promise.all([api('/api/dashboards/' + boardId), api('/api/questions')]);
  }
  questionMap = Object.fromEntries(questions.map(q => [q.id, q]));
  if ($('#title')) $('#title').textContent = board.name;
  activeTab = board.tabs?.[0]?.id || '';
  editing = !isPublicDashboard && !(board.cards || []).length;
  renderTabs();
  renderFilters();
  render();
  await loadCards();
}

if ($('#edit')) $('#edit').onclick = async () => {
  if (editing) {
    try {
      await saveBoard();
      if (!(board.cards || []).length) {
        editing = true;
        toast('空仪表盘会保持编辑状态，请从左侧添加内容');
        render();
        return;
      }
      editing = false;
      toast('已保存');
      render();
      await loadCards();
    } catch (e) { toast(e.message); }
    return;
  }
  editing = true;
  render();
  await loadCards();
};
$('#apply').onclick = loadCards;
if ($('#qsearch')) $('#qsearch').oninput = renderPalette;
$$('[data-add]').forEach(button=>{
  button.onclick=()=>addStatic(button.dataset.add);
  button.ondragstart=event=>{
    event.dataTransfer.setData('text/plain','component:'+button.dataset.add);
    event.dataTransfer.effectAllowed='copy';
  };
});
window.addEventListener('resize', () => { layoutEls(); paintCached(); });

function closeActionModal(){ $('#modal').hidden=true;$('#modal').innerHTML=''; }
async function openShare(){
  try{
    const shared=await api(`/api/dashboards/${boardId}/public-link`,'POST',{});
    const publicURL=shared.public_url||'';
    const embedURL=shared.embed_url||'';
    $('#modal').hidden=false;
    $('#modal').innerHTML=`<section class="action-dialog"><header><div><small>非编辑状态功能</small><h2>分享与嵌入</h2><p>公开链接可直接访问；嵌入地址适合放入门户或业务系统。</p></div><button id="close-action" class="secondary" type="button">关闭</button></header><label>公开链接<div class="copy-row"><input readonly value="${esc(publicURL)}"><button data-copy="${esc(publicURL)}" class="secondary" type="button">复制</button></div></label><label>嵌入地址<div class="copy-row"><input readonly value="${esc(embedURL)}"><button data-copy="${esc(embedURL)}" class="secondary" type="button">复制</button></div></label><button id="disable-public" class="danger-text" type="button">关闭公开访问</button></section>`;
    $('#close-action').onclick=closeActionModal;
    $$('[data-copy]').forEach(button=>button.onclick=async()=>{await navigator.clipboard.writeText(button.dataset.copy);toast('已复制')});
    $('#disable-public').onclick=async()=>{await api(`/api/dashboards/${boardId}/public-link`,'DELETE');closeActionModal();toast('已关闭公开访问')};
  }catch(error){toast(error.message)}
}
async function addBookmark(){try{await api('/api/bookmarks','POST',{target_type:'dashboard',target_id:boardId});toast('已加入书签')}catch(error){toast(error.message)}}
async function archiveBoard(){if(!await confirmDialog({kicker:'仪表盘管理',title:'将这个仪表盘移到回收站？',description:'归档后会从仪表盘列表中隐藏，之后仍可在回收站恢复。',confirmText:'移到回收站',tone:'danger'}))return;try{await api('/api/dashboards/'+boardId,'DELETE');location.href='/dashboard/'}catch(error){toast(error.message)}}
async function duplicateBoard(){try{const copy=await api(`/api/dashboards/${boardId}/copy`,'POST',{});location.href='/dashboard/'+copy.id+'/'}catch(error){toast(error.message)}}
async function createAlert(){
  const card = (board.cards || []).find(c => c.type === 'question' && c.question_id);
  if (!card) return toast('这张板没有分析卡');
  try {
    const alert = await api('/api/alerts', 'POST', { name: board.name + ' 有结果', question_id: card.question_id, kind: 'results', channel: 'inbox' });
    const note = await api('/api/alerts/' + alert.id + '/run', 'POST', {});
    toast(note.title + '：' + note.body);
  } catch (e) { toast(e.message); }
}
async function subscribeBoard(){
  const channel = await choiceDialog({kicker:'仪表盘订阅',title:'选择每天的推送渠道',description:'订阅将在每天 09:00 执行，可以在管理后台查看运行状态。',label:'通知渠道',value:'inbox',confirmText:'创建订阅',options:[{value:'inbox',label:'站内通知',description:'在 Topbase 通知中心接收结果'},{value:'feishu',label:'飞书通知',description:'发送到已配置的飞书群机器人 Webhook'}]});
  if(channel===null)return;
  try {
    const sub = await api(`/api/dashboards/${boardId}/subscriptions`, 'POST', { cron: '0 9 * * *', channel });
    toast('已订阅每天 09:00（' + sub.channel + '）');
  } catch (e) { toast(e.message); }
}
function toggleMore(){
  const menu=$('#more-menu');
  menu.hidden=!menu.hidden;
  if(menu.hidden)return;
  menu.innerHTML='<button data-more="bookmark" type="button">加入书签</button><button data-more="duplicate" type="button">创建副本</button><button data-more="alert" type="button">设置结果提醒</button><button data-more="subscribe" type="button">订阅仪表盘</button><hr><button data-more="archive" class="danger-text" type="button">移到回收站</button>';
  $$('[data-more]').forEach(button=>button.onclick=()=>{
    menu.hidden=true;
    ({bookmark:addBookmark,duplicate:duplicateBoard,alert:createAlert,subscribe:subscribeBoard,archive:archiveBoard})[button.dataset.more]();
  });
}
if ($('#fullscreen')) $('#fullscreen').onclick=()=>$('#board-canvas')?.requestFullscreen?.();
if ($('#share')) $('#share').onclick=openShare;
if ($('#more')) $('#more').onclick=event=>{event.stopPropagation();toggleMore()};
if ($('#modal')) $('#modal').onclick=event=>{if(event.target===$('#modal'))closeActionModal()};
document.addEventListener('click',event=>{if(!event.target.closest('#more-menu,#more') && $('#more-menu'))$('#more-menu').hidden=true});
boot().catch(e => { if ($('#title')) $('#title').textContent = e.message; else toast(e.message); });
