const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');

function read(file) {
  return fs.readFileSync(path.resolve(file), 'utf8');
}

const script = read('internal/platform/httpapi/web/dashboard-view.js');
const styles = read('internal/platform/httpapi/web/dashboard.css');

test('dashboard canvas switches drive distinct visual states', () => {
  assert.match(script, /canvas\.classList\.toggle\('canvas-grid-off', !a\.grid\)/);
  assert.match(script, /canvas\.classList\.toggle\('canvas-snap-off', !a\.snap\)/);
  assert.match(script, /canvas\.classList\.toggle\('canvas-glow-off', !a\.glow\)/);
  assert.match(styles, /canvas-grid-off \.board-grid\{background-image:none!important\}/);
  assert.match(styles, /canvas-glow-off \.board-grid\{box-shadow:none\}/);
});

test('free positioning uses finer increments when snap is disabled', () => {
  assert.match(script, /const precision = appearance\(\)\.snap \? 1 : 4/g);
  assert.match(script, /Math\.round\(\(ev\.clientX - drag\.startX\) \/ cw \* precision\) \/ precision/);
  assert.match(script, /Math\.round\(\(ev\.clientY - drag\.startY\) \/ \(ROW \+ Y_GAP\) \* precision\) \/ precision/);
});

test('empty dashboard presents an explicit composition canvas', () => {
  assert.match(script, /开始搭建你的数据画布/);
  assert.match(script, /24 列精细网格/);
  assert.match(styles, /DATA CANVAS · 24 COL/);
  assert.match(styles, /\.board-canvas\.canvas-editing/);
});

test('small number cards fit their value instead of scaling the whole visualization', () => {
  assert.match(script, /cardVizType\(card\) === 'scalar'/);
  assert.match(script, /el\.classList\.remove\('card-content-scaled'\)/);
  assert.match(script, /function fitScalarContent\(el\)/);
  assert.match(script, /--scalar-number-size/);
  assert.match(script, /minWidth\(card\).*'scalar' \? 4 : MIN_W/);
  assert.match(styles, /\.board-card\.card-content-scalar \.viz-scalar b/);
  assert.match(styles, /font-size:var\(--scalar-number-size,36px\)/);
});
