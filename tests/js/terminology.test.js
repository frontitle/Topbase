const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');

function sourceFiles(root) {
  return fs.readdirSync(root, { withFileTypes: true }).flatMap(entry => {
    const full = path.join(root, entry.name);
    if (entry.isDirectory()) return sourceFiles(full);
    return /\.(?:go|js|html|md)$/.test(entry.name) && entry.name !== 'echarts.min.js' ? [full] : [];
  });
}

test('product source uses the unified analysis and data-group terminology', () => {
  const roots = [path.resolve('internal'), path.resolve('docs')];
  const files = roots.flatMap(sourceFiles).concat(path.resolve('README.md'));
  const legacyTerms = new RegExp(['\\u95ee\\u6570', '\\u9879\\u76ee\\u7a7a\\u95f4', '\\u5408\\u96c6'].join('|'), 'g');
  const failures = files.flatMap(file => {
    const source = fs.readFileSync(file, 'utf8');
    return [...source.matchAll(legacyTerms)].map(match => `${path.relative(process.cwd(), file)}:${source.slice(0, match.index).split('\n').length}`);
  });
  assert.deepEqual(failures, [], `Found legacy product terminology:\n${failures.join('\n')}`);
});
