const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');

const root = path.resolve(__dirname, '../..');
const html = fs.readFileSync(path.join(root, 'internal/platform/httpapi/web/admin/settings/index.html'), 'utf8');
const script = fs.readFileSync(path.join(root, 'internal/platform/httpapi/web/admin/settings/settings.js'), 'utf8');

test('admin settings exposes governed developer mode controls', () => {
	assert.match(html, /id="public_scheme"/);
	assert.match(html, /id="custom_domain"/);
	assert.match(html, /id="public_port"/);
	assert.match(html, /id="public-address-preview"/);
  assert.match(html, /id="developer-enabled"/);
  assert.match(html, /id="developer-public-url"/);
  assert.match(html, /id="developer-max-rows"/);
  assert.match(html, /id="developer-key-ttl"/);
  assert.match(html, /id="developer-personal-keys"/);
  assert.match(html, /id="developer-analysis-write"/);
  assert.match(html, /id="developer-key-list"/);
});

test('developer settings use real APIs and custom confirmations', () => {
  assert.match(script, /api\('\/api\/admin\/developer-settings'/);
  assert.match(script, /api\('\/api\/admin\/api-keys'/);
  assert.match(script, /confirmDialog\(/);
  assert.doesNotMatch(script, /\b(?:alert|confirm|prompt)\s*\(/);
});
