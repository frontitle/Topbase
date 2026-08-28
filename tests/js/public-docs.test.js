const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');

const publicFiles = [
  'docs/README.md',
  'docs/architecture.md',
  'docs/assets/topbase-dashboard-demo.jpg',
  'docs/configuration.md',
  'docs/database-drivers.md',
  'docs/deployment.md',
  'docs/frontend-components.md',
  'docs/getting-started.md',
  'docs/information-architecture.md',
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

test('public documentation has no internal research references or broken local links', () => {
  const markdownFiles = ['README.md', 'CONTRIBUTING.md', 'SECURITY.md', ...publicFiles.filter(file => file.endsWith('.md'))];
  for (const file of markdownFiles) {
    const source = fs.readFileSync(file, 'utf8');
    assert.doesNotMatch(source, /metabase|metabase-master|parity-matrix|\/Users\/wensky/i, file);
    for (const match of source.matchAll(/!?\[[^\]]*\]\(([^)]+)\)/g)) {
      const target = match[1].split('#')[0];
      if (!target || /^(https?:|mailto:)/.test(target)) continue;
      assert.ok(fs.existsSync(path.resolve(path.dirname(file), target)), `${file}: missing ${target}`);
    }
  }
});
