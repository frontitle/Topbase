const assert = require('node:assert/strict');
const fs = require('node:fs');
const test = require('node:test');
const vm = require('node:vm');

const script = fs.readFileSync('internal/platform/httpapi/web/dashboard-view.js', 'utf8');
const publicPage = fs.readFileSync('internal/platform/httpapi/web/embed/dashboard.html', 'utf8');

function dashboard(pathname) {
  const listeners = {};
  const filter = { value: '华东' };
  const elements = {
    '#grid': { addEventListener() {} },
    '#preview': { focus() {} }, '#edit': {}, '#view-actions': {}
  };
  const context = vm.createContext({
    location: { pathname }, URLSearchParams,
    document: { addEventListener(event, fn) { listeners[event] = fn; } },
    window: { addEventListener() {} },
    $: key => elements[key],
    $$: key => key === '[data-filter]' ? [filter] : [],
    api: () => new Promise(() => {})
  });
  const run = code => vm.runInContext(code, context);
  run(fs.readFileSync('internal/platform/httpapi/web/ui.js', 'utf8'));
  context.TopbaseUI = context.window.TopbaseUI;
  run(script);
  run(`
    board = { cards: [{ id: 'chart', type: 'question', tab_id: 'one' }] };
    selectedCard = 'chart'; activeTab = 'one'; cardData.chart = {}; editing = true;
    let loads = 0; render = renderModeActions; renderTabs = () => {};
    loadCards = () => { loads++; }; render();
  `);
  return { run, filter, elements, listeners };
}

test('preview returns to the selected tab, inspector and filters after interacting with the canvas', () => {
  const { run, elements, listeners, filter } = dashboard('/dashboard/private/');
  elements['#preview'].onclick();
  assert.equal(run('editing'), false);
  assert.equal(elements['#edit'].hidden, true);
  assert.equal(elements['#view-actions'].hidden, true);
  assert.equal(elements['#preview'].textContent, '返回编辑');
  assert.equal(run('loads'), 0);
  run("activeTab = 'two'");
  filter.value = '西南';
  listeners.keydown({ key: 'Escape', preventDefault() {} });
  assert.equal(run('editing'), true);
  assert.equal(run('activeTab'), 'one');
  assert.equal(run('selectedCard'), 'chart');
  assert.equal(filter.value, '华东');
  assert.equal(elements['#preview'].textContent, '预览');
  assert.equal(elements['#edit'].hidden, false);
  assert.equal(run('loads'), 1);
});

test('preview only opens from the private editor, including while saving or dragging', () => {
  assert.doesNotMatch(publicPage, /id="preview"/);
  for (const pathname of ['/public/dashboard/shared/', '/embed/dashboard/shared/']) {
    const { run } = dashboard(pathname);
    run('togglePreview()');
    assert.equal(run('previewOrigin'), null);
  }
  const { run } = dashboard('/dashboard/private/');
  for (const setup of ['editing = false', 'editing = true; drag = {}', 'drag = null; finishingEdit = true']) {
    run(setup);
    run('togglePreview()');
    assert.equal(run('previewOrigin'), null);
  }
});
test('canvas defaults follow the theme in legacy boards and keep explicit custom backgrounds', () => {
  const { run } = dashboard('/dashboard/private/');
  assert.equal(run('appearance().theme'), 'light');
  assert.equal(run('appearance().background'), '#ffffff');
  run("board.appearance = { theme: 'light', background: '#07111F', snap: false }");
  assert.equal(run('appearance().background'), '#ffffff');
  assert.equal(run('appearance().snap'), false);
  run("useCanvasTheme('aurora')");
  assert.equal(run('appearance().background'), '#08284a');
  run("board.appearance = { theme: 'deep', background: '#55336a' }; useCanvasTheme('light')");
  assert.equal(run('appearance().background'), '#55336a');
  run("board.appearance = { theme: 'light', background: '#07111f', background_custom: true }");
  assert.equal(run('appearance().background'), '#07111f');
  run("board.appearance = { theme: 'light', background: '#f5fbfb' }");
  assert.equal(run('appearance().background'), '#f5fbfb');
  run("useCanvasTheme('deep')");
  assert.equal(run('appearance().background'), '#07111f');
});

