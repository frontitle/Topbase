const assert = require('node:assert/strict');
const path = require('node:path');
const test = require('node:test');

global.window = global;
require(path.resolve('internal/platform/httpapi/web/viz.js'));

function host(tab) {
  return {
    dataset: { tab },
    innerHTML: '',
    querySelectorAll() { return []; },
    querySelector() { return null; }
  };
}

test('line chart settings render without reference errors', () => {
  const element = host('data');
  const columns = ['report_date', 'order_count', 'refund_amount'];
  const rows = [['2026-08-26', 10, 2], ['2026-08-27', 12, 1]];
  const spec = {
    type: 'line',
    x: 'report_date',
    y: ['order_count', 'refund_amount'],
    series: { order_count: { display: 'line' } }
  };

  TopbaseViz.renderSettings(element, columns, rows, spec, () => {});

  assert.match(element.innerHTML, /展示类型/);
  assert.match(element.innerHTML, /Y 轴位置/);
});

test('time-series lines are ordered chronologically and use linear interpolation by default', () => {
  let option;
  const chartElement = {};
  const element = {
    innerHTML: '',
    querySelector(selector) {
      return selector === '.viz-chart' && this.innerHTML.includes('viz-chart') ? chartElement : null;
    }
  };
  const previousEcharts = global.echarts;
  const previousResizeObserver = global.ResizeObserver;
  global.echarts = {
    init() {
      return {
        setOption(value) { option = value; },
        resize() {},
        dispose() {}
      };
    }
  };
  global.ResizeObserver = class {
    observe() {}
    disconnect() {}
  };

  try {
    TopbaseViz.render(element, {
      columns: ['report_date', 'order_count'],
      rows: [
        ['2026-05-19', 19],
        ['2026-04-18', 18],
        ['2026-05-12', 12],
        ['2026-04-19', 20]
      ],
      spec: { type: 'line', x: 'report_date', y: ['order_count'] }
    });
  } finally {
    global.echarts = previousEcharts;
    global.ResizeObserver = previousResizeObserver;
  }

  assert.deepEqual(
    option.series[0].data.map((point) => new Date(point[0]).toISOString().slice(0, 10)),
    ['2026-04-18', '2026-04-19', '2026-05-12', '2026-05-19']
  );
  assert.deepEqual(option.series[0].data.map((point) => point[1]), [18, 20, 12, 19]);
  assert.equal(option.series[0].smooth, false);
});

test('saved fields missing from fresh results are removed from chart mappings', () => {
  const merged = TopbaseViz.merge(
    { type: 'line', x: 'removed_dimension', y: ['removed_metric', 'order_count'] },
    { type: 'line', x: 'report_date', y: ['order_count'] },
    ['report_date', 'order_count'],
    [['2026-08-26', 10]]
  );

  assert.equal(merged.x, 'report_date');
  assert.deepEqual(merged.y, ['order_count']);
});

test('table settings do not duplicate page-level filters', () => {
  const element = host('display');

  TopbaseViz.renderSettings(
    element,
    ['report_date', 'order_count'],
    [['2026-08-26', 10]],
    { type: 'table', columns: { order_count: { filter: '> 5' } } },
    () => {}
  );

  assert.doesNotMatch(element.innerHTML, /data-ck="filter"/);
  assert.doesNotMatch(element.innerHTML, /data-k="search"/);
  assert.match(element.innerHTML, /页面顶部的筛选器/);
});
