import {cp, mkdir, readFile, readdir, rm, writeFile} from 'node:fs/promises';
import {dirname, extname, join, relative, resolve} from 'node:path';
import {fileURLToPath} from 'node:url';
import {marked} from 'marked';

const here = dirname(fileURLToPath(import.meta.url));
const website = dirname(here);
const repo = dirname(website);
const source = join(website, 'src');
const output = join(website, 'dist');
const docsRoot = join(repo, 'docs');
const version = (await readFile(join(repo, 'VERSION'), 'utf8')).trim();

await rm(output, {recursive: true, force: true});
await mkdir(output, {recursive: true});
await cp(source, output, {recursive: true});
await mkdir(join(output, 'assets'), {recursive: true});
for (const asset of ['topbase-logo.png', 'topbase-dashboard-demo.jpg']) {
  await cp(join(docsRoot, 'assets', asset), join(output, 'assets', asset));
}
const indexPath = join(output, 'index.html');
const chinese = join(output, 'zh');
await mkdir(chinese);
await cp(join(output, 'assets'), join(chinese, 'assets'), {recursive: true});
const zhHome = (await readFile(indexPath, 'utf8')).replaceAll('__VERSION__', version);
await writeFile(join(chinese, 'index.html'), zhHome.replace('<nav aria-label', '<a href="../" lang="en">English</a><nav aria-label'));
await writeFile(indexPath, (await readFile(join(source, 'home-en.html'), 'utf8')).replaceAll('__VERSION__', version));
await rm(join(output, 'home-en.html'));

const docFiles = [];
async function collect(directory) {
  for (const entry of await readdir(directory, {withFileTypes: true})) {
    if (entry.name === 'assets') continue;
    const path = join(directory, entry.name);
    if (entry.isDirectory()) await collect(path);
    else if (extname(entry.name) === '.md') docFiles.push(path);
  }
}
await collect(docsRoot);

const labels = {
  'README.md': '文档首页', 'getting-started.md': '快速开始', 'configuration.md': '配置参考',
  'deployment.md': '部署与备份', 'upgrading.md': '升级与备份', 'database-drivers.md': '数据库驱动',
  'ai-integrations.md': 'AI、MCP 与 CLI', 'performance.md': '性能与容量', 'architecture.md': '工程架构',
  'information-architecture.md': '信息架构', 'frontend-components.md': '前端组件'
};
const preferred = ['README.md','getting-started.md','configuration.md','deployment.md','upgrading.md','database-drivers.md','ai-integrations.md','performance.md','architecture.md','information-architecture.md','frontend-components.md'];
const ordered = [...docFiles].sort((a, b) => {
  const ar = relative(docsRoot, a); const br = relative(docsRoot, b);
  const ai = preferred.indexOf(ar); const bi = preferred.indexOf(br);
  return (ai < 0 ? 999 : ai) - (bi < 0 ? 999 : bi) || ar.localeCompare(br, 'zh-CN');
});

const routeFor = (path) => {
  const rel = relative(docsRoot, path).replace(/\.md$/, '.html');
  return rel === 'README.html' ? 'index.html' : rel;
};
const nav = ordered.map((path) => {
  const rel = relative(docsRoot, path);
  const label = labels[rel] || rel.replace(/\.md$/, '').replaceAll('-', ' ');
  return `<a data-route="${routeFor(path)}" href="__ROOT__${routeFor(path)}">${label}</a>`;
}).join('');

marked.use({gfm: true, breaks: false});
for (const file of ordered) {
  const route = routeFor(file);
  const depth = route.split('/').length - 1;
  const root = '../'.repeat(depth);
  const markdown = await readFile(file, 'utf8');
  let content = await marked.parse(markdown);
  content = content.replace(/href="([^"#]+)\.md(#[^"]*)?"/g, (_match, target, hash = '') => {
    if (/^[a-z]+:/i.test(target)) return _match;
    if (relative(docsRoot, resolve(dirname(file), target)).startsWith('..')) {
      return `href="https://github.com/frontitle/Topbase/blob/main/${relative(repo, resolve(dirname(file), target))}.md${hash}"`;
    }
    const htmlTarget = target.endsWith('README') ? `${target.slice(0, -6)}index` : target;
    return `href="${htmlTarget}.html${hash}"`;
  });
  const title = (markdown.match(/^#\s+(.+)$/m)?.[1] || labels[relative(docsRoot, file)] || 'Topbase 文档').trim();
  const page = `<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta name="description" content="Topbase 在线文档：${title}"><title>${title}｜Topbase 文档</title><link rel="icon" href="${root}../assets/topbase-logo.png"><link rel="stylesheet" href="${root}../assets/styles.css"><link rel="stylesheet" href="${root}../assets/docs.css"></head><body class="docs-body"><header class="docs-header"><a class="brand" href="${root}../"><img src="${root}../assets/topbase-logo.png" alt=""><span>Topbase</span><b>Docs</b></a><button type="button" aria-label="打开文档导航">目录</button><nav><a href="${root}../">官网</a><a href="https://github.com/frontitle/Topbase">GitHub</a></nav></header><div class="docs-layout"><aside>${nav.replaceAll('__ROOT__', root)}</aside><main class="docs-content">${content}</main><div class="docs-toc"><strong>本页目录</strong><nav id="toc"></nav></div></div><script src="${root}../assets/docs.js"></script></body></html>`;
  const destination = join(chinese, 'docs', route);
  await mkdir(dirname(destination), {recursive: true});
  await writeFile(destination, page.replace('<button type="button"', `<a lang="en" href="${root}../../docs/${route}">English</a><button type="button"`));
  const english = await readFile(join(website, 'en', route.replace(/\.html$/, '.md')), 'utf8');
  const heading = english.split('\n')[0].replace(/^# /, '');
  const links = ordered.map(path => {
    const href = routeFor(path);
    const name = href.replace(/\.html$/, '').replaceAll('-', ' ').replace('index', 'Documentation').replace('releases/', 'Release ');
    return `<a data-route="${href}" href="${root}${href}">${name}</a>`;
  }).join('\n');
  const englishPath = join(output, 'docs', route);
  await mkdir(dirname(englishPath), {recursive: true});
  await writeFile(englishPath, `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>${heading} | Topbase Docs</title>
<link rel="stylesheet" href="${root}../assets/styles.css"><link rel="stylesheet" href="${root}../assets/docs.css"></head>
<body class="docs-body"><header class="docs-header"><a class="brand" href="${root}../"><img src="${root}../assets/topbase-logo.png" alt="">Topbase Docs</a><a lang="zh-CN" href="${root}../zh/docs/${route}">简体中文</a><button type="button">Contents</button></header>
<div class="docs-layout"><aside>${links}</aside><main class="docs-content">${await marked.parse(english)}</main><div class="docs-toc">On this page<nav id="toc"></nav></div></div><script src="${root}../assets/docs.js"></script></body></html>`);
}

console.log(`Built Topbase website v${version}: ${ordered.length} documentation pages`);
