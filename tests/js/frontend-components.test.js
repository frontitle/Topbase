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
  assert.match(questionHTML, /<h1 id="heading">/);
  assert.match(questionJS, /heading\.classList\.add\('editable-analysis-name'\)/);
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

test('page chrome reserves the context bar for object workflows', () => {
  const uiCSS = read('internal/platform/httpapi/web/ui.css');
  const contextPages = [
    'internal/platform/httpapi/web/questions/new/index.html',
    'internal/platform/httpapi/web/questions/view/index.html',
    'internal/platform/httpapi/web/collections/view/index.html',
    'internal/platform/httpapi/web/dashboard/view/index.html',
  ];
  const topLevelPages = [
    'internal/platform/httpapi/web/index.html',
    'internal/platform/httpapi/web/questions/index.html',
    'internal/platform/httpapi/web/dashboard/index.html',
    'internal/platform/httpapi/web/data/index.html',
    'internal/platform/httpapi/web/warehouse/index.html',
    'internal/platform/httpapi/web/trash/index.html',
    'internal/platform/httpapi/web/account/index.html',
    'internal/platform/httpapi/web/admin/index.html',
    'internal/platform/httpapi/web/admin/settings/index.html',
  ];
  for (const file of contextPages) assert.match(read(file), /class="context-bar(?:\s|")/, file);
  for (const file of topLevelPages) assert.doesNotMatch(read(file), /class="(?:topbar|context-bar)/, file);
  assert.match(uiCSS, /\.context-bar\s*\{/);
  assert.doesNotMatch(uiCSS, /\.topbar(?:\s|\{|>|\.)/);
  assert.doesNotMatch(dataJS, /section-title|\$\('#crumb'\)/);
});

test('account access separates organization sync from personal login in one layer', () => {
  assert.match(peopleHTML, /企业组织接入/);
  assert.match(peopleHTML, /个人第三方登录/);
  assert.match(peopleHTML, /data-provider-create="org"/);
  assert.match(peopleHTML, /data-provider-create="login"/);
  assert.match(peopleHTML, /data-login-method="password"/);
  assert.match(peopleHTML, /至少保留一种登录方式/);
  assert.match(peopleHTML, /\/api\/identity\/providers\/validate/);
  assert.match(peopleHTML, /'DELETE'/);
  assert.match(peopleHTML, /\/api\/identity\/providers\/'\+encodeURIComponent\(id\)\+'\/sync/);
  assert.doesNotMatch(peopleHTML, /data-provider-save|id="save-auth-settings"|id="sync-feishu"|>组织绑定</);
  const addHandler = peopleHTML.match(/\$\$\('\[data-add-provider\]'\)\.forEach\(button=>button\.onclick=([^;]+);/);
  assert.ok(addHandler);
  assert.doesNotMatch(addHandler[0], /formDialog|showModal/);
});

test('compact action controls use shared geometric icons instead of font glyphs', () => {
  const adminHTML = read('internal/platform/httpapi/web/admin/index.html');
  const uiCSS = read('internal/platform/httpapi/web/ui.css');
  const gridJS = read('internal/platform/httpapi/web/grid.js');
  assert.match(peopleHTML, /tb-icon-trigger tb-more-trigger/);
  assert.doesNotMatch(peopleHTML, />•••<\/summary>/);
  assert.match(adminHTML, /tb-button-with-caret/);
  assert.doesNotMatch(adminHTML, /<span>⌄<\/span>/);
  assert.match(uiCSS, /\.tb-more-trigger::before/);
  assert.match(uiCSS, /\.tb-caret\{/);
  assert.match(uiCSS, /\.primary\.tb-button-with-caret \.tb-caret/);
  assert.match(gridJS, /class="tb-caret"/);
  assert.doesNotMatch(gridJS, />⌄<\/span>/);
});

test('admin data page exposes Excel upload from a top-level local warehouse tab', () => {
  const adminHTML = read('internal/platform/httpapi/web/admin/index.html');
  const adminJS = read('internal/platform/httpapi/web/admin/admin.js');
  assert.ok(adminHTML.indexOf('class="tb-tabs data-admin-tabs"') < adminHTML.indexOf('id="remote-data"'));
  assert.match(adminHTML, /data-data-tab="local">本地数仓/);
  assert.match(adminHTML, /id="local-warehouse" hidden/);
  assert.match(adminHTML, /id="upload-excel" type="button">上传 Excel/);
  assert.match(adminJS, /panelSelector:'#remote-data,#local-warehouse'/);
  assert.match(adminJS, /fetch\('\/api\/warehouse\/uploads'/);
});

test('data model field editor supports an in-page fullscreen workspace', () => {
  assert.match(dataModelHTML, /id="toggle-field-fullscreen"/);
  assert.match(dataModelJS, /function setFieldFullscreen\(enabled\)/);
  assert.match(dataModelJS, /event\.key==='Escape'/);
  assert.match(dataModelCSS, /\.schema-fields\.is-fullscreen\{position:fixed;inset:0/);
  assert.match(dataModelCSS, /\.schema-fields\.is-fullscreen \.field-scroll/);
});
