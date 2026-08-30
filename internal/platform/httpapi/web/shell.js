window.topbasePickDatabase = function (dbs, preferred) {
  if (!Array.isArray(dbs) || !dbs.length) return '';
  if (preferred && dbs.some(function (d) { return d.id === preferred; })) return preferred;
  if (dbs.length === 1) return dbs[0].id;
  try {
    var saved = localStorage.getItem('topbase.database_id');
    if (saved && dbs.some(function (d) { return d.id === saved; })) return saved;
  } catch (_) {}
  return dbs[0].id;
};
window.topbaseRememberDatabase = function (id) {
  if (!id) return;
  try { localStorage.setItem('topbase.database_id', id); } catch (_) {}
};

(function () {
  const appItems = [
    { id: 'home', href: '/', icon: 'house', label: '首页' },
    { id: 'questions', href: '/questions/', icon: 'chart-no-axes-combined', label: '分析' },
    { id: 'dashboards', href: '/dashboard/', icon: 'layout-dashboard', label: '仪表盘' },
    { id: 'data', href: '/data/', icon: 'database', label: '源数据' },
    { id: 'warehouse', href: '/warehouse/', icon: 'warehouse', label: '数据沉淀' },
    { id: 'trash', href: '/trash/', icon: 'trash-2', label: '回收站' }
  ];
  const adminItems = [
    { id: 'back', href: '/', icon: 'arrow-left', label: '返回工作台' },
    { id: 'databases', href: '/admin/', icon: 'database', label: '数据' },
    { id: 'datamodel', href: '/admin/datamodel/', icon: 'network', label: '数据模型' },
    { id: 'people', href: '/admin/people/', icon: 'users', label: '人员与组' },
    { id: 'integrations', href: '/admin/integrations/', icon: 'webhook', label: '通知与订阅' },
    { id: 'embedding', href: '/admin/embedding/', icon: 'panels-top-left', label: '嵌入与公开链接' },
    { id: 'settings', href: '/admin/settings/', icon: 'settings', label: '设置' }
  ];

  function icon(name, className) {
    return `<svg class="${className || ''}" aria-hidden="true" focusable="false"><use href="/vendor/lucide-nav.svg#${name}"></use></svg>`;
  }

  function loadShellStyle() {
    if (document.querySelector('link[data-shell-style]')) return;
    const link = document.createElement('link');
    link.rel = 'stylesheet';
    link.href = '/shell.css';
    link.dataset.shellStyle = '';
    document.head.appendChild(link);
  }

  function links(items, active) {
    return items.map(item => {
      const current = item.id === active ? ' active' : '';
      const extra = item.id === 'back' ? 'back' : (item.extra || '');
      return `<a class="${extra}${current}" href="${item.href}" title="${item.label}">${icon(item.icon, 'nav-icon')}<span class="nav-label">${item.label}</span></a>`;
    }).join('');
  }

  function initial(name) {
    const text = (name || 'T').trim();
    return text.slice(0, 1).toUpperCase();
  }

  function avatar(user, name) {
    if (user && user.avatar_url) return `<span class="avatar avatar-image"><img src="${esc(user.avatar_url)}" alt=""></span>`;
    return `<span class="avatar">${esc(initial(name))}</span>`;
  }

  async function currentUser() {
    try {
      const r = await fetch('/api/user/current', { credentials: 'same-origin' });
      if (!r.ok) return null;
      return await r.json();
    } catch (_) {
      return null;
    }
  }

  async function mount() {
    const host = document.querySelector('[data-shell]');
    if (!host) return;
    const kind = host.dataset.shell || 'app';
    loadShellStyle();
    const active = host.dataset.active || '';
    const user = await currentUser();

    if (kind === 'admin') {
      if (!user) {
        location.replace('/auth/login/');
        return;
      }
      if (!user.is_admin) {
        location.replace('/');
        return;
      }
    }

    const name = user ? (user.name || user.email) : '未登录';

    const items = kind === 'admin' ? adminItems : appItems.concat(
      user && user.is_admin ? [{ id: 'admin', href: '/admin/', icon: 'settings-2', label: '进入管理后台', extra: 'admin-entry' }] : []
    );
    const heading = 'Topbase';
    const plan = '';
    const collapsed = localStorage.getItem('topbase.sidebar.collapsed') === 'true';
    document.body.classList.toggle('sidebar-collapsed', collapsed);
    host.classList.add('sidebar', kind === 'admin' ? 'sidebar-admin' : 'sidebar-app');
    const context = kind === 'admin' ? '<p>管理后台</p>' : '<p>数据工作台</p>';
    host.innerHTML = `<div class="sidebar-panel"><div class="sidebar-brand"><a class="workspace" href="${kind === 'admin' ? '/admin/' : '/'}"><img class="mark mark-image" src="/assets/topbase-logo.png" alt="Topbase logo"><strong>${heading}</strong><span class="shell-context">${kind === 'admin' ? '管理后台' : '工作台'}</span></a><button class="sidebar-toggle" type="button" aria-label="折叠侧边栏" title="折叠侧边栏">${icon(collapsed ? 'panel-left-open' : 'panel-left-close', 'sidebar-toggle-icon')}</button></div><label class="sidebar-search">${icon('search', 'sidebar-search-icon')}<input type="search" placeholder="搜索功能" aria-label="搜索功能"></label><nav>${context}${links(items, active)}</nav><footer><details class="sidebar-account" data-active="${active === 'account'}"><summary class="sidebar-profile" title="账户菜单">${avatar(user, name)}<strong>${esc(name)}</strong><span aria-hidden="true">›</span></summary><div class="account-menu"><a href="/account/">个人中心</a><button type="button" data-logout>退出登录</button></div></details></footer></div>`;
    if (kind === 'admin' && active === 'integrations') {
      const script = document.createElement('script'); script.src = '/admin/integrations/tabs.js?v=1'; document.body.appendChild(script);
    }
    if (kind === 'admin') {
      const versionLink = document.createElement('a'); versionLink.className = 'sidebar-version'; versionLink.href = '/admin/updates/'; versionLink.textContent = '版本'; host.querySelector('footer').before(versionLink);
      fetch('/api/version').then(r => r.ok ? r.json() : null).then(info => { if (!info) return; versionLink.textContent = info.version || '版本'; if (info.latest_version && info.latest_version !== info.version) versionLink.innerHTML = esc(info.version) + '<b>NEW</b>'; }).catch(() => {});
    }
    host.querySelector('.sidebar-toggle').onclick = () => {
      const next = !document.body.classList.contains('sidebar-collapsed');
      document.body.classList.toggle('sidebar-collapsed', next);
      localStorage.setItem('topbase.sidebar.collapsed', String(next));
      host.querySelector('.sidebar-toggle use').setAttribute('href', `/vendor/lucide-nav.svg#panel-left-${next ? 'open' : 'close'}`);
    };
    const navSearch = host.querySelector('.sidebar-search input');
    navSearch.oninput = () => {
      const term = navSearch.value.trim().toLowerCase();
      host.querySelectorAll('nav a').forEach(link => { link.hidden = !!term && !link.textContent.toLowerCase().includes(term); });
    };
    const logout = host.querySelector('[data-logout]');
    logout.onclick = async () => {
      logout.disabled = true;
      try { await api('/api/session', 'DELETE'); } catch (_) {}
      location.replace('/auth/login/');
    };
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', mount);
  } else {
    mount();
  }
})();
