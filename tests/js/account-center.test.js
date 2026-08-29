const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');

const root = path.resolve(__dirname, '../..');
const html = fs.readFileSync(path.join(root, 'internal/platform/httpapi/web/account/index.html'), 'utf8');
const script = fs.readFileSync(path.join(root, 'internal/platform/httpapi/web/account/account.js'), 'utf8');
const shell = fs.readFileSync(path.join(root, 'internal/platform/httpapi/web/shell.js'), 'utf8');

test('personal center exposes profile, security and account binding tabs', () => {
  assert.match(html, /data-tab="profile"/);
  assert.match(html, /data-tab="security"/);
  assert.match(html, /data-tab="bindings"/);
  assert.match(html, /id="avatar-file"/);
  assert.match(html, /id="password-form"/);
  assert.match(html, /data-tab="api"/);
  assert.match(html, /id="developer-notice"/);
});

test('personal center uses real profile APIs and custom interaction components', () => {
  assert.match(script, /api\('\/api\/user\/profile'/);
  assert.match(script, /api\('\/api\/user\/password', 'PUT'/);
  assert.match(script, /\/api\/user\/external-identities\//);
  assert.match(script, /confirmDialog\(/);
  assert.doesNotMatch(script, /\b(?:alert|confirm|prompt)\s*\(/);
  assert.match(script, /canvas\.toDataURL\('image\/jpeg'/);
  assert.match(script, /api\('\/api\/developer\/status'/);
  assert.match(script, /expires_at/);
});

test('global account menu links to the personal center and signs out through the session API', () => {
  assert.match(shell, /class="sidebar-profile"/);
  assert.match(shell, /href="\/account\/">个人中心/);
  assert.match(shell, /data-logout/);
  assert.match(shell, /api\('\/api\/session', 'DELETE'\)/);
  assert.match(shell, /user\.avatar_url/);
});
