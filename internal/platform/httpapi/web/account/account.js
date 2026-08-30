let profileState = null;
let avatarValue = '';
let originalEmail = '';
let apiKeysLoaded = false;
let developerStatus = null;

function initials(value) {
  return String(value || 'T').trim().slice(0, 1).toUpperCase();
}

function friendlyError(error) {
  const message = String(error && error.message || error || '操作失败');
  const translations = [
    [/current password is incorrect/i, '当前密码不正确。'],
    [/current password is required to change email/i, '修改登录邮箱前，请填写当前密码。'],
    [/password must be at least 8 characters/i, '新密码至少需要 8 位。'],
    [/new password must be different/i, '新密码不能与当前密码相同。'],
    [/email is already in use/i, '这个邮箱已经被其他账户使用。'],
    [/avatar image is too large/i, '头像文件过大，请重新选择。'],
    [/avatar image is invalid/i, '头像格式无效，请选择 PNG、JPEG 或 WebP 图片。']
  ];
  const hit = translations.find(([pattern]) => pattern.test(message));
  return hit ? hit[1] : message;
}

function setStatus(selector, message, isError) {
  const node = $(selector);
  node.textContent = message || '';
  node.classList.toggle('error', !!isError);
}

function renderAvatars() {
  const name = $('#profile-name').value || (profileState && profileState.user.name) || 'T';
  $$('[data-avatar]').forEach(node => {
    const image = node.querySelector('img');
    const fallback = node.querySelector('span');
    image.hidden = !avatarValue;
    fallback.hidden = !!avatarValue;
    if (avatarValue) image.src = avatarValue;
    else image.removeAttribute('src');
    fallback.textContent = initials(name);
  });
}

function providerMeta(type) {
  return ({
    google: { icon: 'G', label: 'Google' },
    wechat: { icon: '微', label: '微信' },
    feishu: { icon: '飞', label: '飞书' },
    dingtalk: { icon: '钉', label: '钉钉' },
    wecom: { icon: '企', label: '企业微信' },
    oidc: { icon: 'ID', label: '企业身份' }
  })[type] || { icon: 'ID', label: '第三方账号' };
}

function renderBindings() {
  const list = $('#binding-list');
  const passwordDescription = profileState.password_login_enabled
    ? '使用登录邮箱和密码访问当前工作区。'
    : '管理员已关闭邮箱密码登录，仍可在这里维护密码。';
  const cards = [`<article class="binding-card"><span class="binding-icon">@</span><div class="binding-copy"><strong>邮箱与密码</strong><small>${esc(passwordDescription)}</small></div><span class="account-state">${profileState.password_configured ? '已设置' : '未设置'}</span></article>`];
  (profileState.providers || []).forEach(provider => {
    const meta = providerMeta(provider.type);
    let action = '<button class="secondary binding-action" type="button" disabled>需管理员配置</button>';
    let description = '该登录方式尚未完成系统配置。';
    if (provider.linked) {
      action = provider.can_unbind
        ? `<button class="secondary binding-action linked" type="button" data-unbind="${esc(provider.id)}">已绑定 · 解除</button>`
        : '<button class="secondary binding-action linked" type="button" disabled>唯一登录方式</button>';
      description = provider.can_unbind ? '已与当前 Topbase 账户关联，可用于识别登录身份。' : '密码登录已关闭，为避免账户被锁定，请先绑定另一种登录方式。';
    } else if (provider.self_bindable) {
      action = `<a class="secondary binding-action" href="${esc(provider.login_url)}">立即绑定</a>`;
      description = '完成平台授权后，即可使用该身份登录。';
    } else if (provider.enabled) {
      action = '<button class="secondary binding-action" type="button" disabled>组织统一管理</button>';
      description = '该办公平台账号由管理员通过组织同步进行绑定。';
    }
    cards.push(`<article class="binding-card"><span class="binding-icon">${esc(meta.icon)}</span><div class="binding-copy"><strong>${esc(provider.name || meta.label)}</strong><small>${esc(description)}</small></div>${action}</article>`);
  });
  if (!(profileState.providers || []).length) {
    cards.push('<div class="account-loading">当前工作区还没有配置第三方登录方式；邮箱与密码登录不受影响。</div>');
  }
  list.innerHTML = cards.join('');
  $$('[data-unbind]', list).forEach(button => {
    button.onclick = async () => {
      const provider = profileState.providers.find(item => item.id === button.dataset.unbind);
      const approved = await confirmDialog({
        kicker: '账号绑定',
        title: `解除“${provider ? provider.name : '第三方账号'}”绑定？`,
        description: profileState.password_login_enabled ? '解除后将不能再用该平台身份登录，但邮箱密码登录不受影响。' : '解除后将不能再用该平台身份登录，请确认仍有其他已绑定的登录方式。',
        confirmText: '解除绑定',
        tone: 'danger'
      });
      if (!approved) return;
      button.disabled = true;
      try {
        await api(`/api/user/external-identities/${encodeURIComponent(button.dataset.unbind)}`, 'DELETE');
        toast('账号绑定已解除');
        await loadProfile();
      } catch (error) {
        toast(friendlyError(error));
        button.disabled = false;
      }
    };
  });
}

