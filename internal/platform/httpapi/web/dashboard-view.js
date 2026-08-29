// A denser grid gives authors enough precision for data-screen compositions,
// while grid-version migration keeps existing 12-column dashboards unchanged.
const COLS = 24, ROW = 34, X_GAP = 6, Y_GAP = 10, MIN_W = 2, MIN_H = 2, GRID_VERSION = 2;
const pathParts = location.pathname.split('/').filter(Boolean);
const isPublicDashboard = ['public', 'embed'].includes(pathParts[0]) && pathParts[1] === 'dashboard';
const boardId = isPublicDashboard ? '' : (pathParts[1] || new URLSearchParams(location.search).get('id'));
const publicUUID = isPublicDashboard ? pathParts[2] : '';
let board = null, questions = [], collections = [], questionMap = {}, activeTab = '', editing = false, saveTimer = 0, drag = null, cardData = {}, selectedCard = '', styleCard = '', paletteLimit = 60, paletteTab = 'analysis', motionTimers = [], layoutFrame = 0, canvasResizeObserver = null, viewer = null;
const APPEARANCE_DEFAULTS = { theme: 'deep', background: '#07111f', grid: true, snap: true, glow: true };

function newID(prefix) {
  return prefix + '_' + Math.random().toString(16).slice(2) + Date.now().toString(16).slice(-4);
}
function tabCards() {
  return (board.cards || []).filter(c => !activeTab || !c.tab_id || c.tab_id === activeTab);
}
function cellW() {
  const grid = $('#grid');
  const width = grid.getBoundingClientRect().width || grid.clientWidth;
  return (width - X_GAP * (COLS - 1)) / COLS;
}
function rect(layout) {
  const cw = cellW();
  return {
    left: Math.round(layout.x * (cw + X_GAP)),
    top: Math.round(layout.y * (ROW + Y_GAP)),
    width: Math.round(layout.w * cw + (layout.w - 1) * X_GAP),
    height: Math.round(layout.h * ROW + (layout.h - 1) * Y_GAP)
  };
}
function clamp(n, a, b) { return Math.max(a, Math.min(b, n)); }
function minHeight(card) { return ['heading', 'text', 'divider'].includes(card.type) ? 1 : MIN_H; }
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
  let x = Math.floor((clientX - box.left) / (cw + X_GAP));
  let y = Math.floor((clientY - box.top) / (ROW + Y_GAP));
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
function upgradeCanvasGrid() {
  const a = appearance();
  if (Number(a.grid_version) >= GRID_VERSION) return;
  (board.cards || []).forEach(card => {
    if (!card.layout) return;
    card.layout.x = Math.max(0, Math.round(Number(card.layout.x || 0) * 2));
    card.layout.y = Math.max(0, Math.round(Number(card.layout.y || 0) * 2));
    card.layout.w = Math.max(1, Math.round(Number(card.layout.w || 4) * 2));
    card.layout.h = Math.max(1, Math.round(Number(card.layout.h || 4) * 2));
  });
  a.grid_version = GRID_VERSION;
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
function renderPublicBadge() {
  const badge = $('#public-badge');
  if (badge) badge.hidden = !board?.public_uuid;
}
function renderCanvasControls() {
  const host = $('#canvas-controls');
  if (!host) return;
  host.hidden = !editing;
  if (!editing) return;
  const a = appearance();
  host.innerHTML = `<span class="canvas-label">画布</span><div class="canvas-control-group"><button data-theme="deep" class="${a.theme==='deep'?'active':''}" type="button">深空</button><button data-theme="aurora" class="${a.theme==='aurora'?'active':''}" type="button">极光</button><button data-theme="light" class="${a.theme==='light'?'active':''}" type="button">明亮</button></div><label class="canvas-color">背景 <input id="canvas-background" type="color" value="${esc(a.background)}"></label><label class="canvas-switch"><input id="canvas-grid-toggle" type="checkbox" ${a.grid?'checked':''}> 网格</label><label class="canvas-switch"><input id="canvas-snap-toggle" type="checkbox" ${a.snap?'checked':''}> 吸附</label><label class="canvas-switch"><input id="canvas-glow-toggle" type="checkbox" ${a.glow?'checked':''}> 微光</label>`;
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
  const tabs = board.tabs || [];
  $('#tabs').hidden = tabs.length <= 1;
  $('#tabs').innerHTML = tabs.length <= 1 ? '' : tabs.map(t => `<button data-tab="${t.id}" class="${t.id === activeTab ? 'active' : ''}">${esc(t.name)}</button>`).join('');
  $('.board-toolbar').hidden = tabs.length <= 1 && !(board.filters || []).length;
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
  const collectionID = $('#qcollection').value;
  const used = usedQuestionIDs();
  const items = questions.filter(item => (!collectionID || (collectionID === 'unassigned' ? !item.collection_id : item.collection_id === collectionID)) && ((item.name || '') + ' ' + (item.description || '') + ' ' + (item.query_type || '')).toLowerCase().includes(q));
  const shown = items.slice(0, paletteLimit);
  $('#palette-questions').innerHTML = shown.map(item => {
    const added=used.has(item.id);
    const group = collections.find(collection => collection.id === item.collection_id);
    return `<button class="palette-item ${added?'added':''}" draggable="${added?'false':'true'}" data-qid="${item.id}" type="button" ${added?'disabled':''}><b>${esc(item.name)}</b><small>${group ? esc(group.name) + ' · ' : ''}${item.query_type === 'native' ? 'SQL 分析' : '可视化分析'} · ${added?'已添加':'点击添加'}</small></button>`;
  }).join('') || '<div class="palette-empty"><b>还没有分析</b><p>选择起始数据，通过可视化构建器保存第一条分析。</p><a href="/questions/new/">创建分析</a></div>';
  $('#palette-more').hidden = items.length <= shown.length;
  $('#palette-more').textContent = `加载更多（还有 ${items.length - shown.length} 条）`;
  $$('#palette-questions .palette-item:not([disabled])').forEach(btn => {
    btn.ondragstart = ev => {
      ev.dataTransfer.setData('text/plain', btn.dataset.qid);
      ev.dataTransfer.effectAllowed = 'copy';
    };
    btn.onclick = () => addQuestion(btn.dataset.qid);
  });
}
function renderPaletteTabs() {
  $$('[data-palette-tab]').forEach(button => {
    const active = button.dataset.paletteTab === paletteTab;
    button.classList.toggle('active', active);
    button.setAttribute('aria-selected', String(active));
  });
  $$('[data-palette-panel]').forEach(panel => { panel.hidden = panel.dataset.palettePanel !== paletteTab; });
}
function renderCollectionFilter() {
  const select = $('#qcollection');
  if (!select) return;
  const selected = select.value;
  const counts = new Map();
  questions.forEach(question => counts.set(question.collection_id || '', (counts.get(question.collection_id || '') || 0) + 1));
  select.innerHTML = `<option value="">全部分组（${questions.length}）</option>` + collections.filter(collection => counts.has(collection.id)).map(collection => `<option value="${esc(collection.id)}">${esc(collection.name)}（${counts.get(collection.id)}）</option>`).join('') + (counts.has('') ? `<option value="unassigned">未分组（${counts.get('')}）</option>` : '');
  select.value = selected;
}

function cardTitle(card) {
  if (card.title) return card.title;
  if (card.question_id && questionMap[card.question_id]) return questionMap[card.question_id].name;
  if (card.type === 'heading') return '标题';
  if (card.type === 'text') return '文本';
  if (card.type === 'link') return card.title || '链接';
  if (card.type === 'iframe') return card.title || '网页';
  if (card.type === 'divider') return card.title || '分隔标题';
  if (card.type === 'metric') return card.title || '核心指标';
  return '分析卡';
}
function richText(card) {
  const raw = card.config && card.config.rich_text;
  if (!raw) return esc(card.body || '');
  const doc = new DOMParser().parseFromString(raw, 'text/html');
  doc.body.querySelectorAll('*').forEach(node => {
    if (!['B','STRONG','I','EM','U','UL','OL','LI','BR','A','P','DIV'].includes(node.tagName)) node.replaceWith(...node.childNodes);
    else [...node.attributes].forEach(attr => { if (!(node.tagName === 'A' && attr.name === 'href')) node.removeAttribute(attr.name); });
  });
  doc.body.querySelectorAll('a').forEach(link => { link.target = '_blank'; link.rel = 'noopener'; });
  return doc.body.innerHTML;
}
function typography(card) {
  const style = (card.config && card.config.typography) || {};
  return { font: ['system','serif','mono'].includes(style.font) ? style.font : 'system', size: [12,14,16,18,22,28].includes(Number(style.size)) ? Number(style.size) : (card.type === 'heading' ? 22 : 14) };
}
function typographyStyle(card) {
  const style = typography(card);
  const font = { system: 'var(--tb-font)', serif: 'Georgia, "Noto Serif SC", serif', mono: 'ui-monospace, SFMono-Regular, Menlo, monospace' }[style.font];
  return `font-family:${font};font-size:${style.size}px`;
}
function typographyTools(card) {
  const style = typography(card);
  return `<div class="rich-text-tools typography-tools"><button data-format="bold" type="button"><b>B</b></button><button data-format="italic" type="button"><i>I</i></button><button data-format="underline" type="button"><u>U</u></button>${card.type === 'text' ? '<button data-format="insertUnorderedList" type="button">• 列表</button>' : ''}<select data-typography="font" data-card-id="${card.id}"><option value="system" ${style.font==='system'?'selected':''}>无衬线</option><option value="serif" ${style.font==='serif'?'selected':''}>衬线</option><option value="mono" ${style.font==='mono'?'selected':''}>等宽</option></select><select data-typography="size" data-card-id="${card.id}">${[12,14,16,18,22,28].map(size => `<option value="${size}" ${style.size===size?'selected':''}>${size}px</option>`).join('')}</select></div>`;
}
function presentation(card) {
  return Object.assign({ border: 'default', surface: 'solid', padding: 'normal', radius: 'round', shadow: 'default', align: 'left', show_header: true }, (card.config && card.config.presentation) || {});
}
function cardVizType(card) {
  if (card.type !== 'question') return '';
  const own = card.config && card.config.chartspec && card.config.chartspec.type;
  const source = questionMap[card.question_id] && questionMap[card.question_id].chartspec;
  // 表格是未指定图形时的默认呈现；因此没有明确图形类型的分析卡同样按表格处理。
  return own || (source && source.type) || 'table';
}
function presentationPanel(card) {
  const p = presentation(card);
  const select = (key, values) => `<label>${key === 'border' ? '边框' : key === 'surface' ? '底色' : key === 'padding' ? '留白' : key === 'radius' ? '圆角' : '对齐'}<select data-presentation="${key}" data-card-id="${card.id}">${values.map(([value, label]) => `<option value="${value}" ${p[key] === value ? 'selected' : ''}>${label}</option>`).join('')}</select></label>`;
  const hasBorder = p.border !== 'none';
  const canShadow = hasBorder && p.surface !== 'transparent';
  let componentOptions = '';
  if (card.type === 'text') componentOptions = `<section class="card-style-section"><b>内容效果</b><label class="card-style-switch"><span>文字效果</span><select data-effect="text" data-card-id="${card.id}"><option value="none">静态</option><option value="marquee-left">横向走马灯</option><option value="marquee-up">纵向走马灯</option></select></label></section>`;
  else if (card.type === 'image') componentOptions = `<section class="card-style-section"><b>播放效果</b><label class="card-style-switch"><input data-carousel="enabled" data-card-id="${card.id}" type="checkbox" ${card.config && card.config.carousel ? 'checked' : ''}> 自动轮播</label><label class="card-style-switch"><span>间隔</span><select data-carousel="interval" data-card-id="${card.id}">${[3,5,8,12].map(value => `<option value="${value}" ${card.config && Number(card.config.carousel_interval) === value ? 'selected' : ''}>${value} 秒</option>`).join('')}</select></label></section>`;
  else if (card.type === 'time') componentOptions = `<section class="card-style-section"><b>时间内容</b><label class="card-style-switch"><span>显示内容</span><select data-time-mode data-card-id="${card.id}"><option value="datetime">日期和时间</option><option value="date">仅日期</option><option value="time">仅时间</option><option value="weekday">星期和日期</option></select></label></section>`;
  else if (cardVizType(card) === 'table') componentOptions = `<section class="card-style-section"><b>表格呈现</b><label class="card-style-switch"><span>数据动效</span><select data-table-motion data-card-id="${card.id}"><option value="none">静态</option><option value="scroll">数据滚动</option><option value="page">自动翻页</option></select></label></section>`;
  return `<div class="card-style-panel"><header><b>组件样式</b><button data-close-style type="button" aria-label="关闭">×</button></header><section class="card-style-section"><b>布局</b><div class="card-style-grid">${select('border', [['default','默认边框'],['none','无边框'],['glow','发光边框']])}${select('surface', [['solid','实色底'],['transparent','透明'],['glass','玻璃']])}${select('padding', [['normal','标准'],['compact','紧凑'],['none','无内边距']])}${hasBorder ? select('radius', [['round','圆角'],['square','直角']]) : ''}${select('align', [['left','居左'],['center','居中'],['right','居右']])}</div>${canShadow ? `<label class="card-style-switch"><input data-presentation="shadow" data-card-id="${card.id}" type="checkbox" ${p.shadow !== 'none' ? 'checked' : ''}> 显示阴影</label>` : ''}<label class="card-style-switch"><input data-presentation="show_header" data-card-id="${card.id}" type="checkbox" ${p.show_header !== false ? 'checked' : ''}> 展示标题栏</label></section>${componentOptions}</div>`;
}

function cardHTML(card) {
  const p = presentation(card);
  const tools = editing ? `<div class="card-tools">
      ${card.type === 'question' ? `<select data-viz="${card.id}">${TopbaseViz.types.map(t => `<option value="${t.id}">${t.label}</option>`).join('')}</select>` : ''}
      <button type="button" data-card-style="${card.id}" title="组件样式" aria-label="组件样式">◈</button><button class="card-delete" type="button" data-del="${card.id}" title="删除组件" aria-label="删除组件">×</button>
    </div>` : '';
  const body = card.type === 'heading'
    ? `<div class="heading-wrap">${editing ? typographyTools(card) : ''}<h2 style="${typographyStyle(card)}" ${editing ? 'contenteditable="true" data-edit="title"' : ''}>${esc(card.title || '标题')}</h2></div>`
      : card.type === 'text'
      ? `<div class="rich-text-wrap">${editing ? typographyTools(card) : ''}<div class="rich-text text-effect-${editing ? 'none' : (card.config && card.config.text_effect || 'none')}" style="${typographyStyle(card)}" ${editing ? 'contenteditable="true" data-edit="body"' : ''}>${richText(card)}</div></div>`
      : card.type === 'link'
        ? (editing ? `<div class="card-config"><label>链接名称<input data-card-field="title" value="${esc(card.title||'链接')}"></label><label>跳转地址<input data-card-field="body" value="${esc(card.body||'https://')}"></label></div>` : `<a class="dashboard-link" href="${esc(card.body||'#')}" target="_blank" rel="noopener">${esc(card.title||card.body||'打开链接')}</a>`)
      : card.type === 'plugin'
        ? (editing ? `<div class="plugin-editor"><div class="card-config"><label>组件名称<input data-card-field="title" value="${esc(card.title||'自定义组件')}"></label><label>组件运行地址<input data-card-field="plugin_url" value="${esc(card.config&&card.config.url||'https://')}"></label><p class="embed-validation" data-embed-validation="${card.id}">组件运行在隔离 iframe 中；配置变化会实时传递给预览。</p></div>${pluginHTML(card)}</div>` : pluginHTML(card))
      : card.type === 'iframe'
          ? (editing ? `<div class="card-config"><label>网页标题<input data-card-field="title" value="${esc(card.title||'嵌入网页')}"></label><label>网页地址<input data-card-field="iframe_url" value="${esc(card.config&&card.config.url||'https://')}"></label><p class="embed-validation" data-embed-validation="${card.id}">输入地址后会自动检查是否允许嵌入。</p></div>` : `<div class="dashboard-frame-wrap"><iframe class="dashboard-frame" src="${esc(card.config&&card.config.url||'about:blank')}" title="${esc(card.title||'嵌入网页')}" loading="lazy"></iframe><a class="iframe-open" href="${esc(card.config&&card.config.url||'#')}" target="_blank" rel="noopener">无法显示？新窗口打开 ↗</a></div>`)
          : card.type === 'image'
            ? imageHTML(card)
          : card.type === 'time'
            ? `<div class="dashboard-time" data-clock="${card.id}"></div>`
          : card.type === 'divider'
              ? `<div class="dashboard-divider"></div>`
          : `<div class="viz-stage" data-vizhost="${card.id}"><div class="viz-empty"><b>加载中</b><p>正在运行分析。</p></div></div>`;
  const hasTitle = card.question_id || card.type === 'divider';
  const title = !hasTitle || (!editing && p.show_header === false) ? '' : `<div class="card-head" data-drag="${card.id}"><h3 ${card.type === 'divider' && editing ? 'contenteditable="true" data-edit="title"' : ''}>${card.question_id ? `<a href="/questions/${card.question_id}/">${esc(cardTitle(card))}</a>` : esc(cardTitle(card))}</h3>${tools}</div>`;
  return `<article class="board-card ${editing ? 'editing' : ''} ${styleCard===card.id ? 'style-open' : ''} ${selectedCard===card.id ? 'selected' : ''} ${card.type === 'divider' ? 'board-divider-card' : ''} card-border-${p.border} card-surface-${p.surface} card-padding-${p.padding} card-radius-${p.radius} card-shadow-${p.shadow} card-align-${p.align}" data-card="${card.id}">
    ${title || (editing ? `<div class="card-head card-head-tools-only" data-drag="${card.id}">${tools}</div>` : '')}
    <div class="card-body">${body}</div>
    <div class="resize-handle" data-resize="${card.id}"></div>
    ${editing && styleCard === card.id ? presentationPanel(card) : ''}
  </article>`;
}
function pluginHTML(card) {
  const url = card.config && card.config.url;
  if (!url || url === 'https://') return '<div class="plugin-placeholder"><b>尚未配置自定义组件</b><span>在编辑状态输入组件运行地址。</span></div>';
  return `<div class="dashboard-plugin-wrap"><iframe class="dashboard-plugin-frame" data-plugin-card="${card.id}" src="${esc(url)}" title="${esc(card.title || '自定义组件')}" sandbox="allow-scripts allow-forms" referrerpolicy="no-referrer" loading="lazy"></iframe></div>`;
}
function imageHTML(card) {
  const images = (card.config && (card.config.images || (card.config.data_url ? [card.config.data_url] : []))) || [];
  if (!images.length) return '<div class="image-placeholder">选择图片后在此展示</div>';
  return `<div class="image-carousel" data-carousel="${card.id}" data-interval="${Number(card.config && card.config.carousel_interval) || 5}" data-enabled="${card.config && card.config.carousel ? 'true' : 'false'}">${images.map((src, index) => `<img class="dashboard-image ${index ? '' : 'active'}" src="${esc(src)}" alt="${esc(card.title || '仪表盘图片')}">`).join('')}</div>`;
}
function renderClock(card) {
  const target = document.querySelector(`[data-clock="${card.id}"]`);
  if (!target) return;
  const mode = card.config && card.config.time_mode || 'datetime';
  const draw = () => { const now = new Date(); target.textContent = mode === 'date' ? now.toLocaleDateString('zh-CN') : mode === 'time' ? now.toLocaleTimeString('zh-CN', { hour12:false }) : mode === 'weekday' ? now.toLocaleDateString('zh-CN', { weekday:'long', year:'numeric', month:'long', day:'numeric' }) : now.toLocaleString('zh-CN', { hour12:false }); };
  draw(); motionTimers.push(setInterval(draw, 1000));
}

function layoutEls() {
  const grid = $('#grid');
  if (!grid || !grid.getBoundingClientRect().width) return;
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
    applyCardContentScale(el, card, r);
    maxY = Math.max(maxY, (card.layout.y || 0) + (card.layout.h || 1));
  });
  grid.style.minHeight = Math.max(420, maxY * (ROW + Y_GAP) + 40) + 'px';
}
function applyCardContentScale(el, card, box) {
  if (card.type !== 'question') return;
  // A chart has a comfortable design size, but on a data screen a deliberately
  // small tile should still behave like one visual, not like a scrollable app.
  // Render it at that size and scale the whole result into the assigned tile.
  const compactHead = !editing && box.height < 104;
  const usableWidth = Math.max(1, box.width - (compactHead ? 0 : 24));
  const usableHeight = Math.max(1, box.height - (compactHead ? 0 : 42));
  const scale = Math.min(1, usableWidth / 280, usableHeight / 180);
  const scaled = scale < .995;
  el.classList.toggle('card-content-scaled', scaled);
  el.classList.toggle('card-content-mini', compactHead && scaled);
  el.style.setProperty('--card-content-scale', String(Math.max(.18, scale)));
}
function scheduleCanvasLayout() {
  if (layoutFrame) cancelAnimationFrame(layoutFrame);
  layoutFrame = requestAnimationFrame(() => {
    layoutFrame = 0;
    layoutEls();
    paintCached();
  });
}
function observeCanvasLayout() {
  const canvas = $('#board-canvas');
  if (!canvas || canvasResizeObserver || !window.ResizeObserver) return;
  canvasResizeObserver = new ResizeObserver(scheduleCanvasLayout);
  canvasResizeObserver.observe(canvas);
}

