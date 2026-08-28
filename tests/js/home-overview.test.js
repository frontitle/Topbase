const assert = require('node:assert/strict');
const fs = require('node:fs');
const test = require('node:test');

const shell = fs.readFileSync('internal/platform/httpapi/web/shell.js', 'utf8');
const home = fs.readFileSync('internal/platform/httpapi/web/index.html', 'utf8');
const app = fs.readFileSync('internal/platform/httpapi/web/app.js', 'utf8');

test('application sidebar omits model and search entries', () => {
  const appMenu = shell.slice(shell.indexOf('const appItems'), shell.indexOf('const adminItems'));
  assert.doesNotMatch(appMenu, /browse\/models|href:\s*['"]\/search\/|label:\s*['"](?:模型|搜索)['"]/);
  assert.doesNotMatch(appMenu, /label:\s*'数据组'/);
  assert.match(appMenu, /label:\s*'源数据'/);
  assert.match(appMenu, /label:\s*'数据沉淀'/);
  assert.ok(appMenu.indexOf("label: '仪表盘'") < appMenu.indexOf("label: '源数据'"));
});

test('home is a data-driven overview with core statistics and working shortcuts', () => {
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
