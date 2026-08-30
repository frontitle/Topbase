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

test('default line chart maps exactly one dimension and one metric', () => {
  const spec = TopbaseViz.infer(
    ['report_date', 'order_count', 'refund_amount'],
    [['2026-08-26', 10, 2], ['2026-08-27', 12, 1]]
  );

  assert.equal(spec.type, 'line');
  assert.equal(spec.x, 'report_date');
  assert.deepEqual(spec.y, ['order_count']);
});

test('number visualization omits the field name and can compare another value', () => {
  const element = { innerHTML: '' };
  TopbaseViz.render(element, {
    columns: ['sales', 'previous_sales'],
    rows: [[120, 100]],
    spec: { type: 'scalar', y: ['sales'], comparison_field: 'previous_sales' }
  });

  assert.match(element.innerHTML, />120</);
  assert.match(element.innerHTML, /\+20/);
  assert.match(element.innerHTML, /\+20\.0%/);
  assert.doesNotMatch(element.innerHTML, /sales/);
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
  assert.equal(option.color[0], '#0B66C3');
  assert.equal(option.series[0].lineStyle.shadowBlur, 6);
  assert.equal(option.tooltip.backgroundColor, 'rgba(255,255,255,.97)');
});

test('China map loads bundled province boundaries and accepts names or adcodes', async () => {
  let option;
  let registered;
  let requestedURL;
  const chartElement = {};
  const element = {
    innerHTML: '',
    querySelector(selector) {
      return selector === '.viz-chart' && this.innerHTML.includes('viz-chart') ? chartElement : null;
    }
  };
  const previousEcharts = global.echarts;
  const previousResizeObserver = global.ResizeObserver;
  const previousFetch = global.fetch;
  global.echarts = {
    getMap() { return registered; },
    registerMap(name, geoJSON) { registered = { name, geoJSON }; },
    init() {
      return {
        showLoading() {},
        hideLoading() {},
        setOption(value) { option = value; },
        resize() {},
        dispose() {}
      };
    }
  };
  global.fetch = async (url) => {
    requestedURL = url;
    return { ok: true, json: async () => ({ type: 'FeatureCollection', features: [] }) };
  };
  global.ResizeObserver = class {
    observe() {}
    disconnect() {}
  };

  try {
    TopbaseViz.render(element, {
      columns: ['province', 'sales'],
      rows: [['广东', 120], ['110000', 80]],
      spec: { type: 'map', x: 'province', y: ['sales'], show_labels: true }
    });
    await new Promise((resolve) => setImmediate(resolve));
  } finally {
    global.echarts = previousEcharts;
    global.ResizeObserver = previousResizeObserver;
    global.fetch = previousFetch;
  }

  assert.equal(requestedURL, '/assets/maps/china-provinces.json');
  assert.equal(registered.name, 'topbase-china-provinces');
  assert.equal(option.series[0].type, 'map');
  assert.deepEqual(option.series[0].data, [
    { name: '广东省', value: 120 },
    { name: '北京市', value: 80 }
  ]);
  assert.deepEqual(option.visualMap.inRange.color, ['#E6F8F4', '#4CCFBA', '#0B66C3']);
});

test('China map keeps an administrative code field separate from its value', () => {
  const spec = TopbaseViz.merge(
    { type: 'map', x: 'adcode', y: ['adcode', 'sales'] },
    { type: 'map', x: 'adcode', y: ['adcode', 'sales'] },
    ['adcode', 'sales'],
    [[440000, 120], [110000, 80]]
  );

  assert.equal(spec.x, 'adcode');
  assert.deepEqual(spec.y, ['sales']);
});

test('third parties can register and render another map asset package', async () => {
  let option;
  let requestedURL;
  const maps = {};
  const chartElement = {};
  const element = {
    innerHTML: '',
    querySelector(selector) {
      return selector === '.viz-chart' && this.innerHTML.includes('viz-chart') ? chartElement : null;
    }
  };
  TopbaseViz.registerMapPackage({
    id: 'custom-regions',
    label: 'Custom Regions',
    url: '/assets/maps/custom-regions.json',
    nameProperty: 'display_name',
    codeProperty: 'region_code',
    labelSuffixes: [' Zone']
  });
  const previousEcharts = global.echarts;
  const previousResizeObserver = global.ResizeObserver;
  const previousFetch = global.fetch;
  global.echarts = {
    getMap(name) { return maps[name]; },
    registerMap(name, geoJSON) { maps[name] = { geoJSON }; },
    init() {
      return {
        showLoading() {},
        hideLoading() {},
        setOption(value) { option = value; },
        resize() {},
        dispose() {}
      };
    }
  };
  global.fetch = async (url) => {
    requestedURL = url;
    return {
      ok: true,
      json: async () => ({
        type: 'FeatureCollection',
        features: [{ type: 'Feature', properties: { display_name: 'North Zone', region_code: 'N' }, geometry: null }]
      })
    };
  };
  global.ResizeObserver = class {
    observe() {}
    disconnect() {}
  };

  try {
    TopbaseViz.render(element, {
      columns: ['region', 'sales'],
      rows: [['N', 42]],
      spec: { type: 'map', map_package: 'custom-regions', x: 'region', y: ['sales'] }
    });
    await new Promise((resolve) => setImmediate(resolve));
  } finally {
    global.echarts = previousEcharts;
    global.ResizeObserver = previousResizeObserver;
    global.fetch = previousFetch;
  }

  assert.equal(requestedURL, '/assets/maps/custom-regions.json');
  assert.ok(maps['topbase-map-custom-regions']);
  assert.equal(option.series[0].map, 'topbase-map-custom-regions');
  assert.deepEqual(option.series[0].data, [{ name: 'North Zone', value: 42 }]);

  const settings = host('data');
  TopbaseViz.renderSettings(settings, ['region', 'sales'], [['N', 42]], { type: 'map', map_package: 'custom-regions', x: 'region', y: ['sales'] }, () => {});
  assert.match(settings.innerHTML, /图资包/);
  assert.match(settings.innerHTML, /Custom Regions/);
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