function bindCardChrome() {
  $$('[data-card-style]').forEach(button => button.onclick = event => { event.stopPropagation(); styleCard = styleCard === button.dataset.cardStyle ? '' : button.dataset.cardStyle; render(); });
  $$('[data-close-style]').forEach(button => button.onclick = event => { event.stopPropagation(); styleCard = ''; render(); });
  $$('[data-presentation]').forEach(input => input.onchange = () => {
    const card = board.cards.find(item => item.id === input.dataset.cardId);
    if (!card) return;
    const value = input.type === 'checkbox' ? (input.checked ? (input.dataset.presentation === 'shadow' ? 'default' : true) : (input.dataset.presentation === 'shadow' ? 'none' : false)) : input.value;
    const next = Object.assign({}, presentation(card), { [input.dataset.presentation]: value });
    if (input.dataset.presentation === 'border' && value === 'none') next.shadow = 'none';
    if (input.dataset.presentation === 'border' && value !== 'none' && next.shadow === 'none') next.shadow = 'default';
    card.config = Object.assign({}, card.config || {}, { presentation: next });
    render(); scheduleSave();
  });
  $$('[data-effect]').forEach(input => { const card = board.cards.find(item => item.id === input.dataset.cardId); if (!card) return; input.value = card.config && card.config.text_effect || 'none'; input.onchange = () => { card.config = Object.assign({}, card.config || {}, { text_effect: input.value }); render(); scheduleSave(); }; });
  $$('[data-carousel]').forEach(input => { const card = board.cards.find(item => item.id === input.dataset.cardId); if (!card) return; if (input.dataset.carousel === 'interval') input.value = String(Number(card.config && card.config.carousel_interval) || 5); input.onchange = () => { card.config = Object.assign({}, card.config || {}, input.dataset.carousel === 'enabled' ? { carousel: input.checked } : { carousel_interval: Number(input.value) }); render(); scheduleSave(); }; });
  $$('[data-time-mode]').forEach(input => { const card = board.cards.find(item => item.id === input.dataset.cardId); if (!card) return; input.value = card.config && card.config.time_mode || 'datetime'; input.onchange = () => { card.config = Object.assign({}, card.config || {}, { time_mode: input.value }); render(); scheduleSave(); }; });
  $$('[data-table-motion]').forEach(input => { const card = board.cards.find(item => item.id === input.dataset.cardId); if (!card) return; input.value = card.config && card.config.table_motion || 'none'; input.onchange = () => { card.config = Object.assign({}, card.config || {}, { table_motion: input.value }); render(); paintCached(); scheduleSave(); }; });
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
      else {
        card.body = el.innerText.trim();
        card.config = Object.assign({}, card.config || {}, { rich_text: richText({ config: { rich_text: el.innerHTML } }) });
      }
      scheduleSave();
    };
  });
  $$('[data-format]').forEach(button => button.onmousedown = event => {
    event.preventDefault();
    const editor = button.closest('.rich-text-wrap, .heading-wrap').querySelector('[contenteditable]');
    editor.focus(); document.execCommand(button.dataset.format, false, null);
  });
  $$('[data-typography]').forEach(input => input.onchange = () => {
    const card = board.cards.find(item => item.id === input.dataset.cardId);
    if (!card) return;
    const style = typography(card);
    style[input.dataset.typography] = input.dataset.typography === 'size' ? Number(input.value) : input.value;
    card.config = Object.assign({}, card.config || {}, { typography: style });
    render(); scheduleSave();
  });
  $$('[data-card-field]').forEach(input => {
    input.onchange = () => {
      const card = board.cards.find(item => item.id === input.closest('[data-card]').dataset.card);
      if (!card) return;
      if (input.dataset.cardField === 'iframe_url' || input.dataset.cardField === 'plugin_url') {
        card.config = Object.assign({}, card.config || {}, { url: input.value.trim() });
        render();
        validateEmbed(input.value.trim(), card.id);
      }
      else card[input.dataset.cardField] = input.value.trim();
      scheduleSave();
    };
  });
  $$('[data-drag]').forEach(el => el.onpointerdown = ev => startDrag(ev, el.dataset.drag, 'move'));
  $$('[data-resize]').forEach(el => el.onpointerdown = ev => startDrag(ev, el.dataset.resize, 'resize'));
  $$('[data-card]').forEach(el => el.onclick = ev => { if (!editing || ev.target.closest('a,button,select,input,[contenteditable]')) return; selectedCard = el.dataset.card; render(); });
}
async function validateEmbed(url, cardID) {
  const status = document.querySelector(`[data-embed-validation="${cardID}"]`);
  if (!status || !url) return;
  status.textContent = '正在检查网站的嵌入策略…'; status.className = 'embed-validation checking';
  try {
    const result = await api('/api/embed/validate', 'POST', { url });
    status.textContent = result.embeddable ? '✓ ' + result.reason : '无法嵌入：' + result.reason;
    status.className = 'embed-validation ' + (result.embeddable ? 'valid' : 'invalid');
  } catch (error) { status.textContent = '无法校验：' + error.message; status.className = 'embed-validation invalid'; }
}

