const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');

function read(file) {
  return fs.readFileSync(path.resolve(file), 'utf8');
}

const dataHTML = read('internal/platform/httpapi/web/data/index.html');
const dataJS = read('internal/platform/httpapi/web/data/data.js');
const questionHTML = read('internal/platform/httpapi/web/questions/view/index.html');
const questionJS = read('internal/platform/httpapi/web/question-view.js');
const queryEditor = read('internal/platform/httpapi/web/components/query-editor/query-editor.js');
const code = read('internal/platform/httpapi/web/components/code/code.js');
const docs = read('docs/frontend-components.md');

test('query editor is reusable and keeps visual and SQL execution separate', () => {
  assert.match(queryEditor, /global\.TopbaseQueryEditor = \{ mount \}/);
  assert.match(dataHTML, /data-query-panel="visual"/);
  assert.match(dataHTML, /data-query-panel="sql"/);
  assert.match(dataJS, /query_type:'native'/);
  assert.match(dataJS, /query_type:'queryir'/);
  assert.match(dataJS, /database_id:currentDB\.id,sql/);
});

test('SQL surfaces use the shared code component with copy feedback', () => {
  assert.match(code, /global\.TopbaseCode/);
  assert.match(code, /复制代码/);
  assert.match(code, /已复制/);
  assert.match(dataHTML, /components\/code\/code\.js/);
  assert.match(questionHTML, /components\/code\/code\.js/);
  assert.match(dataJS, /queryEditor\.setGeneratedSQL/);
  assert.match(questionJS, /TopbaseCode\.setCode\('#query-json'/);
  assert.doesNotMatch(dataHTML, /id="generated-sql"|查看本次执行的 SQL|class="builder-head"/);
  assert.doesNotMatch(questionHTML, /<pre id="query-json"/);
});

test('developer documentation registers every shared functional component', () => {
  for (const name of ['应用外壳', 'UI 基础设施', '查询编辑器', '代码展示与编辑', '筛选构建器', '数据表格', '可视化渲染器']) {
    assert.match(docs, new RegExp(name));
  }
});

test('application shell uses the vendored icon system instead of text glyphs', () => {
  const shell = read('internal/platform/httpapi/web/shell.js');
  const icons = read('internal/platform/httpapi/web/vendor/lucide-nav.svg');
  const shellCSS = read('internal/platform/httpapi/web/shell.css');
  assert.match(shell, /lucide-nav\.svg#/);
  assert.match(shell, /shell\.css/);
  assert.match(shellCSS, /--tb-admin-sidebar:\s*190px/);
  const iconNames = Array.from(shell.matchAll(/icon:\s*'([^']+)'/g), match => match[1]);
  for (const name of [...iconNames, 'panel-left-close', 'panel-left-open', 'search']) {
    assert.match(icons, new RegExp(`symbol id="${name}"`), `missing icon ${name}`);
  }
  assert.doesNotMatch(shell, /icon:\s*'[▦◇☷▣▤⌫☺⚿◴⚙]'/);
});
