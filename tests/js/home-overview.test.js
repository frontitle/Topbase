const assert = require('node:assert/strict');
const fs = require('node:fs');
const test = require('node:test');
const vm = require('node:vm');

const shell = fs.readFileSync('internal/platform/httpapi/web/shell.js', 'utf8');
const home = fs.readFileSync('internal/platform/httpapi/web/index.html', 'utf8');
const app = fs.readFileSync('internal/platform/httpapi/web/app.js', 'utf8');

test('home shortcut and sidebar show warehouse management only to authorized users', async () => {
  assert.match(home, /id="warehouse-link"[^>]+hidden/);
  for (const isAdmin of [false, true]) {
    const user = { name: 'QA', is_admin: isAdmin };
    const nodes = new Map();
    function element(selector) {
      if (!nodes.has(selector)) nodes.set(selector, {
        dataset: {}, value: '', classList: { add() {}, toggle() {} }, setAttribute() {},
        querySelector: element, querySelectorAll: () => [], addEventListener() {}
      });
      return nodes.get(selector);
    }
    let mount;
    const menus = [];
    const context = vm.createContext({
      TopbaseUI: { popover: (trigger, panel) => menus.push([trigger, panel]) },
      window: { matchMedia: () => ({ matches: false, addEventListener() {} }) },
      document: {
        readyState: 'loading', querySelector: element, body: element('body'),
        addEventListener(type, fn) { if (type === 'DOMContentLoaded') mount = fn; }
      },
      localStorage: { getItem: () => null },
      fetch: async () => ({ ok: true, json: async () => user }),
      $: element, esc: String, emptyHTML: () => '',
      api: async path => {
        if (path === '/api/setup/status') return { completed: true };
        if (path === '/api/user/current') return user;
        return [];
      }
    });
    vm.runInContext(shell, context);
    await mount();
    await vm.runInContext(app, context);
    assert.deepEqual(menus, [[element('.sidebar-profile'), element('.account-menu')], [element('#home-create'), element('#home-create-menu')]]);
    assert.equal(element('[data-shell]').innerHTML.includes('href="/warehouse/"'), isAdmin);
    assert.equal(element('#warehouse-link').hidden, !isAdmin);
    assert.match(element('[data-shell]').innerHTML, /href="\/data\/"/);
  }
});

test('application sidebar omits model and search entries', () => {
  const appMenu = shell.slice(shell.indexOf('const appItems'), shell.indexOf('const adminItems'));
  assert.doesNotMatch(appMenu, /browse\/models|href:\s*['"]\/search\/|label:\s*['"](?:模型|搜索)['"]/);
  assert.doesNotMatch(appMenu, /label:\s*'数据组'/);
  assert.match(appMenu, /label:\s*'源数据'/);
  assert.match(appMenu, /label:\s*'数据沉淀'/);
  assert.ok(appMenu.indexOf("label: '仪表盘'") < appMenu.indexOf("label: '源数据'"));
});

test('workspace and admin navigation share a safe, stateful sidebar shell', () => {
  assert.doesNotMatch(shell, /product-rail|data-rail-action/);
  assert.match(shell, /active === 'account'/);
  assert.match(shell, /shell-context/);
  assert.match(shell, /<p>管理后台<\/p>/);
});

test('home is a data-driven overview with core statistics and working shortcuts', () => {
  assert.doesNotMatch(home, /auth-link|个人中心/);
  for (const id of ['stat-databases', 'stat-questions', 'stat-dashboards', 'stat-collections']) {
    assert.match(home, new RegExp(`id="${id}"`));
  }
  assert.match(home, /href="\/questions\/new\/"/);
  assert.match(home, /href="\/data\/"/);
  assert.match(home, /id="create-dashboard"/);
  assert.match(home, /href="\/questions\/\?view=groups"/);
  for (const endpoint of ['/api/databases', '/api/questions', '/api/dashboards', '/api/collections', '/api/notifications']) {
    assert.ok(app.includes(endpoint), `missing overview endpoint ${endpoint}`);
  }
  assert.match(app, /api\('\/api\/dashboards',\s*'POST'/);
});
