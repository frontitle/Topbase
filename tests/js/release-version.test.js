const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');

const version = fs.readFileSync('VERSION', 'utf8').trim();

test('release version is synchronized across build and public documentation', () => {
  assert.match(version, /^\d+\.\d+\.\d+$/);
  assert.match(fs.readFileSync('internal/buildinfo/info.go', 'utf8'), new RegExp(`Version\\s+= "${version.replaceAll('.', '\\.')}"`));
  assert.match(fs.readFileSync('Dockerfile', 'utf8'), new RegExp(`ARG VERSION=${version.replaceAll('.', '\\.')}`));
  assert.match(fs.readFileSync('.env.example', 'utf8'), new RegExp(`TOPBASE_VERSION=${version.replaceAll('.', '\\.')}`));
  assert.ok(fs.existsSync(`docs/releases/${version}.md`));
  assert.match(fs.readFileSync('CHANGELOG.md', 'utf8'), new RegExp(`## \\[${version.replaceAll('.', '\\.')}\\]`));
  assert.match(fs.readFileSync('README.md', 'utf8'), new RegExp(`version-${version.replaceAll('.', '\\.')}`));
});

test('every compose deployment uses the current release default', () => {
  for (const file of ['docker-compose.yml', 'docker-compose.postgres.yml', 'docker-compose.mysql.yml', 'docker-compose.rds.yml', 'docker-compose.release.yml']) {
    const source = fs.readFileSync(file, 'utf8');
    assert.match(source, new RegExp(version.replaceAll('.', '\\.')), file);
    assert.doesNotMatch(source, /0\.1\.0-alpha/, file);
  }
});

test('tag release workflow verifies VERSION and publishes multi-architecture images', () => {
  const workflow = fs.readFileSync('.github/workflows/release.yml', 'utf8');
  assert.match(workflow, /GITHUB_REF_NAME#v/);
  assert.match(workflow, /linux\/amd64,linux\/arm64/);
  assert.match(workflow, /ghcr\.io/);
  assert.match(workflow, /docs\/releases/);
});