test('the shared canvas surface uses the stored background in each theme', () => {
  const css = fs.readFileSync('internal/platform/httpapi/web/dashboard.css', 'utf8');
  assert.match(publicPage, /id="board-canvas"/);
  assert.match(css, /#board-canvas\s*\{[^}]*background-color: var\(--canvas-custom-bg, #ffffff\)/);
});

test('inspector exposes controls that apply to the selected widget and maps legacy surfaces', () => {
  const { run } = dashboard('/dashboard/private/');
  run('const esc = value => String(value)');
  const panel = card => run('presentationPanel(' + JSON.stringify(card) + ')');
  const text = panel({ type: 'text', config: { presentation: { border: 'none' } } });
  assert.match(text, /data-typography="font"/);
  assert.match(text, /data-format="bold"/);
  assert.doesNotMatch(text, /data-presentation="(?:radius|shadow)"/);
  assert.doesNotMatch(panel({ type: 'image', config: { images: ['one'] } }), /data-carousel/);
  assert.match(panel({ type: 'image', config: { images: ['one','two'] } }), /data-carousel="enabled"/);
  assert.doesNotMatch(panel({ type: 'image', config: { images: ['one','two'] } }), /data-carousel="interval"/);
  const table = panel({ type: 'question', config: { chartspec: { type: 'table' } } });
  assert.match(table, /data-table-motion/);
  assert.doesNotMatch(table, /data-typography|data-presentation="align"/);
  const legacy = run("presentation({ type: 'time', config: { presentation: { border: 'glow', surface: 'glass' } } })");
  assert.equal(legacy.surface, 'solid');
  assert.equal(legacy.border, 'default');
});

test('publishing settings default to permanent and combine expiry and password in one request', () => {
  const {run} = dashboard('/dashboard/private/');
  const now = Date.parse('2026-11-27T00:00:00Z');
  const publish = (policy, draft) => JSON.parse(run('JSON.stringify(sharePolicyUpdate(' + JSON.stringify(policy) + ',' + JSON.stringify(draft) + ',' + now + '))'));
  const basic = {expiry: 'permanent', protected: false, password: ''};
  assert.deepEqual(publish({}, basic).body, {expires_at: null});
  assert.deepEqual(publish({}, {...basic, expiry:'7', protected:true, password:'分享访问密码测试'}).body,
    {expires_at: '2026-12-04T00:00:00.000Z', password: '分享访问密码测试'});
  assert.equal(publish({}, {...basic, expiry:'custom', until:'invalid'}).field, 'share-until');
  assert.equal(publish({}, {...basic, expiry:'custom', until:'2000-01-01T10:00'}).field, 'share-until');
  for (const password of ['', '1234567', '        ', '密'.repeat(129)]) {
    assert.equal(publish({}, {...basic, protected:true, password}).field, 'share-password');
  }
});

test('existing protected links retain precise deadlines and unused password fields, and expired links can be renewed', () => {
  const {run} = dashboard('/dashboard/private/');
  const policy = {public_uuid:'already-published', expires_at:'2020-01-01T00:01:45Z', password_required:true};
  const draft = {expiry:'custom', until:run("shareLocalTime('2020-01-01T00:01:45Z')"), protected:true, password:''};
  const update = changes => JSON.parse(run('JSON.stringify(sharePolicyUpdate(' + JSON.stringify(policy) + ',' + JSON.stringify({...draft,...changes}) + '))')).body;
  assert.deepEqual(update({}), {});
  assert.deepEqual(update({password:'新的分享访问密码'}), {password:'新的分享访问密码'});
  assert.deepEqual(update({expiry:'permanent'}), {expires_at:null});
  assert.deepEqual(update({protected:false}), {password:''});
});
