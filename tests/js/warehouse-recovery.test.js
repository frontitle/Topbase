const assert = require('node:assert/strict');
const fs = require('node:fs');
const test = require('node:test');
const vm = require('node:vm');

const source = fs.readFileSync('internal/platform/httpapi/web/warehouse/warehouse.js', 'utf8');

test('analysis deep link opens once, first failure stays visible and retry reuses the schedule', async () => {
  const nodes = new Map();
  const element = selector => {
    if (!nodes.has(selector)) nodes.set(selector, {});
    return nodes.get(selector);
  };
  let tables = [], plans = [], creations = 0, dialogs = 0, runs = 0;
  const notices = [];
  const button = {dataset: {run: 'orders'}, textContent: '重试更新'};
  const context = vm.createContext({
    $: element,
    $$: () => [button],
    esc: value => String(value).replaceAll('<', '&lt;'),
    toast: value => notices.push(value),
    URLSearchParams,
    crypto: require('node:crypto').webcrypto,
    history: {replaceState(_state, _title, url) { assert.equal(url, '/warehouse/'); }},
    location: {hash:'#create-materialization', pathname:'/warehouse/', search:'?question=q'},
    formDialog: async options => {
      dialogs++;
      assert.equal(options.fields.find(field => field.name === 'question_id').value, 'q');
      return {question_id:'q', name:'每日订单', target:'orders', strategy:'replace', frequency:'daily'};
    },
    api: async (url, method, values) => {
      switch (url) {
        case '/api/questions': return [{id:'another', name:'库存'}, {id:'q', name:'订单'}];
        case '/api/warehouse/tables': return tables;
        case '/api/schedules?include=last_run': return plans;
        case '/api/schedules':
          assert.equal(method, 'POST');
          creations++;
          plans = [{...values, id:'orders'}];
          return plans[0];
        case '/api/schedules/orders/run':
          assert.equal(method, 'POST');
          if (++runs === 1) {
            plans[0].last_status = 'failed';
            plans[0].last_error = '<timeout>';
            throw Error('timeout');
          }
          plans[0].last_status = 'succeeded';
          plans[0].last_error = '';
          tables = [{schedule_id:'orders', database_id:'db', schema:'warehouse', name:'wh_orders'}];
          return {row_count:0};
        default: throw Error('unexpected request ' + url);
      }
    }
  });
  await vm.runInContext(source, context);
  const defaults = vm.runInContext("['订单', '库存', '订单'].map(name => suggestedTarget({name}))", context);
  assert.equal(new Set(defaults).size, 3);
  for (const value of defaults) assert.match(value, /^analysis_result_[a-f0-9]{12}$/);
  assert.match(notices[0], /计划已创建，首次更新失败/);
  assert.match(element('#materializations').innerHTML, /每日订单/);
  assert.match(element('#materializations').innerHTML, /重试更新/);
  assert.match(element('#materializations').innerHTML, /&lt;timeout>/);
  assert.doesNotMatch(element('#materializations').innerHTML, /href=/);
  await button.onclick();
  assert.equal(dialogs, 1);
  assert.equal(creations, 1);
  assert.equal(runs, 2);
  assert.equal(element('#materialization-count').textContent, '1 项');
  assert.match(element('#materializations').innerHTML, /更新成功/);
  assert.match(element('#materializations').innerHTML, /已保存 0 行/);
  assert.match(element('#materializations').innerHTML, /创建分析/);
  plans[0].last_status = 'failed';
  await vm.runInContext('load()', context);
  assert.match(element('#materializations').innerHTML, /更新失败/);
  assert.match(element('#materializations').innerHTML, /创建分析/);
});