function renderProfile() {
  const user = profileState.user;
  originalEmail = user.email;
  avatarValue = user.avatar_url || '';
  $('#account-name').textContent = user.name;
  $('#account-email').textContent = user.email;
  $('#profile-name').value = user.name;
  $('#profile-email').value = user.email;
  $('#profile-current-password').value = '';
  $('#email-password').hidden = true;
  const badges = [...new Set([user.is_admin ? '管理员' : '成员'].concat(profileState.groups || []))];
  $('#account-badges').innerHTML = badges.map(item => `<span>${esc(item)}</span>`).join('');
  $('#password-status').textContent = profileState.password_login_enabled ? '密码登录已启用' : '密码登录已关闭';
  renderAvatars();
  renderBindings();
}

async function loadProfile() {
  profileState = await api('/api/user/profile');
  renderProfile();
}

function resizeAvatar(file) {
  return new Promise((resolve, reject) => {
    if (!['image/png', 'image/jpeg', 'image/webp'].includes(file.type)) {
      reject(new Error('请选择 PNG、JPEG 或 WebP 图片。'));
      return;
    }
    if (file.size > 8 * 1024 * 1024) {
      reject(new Error('原始图片不能超过 8 MB。'));
      return;
    }
    const url = URL.createObjectURL(file);
    const image = new Image();
    image.onload = () => {
      const side = Math.min(image.naturalWidth, image.naturalHeight);
      const sx = Math.max(0, (image.naturalWidth - side) / 2);
      const sy = Math.max(0, (image.naturalHeight - side) / 2);
      const canvas = document.createElement('canvas');
      canvas.width = 256;
      canvas.height = 256;
      const context = canvas.getContext('2d');
      context.fillStyle = '#ffffff';
      context.fillRect(0, 0, 256, 256);
      context.drawImage(image, sx, sy, side, side, 0, 0, 256, 256);
      URL.revokeObjectURL(url);
      resolve(canvas.toDataURL('image/jpeg', .86));
    };
    image.onerror = () => { URL.revokeObjectURL(url); reject(new Error('无法读取这张图片。')); };
    image.src = url;
  });
}

TopbaseTabs.mount('.account-tabs', { onChange: function (tab) {
  history.replaceState(null, '', `#${tab}`);
  if (tab === 'api' && !apiKeysLoaded) loadAPIKeys();
} });

