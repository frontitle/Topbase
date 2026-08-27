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
    if (!status.completed) location.replace('/setup/');
    const options = await api('/api/auth/options', 'GET');
    form.hidden = !options.password_login_enabled;
    const providers = document.querySelector('#providers');
    if (options.providers && options.providers.length) {
      providers.hidden = false;
      providers.innerHTML = options.providers.map(p => '<a class="provider" href="'+p.login_url+'">使用 '+escapeHTML(p.name)+' 登录</a>').join('');
    }
    if (!options.password_login_enabled) document.querySelector('#lead').textContent = '此工作区仅允许使用已配置的第三方账号登录。';
  } catch (e) {
    errorEl.hidden = false;
    errorEl.textContent = e.message;
  }
})();

function escapeHTML(value) { const el = document.createElement('span'); el.textContent = value || ''; return el.innerHTML; }

form.onsubmit = async (event) => {
  event.preventDefault();
  errorEl.hidden = true;
  try {
    await api('/api/session', 'POST', {
      email: document.querySelector('#email').value,
      password: document.querySelector('#password').value,
    });
    location.replace('/');
  } catch (e) {
    errorEl.hidden = false;
    errorEl.textContent = e.message;
  }
};
