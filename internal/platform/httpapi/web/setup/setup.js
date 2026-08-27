const form = document.querySelector('#form');
const errorEl = document.querySelector('#error');

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
  } catch (e) {
    errorEl.hidden = false;
    errorEl.textContent = e.message;
  }
})();

form.onsubmit = async (event) => {
  event.preventDefault();
  errorEl.hidden = true;
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