function pluginOrigin(url) {
  try { return new URL(url, location.href).origin; } catch (_) { return ''; }
}
function sendPluginContext(frame, card) {
  const origin = pluginOrigin(card.config && card.config.url);
  if (!origin || !frame.contentWindow) return;
  frame.contentWindow.postMessage({
    type: 'topbase.dashboard.context',
    version: 1,
    payload: {
      state: editing ? 'config' : (document.fullscreenElement ? 'fullscreen' : 'view'),
      theme: appearance().theme,
      locale: document.documentElement.lang || 'zh-CN',
      card: { id: card.id, title: card.title || '', layout: card.layout || {} },
      config: (card.config && card.config.plugin_config) || {}
    }
  }, origin);
}
function bindPluginFrames() {
  $$('[data-plugin-card]').forEach(frame => {
    const card = board.cards.find(item => item.id === frame.dataset.pluginCard);
    if (!card) return;
    frame.onload = () => sendPluginContext(frame, card);
    sendPluginContext(frame, card);
  });
}
window.addEventListener('message', event => {
  const message = event.data;
  if (!message || typeof message !== 'object' || typeof message.type !== 'string' || !board) return;
  const frame = $$('[data-plugin-card]').find(item => item.contentWindow === event.source);
  if (!frame) return;
  const card = board.cards.find(item => item.id === frame.dataset.pluginCard);
  if (!card || event.origin !== pluginOrigin(card.config && card.config.url)) return;
  if (message.type === 'topbase.dashboard.ready') return sendPluginContext(frame, card);
  if (message.type === 'topbase.dashboard.set-config' && editing && message.config && typeof message.config === 'object' && !Array.isArray(message.config)) {
    card.config = Object.assign({}, card.config || {}, { plugin_config: message.config });
    scheduleSave();
  }
});
document.addEventListener('fullscreenchange', () => bindPluginFrames());

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
  const cw = cellW() + X_GAP;
  const dx = Math.round((ev.clientX - drag.startX) / cw);
  const dy = Math.round((ev.clientY - drag.startY) / (ROW + Y_GAP));
  if (drag.mode === 'move') {
    drag.card.layout.x = clamp(drag.layout.x + dx, 0, COLS - drag.card.layout.w);
    drag.card.layout.y = Math.max(0, drag.layout.y + dy);
  } else {
    drag.card.layout.w = clamp(drag.layout.w + dx, MIN_W, COLS - drag.card.layout.x);
    drag.card.layout.h = clamp(drag.layout.h + dy, minHeight(drag.card), 48);
  }
  layoutEls();
  requestAnimationFrame(scheduleCanvasLayout);
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
  motionTimers.forEach(clearInterval); motionTimers = [];
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
  renderPaletteTabs();
  paintCached();
  bindPluginFrames();
  tabCards().filter(card => card.type === 'time').forEach(renderClock);
  document.querySelectorAll('[data-carousel][data-enabled="true"]').forEach(host => {
    const slides = [...host.querySelectorAll('.dashboard-image')];
    if (slides.length < 2) return;
    let current = 0;
    motionTimers.push(setInterval(() => { slides[current].classList.remove('active'); current = (current + 1) % slides.length; slides[current].classList.add('active'); }, Number(host.dataset.interval || 5) * 1000));
  });
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
    const cell = cellAt(ev.clientX, ev.clientY, 12, 10);
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
    const cell = cellAt(ev.clientX, ev.clientY, 12, 10);
    if (value.startsWith('component:')) addStatic(value.slice('component:'.length), cell);
    else addQuestion(value, cell);
  };
}

