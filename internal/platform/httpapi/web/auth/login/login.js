const form = document.querySelector('#form');
const errorEl = document.querySelector('#error');

(async () => {
  try {
    const status = await api('/api/setup/status', 'GET');
    if (!status.completed) location.replace('/setup/');
    const options = await api('/api/auth/options', 'GET');
    function updateSignInOptions() {
      form.hidden = !options.password_login_enabled;
      const providers = document.querySelector('#providers');
      if (options.providers && options.providers.length) {
        providers.hidden = false;
        providers.innerHTML = options.providers.map(p => '<a class="provider tb-btn tb-btn-secondary" href="'+esc(p.login_url)+'" translate="no">'+esc(window.TopbaseLocale.t('使用 {provider} 登录', {provider:p.name}))+'</a>').join('');
      }
      const lead = document.querySelector('#lead');
      lead.hidden = options.password_login_enabled;
      if (!lead.hidden) lead.textContent = window.TopbaseLocale.t('请使用组织账号登录。');
    }
    updateSignInOptions();
    window.addEventListener('topbase:localechange', updateSignInOptions);
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
