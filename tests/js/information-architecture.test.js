const assert = require('node:assert/strict');
const fs = require('node:fs');
const test = require('node:test');

const shell = fs.readFileSync('internal/platform/httpapi/web/shell.js', 'utf8');
const questions = fs.readFileSync('internal/platform/httpapi/web/questions/index.html', 'utf8');
const questionScript = fs.readFileSync('internal/platform/httpapi/web/questions-list.js', 'utf8');
const collections = fs.readFileSync('internal/platform/httpapi/web/collections/index.html', 'utf8');
const data = fs.readFileSync('internal/platform/httpapi/web/data/index.html', 'utf8');
const warehouse = fs.readFileSync('internal/platform/httpapi/web/warehouse/index.html', 'utf8');
const docs = fs.readFileSync('docs/information-architecture.md', 'utf8');

test('sidebar merges groups into analyses and puts dashboards before data', () => {
  const menu = shell.slice(shell.indexOf('const appItems'), shell.indexOf('const adminItems'));
  assert.doesNotMatch(menu, /id:\s*'collections'/);
  for (const label of ['分析', '仪表盘', '源数据', '数据沉淀']) assert.match(menu, new RegExp(`label: '${label}'`));
  assert.ok(menu.indexOf("label: '分析'") < menu.indexOf("label: '仪表盘'"));
  assert.ok(menu.indexOf("label: '仪表盘'") < menu.indexOf("label: '源数据'"));
});

test('analysis page owns grouping and old collection list redirects there', () => {
  assert.match(questions, /data-analysis-view="items"/);
  assert.match(questions, /data-analysis-view="groups"/);
  assert.match(questions, /id="create-group"/);
  assert.match(questionScript, /api\('\/api\/collections','POST'/);
  assert.match(collections, /location\.replace\('\/questions\/\?view=groups'\)/);
});

test('source data and persisted data explain location and freshness', () => {
  assert.match(data, /实时 · 只读/);
  assert.match(data, /数据仍保存在远程数据库/);
  assert.match(warehouse, /本地保存 · 按计划更新/);
  assert.match(warehouse, /这是 Topbase 管理的数据/);
  assert.match(docs, /源数据（远程、实时、只读）/);
  assert.match(docs, /数据沉淀（本地保存、按计划更新）/);
});