function renderAPIKeys(keys) {
  const list = $('#api-key-list');
  if (!keys.length) {
    list.innerHTML = '<div class="account-loading">还没有 API Key。为每个 AI 客户端创建独立密钥，停用时可单独撤销。</div>';
    return;
  }
  const now = Date.now();
  list.innerHTML = keys.map(key => { const expired = key.expires_at && new Date(key.expires_at).getTime() <= now; const expiry = key.expires_at ? `${expired ? '已过期' : '有效至'} ${new Date(key.expires_at).toLocaleDateString('zh-CN')}` : '永久有效'; return `<article class="binding-card${expired ? ' is-expired' : ''}"><span class="binding-icon">API</span><div class="binding-copy"><strong>${esc(key.name)}</strong><small>${esc(key.prefix)}•••• · 创建于 ${esc(new Date(key.created_at).toLocaleString('zh-CN'))} · <span class="${expired ? 'key-expired' : ''}">${esc(expiry)}</span></small></div><button class="secondary danger-subtle" type="button" data-revoke-key="${esc(key.id)}">撤销</button></article>`; }).join('');
  $$('[data-revoke-key]', list).forEach(button => {
    button.onclick = async () => {
      const approved = await confirmDialog({kicker:'API Key',title:'撤销这个 API Key？',description:'使用它的 MCP、CLI 和自动化程序会立即失去访问权限，此操作不会删除已经创建的分析。',confirmText:'撤销密钥',tone:'danger'});
      if (!approved) return;
      button.disabled = true;
      try { await api(`/api/api-keys/${encodeURIComponent(button.dataset.revokeKey)}`, 'DELETE'); toast('API Key 已撤销'); await loadAPIKeys(true); }
      catch (error) { toast(friendlyError(error)); button.disabled = false; }
    };
  });
}

async function loadAPIKeys(force) {
  if (apiKeysLoaded && !force) return;
  try {
    const result = await Promise.all([api('/api/developer/status'), api('/api/api-keys')]);
    developerStatus = result[0];
    const enabled = !!developerStatus.enabled;
    const canCreate = !!developerStatus.can_create_key;
    const notice = $('#developer-notice');
    notice.classList.toggle('is-disabled', !enabled);
    notice.querySelector('strong').textContent = enabled ? '开发者模式已启用' : '开发者模式未启用';
    notice.querySelector('small').textContent = enabled ? `API 查询最多返回 ${developerStatus.max_query_rows} 行，新密钥默认有效 ${developerStatus.default_key_ttl_days ? developerStatus.default_key_ttl_days + ' 天' : '期为永久'}。${developerStatus.allow_analysis_write ? '允许创建可视化分析。' : '当前仅允许只读查询。'}` : '请联系管理员在“管理后台 → 设置 → 开发者模式”中启用。已有密钥会保留，但目前不能访问 API。';
    notice.querySelector('.account-state').textContent = enabled ? `${developerStatus.active_keys} 个可用 Key` : '已停用';
    $('#create-api-key').disabled = !canCreate;
    $('#create-api-key').title = canCreate ? '' : (enabled ? '管理员未允许成员创建个人 Key' : '开发者模式未启用');
    const baseURL = developerStatus.public_base_url || location.origin;
    $('#developer-command-code').textContent = `TOPBASE_URL=${baseURL} TOPBASE_API_KEY=•••• ./bin/topbase-mcp`;
    $('#developer-command').hidden = !enabled;
    renderAPIKeys(result[1]); apiKeysLoaded = true;
  }
  catch (error) { toast(friendlyError(error)); }
}

$('#create-api-key').onclick = async () => {
  const input = await formDialog({kicker:'AI 与自动化',title:'创建 API Key',description:'密钥继承你的 Topbase 权限。建议按客户端分别命名，便于独立撤销。',confirmText:'创建密钥',fields:[{name:'name',label:'密钥名称',placeholder:'例如：我的 Claude MCP',required:true,help:'不要在名称中填写密码或其他秘密。'}]});
  if (!input) return;
  try {
    const key = await api('/api/api-keys', 'POST', {name: input.name});
    $('#api-key-secret').textContent = key.key;
    $('#api-key-reveal').hidden = false;
    apiKeysLoaded = false;
    await loadAPIKeys();
    toast('API Key 已创建');
  } catch (error) { toast(friendlyError(error)); }
};

