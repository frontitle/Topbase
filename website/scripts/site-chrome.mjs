const repository = 'https://github.com/frontitle/Topbase';

export function siteChrome({english, version, chapter}) {
  const home = chapter ? '../' : './';
  const docs = chapter ? './' : 'docs/';
  const section = id => (chapter ? home : '') + '#' + id;
  const logo = (chapter ? '../' : '') + 'assets/topbase-logo.png';
  const alternate = chapter
    ? (english ? '../zh/docs/' : '../../docs/') + chapter
    : (english ? 'zh/' : '../');
  const t = (en, zh) => english ? en : zh;
  const homeLabel = t('Topbase homepage', 'Topbase 首页');
  const navLabel = t('Main navigation', '主导航');
  const start = t('Get started', '开始使用');
  const tutorial = docs + 'getting-started.html';
  const navLinks = [
    [section('product'), t('Product', '产品')],
    [section('workflow'), t('Workflow', '工作方式')],
    [docs, t('Docs', '文档')],
    [repository, 'GitHub'],
  ].map(([href, label]) => `<a href="${href}"${chapter && href === docs ? ' aria-current="location"' : ''}>${label}</a>`).join('\n');
  const languages = [
    {lang: 'en', label: 'EN', name: 'English', current: english},
    {lang: 'zh-CN', label: '中文', name: '简体中文', current: !english},
  ].map(({lang, label, name, current}) => current
    ? `<span lang="${lang}" aria-current="page" title="${name}">${label}</span>`
    : `<a href="${alternate}" lang="${lang}" hreflang="${lang}" aria-label="${name}" title="${name}">${label}</a>`
  ).join('');
  const header = `<header class="site-header">
    <div class="header-inner shell">
      <a class="brand" href="${home}" aria-label="${homeLabel}"><img src="${logo}" alt=""><span>Topbase</span></a>
      <nav class="primary-nav" aria-label="${navLabel}">${navLinks}</nav>
      <nav class="language-switch" aria-label="${t('Language', '语言')}">${languages}</nav>
      <a class="header-start button button-small button-dark" href="${tutorial}">${start}</a>
      <details class="mobile-nav">
        <summary aria-label="${t('Navigation menu', '导航菜单')}"><span class="menu-bars" aria-hidden="true"></span></summary>
        <nav aria-label="${navLabel}">${navLinks}<a class="mobile-start" href="${tutorial}">${start}</a></nav>
      </details>
    </div>
  </header>`;
  const install = `<section class="install-band" id="install" aria-labelledby="install-title">
    <div class="install-panel shell">
      <div class="install-copy">
        <h2 id="install-title">${t('Build your first <em>dashboard.</em>', '创建第一张<em>仪表盘</em>')}</h2>
        <p>${t('Follow six sample orders from a database to a sales dashboard on your own server.', '从数据库里的六笔示例订单出发，在自己的服务器上做出一张销售仪表盘。')}</p>
        <div class="install-actions">
          <a class="button button-primary" href="${docs}installation.html">${t('Install Topbase', '安装 Topbase')}</a>
          <a class="tutorial-link" href="${tutorial}">${t('View tutorial', '查看示例教程')}<span aria-hidden="true">→</span></a>
        </div>
      </div>
      <div class="install-art" aria-hidden="true"><img src="${logo}" width="208" height="208" alt="" loading="lazy"></div>
    </div>
  </section>`;
  const groups = [
    [t('Product', '产品'), [
      [section('product'), t('Dashboard preview', '仪表盘预览')],
      [section('workflow'), t('Data workflow', '数据工作流程')],
      [section('deployment'), t('Self-hosting', '自托管部署')],
    ]],
    [t('Documentation', '使用文档'), [
      [docs, t('User guide', '使用手册')],
      [docs + 'database-drivers.html', t('Connect data', '连接数据库')],
      [docs + 'dashboards.html', t('Dashboards', '创建仪表盘')],
    ]],
    [t('Open source', '开源项目'), [
      [repository, t('GitHub repository', 'GitHub 仓库')],
      [repository + '/issues', t('Report an issue', '反馈问题')],
      [repository + '/blob/main/' + t('CONTRIBUTING.md', 'CONTRIBUTING.zh-CN.md'), t('Contributing', '参与贡献')],
    ]],
  ];
  const footer = `<footer class="site-footer" id="footer">
    <div class="shell">
      <div class="footer-main">
        <div class="footer-identity">
          <a class="footer-wordmark" href="${home}" aria-label="${homeLabel}">Topbase</a>
          <p>${t('Open-source, self-hosted analytics.', '开源、自托管的数据分析平台。')}</p>
        </div>
        <nav class="footer-nav" aria-label="${t('Footer navigation', '页脚导航')}">
          ${groups.map(([label, links]) => `<div class="footer-group"><h3>${label}</h3><ul>
            ${links.map(([href, text]) => `<li><a href="${href}">${text}</a></li>`).join('\n')}
          </ul></div>`).join('\n')}
        </nav>
      </div>
      <div class="footer-meta">
        <a href="${repository}/releases/tag/v${version}">${t('Release', '发行版本')} <span>v${version}</span></a>
        <a href="${repository}/blob/main/LICENSE">${t('Apache 2.0 license', 'Apache 2.0 许可证')}</a>
        <a class="return-link" href="${chapter ? '#top' : '#overview'}">${t('Back to top', '返回顶部')}<span aria-hidden="true">↑</span></a>
      </div>
    </div>
  </footer>`;
  return {header, install, footer};
}
