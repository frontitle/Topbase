const form = document.querySelector('#form');
const errorEl = document.querySelector('#error');

(async () => {
  try {
    const status = await api('/api/setup/status', 'GET');
    if (!status.completed) location.replace('/setup/');
    const options = await api('/api/auth/options', 'GET');
    form.hidden = !options.password_login_enabled;
    const providers = document.querySelector('#providers');
    if (options.providers && options.providers.length) {
      providers.hidden = false;
      providers.innerHTML = options.providers.map(p => '<a class="provider tb-btn tb-btn-secondary" href="'+esc(p.login_url)+'">使用 '+esc(p.name)+' 登录</a>').join('');
    }
    if (!options.password_login_enabled) document.querySelector('#lead').textContent = '此工作区仅允许使用已配置的第三方账号登录。';
  } catch (e) {
    errorEl.hidden = false;
    errorEl.textContent = e.message;
  }
})();

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