$('#copy-api-key').onclick = async () => {
  const value = $('#api-key-secret').textContent;
  try { await navigator.clipboard.writeText(value); toast('密钥已复制'); }
  catch (_) { toast('复制失败，请手动复制'); }
};

$('#copy-developer-command').onclick = async () => {
  try { await navigator.clipboard.writeText($('#developer-command-code').textContent); toast('MCP 启动参数已复制'); }
  catch (_) { toast('复制失败，请手动复制'); }
};

$('#profile-name').oninput = renderAvatars;
$('#profile-email').oninput = () => {
  const changed = $('#profile-email').value.trim().toLowerCase() !== originalEmail.toLowerCase();
  $('#email-password').hidden = !changed;
  $('#profile-current-password').required = changed;
};
$('#avatar-file').onchange = async event => {
  const file = event.target.files && event.target.files[0];
  if (!file) return;
  try {
    avatarValue = await resizeAvatar(file);
    renderAvatars();
    setStatus('#profile-status', '新头像将在保存后生效。');
  } catch (error) {
    toast(friendlyError(error));
  } finally {
    event.target.value = '';
  }
};
$('#remove-avatar').onclick = () => {
  avatarValue = '';
  renderAvatars();
  setStatus('#profile-status', '头像将在保存后移除。');
};

$('#profile-form').onsubmit = async event => {
  event.preventDefault();
  const button = event.submitter;
  button.disabled = true;
  setStatus('#profile-status', '正在保存…');
  try {
    const updated = await api('/api/user/profile', 'PUT', {
      name: $('#profile-name').value,
      email: $('#profile-email').value,
      locale: profileState.user.locale || 'zh-CN',
      theme: profileState.user.theme || 'light',
      avatar_url: avatarValue,
      current_password: $('#profile-current-password').value
    });
    profileState.user = updated;
    renderProfile();
    setStatus('#profile-status', '个人资料已保存。');
    toast('个人资料已保存');
  } catch (error) {
    setStatus('#profile-status', friendlyError(error), true);
  } finally {
    button.disabled = false;
  }
};

$('#password-form').onsubmit = async event => {
  event.preventDefault();
  const button = event.submitter;
  const currentPassword = $('#current-password').value;
  const newPassword = $('#new-password').value;
  if (newPassword !== $('#confirm-password').value) {
    setStatus('#security-status', '两次输入的新密码不一致。', true);
    return;
  }
  button.disabled = true;
  setStatus('#security-status', '正在更新…');
  try {
    await api('/api/user/password', 'PUT', { current_password: currentPassword, new_password: newPassword });
    event.currentTarget.reset();
    setStatus('#security-status', '密码已更新，下次登录请使用新密码。');
    toast('密码已安全更新');
  } catch (error) {
    setStatus('#security-status', friendlyError(error), true);
  } finally {
    button.disabled = false;
  }
};

$('#logout').onclick = async () => {
  $('#logout').disabled = true;
  try { await api('/api/session', 'DELETE'); } catch (_) {}
  location.replace('/auth/login/');
};

const requestedTab = location.hash.slice(1);
if (requestedTab && $(`[data-tab="${requestedTab}"]`)) $(`[data-tab="${requestedTab}"]`).click();
const bindingResult = new URLSearchParams(location.search).get('binding');
if (bindingResult === 'success') toast('第三方账号绑定成功');
if (bindingResult === 'failed') toast('账号绑定失败，该身份可能已绑定其他成员');
if (bindingResult) history.replaceState(null, '', location.pathname + location.hash);

loadProfile().catch(error => {
  toast(friendlyError(error));
  if (/not signed in/i.test(error.message)) location.replace('/auth/login/');
});
