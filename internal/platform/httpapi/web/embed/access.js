(function (global) {
  function mount(uuid, embedded, { clearCanvas, openDashboard }) {
    const root = document.querySelector('#share-access');
    const canvas = document.querySelector('#board-canvas');
    const form = root.querySelector('form');
    const password = root.querySelector('input');
    const submit = form.querySelector('button');
    const title = root.querySelector('h1');
    const message = root.querySelector('[role="status"]');
    const errorText = root.querySelector('[role="alert"]');
    const retry = root.querySelector('#share-access-retry');
    const base = '/api/public/dashboard/' + encodeURIComponent(uuid);
    const key = 'topbase-share:' + uuid;
    let token = '';
    let deadline = 0;
    let timer;
    try { token = sessionStorage.getItem(key) || ''; } catch { /* Embedded storage can be disabled. */ }

    function remember(value) {
      token = value;
      try {
        if (value) sessionStorage.setItem(key, value);
        else sessionStorage.removeItem(key);
      } catch { /* The current frame can still use the in-memory grant. */ }
    }

    async function request(path, body, signal) {
      const headers = {};
      if (token && path !== '/verify') headers['X-Topbase-Share-Token'] = token;
      if (body !== undefined) headers['Content-Type'] = 'application/json';
      const response = await fetch(base + path + (embedded ? '?embed=1' : ''), {
        method: body === undefined ? 'GET' : 'POST',
        headers, body: JSON.stringify(body), signal, credentials: 'omit', cache: 'no-store'
      });
      const data = await response.json();
      if (!response.ok) {
        const error = new Error(data.error || '请求失败，请重试。');
        error.status = response.status;
        error.code = data.code;
        throw error;
      }
      if (path === '') {
        const expires = Date.parse(data.dashboard.public_expires_at || '');
        // Use the server clock so the visitor's timezone and system clock
        // cannot extend the lifetime of an already open, text-only canvas.
        const serverDate = Date.parse(response.headers.get('Date'));
        deadline = Number.isFinite(expires) ? Date.now() + expires - (Number.isFinite(serverDate) ? serverDate : Date.now()) : 0;
      }
      return data;
    }

    function scheduleDeadline() {
      clearTimeout(timer);
      if (!deadline) return;
      const remaining = deadline - Date.now();
      if (remaining <= 0) {
        if (!canvas.hidden) clearCanvas();
        canvas.hidden = true;
        location.reload();
        return;
      }
      timer = setTimeout(scheduleDeadline, Math.min(remaining, 2147483647));
    }

    function show(state, description) {
      if (!canvas.hidden) {
        canvas.hidden = true;
        clearCanvas();
      }
      root.hidden = false;
      form.hidden = state !== 'password';
      retry.hidden = state !== 'retry';
      errorText.hidden = true;
      password.value = '';
      password.removeAttribute('aria-invalid');
      title.textContent = state === 'password' ? '验证访问密码' : state === 'retry' ? '仪表盘暂时无法加载' : '分享链接不可用';
      message.textContent = description;
      if (state === 'password' && !embedded) password.focus();
    }

    function handleDenial(error) {
      if (error.status === 401 && error.code === 'password_required') {
        remember('');
        show('password', '请输入分享者提供的密码。');
        return true;
      }
      if (error.status === 403 || error.status === 404 || error.status === 410) {
        remember('');
        show('unavailable', '请向分享者索取新链接。');
        return true;
      }
      return false;
    }

    async function load() {
      try {
        const data = await request('');
        if (deadline && deadline <= Date.now()) {
          show('unavailable', '请向分享者索取新链接。');
          return null;
        }
        scheduleDeadline();
        root.hidden = true;
        canvas.hidden = false;
        return data;
      } catch (error) {
        if (!handleDenial(error)) show('retry', '请检查网络连接后重试。');
        return null;
      }
    }

    form.onsubmit = async event => {
      event.preventDefault();
      if (!password.value || submit.disabled) return;
      submit.disabled = true;
      errorText.hidden = true;
      password.removeAttribute('aria-invalid');
      try {
        const result = await request('/verify', { password: password.value });
        password.value = '';
        remember(result.token || '');
        await openDashboard();
      } catch (error) {
        if (handleDenial(error)) return;
        errorText.textContent = error.status ? error.message : '网络连接失败，请重试。';
        errorText.hidden = false;
        if (error.code === 'invalid_password') {
          password.setAttribute('aria-invalid', 'true');
          password.focus();
          password.select();
        }
      } finally { submit.disabled = false; }
    };
    retry.onclick = openDashboard;
    global.addEventListener('pageshow', event => { if (event.persisted) location.reload(); });
    document.addEventListener('visibilitychange', () => { if (!document.hidden) scheduleDeadline(); });
    return { load, requestCard: (id, filters, signal) => request('/cards/' + encodeURIComponent(id) + '/dataset', { filters }, signal), handleDenial };
  }
  global.TopbaseSharedDashboard = { mount };
})(window);
