const form = document.querySelector('#form');
const errorEl = document.querySelector('#error');
let storageMode = 'development';

(async () => {
  try {
    const status = await api('/api/setup/status', 'GET');
    if (status.completed) location.replace('/auth/login/');
    const storage = status.application_database || {};
    storageMode = storage.mode || 'development';
    const development = storageMode === 'development';
    function renderStorage() {
      document.querySelector('#storage-title').textContent = development ? 'SQLite · 开发体验模式' : window.TopbaseLocale.t('{engine} · 生产模式', {engine:storage.engine === 'mysql' ? 'MySQL' : 'PostgreSQL'});
      document.querySelector('#storage-description').textContent = development
        ? '数据保存在本机。生产环境请迁移至 PostgreSQL 或 MySQL。'
        : '工作区数据保存在独立的应用数据库中。';
    }
    renderStorage();
    window.addEventListener('topbase:localechange', renderStorage);
    document.querySelector('#storage-mode').classList.toggle('is-risk', development);
    document.querySelector('#storage-risk').hidden = !development;
    document.querySelector('#accept_storage_risk').required = development;
  } catch (e) {
    errorEl.hidden = false;
    errorEl.textContent = e.message;
  }
})();

form.onsubmit = async (event) => {
  event.preventDefault();
  errorEl.hidden = true;
  if (storageMode === 'development' && !document.querySelector('#accept_storage_risk').checked) {
    errorEl.hidden = false;
    errorEl.textContent = '请先确认存储风险。';
    return;
  }
  try {
    await api('/api/setup', 'POST', {
      language: window.TopbaseLocale.locale(),
      site_name: document.querySelector('#site_name').value,
      admin_name: document.querySelector('#admin_name').value,
      admin_email: document.querySelector('#admin_email').value,
      admin_password: document.querySelector('#admin_password').value,
    });
    location.replace('/');
  } catch (e) {
    errorEl.hidden = false;
    errorEl.textContent = e.message;
  }
};