async function addQuestion(questionId, layout) {
  if (!questionId || usedQuestionIDs().has(questionId)) return toast('这条分析已经在看板上');
  const q = questionMap[questionId];
  const place = layout || { x: 0, y: nextY(12), w: 12, h: 10 };
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
    heading: { type:'heading', title:'标题', layout:{x:0,y:nextY(COLS),w:COLS,h:2} },
    text: { type:'text', body:'说明文字', layout:{x:0,y:nextY(COLS),w:COLS,h:2} },
    link: { type:'link', title:'链接', body:'https://', layout:{x:0,y:nextY(12),w:12,h:4} },
    iframe: { type:'iframe', title:'嵌入网页', config:{url:'https://'}, layout:{x:0,y:nextY(12),w:12,h:10} },
    plugin: { type:'plugin', title:'自定义组件', config:{url:'', plugin_config:{}}, layout:{x:0,y:nextY(12),w:12,h:10} },
    time: { type:'time', title:'时间', config:{time_mode:'datetime'}, layout:{x:0,y:nextY(6),w:6,h:4} },
    divider: { type:'divider', title:'', layout:{x:0,y:nextY(COLS),w:COLS,h:2} }
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
async function addImage(files) {
  const valid = [...files].filter(file => file.type.startsWith('image/') && file.size <= 3 * 1024 * 1024);
  if (!valid.length) return toast('请选择 3MB 以内的图片文件');
  const images = await Promise.all(valid.map(file => new Promise((resolve, reject) => { const reader = new FileReader(); reader.onload = () => resolve(reader.result); reader.onerror = reject; reader.readAsDataURL(file); })));
  const card = { id:newID('crd'), tab_id:activeTab, type:'image', title:valid[0].name.replace(/\.[^.]+$/, ''), config:{images, carousel:images.length > 1, carousel_interval:5}, layout:{x:0,y:nextY(12),w:12,h:8} };
  board.cards = (board.cards || []).concat([card]); render();
  try { await saveBoard(); toast(images.length > 1 ? '图片轮播已添加' : '图片已添加'); } catch (error) { toast(error.message); }
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
  TopbaseViz.render(host, { columns: d.columns || [], rows: d.rows || [], spec, queryir: q && q.queryir, compact: true, dashboardOnly: true });
  if (card.config && card.config.table_motion && card.config.table_motion !== 'none') {
    const scroller = host.querySelector('.tb-grid-scroll');
    if (scroller) {
      const step = card.config.table_motion === 'page' ? Math.max(scroller.clientHeight - 12, 1) : 1;
      motionTimers.push(setInterval(() => { scroller.scrollTop = scroller.scrollTop + step >= scroller.scrollHeight - scroller.clientHeight ? 0 : scroller.scrollTop + step; }, card.config.table_motion === 'page' ? 3500 : 45));
    }
  }
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
    [board, questions, collections] = await Promise.all([api('/api/dashboards/' + boardId), api('/api/questions'), api('/api/collections')]);
  }
  upgradeCanvasGrid();
  questionMap = Object.fromEntries(questions.map(q => [q.id, q]));
  if (!isPublicDashboard) viewer = await api('/api/user/current').catch(() => null);
  if ($('#title')) $('#title').textContent = board.name;
  renderPublicBadge();
  activeTab = board.tabs?.[0]?.id || '';
  editing = !isPublicDashboard && !(board.cards || []).length;
  renderCollectionFilter();
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
if ($('#apply')) $('#apply').onclick = loadCards;
if ($('#qsearch')) $('#qsearch').oninput = renderPalette;
if ($('#qcollection')) $('#qcollection').onchange = () => { paletteLimit = 60; renderPalette(); };
if ($('#palette-more')) $('#palette-more').onclick = () => { paletteLimit += 60; renderPalette(); };
$('#palette-tabs')?.addEventListener('click', event => {
  const button = event.target.closest('[data-palette-tab]');
  if (!button) return;
  paletteTab = button.dataset.paletteTab;
  renderPaletteTabs();
});
if ($('#add-image')) $('#add-image').onclick = () => $('#image-upload').click();
if ($('#image-upload')) $('#image-upload').onchange = event => { addImage(event.target.files); event.target.value = ''; };
$$('[data-add]').forEach(button=>{
  button.onclick=()=>addStatic(button.dataset.add);
  button.ondragstart=event=>{
    event.dataTransfer.setData('text/plain','component:'+button.dataset.add);
    event.dataTransfer.effectAllowed='copy';
  };
});
observeCanvasLayout();
window.addEventListener('resize', scheduleCanvasLayout);

function closeActionModal(){ $('#modal').hidden=true;$('#modal').innerHTML=''; }
function shareURL(kind) {
  if (!board?.public_uuid) return '';
  return location.origin + '/' + kind + '/dashboard/' + encodeURIComponent(board.public_uuid) + '/';
}
function copyValue(value) {
  if (!value) return;
  navigator.clipboard.writeText(value).then(() => toast('已复制')).catch(() => toast('复制失败，请手动复制'));
}
function renderShareDialog() {
  const published = !!board?.public_uuid;
  const publicURL = shareURL('public');
  const embedURL = shareURL('embed');
  const iframe = `<iframe src="${embedURL}" width="100%" height="800" frameborder="0" allowtransparency="true" title="${board?.name || 'Topbase 仪表盘'}"></iframe>`;
  const canEmbed = !!viewer?.is_admin;
  const publicDetails = published
    ? `<label>公开链接<div class="copy-row"><input readonly value="${esc(publicURL)}"><button data-copy="${esc(publicURL)}" class="secondary" type="button">复制</button></div></label>`
    : `<p class="share-help">开启后会创建一个无需登录即可访问的公开链接。</p>`;
  const embedDetails = !canEmbed
    ? `<div class="share-restricted"><b>嵌入仅管理员可配置</b><p>管理员启用后，才会生成可供第三方网页动态加载的 iframe 地址。</p></div>`
    : `<section class="embed-share-settings ${published ? '' : 'disabled'}"><label class="share-toggle"><input id="embed-enabled" type="checkbox" ${board.public_embed_enabled ? 'checked' : ''} ${published ? '' : 'disabled'}><span><b>允许 iframe 嵌入</b><small>仅管理员可以启用。关闭分享后会自动关闭嵌入。</small></span></label>${board.public_embed_enabled ? `<label>动态加载地址<div class="copy-row"><input readonly value="${esc(embedURL)}"><button data-copy="${esc(embedURL)}" class="secondary" type="button">复制</button></div></label><label>iframe 代码<div class="copy-row"><input readonly value="${esc(iframe)}"><button data-copy="${esc(iframe)}" class="secondary" type="button">复制</button></div></label>` : ''}</section>`;
  $('#modal').hidden = false;
  $('#modal').innerHTML = `<section class="action-dialog"><header><div><small>访问控制</small><h2>分享与嵌入</h2><p>分享默认关闭。公开链接与 iframe 都会始终加载仪表盘的最新数据。</p></div><button id="close-action" class="secondary" type="button">关闭</button></header><label class="share-toggle"><input id="public-sharing-toggle" type="checkbox" ${published ? 'checked' : ''}><span><b>公开分享</b><small>生成无需登录即可访问的仪表盘链接。</small></span></label>${publicDetails}<div class="share-section"><h3>嵌入</h3>${embedDetails}</div></section>`;
  $('#close-action').onclick = closeActionModal;
  $$('[data-copy]').forEach(button => button.onclick = () => copyValue(button.dataset.copy));
  $('#public-sharing-toggle').onchange = async event => {
    try {
      const result = event.target.checked ? await api(`/api/dashboards/${boardId}/public-link`, 'POST', {}) : await api(`/api/dashboards/${boardId}/public-link`, 'DELETE');
      board = result.dashboard || result;
      renderPublicBadge();
      renderShareDialog();
      toast(event.target.checked ? '公开分享已开启' : '公开分享已关闭');
    } catch (error) { renderShareDialog(); toast(error.message); }
  };
  $('#embed-enabled')?.addEventListener('change', async event => {
    try {
      board = await api(`/api/dashboards/${boardId}/embedding`, 'PUT', { enabled:event.target.checked });
      renderShareDialog();
      toast(event.target.checked ? 'iframe 嵌入已启用' : 'iframe 嵌入已关闭');
    } catch (error) { renderShareDialog(); toast(error.message); }
  });
}
function openShare(){ renderShareDialog(); }
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
  const hooks = await api('/api/subscription-channels');
  const channels = hooks.filter(h=>h.enabled).map(h=>({value:'webhook:'+h.id,label:h.name,description:(h.provider||'Webhook')+' · 发送到已配置群'}));
  if(!channels.length){toast('请联系管理员先在“通知与订阅”中创建并启用 Webhook 通道');return}
  const channel = await choiceDialog({kicker:'仪表盘订阅',title:'选择每天的推送通道',description:'订阅将在每天 09:00 执行。管理员可在“通知与订阅”统一管理。',label:'Webhook 通道',value:channels[0].value,confirmText:'创建订阅',options:channels});
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
