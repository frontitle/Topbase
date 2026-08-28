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
    { id: 'home', href: '/', icon: '▦', label: '首页' },
    { id: 'collections', href: '/collections/', icon: '☰', label: '数据组' },
    { id: 'questions', href: '/questions/', icon: '◇', label: '分析' },
    { id: 'data', href: '/data/', icon: '▣', label: '数据浏览' },
    { id: 'dashboards', href: '/dashboard/', icon: '☷', label: '仪表盘' },
    { id: 'warehouse', href: '/warehouse/', icon: '▣', label: '数仓' },
    { id: 'trash', href: '/trash/', icon: '⌫', label: '回收站' }
  ];
  const adminItems = [
    { id: 'back', href: '/', icon: '←', label: '返回工作台' },
    { id: 'databases', href: '/admin/', icon: '▣', label: '数据源' },
    { id: 'datamodel', href: '/admin/datamodel/', icon: '◎', label: '数据模型' },
    { id: 'people', href: '/admin/people/', icon: '☺', label: '人员与组' },
    { id: 'permissions', href: '/admin/permissions/', icon: '⚿', label: '权限' },
    { id: 'integrations', href: '/admin/integrations/', icon: '↔', label: '通知与订阅' },
    { id: 'embedding', href: '/admin/embedding/', icon: '⌗', label: '嵌入与公开链接' },
    { id: 'monitor', href: '/admin/monitor/', icon: '◴', label: '监控与任务' },
    { id: 'settings', href: '/admin/settings/', icon: '⚙', label: '设置' }
  ];

  function links(items, active) {
    return items.map(item => {
      const current = item.id === active ? ' active' : '';
      const extra = item.id === 'back' ? 'back' : (item.extra || '');
      return `<a class="${extra}${current}" href="${item.href}" title="${item.label}"><span class="nav-icon">${item.icon}</span><span class="nav-label">${item.label}</span></a>`;
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
      user && user.is_admin ? [{ id: 'admin', href: '/admin/', icon: '⚙', label: '进入管理后台', extra: 'admin-entry' }] : []
    );
    const heading = 'Topbase';
    const plan = '';
    const collapsed = localStorage.getItem('topbase.sidebar.collapsed') === 'true';
    document.body.classList.toggle('sidebar-collapsed', collapsed);
    host.classList.add('sidebar', kind === 'admin' ? 'sidebar-admin' : 'sidebar-app');
    host.innerHTML = `<div class="sidebar-brand"><a class="workspace" href="${kind === 'admin' ? '/admin/' : '/'}"><span class="mark">T</span><strong>${heading}</strong><span class="plan">${plan}</span></a><button class="sidebar-toggle" type="button" aria-label="折叠侧边栏" title="折叠侧边栏">‹</button></div><nav>${kind === 'admin' ? '<p>系统管理</p>' : '<p>数据工作台</p>'}${links(items, active)}</nav><footer><a class="sidebar-profile" href="/account/" title="个人中心">${avatar(user, name)}<strong>${esc(name)}</strong><span aria-hidden="true">›</span></a></footer>`;
    host.querySelector('.sidebar-toggle').onclick = () => {
      const next = !document.body.classList.contains('sidebar-collapsed');
      document.body.classList.toggle('sidebar-collapsed', next);
      localStorage.setItem('topbase.sidebar.collapsed', String(next));
    };
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', mount);
  } else {
    mount();
  }
})();
