import {cp, mkdir, readFile, rm, writeFile} from 'node:fs/promises';
import {dirname, join} from 'node:path';
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

// Only paired product guides are published in the online manual.
const slugs = ["getting-started","excel","database-drivers","browse","model","analysis","sql","charts","dashboards","sharing","warehouse","notifications","permissions","ai-integrations"];
for (const locale of ['en', 'zh']) {
  const english = locale === 'en';
  const folder = join(english ? output : chinese, 'docs');
  await mkdir(folder, {recursive: true});
  const chapters = [];
  for (const slug of slugs) {
    const markdown = await readFile(join(website, 'guides', locale, slug + '.md'), 'utf8');
    chapters.push({slug, title: markdown.split('\n')[0].slice(2), markdown});
  }
  const title = english ? 'Topbase user guide' : 'Topbase 使用手册';
  const overview = english
    ? 'Follow an orders table from import to a shared sales dashboard, or choose the feature you need below.'
    : '跟随订单表示例，从接入数据一路做到分享销售仪表盘，也可以直接选择要使用的功能。';
  const index = '# ' + title + '\n\n' + overview + '\n\n' + chapters.map(c =>
    '### [' + c.title + '](' + c.slug + '.html)\n\n' + c.markdown.split('\n\n')[1]
  ).join('\n\n');
  chapters.unshift({slug: 'index', title, markdown: index});
  const nav = chapters.map(c => `<a data-route="${c.slug}.html" href="${c.slug}.html">${c.title}</a>`).join('\n');
  for (const chapter of chapters) {
    const route = chapter.slug + '.html';
    const alternate = (english ? '../zh/docs/' : '../../docs/') + route;
    await writeFile(join(folder, route), `<!doctype html>
<html lang="${english ? 'en' : 'zh-CN'}"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>${chapter.title} | Topbase Docs</title><link rel="stylesheet" href="../assets/styles.css"><link rel="stylesheet" href="../assets/docs.css"></head>
<body class="docs-body"><header class="docs-header"><a class="brand" href="../"><img src="../assets/topbase-logo.png" alt="">Topbase Docs</a>
<a href="${alternate}">${english ? '简体中文' : 'English'}</a><button type="button">${english ? 'Contents' : '目录'}</button></header>
<div class="docs-layout"><aside>${nav}</aside><main class="docs-content">${await marked.parse(chapter.markdown)}</main>
<div class="docs-toc">${english ? 'On this page' : '本页目录'}<nav id="toc"></nav></div></div><script src="../assets/docs.js"></script></body></html>`);
  }
}
