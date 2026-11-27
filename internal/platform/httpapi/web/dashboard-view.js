// A denser grid gives authors enough precision for data-screen compositions,
// while grid-version migration keeps existing 12-column dashboards unchanged.
const COLS = 24, ROW = 34, X_GAP = 6, Y_GAP = 10, MIN_W = 2, MIN_H = 2, GRID_VERSION = 2;
const pathParts = location.pathname.split('/').filter(Boolean);
const isPublicDashboard = ['public', 'embed'].includes(pathParts[0]) && pathParts[1] === 'dashboard';
const boardId = isPublicDashboard ? '' : (pathParts[1] || new URLSearchParams(location.search).get('id'));
const publicUUID = isPublicDashboard ? pathParts[2] : '';
let board = null, questions = [], collections = [], questionMap = {}, activeTab = '', editing = false, saveTimer = 0, drag = null, cardData = {}, selectedCard = '', inspectorSection = 'appearance', paletteLimit = 60, paletteTab = 'analysis', motionTimers = [], layoutFrame = 0, canvasResizeObserver = null, loadController = null, loadSequence = 0;
let saveQueue = Promise.resolve();
let previewOrigin = null;
let finishingEdit = false;
const CANVAS_BACKGROUNDS = { light: '#ffffff', deep: '#07111f', aurora: '#08284a' };
const APPEARANCE_DEFAULTS = { theme: 'light', background: CANVAS_BACKGROUNDS.light, grid: true, snap: true };

