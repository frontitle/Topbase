const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');

test('admin monitor uses the shared HTML escaping helper for dynamic values', () => {
  const page = fs.readFileSync(path.resolve('internal/platform/httpapi/web/admin/monitor/index.html'), 'utf8');
  const script = fs.readFileSync(path.resolve('internal/platform/httpapi/web/admin/monitor/monitor.js'), 'utf8');
  assert.match(page, /src="\/admin\/monitor\/monitor\.js(?:\?[^\"]*)?"/);
  assert.match(script, /esc\((?:item|x)\.schedule_id\)/);
  assert.match(script, /esc\((?:item|x)\.error\s*\|\|\s*(?:item|x)\.message\s*\|\|\s*'—'\)/);
  assert.match(script, /runtime\.heap_alloc_bytes/);
  assert.match(script, /runtime\.memory_limit_bytes/);
  assert.doesNotMatch(script, /const esc=/);
});
