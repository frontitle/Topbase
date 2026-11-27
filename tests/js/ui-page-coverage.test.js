const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const vm = require('node:vm');

const root = path.resolve('internal/platform/httpapi/web');
const read = file => fs.readFileSync(path.join(root, file), 'utf8');

function templates(dir = '', extensions = ['.html']) {
  return fs.readdirSync(path.join(root, dir), { withFileTypes: true }).flatMap(entry => {
    const file = path.join(dir, entry.name);
    if (entry.isDirectory()) return entry.name === 'vendor' ? [] : templates(file, extensions);
    return extensions.includes(path.extname(file)) ? [file] : [];
  });
}

test('application pages load the shared controls before their business scripts', () => {
  const standalone = new Set(['collections/index.html', 'developers/dashboard-components/example/index.html']);
  const staticContent = new Set(['developers/dashboard-components/index.html', 'embed/unavailable.html']);
  for (const file of templates()) {
    if (standalone.has(file)) continue;
    const html = read(file);
    assert.match(html, /<link\b[^>]*href="\/ui\.css(?:\?[^"]*)?"/, file);
    if (staticContent.has(file)) continue;
    const scripts = [...html.matchAll(/<script\b[^>]*src="([^"?]+)(?:\?[^"]*)?"/g)].map(match => match[1]);
    assert.equal(scripts.filter(src => src === '/ui.js').length, 1, file);
    const controls = scripts.filter(src => src !== '/locale.js' && !src.startsWith('/locales/'));
    assert.equal(controls[0], '/ui.js', file);
  }
});

test('native selects in application templates and generated controls use the shared styling', () => {
  for (const file of templates('', ['.html', '.js'])) {
    if (file.startsWith('developers/dashboard-components/example/')) continue;
    for (const match of read(file).matchAll(/<select(?=\s|>)[^>]*>/gi)) {
      assert.match(match[0], /\btb-select\b/, file + ' — ' + match[0]);
    }
  }
});

test('shared controls preserve provider-only sign-in and setup consent', async () => {
  async function boot(file, replies) {
    const elements = new Map();
    const requests = [];
    const document = {
      cookie: '', documentElement:{}, readyState:'loading', addEventListener() {},
      querySelector(id) {
        if (!elements.has(id)) elements.set(id, { classList: { toggle() {} }, hidden: true });
        return elements.get(id);
      }
    };
    const context = vm.createContext({
      document, location: { replace() {} }, navigator: {language:'en'}, addEventListener() {},
      async fetch(url, options) {
        requests.push([url, options.method, ...(options.body ? [JSON.parse(options.body)] : [])]);
        assert.ok(Object.hasOwn(replies, url), url);
        return { ok: true, status: 200, json: async () => replies[url] };
      }
    });
    context.window = context;
    vm.runInContext(read('locale.js'), context);
    vm.runInContext(read('locales/analytics.js'), context);
    vm.runInContext(read('ui.js'), context);
    vm.runInContext(read(file), context);
    await new Promise(resolve => setImmediate(resolve));
    return { get: id => document.querySelector(id), requests };
  }
  const login = await boot('auth/login/login.js', {
    '/api/setup/status': { completed: true },
    '/api/auth/options': { password_login_enabled: false, providers: [{ name: 'SSO', login_url: '/auth/test' }] }
  });
  assert.equal(login.get('#form').hidden, true);
  assert.equal(login.get('#providers').hidden, false);
  assert.match(login.get('#providers').innerHTML, /tb-btn-secondary[^>]*href="\/auth\/test"/);
  assert.match(login.get('#providers').innerHTML, /Sign in with SSO/);

  const setup = await boot('setup/setup.js', { '/api/setup/status': { completed: false }, '/api/setup': {} });
  assert.equal(setup.get('#accept_storage_risk').required, true);
  await setup.get('#form').onsubmit({ preventDefault() {} });
  assert.equal(setup.get('#error').hidden, false);
  assert.deepEqual(setup.requests, [['/api/setup/status', 'GET']]);
  setup.get('#accept_storage_risk').checked = true;
  await setup.get('#form').onsubmit({preventDefault() {}});
  assert.equal(setup.requests[1][2].language, 'en');
});
