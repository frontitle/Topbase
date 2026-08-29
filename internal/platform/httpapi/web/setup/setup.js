const form = document.querySelector('#form');
const errorEl = document.querySelector('#error');
let storageMode = 'development';

async function api(path, method, body) {
  const r = await fetch(path, {
    method,
    headers: body ? { 'Content-Type': 'application/json' } : {},
    body: body ? JSON.stringify(body) : undefined,
  });
  const d = await r.json().catch(() => ({}));
  if (!r.ok) throw Error(d.error || '请求失败');
  return d;
}

(async () => {
  try {
    const status = await api('/api/setup/status', 'GET');
    if (status.completed) location.replace('/auth/login/');
    const storage = status.application_database || {};
    storageMode = storage.mode || 'development';
    const development = storageMode === 'development';
    document.querySelector('#storage-title').textContent = development ? 'SQLite · 开发体验模式' : `${storage.engine === 'mysql' ? 'MySQL' : 'PostgreSQL'} · 生产正式模式`;
    document.querySelector('#storage-description').textContent = development
      ? '无需额外数据库即可快速体验；数据保存在当前服务器的本地目录。'
      : '项目数据保存在独立应用数据库，可用于持续升级、RDS 和多节点部署。';
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
    errorEl.textContent = '请先确认开发体验模式的数据持久化风险。';
    return;
  }
  try {
    await api('/api/setup', 'POST', {
      language: 'zh-CN',
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
