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
const dataSaveHTML = read('internal/platform/httpapi/web/data/index.html');
const warehouseHTML = read('internal/platform/httpapi/web/warehouse/index.html');
const warehouseJS = read('internal/platform/httpapi/web/warehouse/warehouse.js');
const queryEditor = read('internal/platform/httpapi/web/components/query-editor/query-editor.js');
const code = read('internal/platform/httpapi/web/components/code/code.js');
const docs = read('docs/frontend-components.md');
const peopleHTML = read('internal/platform/httpapi/web/admin/people/index.html');
const dataModelHTML = read('internal/platform/httpapi/web/admin/datamodel/index.html');
const dataModelJS = read('internal/platform/httpapi/web/admin/datamodel/datamodel.js');
const dataModelCSS = read('internal/platform/httpapi/web/admin/datamodel/datamodel.css');

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

test('analysis names are edited in place instead of through a rename action', () => {
  assert.match(questionHTML, /id="title" type="button" title="点击修改名称"/);
  assert.match(questionHTML, /id="heading" class="editable-analysis-name"/);
  assert.doesNotMatch(questionHTML, /id="rename"/);
  assert.match(questionJS, /function editName\(\)/);
  assert.match(questionJS, /on\('title', editName\)/);
  assert.match(questionJS, /on\('heading', editName\)/);
  assert.doesNotMatch(questionJS, /promptDialog\(\{ kicker: '分析设置', title: '重命名分析'/);
});

test('analysis detail links directly into the materialization flow', () => {
  assert.match(questionHTML, /id="materialize" href="\/warehouse\/"/);
  assert.match(questionJS, /\/warehouse\/\?question=' \+ encodeURIComponent\(question\.id\) \+ '#create-materialization'/);
});

test('warehouse presents materializations as a unified list and creates them in a shared dialog', () => {
  assert.match(warehouseHTML, /id="materializations"/);
  assert.match(warehouseHTML, /id="create-materialization" type="button"/);
  assert.doesNotMatch(warehouseHTML, /id="schedules"|id="tables"/);
  assert.match(warehouseJS, /function createMaterialization\(\)/);
  assert.match(warehouseJS, /formDialog\(\{/);
  assert.match(warehouseJS, /api\('\/api\/schedules\/'.*'\/run','POST'/);
});

test('saving an analysis selects its destination group and follows metadata editing', () => {
  assert.ok(dataSaveHTML.indexOf('id="edit-meta"') < dataSaveHTML.indexOf('id="save-question"'));
  assert.match(dataJS, /api\('\/api\/collections'\)/);
  assert.match(dataJS, /name:'collection_id',label:'保存到分组',type:'select'/);
  assert.match(dataJS, /collection_id:values\.collection_id==='__personal__'\?'':values\.collection_id/);
});

test('field picker closes when clicking outside or pressing Escape', () => {
  assert.match(dataHTML, /<details class="field-details">/);
  assert.match(dataJS, /details\.open&&!details\.contains\(event\.target\)\)details\.open=false/);
  assert.match(dataJS, /event\.key!==['"]Escape['"]/);
  assert.match(dataJS, /details\.querySelector\(['"]summary['"]\)\?\.focus\(\)/);
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

test('enterprise account onboarding stays in one layer and is provider neutral', () => {
  assert.match(peopleHTML, /企业账号接入/);
  assert.match(peopleHTML, /id="provider-create"/);
  assert.match(peopleHTML, /\/api\/identity\/providers\/'\+encodeURIComponent\(id\)\+'\/sync/);
  assert.doesNotMatch(peopleHTML, /id="sync-feishu"|>组织绑定</);
  const addHandler = peopleHTML.match(/\$\('#add-provider'\)\.onclick=([^;]+);/);
  assert.ok(addHandler);
  assert.doesNotMatch(addHandler[0], /formDialog|showModal/);
});

test('data model field editor supports an in-page fullscreen workspace', () => {
  assert.match(dataModelHTML, /id="toggle-field-fullscreen"/);
  assert.match(dataModelJS, /function setFieldFullscreen\(enabled\)/);
  assert.match(dataModelJS, /event\.key==='Escape'/);
  assert.match(dataModelCSS, /\.schema-fields\.is-fullscreen\{position:fixed;inset:0/);
  assert.match(dataModelCSS, /\.schema-fields\.is-fullscreen \.field-scroll/);
});
