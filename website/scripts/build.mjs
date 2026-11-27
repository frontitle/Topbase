import {access, cp, mkdir, readdir, readFile, rm, writeFile} from 'node:fs/promises';
import {createHash} from 'node:crypto';
import {basename, dirname, join} from 'node:path';
import {fileURLToPath} from 'node:url';
import {marked} from 'marked';
import {imageSize} from 'image-size';
import {siteChrome} from './site-chrome.mjs';

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
await cp(join(output, 'assets'), join(chinese, 'assets'), {
  recursive: true,
  filter: path => !['guide-images', 'samples'].includes(basename(path)),
});
const fingerprint = bytes => createHash('sha256').update(bytes).digest('hex').slice(0, 12);
const revisions = new Map();
for (const name of ['styles.css', 'home-layout.css', 'site.js', 'docs.css', 'docs.js']) {
  revisions.set(name, fingerprint(await readFile(join(source, 'assets', name))));
}
function assetUrl(name, base = 'assets/') {
  return base + name + '?v=' + revisions.get(name);
}
for (const english of [true, false]) {
  const template = await readFile(join(source, english ? 'home-en.html' : 'index.html'), 'utf8');
  let home = template.replaceAll('__VERSION__', version);
  for (const [part, html] of Object.entries(siteChrome({english, version}))) {
    home = home.replace(`<!-- site ${part} -->`, html);
  }
  for (const asset of ['styles.css', 'home-layout.css', 'site.js']) {
    home = home.replace(`"assets/${asset}"`, `"${assetUrl(asset)}"`);
  }
  await writeFile(english ? indexPath : join(chinese, 'index.html'), home);
}
await rm(join(output, 'home-en.html'));

// Only paired product guides are published in the online manual.
const sections = [
  {en: 'Getting started', zh: '快速入门', slugs: ['getting-started', 'account']},
  {en: 'Connect and explore data', zh: '接入与浏览数据', slugs: ['database-drivers', 'excel', 'browse', 'model']},
  {en: 'Analyze and visualize', zh: '查询与可视化', slugs: ['analysis', 'sql', 'charts']},
  {en: 'Organize and share', zh: '组织与分享', slugs: ['groups', 'dashboards', 'sharing']},
  {en: 'Scheduled work', zh: '定时任务', slugs: ['warehouse', 'notifications']},
  {en: 'Workspace administration', zh: '工作区管理', slugs: ['workspace', 'permissions', 'ai-integrations']},
];
const slugs = sections.flatMap(section => section.slugs);
const illustrations = new Map();
// Capture guides at 2× device scale. Reserve the original layout size before image load.
const screenshots = new Map();
const imageDir = join(source, 'assets', 'guide-images');
for (const name of await readdir(imageDir)) {
  if (!name.endsWith('.jpg')) continue;
  const bytes = await readFile(join(imageDir, name));
  const {width, height} = imageSize(bytes);
  screenshots.set(name, {revision: fingerprint(bytes), width: Math.round(width / 2), height: Math.round(height / 2)});
}
for (const locale of ['en', 'zh']) {
  const english = locale === 'en';
  const folder = join(english ? output : chinese, 'docs');
  await mkdir(folder, {recursive: true});
  const chapters = [];
  for (const slug of slugs) {
    const markdown = await readFile(join(website, 'guides', locale, slug + '.md'), 'utf8');
    const images = [...markdown.matchAll(/<img src="\.\.\/assets\/guide-images\/([^"]+)"/g)].map(match => match[1]);
    const names = images.map(filename => {
      if (!filename.endsWith(`-${locale}.jpg`)) throw new Error(`Wrong screenshot language in ${locale}/${slug}: ${filename}`);
      return filename.slice(0, -(`-${locale}.jpg`.length));
    });
    if (english) illustrations.set(slug, names.join('\n'));
    else if (names.join('\n') !== illustrations.get(slug)) throw new Error(`Screenshot coverage differs for ${slug}`);
    const samples = [...markdown.matchAll(/\]\(\.\.\/assets\/samples\/([^()]+)\)/g)].map(match => match[1]);
    await Promise.all([
      ...images.map(name => access(join(output, 'assets', 'guide-images', name))),
      ...samples.map(name => access(join(output, 'assets', 'samples', name))),
    ]);
    chapters.push({slug, title: markdown.split('\n')[0].slice(2), markdown});
  }
  const title = english ? 'Topbase user guide' : 'Topbase 使用手册';
  const overview = english
    ? 'New to Topbase? [Build your first sales dashboard](getting-started.html) with six sample orders, then explore a feature below.'
    : '第一次使用 Topbase？用六笔示例订单[创建第一张销售仪表盘](getting-started.html)，或从下方选择要了解的功能。';
  const bySlug = new Map(chapters.map(chapter => [chapter.slug, chapter]));
  const index = '# ' + title + '\n\n' + overview + '\n\n' + sections.map(section =>
    '## ' + section[locale] + '\n\n' + section.slugs.map(slug => {
      const chapter = bySlug.get(slug);
      return '### [' + chapter.title + '](' + slug + '.html)\n\n' + chapter.markdown.split('\n\n')[1];
    }).join('\n\n')
  ).join('\n\n');
  chapters.unshift({slug: 'index', title, markdown: index});
  const navLink = c => `<a data-route="${c.slug}.html" href="${c.slug}.html">${c.title}</a>`;
  const nav = navLink(chapters[0]) + sections.map(section =>
    `<p class="docs-group">${section[locale]}</p>` + section.slugs.map(slug => navLink(bySlug.get(slug))).join('\n')
  ).join('\n');
  for (const chapter of chapters) {
    const route = chapter.slug + '.html';
    const {header, install, footer} = siteChrome({english, version, chapter: route});
    const guideLabel = english ? 'User guide' : '使用手册';
    const assetBase = english ? '../assets/' : '../../assets/';
    const content = (await marked.parse(chapter.markdown))
      .replaceAll('../assets/samples/', assetBase + 'samples/')
      .replace(/(src|href)="\.\.\/assets\/guide-images\/([^"]+)"/g, (_, attr, name) => {
        const {width, height, revision} = screenshots.get(name);
        const link = `${attr}="${assetBase}guide-images/${name}?v=${revision}"`;
        return attr === 'src' ? `${link} width="${width}" height="${height}"` : link;
      });
    await writeFile(join(folder, route), `<!doctype html>
<html lang="${english ? 'en' : 'zh-CN'}"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>${chapter.title} | Topbase Docs</title><link rel="icon" href="${assetBase}topbase-logo.png">
${['styles.css', 'home-layout.css', 'docs.css'].map(name => `<link rel="stylesheet" href="${assetUrl(name, assetBase)}">`).join('\n')}</head>
<body class="docs-body" id="top">${header}
<nav class="docs-chapter-bar" aria-label="${guideLabel}"><a href="./">${guideLabel}</a><button type="button" aria-expanded="false" aria-controls="docs-nav">${english ? 'Chapters' : '章节目录'}</button></nav>
<div class="docs-layout"><aside id="docs-nav"><nav aria-label="${guideLabel}">${nav}</nav></aside><main class="docs-content">${content}</main>
<div class="docs-toc">${english ? 'On this page' : '本页目录'}<nav id="toc"></nav></div></div>
${install}
${footer}
<script src="${assetUrl('docs.js', assetBase)}"></script><script src="${assetUrl('site.js', assetBase)}"></script></body></html>`);
  }
}
