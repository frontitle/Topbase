const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');

const publicFiles = [
  'docs/README.md',
  'docs/architecture.md',
  'docs/ai-integrations.md',
  'docs/assets/topbase-logo.png',
  'docs/configuration.md',
  'docs/database-drivers.md',
  'docs/deployment.md',
  'docs/frontend-components.md',
  'docs/getting-started.md',
  'docs/information-architecture.md',
  'docs/performance.md',
  'docs/releases/0.2.1.md',
  'docs/releases/0.2.2.md',
  'docs/releases/0.2.3.md',
  'docs/releases/0.2.4.md',
  'docs/releases/0.2.5.md',
  'docs/releases/0.2.6.md',
  'docs/releases/0.2.7.md',
  'docs/upgrading.md',
];

function walk(directory) {
  return fs.readdirSync(directory, { withFileTypes: true }).flatMap(entry => {
    const item = path.join(directory, entry.name);
    return entry.isDirectory() ? walk(item) : [item];
  });
}

test('docs contains only reviewed public documentation', () => {
  assert.deepEqual(walk('docs').sort(), publicFiles.slice().sort());
});

const projectMarkdown = [
  'README.md', 'README.zh-CN.md', 'CHANGELOG.md', 'CONTRIBUTING.md', 'CONTRIBUTING.zh-CN.md',
  'SECURITY.md', 'SECURITY.zh-CN.md', ...publicFiles.filter(file => file.endsWith('.md')),
];

test('published documentation does not contain local user paths', () => {
  for (const file of projectMarkdown) {
    assert.doesNotMatch(fs.readFileSync(file, 'utf8'), /\/Users\/[^/\s]+\/|[a-z]:\\Users\\[^\\\s]+\\/i, file);
  }
});

test('repository documentation links resolve in a source checkout', () => {
  for (const file of projectMarkdown) {
    const source = fs.readFileSync(file, 'utf8');
    for (const match of source.matchAll(/!?\[[^\]]*\]\(([^)]+)\)/g)) {
      const target = match[1].split('#')[0];
      if (!target || /^(https?:|mailto:)/.test(target)) continue;
      assert.ok(fs.existsSync(path.resolve(path.dirname(file), target)), `${file}: missing ${target}`);
    }
  }
});
