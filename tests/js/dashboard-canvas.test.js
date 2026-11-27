const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');

function read(file) {
  return fs.readFileSync(path.resolve(file), 'utf8');
}

const script = read('internal/platform/httpapi/web/dashboard-view.js');
const styles = read('internal/platform/httpapi/web/dashboard.css');

test('only the editor draws the switchable alignment grid', () => {
  assert.match(script, /canvas\.classList\.toggle\('canvas-grid-off', !a\.grid\)/);
  assert.match(script, /canvas\.classList\.toggle\('canvas-snap-off', !a\.snap\)/);
  assert.match(styles, /\.board-canvas:not\(\.canvas-grid-off\) \.board-grid\.editing\s*\{\s*background-image:/);
  assert.doesNotMatch(styles, /canvas-glow-off|DATA CANVAS/);
});

test('free positioning uses finer increments when snap is disabled', () => {
  assert.match(script, /const precision = appearance\(\)\.snap \? 1 : 4/g);
  assert.match(script, /Math\.round\(\(ev\.clientX - drag\.startX\) \/ cw \* precision\) \/ precision/);
  assert.match(script, /Math\.round\(\(ev\.clientY - drag\.startY\) \/ \(ROW \+ Y_GAP\) \* precision\) \/ precision/);
});

test('empty canvas offers an insert action in the editor only', () => {
  assert.match(script, /<button type="button" data-empty-insert[^>]*>插入内容<\/button>/);
  assert.match(script, /editing \? emptyCanvas : ''/);
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