function newID(prefix) {
  return prefix + '_' + Math.random().toString(16).slice(2) + Date.now().toString(16).slice(-4);
}
function tabCards() {
  return (board.cards || []).filter(c => !activeTab || !c.tab_id || c.tab_id === activeTab);
}
function cellW() {
  const grid = $('#grid');
  const width = grid.clientWidth || grid.getBoundingClientRect().width;
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
function minWidth(card) { return card.type === 'question' && cardVizType(card) === 'scalar' ? 4 : MIN_W; }
function nextY(w) {
  w = w || 4;
  const cards = tabCards();
  if (!cards.length) return 0;
  return Math.max(...cards.map(c => (c.layout.y || 0) + (c.layout.h || 1)));
}
function resolveOverlap(moved) {
  TopbaseDashboardLayout.resolveOverlap(moved, tabCards());
}
function cellAt(clientX, clientY, w, h) {
  const box = $('#grid').getBoundingClientRect();
  const cw = cellW();
  const precision = appearance().snap ? 1 : 4;
  let x = Math.floor((clientX - box.left) / (cw + X_GAP) * precision) / precision;
  let y = Math.floor((clientY - box.top) / (ROW + Y_GAP) * precision) / precision;
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
  const saved = board.appearance || {};
  const a = { ...APPEARANCE_DEFAULTS, ...saved };
  if (!Object.hasOwn(CANVAS_BACKGROUNDS, a.theme)) a.theme = APPEARANCE_DEFAULTS.theme;
  const preset = CANVAS_BACKGROUNDS[a.theme];
  const color = String(saved.background || '').toLowerCase();
  const legacyDefault = !saved.background_custom && color === CANVAS_BACKGROUNDS.deep && a.theme !== 'deep';
  a.background = color && !legacyDefault ? color : preset;
  // Older light dashboards used a tinted paper. Preserve their stored color
  // while still allowing the next preset selection to change it.
  if (a.background_custom == null) {
    const oldLight = a.theme === 'light' && color === '#f5fbfb';
    a.background_custom = a.background !== preset && !oldLight;
  }
  delete a.glow;
  return board.appearance = a;
}
function useCanvasTheme(theme) {
  if (!Object.hasOwn(CANVAS_BACKGROUNDS, theme)) return;
  const a = appearance();
  a.theme = theme;
  if (!a.background_custom) a.background = CANVAS_BACKGROUNDS[theme];
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
  canvas.classList.toggle('canvas-snap-off', !a.snap);
}
function renderPublicBadge() {
  const badge = $('#public-badge');
  if (!badge) return;
  const state = dashboardShareStatus(board);
  badge.hidden = state === 'private';
  badge.classList.toggle('expired', state === 'expired');
  badge.textContent = state === 'expired' ? '分享已到期' : '◉ 已公开';
  badge.title = state === 'expired' ? '在分享设置中可以延长有效期' : '此仪表盘已公开分享';
}
function canvasPropertyFields() {
  const a = appearance();
  return `<div class="inspector-fields canvas-property-fields">
    <section><h3>外观</h3>
      ${inspectorSelect('配色', { id: 'canvas-theme' }, a.theme, [['light','浅色'],['deep','深色'],['aurora','藏蓝']])}
      <label class="background-field">背景颜色<span class="color-input">
        <input id="canvas-background" type="color" value="${esc(a.background)}" aria-label="背景颜色">
        <output for="canvas-background">${esc(a.background.toUpperCase())}</output>
      </span></label>
      <button data-reset-background type="button" ${a.background_custom ? '' : 'hidden'}>恢复配色默认背景</button>
    </section>
    <section><h3>编辑辅助</h3>
      ${TopbaseUI.checkbox({ label: '显示网格', attrs: { 'id': 'canvas-grid-toggle' }, checked: a.grid })}
      ${TopbaseUI.checkbox({ label: '吸附到网格', attrs: { 'id': 'canvas-snap-toggle' }, checked: a.snap })}
    </section>
  </div>`;
}

function updateCanvasBackground(value) {
  const a = appearance();
  a.background = value;
  a.background_custom = value !== CANVAS_BACKGROUNDS[a.theme];
  applyAppearance();
  const host = $('#card-inspector');
  $('#canvas-background', host).value = value;
  $('output', host).textContent = value.toUpperCase();
  $('[data-reset-background]', host).hidden = !a.background_custom;
  scheduleSave();
}
async function saveBoard() {
  clearTimeout(saveTimer);
  const snapshot = structuredClone(payload());
  const path = '/api/dashboards/' + board.id;
  const write = saveQueue.then(() => api(path, 'PUT', snapshot));
  saveQueue = write.catch(() => {});
  await write;
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
  $('#tabs').className = 'tb-tabs tabs';
  $('#tabs').innerHTML = tabs.length <= 1 ? '' : tabs.map(t => `<button translate="no" data-tab="${t.id}" class="${t.id === activeTab ? 'active' : ''}">${esc(t.name)}</button>`).join('');
  $('.board-toolbar').hidden = tabs.length <= 1 && !(board.filters || []).length;
  $$('#tabs button').forEach(b => b.onclick = () => {
    activeTab = b.dataset.tab;
    $$('#tabs button').forEach(button => button.classList.toggle('active', button === b));
    render();
    loadCards();
  });
}
function renderFilters() {
  $('#filters').innerHTML = (board.filters || []).map(f => {
    if (f.type === 'date') return `<label><span translate="no">${esc(f.name)}</span> 起<input data-filter="${f.id}" data-key="start" type="date"></label><label>止<input data-filter="${f.id}" data-key="end" type="date"></label>`;
    return `<label><span translate="no">${esc(f.name)}</span><input data-filter="${f.id}" data-key="value"></label>`;
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
    return `<button class="palette-item ${added?'added':''}" draggable="${added?'false':'true'}" data-qid="${item.id}" type="button" ${added?'disabled':''}><b translate="no">${esc(item.name)}</b><small>${group ? `<span translate="no">${esc(group.name)}</span>` + ' · ' : ''}${dashboardText(item.query_type === 'native' ? 'SQL 分析' : '可视化分析')} · ${dashboardText(added?'已添加':'点击添加')}</small></button>`;
  }).join('') || (questions.length
    ? '<div class="palette-empty" role="status">没有匹配的分析，请调整搜索词或分组。</div>'
    : '<div class="palette-empty"><b>还没有分析</b><p>保存分析后，即可在这里添加。</p><a href="/questions/new/">创建分析</a></div>');
  $('#palette-more').hidden = items.length <= shown.length;
  $('#palette-more').textContent = dashboardText('加载更多（剩余 {n} 条）', {n:items.length - shown.length});
  $$('#palette-questions .palette-item:not([disabled])').forEach(btn => {
    btn.ondragstart = ev => beginPaletteDrag(ev, { questionId: btn.dataset.qid });
    btn.ondragend = clearPaletteDrag;
    btn.onclick = () => addQuestion(btn.dataset.qid);
  });
}
function renderWidgetLibrary() {
  const host = $('#widget-library');
  if (!host) return;
  const groups = new Map();
  for (const item of TopbaseUI.components.dashboardWidgets.list($('#widget-search').value)) {
    if (!groups.has(item.group)) groups.set(item.group, []);
    groups.get(item.group).push(item);
  }
  host.innerHTML = [...groups].map(([group, items]) => `<section class="widget-group"><h3>${esc(group)}</h3><div class="widget-tiles">${items.map(item =>
    `<button type="button" class="widget-tile" data-add="${esc(item.id)}" draggable="${item.id !== 'image'}"><span class="widget-symbol" aria-hidden="true">${esc(item.symbol)}</span><span translate="no">${esc(item.label)}</span></button>`
  ).join('')}</div></section>`).join('') || '<p class="widget-no-match" role="status">没有匹配的小组件</p>';
}
let insertPopover;
function closeInsertPalette(restoreFocus = false) {
  insertPopover?.close(restoreFocus);
}

function syncEditorTools(isEditing) {
  const tools = $('#editor-tools');
  if (!tools) return;
  if (!isEditing) {
    closeInsertPalette();
    clearPaletteDrag();
  }
  tools.hidden = !isEditing;
}
function renderCollectionFilter() {
  const select = $('#qcollection');
  if (!select) return;
  const selected = select.value;
  const counts = new Map();
  questions.forEach(question => counts.set(question.collection_id || '', (counts.get(question.collection_id || '') || 0) + 1));
  select.innerHTML = `<option value="">${dashboardText('全部分组（{n}）', {n:questions.length})}</option>` + collections.filter(collection => counts.has(collection.id)).map(collection => `<option translate="no" value="${esc(collection.id)}">${esc(collection.name)}（${counts.get(collection.id)}）</option>`).join('') + (counts.has('') ? `<option value="unassigned">${dashboardText('未分组（{n}）', {n:counts.get('')})}</option>` : '');
  select.value = selected;
}

function dashboardText(key, values = {}) {
  return globalThis.TopbaseLocale?.t(key, values) ?? key.replace(/\{(\w+)\}/g, (token, name) => values[name] ?? token);
}

function cardTitle(card) {
  if (card.title) return card.title;
  if (card.question_id && questionMap[card.question_id]) return questionMap[card.question_id].name;
  if (card.type === 'heading') return dashboardText('标题');
  if (card.type === 'text') return dashboardText('文本');
  if (card.type === 'link') return card.title || dashboardText('链接');
  if (card.type === 'iframe') return card.title || dashboardText('网页');
  if (card.type === 'divider') return card.title || dashboardText('分隔标题');
  if (card.type === 'metric') return card.title || dashboardText('核心指标');
  return dashboardText('分析卡');
}
function richText(card) {
  const heading = card.type === 'heading';
  const raw = heading ? card.config?.rich_title : card.config?.rich_text;
  if (!raw) return esc(heading ? (card.title || dashboardText('标题')) : (card.body || ''));
  const allowed = heading ? ['B','STRONG','I','EM','U'] : ['B','STRONG','I','EM','U','UL','OL','LI','BR','A','P','DIV'];
  const doc = new DOMParser().parseFromString(raw, 'text/html');
  doc.body.querySelectorAll('*').forEach(node => {
    if (!allowed.includes(node.tagName)) node.replaceWith(...node.childNodes);
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
  return `font-family:${font};font-size:${style.size}px${card.type === 'heading' ? ';font-weight:500' : ''}`;
}
function typographyTools(card) {
  return `<div class="tb-segmented inspector-format-tools" role="group" aria-label="文字强调">
    <button data-format="bold" type="button" aria-label="加粗" title="加粗"><b>B</b></button>
    <button data-format="italic" type="button" aria-label="倾斜" title="倾斜"><i>I</i></button>
    <button data-format="underline" type="button" aria-label="下划线" title="下划线"><u>U</u></button>
    ${card.type === 'text' ? '<button data-format="insertUnorderedList" type="button" title="项目列表">列表</button>' : ''}
  </div>`;
}
function presentation(card) {
  const align = cardVizType(card) === 'scalar' ? 'center' : 'left';
  const saved = card.config?.presentation || {};
  const border = saved.border === 'none' ? 'none' : 'default';
  const surface = saved.surface === 'transparent' ? 'transparent' : 'solid';
  return {
    border, surface,
    padding: ['compact','none'].includes(saved.padding) ? saved.padding : 'normal',
    radius: border === 'none' || saved.radius === 'square' ? 'square' : 'round',
    shadow: border === 'none' || surface === 'transparent' || saved.shadow === 'none' ? 'none' : 'default',
    align: ['left','center','right'].includes(saved.align) ? saved.align : align,
    show_header: saved.show_header !== false
  };
}
function cardVizType(card) {
  if (card.type !== 'question') return '';
  const own = card.config && card.config.chartspec && card.config.chartspec.type;
  const source = questionMap[card.question_id] && questionMap[card.question_id].chartspec;
  // 表格是未指定图形时的默认呈现；因此没有明确图形类型的分析卡同样按表格处理。
  return own || (source && source.type) || 'table';
}
function inspectorSelect(label, attrs, value, choices) {
  return TopbaseUI.select({ label, attrs, value, options: choices.map(([value, label]) => ({ value, label })) });
}

function presentationPanel(card) {
  const p = presentation(card);
  const config = card.config || {};
  const field = (label, key, options) => inspectorSelect(label, { 'data-presentation': key }, p[key], options);
  const section = (title, body) => `<section class="card-style-section"><b>${title}</b>${body}</section>`;
  const grid = fields => `<div class="card-style-grid">${fields.join('')}</div>`;
  const groups = [];
  const viz = cardVizType(card);
  const textCard = card.type === 'text' || card.type === 'heading';

  const content = [];
  if (textCard) {
    const style = typography(card);
    content.push(inspectorSelect('字体', { 'data-typography': 'font' }, style.font, [['system','无衬线'],['serif','衬线'],['mono','等宽']]));
    content.push(inspectorSelect('字号', { 'data-typography': 'size' }, style.size, [12,14,16,18,22,28].map(size => [size, size + ' px'])));
  }
  if (textCard || ['time','link'].includes(card.type) || viz === 'scalar') {
    content.push(field('对齐方式', 'align', [['left','靠左'],['center','居中'],['right','靠右']]));
  }
  if (card.type === 'text') {
    content.push(inspectorSelect('文字效果', { 'data-effect': 'text' }, config.text_effect || 'none',
      [['none','静止'],['marquee-left','向左滚动'],['marquee-up','向上滚动']]));
  }
  if (card.type === 'time') {
    content.push(inspectorSelect('显示内容', { 'data-time-mode': '' }, config.time_mode || 'datetime',
      [['datetime','日期和时间'],['date','仅日期'],['time','仅时间'],['weekday','星期和日期']]));
  }
  if (viz === 'table') {
    content.push(inspectorSelect('数据播放', { 'data-table-motion': '' }, config.table_motion || 'none',
      [['none','静止'],['scroll','连续滚动'],['page','自动翻页']]));
  }
  if (content.length) groups.push(section(textCard ? '文字排版' : '内容显示', grid(content) + (textCard ? typographyTools(card) : '')));

  if (card.type === 'image' && config.images?.length > 1) {
    const interval = inspectorSelect('切换间隔', { 'data-carousel': 'interval' }, Number(config.carousel_interval) || 5,
      [3,5,8,12].map(value => [value, dashboardText('{value} 秒', {value})]));
    groups.push(section('图片播放', TopbaseUI.checkbox({ label: '自动轮播', attrs: { 'data-carousel': 'enabled' }, checked: config.carousel }) +
      (config.carousel ? grid([interval]) : '')));
  }

  const appearanceFields = [];
  if (card.type !== 'divider') {
    appearanceFields.push(field('背景', 'surface', [['solid','实色'],['transparent','透明']]));
    appearanceFields.push(field('边框', 'border', [['default','显示'],['none','隐藏']]));
    if (p.border !== 'none') appearanceFields.push(field('边角', 'radius', [['round','圆角'],['square','直角']]));
  }
  appearanceFields.push(field('内边距', 'padding', [['normal','标准'],['compact','紧凑'],['none','无']]));
  let appearance = grid(appearanceFields);
  if (card.type !== 'divider' && p.border !== 'none' && p.surface !== 'transparent') {
    appearance += TopbaseUI.checkbox({ label: '显示阴影', attrs: { 'data-presentation': 'shadow' }, checked: p.shadow !== 'none' });
  }
  if (card.type === 'question' || card.type === 'divider') {
    appearance += TopbaseUI.checkbox({ label: '显示标题', attrs: { 'data-presentation': 'show_header' }, checked: p.show_header });
  }
  groups.push(section('卡片样式', appearance));
  return groups.join('');
}

function inspectorType(card) {
  if (card.type === 'question') return TopbaseViz.types.find(type => type.id === cardVizType(card))?.label || '分析';
  if (card.type === 'plugin' && card.title) return card.title;
  return TopbaseUI.components.dashboardWidgets.list().find(widget => widget.id === card.type)?.label || '小组件';
}

function inspectorContent(card) {
  const field = (label, key, value) => TopbaseUI.input({ label, value, attrs: { 'data-config-field': key } });
  if (card.type === 'question') {
    return field('卡片标题', 'title', cardTitle(card)) +
      inspectorSelect('显示形式', { 'data-inspector-viz': '' }, cardVizType(card), TopbaseViz.types.map(({ id, label }) => [id, label]));
  }
  if (card.type === 'divider') return field('分隔线标题', 'title', card.title || '');
  if (!['link', 'iframe', 'plugin'].includes(card.type)) return '';
  const url = card.type === 'link' ? card.body : card.config?.url;
  const validation = card.type === 'link' ? '' : `<p class="embed-validation" data-embed-validation="${esc(card.id)}" role="status"></p>`;
  return field('名称', 'title', card.title || '') + field(card.type === 'plugin' ? '组件地址' : '网址', 'url', url || '') + validation;
}

function layoutLimits(card) {
  return { columns: COLS, minW: minWidth(card), minH: minHeight(card), maxH: 48, snap: appearance().snap };
}

function renderInspector() {
  const host = $('#card-inspector');
  if (!host) return;
  host.hidden = !editing;
  if (!editing) { host.replaceChildren(); return; }
  const card = tabCards().find(item => item.id === selectedCard);
  if (!card) selectedCard = '';
  const heading = card ? componentName(card) : '画布设置';
  const icon = card ? 'settings-2' : 'layout-dashboard';
  const header = `<header><div class="inspector-heading"><svg aria-hidden="true"><use href="/vendor/lucide-nav.svg#${icon}"></use></svg>
    <div>${card ? `<button data-close-inspector type="button" title="返回画布设置">画布</button><span class="inspector-type"> / ${esc(inspectorType(card))}</span>` : ''}
    <h2 tabindex="-1" ${card ? 'translate="no"' : ''} title="${esc(heading)}">${esc(heading)}</h2></div></div>
    <button type="button" data-inspector-dismiss aria-label="收起属性">×</button></header>`;
  syncInspectorToggle();
  if (!card) { host.innerHTML = header + canvasPropertyFields(); return; }
  const content = inspectorContent(card);
  const layout = inspectorSection === 'layout';
  host.innerHTML = `${header}
    <div class="inspector-tabs" role="group" aria-label="配置分类">
      <button type="button" data-section="appearance" aria-pressed="${!layout}">外观</button>
      <button type="button" data-section="layout" aria-pressed="${layout}">位置与尺寸</button>
    </div><div class="inspector-fields">${layout ? TopbaseDashboardLayout.section(card.layout, layoutLimits(card)) :
      (content ? `<section class="inspector-content">${content}</section>` : '') + presentationPanel(card)}</div>`;
  if (layout) TopbaseDashboardLayout.bind(host, () => tabCards().find(item => item.id === selectedCard), layoutLimits, (item, next) => {
    item.layout = next;
    resolveOverlap(item);
    scheduleCanvasLayout();
    scheduleSave();
  });
}

function compactInspector() {
  return getComputedStyle($('#layout')).getPropertyValue('--compact-inspector').trim() === '1';
}

function syncInspectorToggle() {
  const toggle = $('#canvas-settings');
  if (!toggle || !editing) return;
  if (compactInspector()) toggle.setAttribute('aria-expanded', String($('#layout').classList.contains('inspector-open')));
  else toggle.removeAttribute('aria-expanded');
}

function openCardInspector() {
  if (!editing) return;
  if (compactInspector()) $('#layout').classList.add('inspector-open');
  syncInspectorToggle();
  $('#card-inspector h2')?.focus({ preventScroll: true });
}

function dismissInspector() {
  $('#layout').classList.remove('inspector-open');
  syncInspectorToggle();
  $('#canvas-settings').focus({ preventScroll: true });
}

function selectCard(id) {
  if (!editing || id === selectedCard) return;
  selectedCard = id;
  $$('#grid [data-card]').forEach(el => el.classList.toggle('selected', el.dataset.card === id));
  renderInspector();
  renderOutline();
}

function syncComponentName(card) {
  if (selectedCard !== card.id) return;
  const title = $('#card-inspector h2');
  if (title) title.textContent = title.title = componentName(card);
  renderOutline();
}

function bindInspector() {
  const host = $('#card-inspector');
  if (!host) return;
  $('#canvas-settings').onclick = () => {
    if (compactInspector() && $('#layout').classList.contains('inspector-open')) return dismissInspector();
    if (!compactInspector()) selectCard('');
    openCardInspector();
  };
  host.onkeydown = event => {
    if (event.key === 'Escape' && compactInspector()) {
      event.preventDefault();
      dismissInspector();
    }
  };
  host.oninput = event => {
    if (event.target.id === 'canvas-background') updateCanvasBackground(event.target.value);
  };
  host.onclick = event => {
    if (event.target.closest('[data-inspector-dismiss]')) return dismissInspector();
    if (event.target.closest('[data-close-inspector]')) {
      selectCard('');
      return openCardInspector();
    }
    if (event.target.closest('[data-reset-background]')) {
      updateCanvasBackground(CANVAS_BACKGROUNDS[appearance().theme]);
      $('#canvas-background').focus();
      return;
    }
    const tab = event.target.closest('[data-section]');
    if (tab) {
      inspectorSection = tab.dataset.section;
      renderInspector();
      $(`[data-section="${inspectorSection}"]`, host)?.focus();
    }
  };
  host.onchange = event => {
    const input = event.target;
    if (!editing) return;
    if (input.id === 'canvas-theme') {
      useCanvasTheme(input.value);
      render();
      $('#canvas-theme').focus({ preventScroll: true });
      scheduleSave();
      return;
    }
    if (input.id === 'canvas-grid-toggle' || input.id === 'canvas-snap-toggle') {
      appearance()[input.id === 'canvas-grid-toggle' ? 'grid' : 'snap'] = input.checked;
      applyAppearance();
      scheduleSave();
      return;
    }
    const card = editing && tabCards().find(item => item.id === selectedCard);
    if (!card || inspectorSection === 'layout') return;
    const config = card.config ||= {};
    const key = input.dataset.presentation;
    let checkURL;
    if (key) {
      const next = presentation(card);
      next[key] = input.type === 'checkbox' ? input.checked : input.value;
      if (key === 'shadow') next.shadow = input.checked ? 'default' : 'none';
      config.presentation = next;
    } else if (input.dataset.typography) {
      const style = typography(card);
      style[input.dataset.typography] = input.dataset.typography === 'size' ? Number(input.value) : input.value;
      config.typography = style;
    } else if (input.hasAttribute('data-inspector-viz')) {
      config.chartspec = { ...config.chartspec, type: input.value };
    } else if (input.dataset.configField === 'title') {
      card.title = input.value.trim();
    } else if (input.dataset.configField === 'url') {
      if (card.type === 'link') card.body = input.value.trim();
      else checkURL = config.url = input.value.trim();
    } else if (input.hasAttribute('data-time-mode')) config.time_mode = input.value;
    else if (input.hasAttribute('data-table-motion')) config.table_motion = input.value;
    else if (input.dataset.effect === 'text') config.text_effect = input.value;
    else if (input.dataset.carousel === 'enabled') config.carousel = input.checked;
    else if (input.dataset.carousel === 'interval') config.carousel_interval = Number(input.value);
    else return;
    const field = [...input.attributes].find(attribute => attribute.name.startsWith('data-'));
    render({ keepInspector: !!input.dataset.configField });
    if (input.dataset.configField === 'title') syncComponentName(card);
    if (!input.isConnected && field) {
      host.querySelector(`[${field.name}="${CSS.escape(field.value)}"]`)?.focus({ preventScroll: true });
    }
    if (checkURL) validateEmbed(checkURL, card.id);
    scheduleSave();
  };
}

function cardHTML(card) {
  const p = presentation(card);
  const frameURL = card.type === 'iframe' ? frameAddress(card.config?.url) : '';
  const tools = editing ? `<div class="card-tools" role="group" aria-label="组件工具">
    ${TopbaseUI.iconButton({ icon: 'grip-vertical', label: '拖动组件，或按回车设置位置', attrs: { 'data-drag': card.id } })}
    ${TopbaseUI.iconButton({ icon: 'settings-2', label: '组件设置', attrs: { 'data-card-style': card.id } })}
    ${TopbaseUI.iconButton({ icon: 'move', label: '位置与尺寸', attrs: { 'data-card-layout': card.id } })}
    ${TopbaseUI.iconButton({ icon: 'trash-2', label: '删除组件', attrs: { 'data-del': card.id }, tone: 'danger' })}
  </div>` : '';
  const body = card.type === 'heading'
    ? `<div class="heading-wrap"><h2 style="${typographyStyle(card)}"><span translate="no" ${editing ? 'contenteditable="true" data-edit="title"' : ''}>${richText(card)}</span></h2></div>`
      : card.type === 'text'
      ? `<div class="rich-text-wrap"><div translate="no" class="rich-text text-effect-${editing ? 'none' : (card.config && card.config.text_effect || 'none')}" style="${typographyStyle(card)}" ${editing ? 'contenteditable="true" data-edit="body"' : ''}>${richText(card)}</div></div>`
      : card.type === 'link'
        ? (editing ? `<span class="dashboard-link" translate="no">${esc(card.title || dashboardText('链接'))}</span>` : `<a class="dashboard-link" translate="no" href="${esc(card.body || '#')}" target="_blank" rel="noopener">${esc(card.title || card.body || dashboardText('链接'))}</a>`)
      : card.type === 'plugin' ? pluginHTML(card)
      : card.type === 'iframe' ? (!frameURL
        ? '<div class="image-placeholder">等待配置网页地址</div>'
        : `<div class="dashboard-frame-wrap"><iframe class="dashboard-frame" title="${esc(card.title || dashboardText('网页'))}" src="${esc(frameURL)}"></iframe>${editing ? '' : `<a class="iframe-open" href="${esc(frameURL)}" target="_blank" rel="noopener">无法显示？新窗口打开 ↗</a>`}</div>`)
          : card.type === 'image'
            ? imageHTML(card)
          : card.type === 'time'
            ? `<div class="dashboard-time" data-clock="${card.id}"><span></span></div>`
          : card.type === 'divider'
              ? `<div class="dashboard-divider"></div>`
          : `<div class="viz-stage" data-vizhost="${card.id}"><div class="viz-empty"><b>加载中</b><p>正在运行分析。</p></div></div>`;
  const hasTitle = card.question_id || card.type === 'divider';
  const title = !hasTitle || p.show_header === false ? '' : `<div class="card-head" ${editing ? `data-drag="${esc(card.id)}"` : ''}><h3 translate="no" ${card.type === 'divider' && editing ? 'contenteditable="true" data-edit="title"' : ''}>${card.question_id && !editing && !isPublicDashboard ? `<a href="/questions/${esc(card.question_id)}/">${esc(cardTitle(card))}</a>` : esc(cardTitle(card))}</h3></div>`;
  return `<article class="board-card ${editing ? 'editing' : ''} ${editing && selectedCard===card.id ? 'selected' : ''} ${card.type === 'divider' ? 'board-divider-card' : ''} card-border-${p.border} card-surface-${p.surface} card-padding-${p.padding} card-radius-${p.radius} card-shadow-${p.shadow} card-align-${p.align}" data-card="${esc(card.id)}" ${editing ? `tabindex="0" aria-label="${esc(componentName(card))}"` : ''}>
    ${tools}${title}
    <div class="card-body">${body}</div>
    ${editing ? `<span class="resize-handle" data-resize="${esc(card.id)}" aria-hidden="true"></span><span class="card-drag-hint" aria-hidden="true" hidden></span>` : ''}
  </article>`;
}
function frameAddress(value) {
  if (typeof value !== 'string' || !value.trim()) return '';
  try {
    const url = new URL(value, location.href);
    return ['http:', 'https:'].includes(url.protocol) ? url.href : '';
  } catch { return ''; }
}
function pluginHTML(card) {
  const url = frameAddress(card.config?.url);
  if (!url) return '<div class="plugin-placeholder"><b>尚未配置自定义组件</b><span>在编辑状态输入组件运行地址。</span></div>';
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
  const options = {
    date: { dateStyle: 'short' },
    time: { timeStyle: 'medium', hour12: false },
    weekday: { weekday: 'long', year: 'numeric', month: 'long', day: 'numeric' },
    datetime: { dateStyle: 'short', timeStyle: 'medium', hour12: false }
  };
  const format = new Intl.DateTimeFormat(globalThis.TopbaseLocale?.locale() || 'zh-CN', options[mode] || options.datetime);
  const draw = () => {
    const text = format.format(new Date());
    if (target.firstElementChild.textContent === text) return;
    target.firstElementChild.textContent = text;
    fitClock(target);
  };
  draw(); motionTimers.push(setInterval(draw, 1000));
}

function fitClock(host) {
  const label = host?.firstElementChild;
  if (!label?.textContent || !host.clientWidth) return;
  const textWidth = label.getBoundingClientRect().width;
  if (!textWidth) return;
  const fontSize = parseFloat(getComputedStyle(label).fontSize);
  const fit = Math.min(34, fontSize * (host.clientWidth - 1) / textWidth, host.clientHeight / 1.2);
  label.style.fontSize = Math.max(1, Math.floor(fit)) + 'px';
}

function layoutEls() {
  const grid = $('#grid');
  if (!board || !grid || !grid.getBoundingClientRect().width) return;
  const cards = tabCards();
  let maxY = 4;
  cards.forEach(card => {
    const el = document.querySelector(`[data-card="${card.id}"]`);
    if (!el) return;
    const layout = drag?.card === card ? drag.draft : card.layout;
    const r = rect(layout || { x: 0, y: 0, w: 4, h: 4 });
    if (editing) el.style.setProperty('--card-offset-x', r.left + 'px');
    el.style.left = r.left + 'px';
    el.style.top = r.top + 'px';
    el.style.width = r.width + 'px';
    el.style.height = r.height + 'px';
    applyCardContentScale(el, card, r);
    if (card.type === 'time') fitClock(el.querySelector('[data-clock]'));
    maxY = Math.max(maxY, (layout?.y || 0) + (layout?.h || 1));
  });
  grid.style.setProperty('--board-content-height', (maxY * (ROW + Y_GAP) + 40) + 'px');
}
function applyCardContentScale(el, card, box) {
  if (card.type !== 'question') return;
  const scalar = cardVizType(card) === 'scalar';
  const compactHead = !editing && box.height < 104;
  el.classList.toggle('card-content-scalar', scalar);
  if (scalar) {
    // A number is already the distilled result. Scaling the whole visualization
    // makes that primary value unnecessarily tiny, so fit the type itself.
    el.classList.remove('card-content-scaled');
    el.classList.toggle('card-content-mini', compactHead);
    el.style.setProperty('--card-content-scale', '1');
    fitScalarContent(el);
    return;
  }
  // A chart has a comfortable design size, but on a data screen a deliberately
  // small tile should still behave like one visual, not like a scrollable app.
  // Render it at that size and scale the whole result into the assigned tile.
  const usableWidth = Math.max(1, box.width - (compactHead ? 0 : 24));
  const usableHeight = Math.max(1, box.height - (compactHead ? 0 : 42));
  const scale = Math.min(1, usableWidth / 280, usableHeight / 180);
  const scaled = scale < .995;
  el.classList.toggle('card-content-scaled', scaled);
  el.classList.toggle('card-content-mini', compactHead && scaled);
  el.style.setProperty('--card-content-scale', String(Math.max(.18, scale)));
}
function fitScalarContent(el) {
  const value = el && el.querySelector('.viz-scalar > b');
  const body = el && el.querySelector('.card-body');
  if (!value || !body) return;
  const box = body.getBoundingClientRect();
  const glyphs = Math.max(3, Array.from((value.textContent || '').trim()).length);
  const hasDetail = !!el.querySelector('.viz-scalar em');
  const widthSize = Math.max(1, box.width - 16) / (glyphs * .6);
  const heightSize = Math.max(1, box.height - 8) * (hasDetail ? .48 : .68);
  const size = clamp(Math.floor(Math.min(64, widthSize, heightSize)), 22, 64);
  el.style.setProperty('--scalar-number-size', size + 'px');
  el.style.setProperty('--scalar-detail-size', clamp(Math.round(size * .3), 10, 13) + 'px');
  el.classList.toggle('scalar-detail-compact', hasDetail && (box.height < 92 || box.width < 150));
}
function scheduleCanvasLayout() {
  if (layoutFrame) cancelAnimationFrame(layoutFrame);
  layoutFrame = requestAnimationFrame(() => {
    layoutFrame = 0;
    layoutEls();
    syncInspectorToggle();
  });
}
function observeCanvasLayout() {
  const canvas = $('#board-canvas');
  if (!canvas || canvasResizeObserver || !window.ResizeObserver) return;
  canvasResizeObserver = new ResizeObserver(scheduleCanvasLayout);
  canvasResizeObserver.observe(canvas);
}

function keepFormatSelection(event) {
  if (event.target.closest('[data-format]')) event.preventDefault();
}
function formatSelectedText(event) {
  const button = event.target.closest('[data-format]');
  if (!button || !editing) return;
  const cardEl = $$('#grid [data-card]').find(el => el.dataset.card === selectedCard);
  const editor = cardEl?.querySelector('.rich-text[contenteditable], .heading-wrap [contenteditable]');
  if (!editor) return;
  const selection = window.getSelection();
  const range = selection.rangeCount ? selection.getRangeAt(0) : null;
  const selected = range && !range.collapsed && editor.contains(range.commonAncestorContainer) ? range.cloneRange() : null;
  editor.focus({ preventScroll: true });
  const target = selected || document.createRange();
  if (!selected) target.selectNodeContents(editor);
  selection.removeAllRanges();
  selection.addRange(target);
  document.execCommand(button.dataset.format, false, null);
}
function bindCardChrome() {
  const showProperties = (id, section) => {
    const alreadySelected = selectedCard === id;
    inspectorSection = section;
    selectCard(id);
    if (alreadySelected) renderInspector();
    openCardInspector();
  };
  $$('[data-card-style], [data-card-layout]').forEach(button => button.onclick = event => {
    event.stopPropagation();
    showProperties(button.dataset.cardStyle || button.dataset.cardLayout, button.hasAttribute('data-card-layout') ? 'layout' : 'appearance');
  });
  $$('[data-del]').forEach(btn => btn.onclick = async ev => {
    ev.stopPropagation();
    board.cards = board.cards.filter(c => c.id !== btn.dataset.del);
    render();
    try { await saveBoard(); }
    catch (e) { toast(e.message); }
  });
  $$('[data-edit]').forEach(el => {
    el.onblur = () => {
      const card = board.cards.find(c => c.id === el.closest('[data-card]').dataset.card);
      if (!card) return;
      if (el.dataset.edit === 'title') {
        card.title = el.textContent.trim();
        if (card.type === 'heading') {
          card.config ||= {};
          card.config.rich_title = richText({ ...card, config: { rich_title: el.innerHTML } });
        }
      }
      else {
        card.body = el.innerText.trim();
        card.config = Object.assign({}, card.config || {}, { rich_text: richText({ config: { rich_text: el.innerHTML } }) });
      }
      syncComponentName(card);
      scheduleSave();
    };
  });
  const inspector = $('#card-inspector');
  inspector?.addEventListener('mousedown', keepFormatSelection);
  inspector?.addEventListener('click', formatSelectedText);
  $$('[data-drag]').forEach(el => {
    el.onpointerdown = ev => startDrag(ev, el.dataset.drag, 'move');
    if (el.matches('button')) el.onclick = ev => {
      ev.stopPropagation();
      if (ev.detail === 0) showProperties(el.dataset.drag, 'layout');
    };
  });
  $$('[data-resize]').forEach(el => el.onpointerdown = ev => startDrag(ev, el.dataset.resize, 'resize'));
  if (editing) $$('[data-card]').forEach(el => {
    el.onclick = ev => {
      if (!ev.target.closest('a,button,select,input')) selectCard(el.dataset.card);
    };
    el.addEventListener('focusin', () => selectCard(el.dataset.card));
    el.onkeydown = ev => {
      if (ev.target === el && ev.key === 'Enter') {
        ev.preventDefault();
        showProperties(el.dataset.card, 'appearance');
      }
    };
  });
}
async function validateEmbed(url, cardID) {
  const status = document.querySelector(`[data-embed-validation="${cardID}"]`);
  if (!status || !url) return;
  status.textContent = '正在检查网站的嵌入策略…'; status.className = 'embed-validation checking';
  try {
    const result = await api('/api/embed/validate', 'POST', { url });
    status.textContent = result.embeddable ? '✓ ' + dashboardText(result.reason) : dashboardText('无法嵌入：{reason}', {reason:dashboardText(result.reason)});
    status.className = 'embed-validation ' + (result.embeddable ? 'valid' : 'invalid');
  } catch (error) { status.textContent = dashboardText('无法校验：{reason}', {reason:dashboardText(error.message)}); status.className = 'embed-validation invalid'; }
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
  if (!editing || finishingEdit || drag || ev.button !== 0 || ev.isPrimary === false) return;
  if (ev.target.closest('a, button:not([data-drag]), select, input, [contenteditable]')) return;
  const card = board.cards.find(item => item.id === cardId);
  if (!card) return;
  const el = ev.currentTarget.closest('[data-card]');
  ev.preventDefault();
  selectCard(cardId);
  drag = {
    mode, card, el, pointer: ev.pointerId, startX: ev.clientX, startY: ev.clientY,
    draft: { ...card.layout }, active: false, hint: el.querySelector('.card-drag-hint')
  };
  el.setPointerCapture(ev.pointerId);
  el.addEventListener('lostpointercapture', cancelDrag);
  window.addEventListener('pointermove', onDrag);
  window.addEventListener('pointerup', endDrag);
  window.addEventListener('pointercancel', cancelDrag);
  window.addEventListener('blur', cancelDrag);
  window.addEventListener('keydown', cancelDragKey, true);
}

function cancelDragKey(ev) {
  if (ev.key !== 'Escape') return;
  ev.preventDefault();
  ev.stopPropagation();
  cancelDrag();
}

function onDrag(ev) {
  if (!drag || ev.pointerId !== drag.pointer) return;
  if (!drag.el.isConnected) return cancelDrag();
  if (!drag.active && Math.hypot(ev.clientX - drag.startX, ev.clientY - drag.startY) < 5) return;
  const { card, mode, draft } = drag;
  if (!drag.active) {
    drag.active = true;
    drag.el.classList.add(mode === 'move' ? 'dragging' : 'resizing');
    drag.hint.hidden = false;
  }
  const cw = cellW() + X_GAP;
  const precision = appearance().snap ? 1 : 4;
  const dx = Math.round((ev.clientX - drag.startX) / cw * precision) / precision;
  const dy = Math.round((ev.clientY - drag.startY) / (ROW + Y_GAP) * precision) / precision;
  if (mode === 'move') {
    draft.x = clamp(card.layout.x + dx, 0, COLS - card.layout.w);
    draft.y = Math.max(0, card.layout.y + dy);
    drag.hint.textContent = `X ${draft.x} · Y ${draft.y}`;
  } else {
    draft.w = clamp(card.layout.w + dx, minWidth(card), COLS - card.layout.x);
    draft.h = clamp(card.layout.h + dy, minHeight(card), 48);
    drag.hint.textContent = dashboardText('{w} × {h} 格', {w:draft.w,h:draft.h});
  }
  scheduleCanvasLayout();
}

function cancelDrag(ev) {
  if (ev?.pointerId != null && ev.pointerId !== drag?.pointer) return;
  finishDrag(false);
}

function endDrag(ev) {
  if (!drag || ev.pointerId !== drag.pointer) return;
  onDrag(ev);
  finishDrag(true);
}

function finishDrag(commit) {
  if (!drag) return;
  const { card, draft, el, hint, pointer, active } = drag;
  drag = null;
  el.removeEventListener('lostpointercapture', cancelDrag);
  window.removeEventListener('pointermove', onDrag);
  window.removeEventListener('pointerup', endDrag);
  window.removeEventListener('pointercancel', cancelDrag);
  window.removeEventListener('blur', cancelDrag);
  window.removeEventListener('keydown', cancelDragKey, true);
  if (el.hasPointerCapture(pointer)) el.releasePointerCapture(pointer);
  el.classList.remove('dragging', 'resizing');
  hint.hidden = true;
  if (!active) return;
  if (commit && el.isConnected && ['x','y','w','h'].some(key => draft[key] !== card.layout[key])) {
    card.layout = draft;
    resolveOverlap(card);
    if (selectedCard === card.id && inspectorSection === 'layout') {
      TopbaseDashboardLayout.sync($('#card-inspector'), card.layout, layoutLimits(card));
    }
    scheduleSave();
  }
  layoutEls();
}

function renderModeActions() {
  const edit = $('#edit');
  const preview = $('#preview');
  if (!edit || !preview) return;
  $('#view-actions').hidden = editing || !!previewOrigin;
  edit.hidden = !!previewOrigin;
  edit.disabled = finishingEdit;
  edit.textContent = editing ? '完成编辑' : '编辑';
  preview.hidden = !editing && !previewOrigin;
  preview.disabled = finishingEdit;
  preview.textContent = previewOrigin ? '返回编辑' : '预览';
  preview.title = previewOrigin ? '返回编辑（Esc）' : '预览仪表盘';
}

function togglePreview() {
  if (isPublicDashboard || !board || finishingEdit || drag || (!editing && !previewOrigin)) return;
  let needsData = false;
  if (previewOrigin) {
    const previous = previewOrigin;
    previewOrigin = null;
    editing = true;
    needsData = activeTab !== previous.activeTab;
    activeTab = previous.activeTab;
    selectedCard = previous.selectedCard;
    for (const [input, value] of previous.filters) {
      if (input.value !== value) needsData = true;
      input.value = value;
    }
    renderTabs();
  } else {
    previewOrigin = { activeTab, selectedCard, filters: $$('[data-filter]').map(input => [input, input.value]) };
    editing = false;
  }
  render();
  // Reuse results for purely visual transitions. A tab or filter change needs
  // fresh data, as does a card whose load was interrupted by the new DOM.
  if (needsData || tabCards().some(card => card.type === 'question' && !cardData[card.id])) loadCards();
  $('#preview').focus({ preventScroll: true });
}

function render({ keepInspector = false } = {}) {
  motionTimers.forEach(clearInterval); motionTimers = [];
  disposeCardVisuals();
  applyAppearance();
  $('#layout')?.classList.toggle('viewing', !editing);
  $('#board-canvas')?.classList.toggle('canvas-editing', editing);
  $('#grid').classList.toggle('editing', editing);
  renderModeActions();
  const cards = tabCards();
  const emptyCanvas = '<div class="board-empty"><h2>空白画布</h2><button type="button" data-empty-insert class="tb-btn tb-btn-primary" aria-controls="palette" aria-haspopup="dialog" aria-expanded="false">插入内容</button></div>';
  $('#grid').innerHTML = cards.length ? cards.map(cardHTML).join('') : (editing ? emptyCanvas : '');
  layoutEls();
  if (!keepInspector) renderInspector();
  bindCardChrome();
  bindGridDrop();
  renderPalette();
  syncEditorTools(editing);
  if ($('#outline-anchor')) {
    if (editing) renderOutline();
    else closeOutline();
  }
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

function disposeCardVisuals() {
  document.querySelectorAll('#grid .viz-chart').forEach(element => {
    if (element._tbRo) { element._tbRo.disconnect(); element._tbRo = null; }
    if (element._tbChart) { element._tbChart.dispose(); element._tbChart = null; }
  });
}

const questionTileSize = { w: 12, h: 10 };
let paletteDrag = null;

function beginPaletteDrag(event, item) {
  if (!editing) { event.preventDefault(); return; }
  paletteDrag = item;
  // During dragover browsers expose the data types but hide the payload.
  // Keep a local descriptor so the preview can use the item's actual size.
  event.dataTransfer.setData('text/plain', item.widgetId ? 'component:' + item.widgetId : item.questionId);
  event.dataTransfer.effectAllowed = 'copy';
  // Let the browser capture the drag image before making the panel transparent.
  if ($('#palette')) requestAnimationFrame(() => {
    if (paletteDrag === item) $('#palette').classList.add('palette-in-flight');
  });
}

function clearPaletteDrag() {
  paletteDrag = null;
  $('#palette')?.classList.remove('palette-in-flight');
  $('#ghost')?.remove();
}

window.addEventListener('blur', clearPaletteDrag);
document.addEventListener('dragend', clearPaletteDrag);

function placementForDrag(event) {
  if (!editing || !paletteDrag) return null;
  const size = paletteDrag.widgetId ? TopbaseDashboardWidgets.size(paletteDrag.widgetId) : questionTileSize;
  return size && cellAt(event.clientX, event.clientY, size.w, size.h);
}

function bindGridDrop() {
  const grid = $('#grid');
  grid.ondragover = ev => {
    const cell = placementForDrag(ev);
    if (!cell) return;
    ev.preventDefault();
    ev.dataTransfer.dropEffect = 'copy';
    let ghost = $('#ghost');
    if (!ghost) {
      ghost = document.createElement('div');
      ghost.id = 'ghost';
      ghost.ariaHidden = 'true';
      grid.appendChild(ghost);
    }
    const geometry = `X ${cell.x} · Y ${cell.y}　${dashboardText('{w} × {h} 格', {w:cell.w,h:cell.h})}`;
    if (ghost.title !== geometry) ghost.title = geometry;
    ghost.className = cell.x >= COLS / 2 ? 'board-ghost hint-right' : 'board-ghost';
    const r = rect(cell);
    ghost.style.left = r.left + 'px';
    ghost.style.top = r.top + 'px';
    ghost.style.width = r.width + 'px';
    ghost.style.height = r.height + 'px';
  };
  grid.ondragleave = ev => {
    if (!grid.contains(ev.relatedTarget)) $('#ghost')?.remove();
  };
  grid.ondrop = ev => {
    const cell = placementForDrag(ev);
    if (!cell) return;
    ev.preventDefault();
    const { widgetId, questionId } = paletteDrag;
    clearPaletteDrag();
    if (widgetId) addStatic(widgetId, cell);
    else addQuestion(questionId, cell);
  };
}

function insertCard(card) {
  const focusNewCard = $('#palette')?.contains(document.activeElement);
  closeInsertPalette();
  resolveOverlap(card);
  board.cards = (board.cards || []).concat(card);
  selectedCard = card.id;
  inspectorSection = 'appearance';
  render();
  if (focusNewCard) $(`[data-card-style="${card.id}"]`)?.focus({ preventScroll: true });
  $(`[data-card="${card.id}"]`)?.scrollIntoView({ block: 'nearest', inline: 'nearest' });
}

async function addQuestion(questionId, layout) {
  if (!questionId || usedQuestionIDs().has(questionId)) return toast('这条分析已经在看板上');
  const q = questionMap[questionId];
  const place = layout || { x: 0, y: nextY(), ...questionTileSize };
  const card = {
    id: newID('crd'),
    type: 'question',
    question_id: questionId,
    title: q ? q.name : '',
    tab_id: activeTab,
    layout: place,
    config: { chartspec: { type: (q && q.chartspec && q.chartspec.type) || 'table' } }
  };
  insertCard(card);
  try {
    await saveBoard();
    await loadCards();
  } catch (e) { toast(e.message); }
}

async function addStatic(id, layout) {
  const defaults = TopbaseDashboardWidgets.create(id);
  if (!editing || !defaults || id === 'image') return;
  const place = layout || { x: 0, y: nextY(), ...TopbaseDashboardWidgets.size(id) };
  const card = { id: newID('crd'), tab_id: activeTab, ...defaults, layout: place };
  insertCard(card);
  try { await saveBoard(); }
  catch (e) { toast(e.message); }
}
async function addImage(files) {
  const valid = [...files].filter(file => file.type.startsWith('image/') && file.size <= 3 * 1024 * 1024);
  if (!valid.length) return toast('请选择 3MB 以内的图片文件');
  const images = await Promise.all(valid.map(file => new Promise((resolve, reject) => { const reader = new FileReader(); reader.onload = () => resolve(reader.result); reader.onerror = reject; reader.readAsDataURL(file); })));
  const card = { id:newID('crd'), tab_id:activeTab, type:'image', title:valid[0].name.replace(/\.[^.]+$/, ''), config:{images, carousel:images.length > 1, carousel_interval:5}, layout:{x:0,y:nextY(),...TopbaseDashboardWidgets.size('image')} };
  insertCard(card);
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
  TopbaseViz.render(host, { columns: d.columns || [], rows: d.rows || [], spec, queryir: q && q.queryir, semanticTypes: d.meta && d.meta.semantic_types || {}, compact: true, dashboardOnly: true });
  if (spec.type === 'scalar') fitScalarContent(host.closest('[data-card]'));
  const limited = queryLimitMessage(d.meta);
  if (limited) host.insertAdjacentHTML('afterbegin', `<div class="dashboard-result-limit" title="${esc(limited)}">部分结果</div>`);
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


function paintCached() {
  tabCards().forEach(card => {
    if (card.type === 'question' && cardData[card.id]) paintCard(card, cardData[card.id]);
  });
}

async function loadCards() {
  if (loadController) loadController.abort();
  const controller = new AbortController();
  loadController = controller;
  const sequence = ++loadSequence;
  const filters = filterValues();
  const cards = (board.cards || []).filter(card => card.type === 'question' && (!activeTab || !card.tab_id || card.tab_id === activeTab));
  const visible = new Set(cards.map(card => card.id));
  Object.keys(cardData).forEach(id => { if (!visible.has(id)) delete cardData[id]; });
  let next = 0;
  const worker = async () => {
    while (next < cards.length && !controller.signal.aborted) {
      const card = cards[next++];
      const host = document.querySelector(`[data-vizhost="${card.id}"]`);
      if (!host) continue;
      try {
        const d = isPublicDashboard
          ? await publicAccess.requestCard(card.id, filters, controller.signal)
          : await api(`/api/dashboards/${board.id}/cards/${card.id}/dataset`, 'POST', { filters }, { signal: controller.signal });
        if (controller.signal.aborted || sequence !== loadSequence) return;
        cardData[card.id] = d;
        paintCard(card, d);
      } catch (e) {
        if (controller.signal.aborted || e.name === 'AbortError') return;
        if (isPublicDashboard && publicAccess.handleDenial(e)) return;
        host.innerHTML = `<div class="viz-error"><b>卡片无法显示</b><p>${esc(e.message)}</p></div>`;
      }
    }
  };
  await Promise.all(Array.from({ length: Math.min(4, cards.length) }, worker));
  if (sequence === loadSequence) loadController = null;
}

async function boot() {
  if (isPublicDashboard) {
    const payload = await publicAccess?.load();
    if (!payload) return;
    board = payload.dashboard;
    questions = payload.questions || [];
  } else {
    [board, questions, collections] = await Promise.all([api('/api/dashboards/' + boardId), api('/api/questions'), api('/api/collections')]);
  }
  upgradeCanvasGrid();
  questionMap = Object.fromEntries(questions.map(q => [q.id, q]));
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

if ($('#preview')) {
  $('#preview').onclick = togglePreview;
  document.addEventListener('keydown', event => {
    if (!previewOrigin || event.key !== 'Escape' || event.defaultPrevented || event.isComposing || document.fullscreenElement) return;
    event.preventDefault();
    togglePreview();
  });
}
if ($('#edit')) $('#edit').onclick = async () => {
  if (!board || previewOrigin || finishingEdit) return;
  if (editing) {
    finishingEdit = true;
    renderModeActions();
    try {
      await saveBoard();
      if (!(board.cards || []).length) {
        editing = true;
        toast('画布还是空的，可以点「插入」添加内容');
        render();
        return;
      }
      editing = false;
      selectedCard = '';
      $('#layout').classList.remove('inspector-open');
      toast('已保存');
      render();
      await loadCards();
    } catch (e) { toast(e.message); }
    finally {
      finishingEdit = false;
      renderModeActions();
    }
    return;
  }
  editing = true;
  render();
  await loadCards();
};
if ($('#apply')) $('#apply').onclick = loadCards;
if ($('#qsearch')) $('#qsearch').oninput = () => { paletteLimit = 60; renderPalette(); };
if ($('#qcollection')) $('#qcollection').onchange = () => { paletteLimit = 60; renderPalette(); };
if ($('#palette-more')) $('#palette-more').onclick = () => { paletteLimit += 60; renderPalette(); };
if ($('#insert-toggle')) {
  const panel = $('#palette');
  insertPopover = TopbaseUI.popover($('#insert-toggle'), panel, {
    type: 'dialog',
    canOpen: () => editing,
    initialFocus: () => $(paletteTab === 'analysis' ? '#qsearch' : '#widget-search'),
    keepOpenOnOutside: () => !!paletteDrag,
    onClose: clearPaletteDrag
  });
  document.addEventListener('click', event => {
    const opener = event.target.closest('[data-empty-insert]');
    if (opener) insertPopover.open({ from: opener });
  });
}
if ($('#palette-tabs')) {
  TopbaseUI.tabs.mount($('#palette-tabs'), {
    panels: $('#palette'), tabSelector: '[data-palette-tab]', panelSelector: '[data-palette-panel]',
    tabValue: tab => tab.dataset.paletteTab, panelValue: panel => panel.dataset.palettePanel,
    initial: paletteTab, onChange: value => { paletteTab = value; }
  });
}
if ($('#image-upload')) $('#image-upload').onchange = event => { addImage(event.target.files); event.target.value = ''; };
if ($('#widget-library')) {
  const library = $('#widget-library');
  renderWidgetLibrary();
  $('#widget-search').oninput = renderWidgetLibrary;
  window.addEventListener('topbase:widgets-changed', renderWidgetLibrary);
  library.onclick = event => {
    const id = event.target.closest('[data-add]')?.dataset.add;
    if (!editing || !id) return;
    if (id === 'image') $('#image-upload').click();
    else addStatic(id);
  };
  library.ondragstart = event => {
    const widgetId = event.target.closest('[draggable="true"]')?.dataset.add;
    if (widgetId) beginPaletteDrag(event, { widgetId });
  };
  library.ondragend = clearPaletteDrag;
}
bindInspector();
if ($('#outline-anchor')) bindOutline();
$('#grid').addEventListener('click', event => {
  if (event.target === $('#grid')) selectCard('');
});
observeCanvasLayout();
window.addEventListener('resize', scheduleCanvasLayout);
window.addEventListener('pagehide', () => {
  if (loadController) loadController.abort();
  motionTimers.forEach(clearInterval); motionTimers = [];
  disposeCardVisuals();
  if (canvasResizeObserver) canvasResizeObserver.disconnect();
});

const shareDialogElement = $('#modal');
let shareExpiryTimer;
const shareDialog = shareDialogElement && TopbaseUI.modal(shareDialogElement, {
  onClose: () => {
    clearTimeout(shareExpiryTimer);
    shareDialogElement.replaceChildren();
  }
});
async function copyValue(value) {
  try { await navigator.clipboard.writeText(value); toast('已复制'); }
  catch { toast('复制失败，请手动复制'); }
}
function copyShareField(label, value) {
  return `<div class="share-link-field"><label>${label}<input class="tb-input" readonly value="${esc(value)}"></label>
    <button data-copy="${esc(value)}" class="tb-btn tb-btn-secondary" aria-label="${esc(dashboardText('复制{label}', {label:dashboardText(label)}))}" type="button">复制</button></div>`;
}
function shareLocalTime(value) {
  if (!value) return '';
  const date = new Date(value);
  return new Date(date.getTime() - date.getTimezoneOffset() * 60000).toISOString().slice(0, 16);
}

// Send only changed settings when editing a live link. In particular, leaving
// the replacement password blank must keep the hash already on the server.
function sharePolicyUpdate(policy, draft, now = Date.now()) {
  const body = {};
  const publish = !policy.public_uuid;
  if (draft.expiry === 'custom') {
    if (publish || !policy.expires_at || draft.until !== shareLocalTime(policy.expires_at)) {
      const end = new Date(draft.until);
      if (!Number.isFinite(end.getTime()) || end.getTime() <= now) return { field: 'share-until', error: '请选择晚于当前时间的截止时间。' };
      body.expires_at = end.toISOString();
    }
  } else if (draft.expiry === 'permanent') {
    if (publish || policy.expires_at) body.expires_at = null;
  } else {
    const days = Number(draft.expiry);
    if (![1, 7, 30].includes(days)) return { field: 'share-expiry', error: '请选择有效期。' };
    body.expires_at = new Date(now + days * 86400000).toISOString();
  }
  if (draft.protected && (!policy.password_required || draft.password)) {
    const length = Array.from(draft.password).length;
    if (length < 8 || length > 128 || !draft.password.trim()) {
      return { field: 'share-password', error: '请输入 8 至 128 个字符的访问密码，不能全为空格。' };
    }
    body.password = draft.password;
  } else if (!draft.protected && policy.password_required) {
    body.password = '';
  }
  return { body };
}
function renderShareDialog(policy, focusId) {
  const modal = shareDialogElement;
  clearTimeout(shareExpiryTimer);
  const published = !!policy.public_uuid;
  const canToggleLink = policy.can_manage && (published || policy.allow_public);
  const canEditPolicy = policy.can_manage && policy.allow_public;
  const path = '/dashboard/' + encodeURIComponent(policy.public_uuid || '') + '/';
  const publicURL = policy.base_url + '/public' + path;
  const embedURL = policy.base_url + '/embed' + path;
  const initialExpiry = policy.expires_at ? 'custom' : 'permanent';
  const initialTime = shareLocalTime(policy.expires_at);
  const expiryText = policy.expires_at ? new Date(policy.expires_at).toLocaleString(globalThis.TopbaseLocale?.locale() || 'zh-CN', { dateStyle: 'medium', timeStyle: 'short' }) : '';
  const settingsLink = policy.is_admin && (!policy.allow_public || !policy.allow_embed)
    ? '<a href="/admin/settings/#access">前往管理后台设置对外访问</a>' : '';
  const expiryChoices = [
    { value: 'permanent', label: '永久有效' }, { value: '1', label: '1 天' }, { value: '7', label: '7 天' },
    { value: '30', label: '30 天' }, { value: 'custom', label: '自定义截止时间' }
  ];
  const publicHelp = !policy.allow_public ? '工作区已暂停公开分享。' : !policy.can_manage ? '需要仪表盘管理权限才能更改分享设置。' : '';
  modal.innerHTML = `<header><h2 id="share-dialog-title">分享与嵌入</h2>${TopbaseUI.iconButton({icon:'x', label:'关闭', attrs:{id:'close-action'}})}</header>
    <form id="share-policy" novalidate><fieldset id="share-controls">
      <label class="share-toggle"><input id="public-sharing-toggle" type="checkbox" ${published ? 'checked' : ''} ${canToggleLink ? '' : 'disabled'}>
        <span><b>公开分享</b><small id="share-status" role="status"></small></span></label>
      ${publicHelp ? `<p class="share-help">${publicHelp}</p>` : ''}
      ${canEditPolicy ? `<div class="share-policy-fields" id="share-fields" hidden>
        ${TopbaseUI.select({label:'链接有效期', options:expiryChoices, value:initialExpiry, attrs:{id:'share-expiry'}})}
        <div id="share-until-field" hidden>${TopbaseUI.input({label:'截止时间（本地时间）', type:'datetime-local', value:initialTime, attrs:{id:'share-until'}})}</div>
        ${TopbaseUI.checkbox({label:'需要访问密码', checked:policy.password_required, attrs:{id:'share-protect'}})}
        <div id="share-password-field" hidden>${TopbaseUI.input({label:policy.password_required ? '更换密码' : '访问密码', type:'password',
          attrs:{id:'share-password', autocomplete:'new-password', placeholder:policy.password_required ? '留空保留现有密码' : '8 至 128 个字符'}})}</div>
      </div>` : ''}
      <p id="share-policy-error" class="share-error" role="alert" hidden></p>
      <div class="share-save" id="share-save-row" hidden><button type="submit" id="share-save" class="tb-btn tb-btn-primary"></button></div>
    </fieldset></form>
    <div id="share-links" hidden>${published ? copyShareField('公开链接', publicURL) : ''}</div>
    <section class="share-section"><label class="share-toggle"><input id="embed-enabled" type="checkbox" ${policy.public_embed_enabled ? 'checked' : ''}>
      <span><b>允许 iframe 嵌入</b><small id="share-embed-help"></small></span></label>
      <div id="share-embed-links" hidden>${policy.is_admin && published && policy.public_embed_enabled
        ? copyShareField('嵌入地址', embedURL) + copyShareField('iframe 代码', `<iframe src="${embedURL}" width="100%" height="800" title="仪表盘"></iframe>`) : ''}</div>
    </section>${settingsLink}`;
  const form = $('#share-policy', modal);
  const toggle = $('#public-sharing-toggle', form);
  const protect = $('#share-protect', form);
  const password = $('#share-password', form);
  const expiry = $('#share-expiry', form);
  const until = $('#share-until', form);
  const save = $('#share-save', form);
  const embed = $('#embed-enabled', modal);
  const error = $('#share-policy-error', form);
  let busy = false;
  function sync() {
    const enabled = toggle.checked;
    const expired = published && policy.expires_at && Date.parse(policy.expires_at) <= Date.now();
    const active = published && policy.allow_public && !expired;
    const dirty = enabled !== published || (enabled && canEditPolicy &&
      (expiry.value !== initialExpiry || (expiry.value === 'custom' && until.value !== initialTime) ||
       protect.checked !== !!policy.password_required || (protect.checked && !!password.value)));
    $('#share-status', modal).textContent = published && !enabled ? '保存后，现有链接和嵌入将停止访问。' :
      !published ? (enabled ? '确认有效期和密码后，点击「开启分享」。' : '当前仅在工作区内可见。') :
      (expired ? dashboardText('已于 {date} 到期', {date:expiryText}) : (expiryText ? dashboardText('有效至 {date}', {date:expiryText}) : dashboardText('永久有效'))) + ' · ' + dashboardText(policy.password_required ? '需要访问密码' : '无需访问密码');
    if (canEditPolicy) {
      $('#share-fields', form).hidden = !enabled;
      $('#share-until-field', form).hidden = expiry.value !== 'custom';
      $('#share-password-field', form).hidden = !protect.checked;
    }
    $('#share-links', modal).hidden = !active || !enabled;
    $('#share-save-row', form).hidden = !dirty;
    save.textContent = enabled ? (published ? '保存设置' : '开启分享') : '撤销链接';
    save.classList.toggle('tb-btn-danger', !enabled);
    save.classList.toggle('tb-btn-primary', enabled);
    $('#share-controls', form).disabled = busy;
    embed.disabled = busy || dirty || !policy.is_admin || (!policy.public_embed_enabled && !(active && policy.allow_embed));
    $('#share-embed-links', modal).hidden = !(policy.is_admin && active && enabled && policy.allow_embed && policy.public_embed_enabled);
    $('#share-embed-help', modal).textContent = !policy.is_admin ? '仅工作区管理员可以设置嵌入。' :
      !policy.allow_public || !policy.allow_embed ? '工作区需要同时允许公开分享和外部嵌入。' :
      !published || !enabled ? '开启公开分享后可设置嵌入。' : expired ? '续期后可继续使用嵌入。' :
      dirty ? '保存分享设置后可更改嵌入。' : '嵌入页面使用相同的有效期和访问密码。';
  }
  function refreshAtExpiry() {
    sync();
    const remaining = Date.parse(policy.expires_at) - Date.now();
    if (remaining > 0) shareExpiryTimer = setTimeout(refreshAtExpiry, Math.min(remaining + 10, 2147483647));
  }
  function clearError() {
    error.hidden = true;
    form.querySelector('[aria-invalid="true"]')?.removeAttribute('aria-invalid');
  }
  function showError(message, field) {
    error.textContent = message;
    error.hidden = false;
    if (field) {
      const input = $('#' + field, form);
      input.setAttribute('aria-invalid', 'true');
      input.setAttribute('aria-describedby', error.id);
      input.focus();
    }
  }
  form.oninput = () => { clearError(); sync(); };
  form.onchange = () => {
    if (protect && !protect.checked) password.value = '';
    clearError(); sync();
  };
  $('#close-action', modal).onclick = () => shareDialog.close();
  $$('[data-copy]', modal).forEach(button => button.onclick = () => copyValue(button.dataset.copy));
  async function change(resource, method, body, label, nextFocus) {
    if (busy) return;
    busy = true;
    clearError(); sync();
    try {
      const result = await api(`/api/dashboards/${boardId}/${resource}`, method, body);
      const updated = result.dashboard || result;
      board.public_uuid = policy.public_uuid = updated.public_uuid || '';
      board.public_embed_enabled = policy.public_embed_enabled = !!updated.public_embed_enabled;
      if (resource === 'public-link') {
        policy.expires_at = method === 'DELETE' ? null : result.expires_at;
        policy.password_required = method !== 'DELETE' && !!result.password_required;
      }
      board.public_expires_at = policy.expires_at;
      renderPublicBadge();
      if (modal.open && modal.contains(form)) renderShareDialog(policy, nextFocus);
      toast(label);
    } catch (failure) {
      if (modal.open && modal.contains(form)) {
        busy = false;
        embed.checked = !!policy.public_embed_enabled;
        sync();
        showError(failure.message);
      } else toast(failure.message);
    }
  }
  form.onsubmit = event => {
    event.preventDefault();
    if (busy || !canToggleLink || $('#share-save-row', form).hidden) return;
    if (!toggle.checked) return change('public-link', 'DELETE', undefined, '公开链接已撤销', toggle.id);
    if (!canEditPolicy) return;
    const result = sharePolicyUpdate(policy, { expiry: expiry.value, until: until.value, protected: protect.checked, password: password.value });
    if (result.error) return showError(result.error, result.field);
    return change('public-link', published ? 'PUT' : 'POST', result.body, published ? '分享设置已保存' : '公开分享已开启', toggle.id);
  };
  embed.onchange = () => change('embedding', 'PUT', {enabled:embed.checked}, embed.checked ? '嵌入已开启' : '嵌入已关闭', embed.id);
  refreshAtExpiry();
  shareDialog.open({ from: $('#share') });
  if (focusId) {
    const next = $('#' + focusId, modal);
    (next && !next.disabled ? next : $('#close-action', modal)).focus({ preventScroll: true });
  }
}
async function openShare() {
  if (!board) return;
  const button = $('#share');
  button.disabled = true;
  try {
    const policy = await api(`/api/dashboards/${boardId}?view=sharing`);
    renderShareDialog(policy);
  } catch (error) {
    toast(error.message);
  } finally {
    button.disabled = false;
  }
}
async function addBookmark(){try{await api('/api/bookmarks','POST',{target_type:'dashboard',target_id:boardId});toast('已加入书签')}catch(error){toast(error.message)}}
async function archiveBoard(){if(!await confirmDialog({kicker:'仪表盘管理',title:'将这个仪表盘移到回收站？',description:'归档后会从仪表盘列表中隐藏，之后仍可在回收站恢复。',confirmText:'移到回收站',tone:'danger'}))return;try{await api('/api/dashboards/'+boardId,'DELETE');location.href='/dashboard/'}catch(error){toast(error.message)}}
async function duplicateBoard(){try{const copy=await api(`/api/dashboards/${boardId}/copy`,'POST',{});location.href='/dashboard/'+copy.id+'/'}catch(error){toast(error.message)}}
async function createAlert(){
  const card = (board.cards || []).find(c => c.type === 'question' && c.question_id);
  if (!card) return toast('这张板没有分析卡');
  try {
    const alert = await api('/api/alerts', 'POST', { name: dashboardText('{name} 有结果', {name:board.name}), question_id: card.question_id, kind: 'results', channel: 'inbox' });
    const note = await api('/api/alerts/' + alert.id + '/run', 'POST', {});
    toast(note.title + '：' + note.body);
  } catch (e) { toast(e.message); }
}
async function subscribeBoard(){
  try {
    const hooks = await api('/api/subscription-channels');
    const channels = hooks.map(hook => ({value: 'webhook:' + hook.id, label: hook.name}));
    if (!channels.length) return toast('请联系管理员先在“通知与订阅”中创建并启用 Webhook 通道');
    const channel = await choiceDialog({kicker:'仪表盘订阅',title:'选择每天的推送通道',description:'订阅将在每天 09:00 执行。管理员可在“通知与订阅”统一管理。',label:'Webhook 通道',value:channels[0].value,confirmText:'创建订阅',options:channels});
    if (channel === null) return;
    await api(`/api/dashboards/${boardId}/subscriptions`, 'POST', { cron: '0 9 * * *', channel });
    toast(dashboardText('已订阅每天 09:00（{channel}）', {channel: channels.find(option => option.value === channel).label}));
  } catch (e) { toast(e.message); }
}
if ($('#more')) {
  const panel = $('#more-menu');
  TopbaseUI.popover($('#more'), panel);
  const actions = { bookmark: addBookmark, duplicate: duplicateBoard, alert: createAlert, subscribe: subscribeBoard, archive: archiveBoard };
  panel.addEventListener('click', event => {
    const item = event.target.closest('[data-more]');
    if (item) actions[item.dataset.more]();
  });
}
if ($('#fullscreen')) $('#fullscreen').onclick=()=>$('#board-canvas')?.requestFullscreen?.();
if ($('#share')) $('#share').onclick=openShare;
const publicAccess = isPublicDashboard && window.TopbaseSharedDashboard?.mount(publicUUID, pathParts[0] === 'embed', {
  openDashboard: boot,
  clearCanvas() {
    loadController?.abort();
    motionTimers.forEach(clearInterval);
    motionTimers = [];
    disposeCardVisuals();
    for (const element of [$('#grid'), $('#tabs'), $('#filters')]) element.replaceChildren();
    board = null;
    cardData = {};
    questions = [];
    questionMap = {};
  }
});
boot().catch(e => { if ($('#title')) $('#title').textContent = e.message; else toast(e.message); });
