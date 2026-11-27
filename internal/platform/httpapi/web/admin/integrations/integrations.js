(() => {
  const platforms = {
    feishu: {label: '飞书', guide: '在目标群添加自定义机器人，然后复制 Webhook 地址。'},
    dingtalk: {label: '钉钉', guide: '在目标群添加自定义机器人，然后复制 Webhook 地址。'},
    wecom: {label: '企业微信', guide: '在目标群添加群机器人，然后复制 Webhook 地址。'},
    slack: {
      label: 'Slack',
      guide: '在 Slack 应用中开启 Incoming Webhooks，选择 Add New Webhook to Workspace，授权目标频道后复制地址。',
      docs: 'https://docs.slack.dev/messaging/sending-messages-using-incoming-webhooks/'
    },
    teams: {
      label: 'Microsoft Teams',
      guide: '在 Workflows 中创建接收 Webhook 后向频道或聊天发送消息的工作流。触发器选择“任何人”，保存后复制 HTTP POST URL。',
      docs: 'https://support.microsoft.com/en-us/workflows/send-messages-in-teams-using-incoming-webhooks'
    },
    google_chat: {
      label: 'Google Chat',
      guide: '在目标聊天室打开“应用和集成”，添加 Webhook 后复制完整地址（含 key 和 token）。',
      docs: 'https://developers.google.com/workspace/chat/quickstart/webhooks'
    },
    generic: {label: '通用 Webhook', guide: '填写接收 HTTP POST 请求的服务地址。'}
  };
  const labels = {text: '文本消息', post: '富文本', card: '消息卡片', custom: '自定义 JSON'};
  const formatDate = value => value ? new Date(value).toLocaleString(TopbaseLocale.locale()) : '未发送';
  const dialog = TopbaseUI.modal('#hook-dialog', {onClose() {
    $('#hook-url').value = '';
    $('#hook-secret').value = '';
  }});
  let hooks = [];
  let subscriptions = [];
  let editing = null;
  let saving = false;

  function nameOfChannel(channel) {
    return hooks.find(hook => 'webhook:' + hook.id === channel)?.name || channel;
  }

  function renderHooks() {
    $('#hooks').innerHTML = hooks.map(hook => {
      const button = (action, label, variant = 'secondary', disabled = false) => TopbaseUI.button({
        label, variant, attrs: {['data-' + action]: hook.id, disabled}
      });
      const style = hook.template_type || (hook.template ? 'custom' : 'text');
      return `<article class="hook-card">
        <h3 translate="no">${esc(hook.name)}</h3>
        <div class="hook-meta"><span>${esc(platforms[hook.provider]?.label || hook.provider)}</span>
          <span>${hook.enabled ? '已启用' : '已暂停'}</span><span>${esc(labels[style] || style)}</span></div>
        <div class="hook-actions">${button('test', '发送测试消息', 'secondary', !hook.enabled)}
          ${button('edit', '编辑')}${button('toggle', hook.enabled ? '暂停' : '启用', 'ghost')}${button('remove', '删除', 'danger')}</div>
      </article>`;
    }).join('') || '<p class="panel-help">尚未创建 Webhook 通道。</p>';
  }

  function renderSubscriptions() {
    const mode = $('#sub-filter').value;
    const word = $('#sub-search').value.toLowerCase();
    const rows = subscriptions.filter(sub => (mode === 'all' || (mode === 'enabled') === sub.enabled)
      && [sub.dashboard_id, sub.channel, nameOfChannel(sub.channel)].join(' ').toLowerCase().includes(word));
    $('#subscriptions').innerHTML = rows.map(sub => {
      const id = esc(sub.id);
      return `<tr><td><b>${esc(sub.dashboard_id)}</b><small>${id}</small></td><td translate="no">${esc(nameOfChannel(sub.channel))}</td>
        <td>${esc(sub.cron)}<small>${esc(sub.timezone)}</small></td><td>${formatDate(sub.last_run_at)}</td>
        <td><span class="status ${sub.enabled ? 'on' : 'off'}">${sub.enabled ? '已启用' : '已暂停'}</span></td>
        <td class="actions"><button class="tb-btn tb-btn-secondary" data-run="${id}">验证</button><button class="tb-btn tb-btn-ghost" data-sub="${id}">${sub.enabled ? '暂停' : '启用'}</button>
          <button class="tb-btn tb-btn-danger" data-del="${id}">删除</button></td></tr>`;
    }).join('') || '<tr><td colspan="6">暂无订阅</td></tr>';
  }

  function syncProviderFields() {
    const provider = $('#hook-provider').value;
    const info = platforms[provider] || platforms.generic;
    $('#hook-instructions').textContent = info.guide;
    const guide = $('#hook-guide-link');
    guide.hidden = !info.docs;
    if (info.docs) guide.href = info.docs;
    else guide.removeAttribute('href');
    const keepsURL = editing?.provider === provider;
    $('#hook-url').required = !keepsURL;
    $('#hook-url').placeholder = keepsURL ? '已保存，留空保留原地址' : '粘贴完整 Webhook URL';
    $('#hook-secret-field').hidden = !!info.docs;
  }

  function syncMessageFields() {
    const style = $('#hook-style').value;
    const custom = style === 'custom';
    $('#hook-content').hidden = custom;
    $('#hook-json-field').hidden = !custom;
    $('#hook-json').required = custom;
    $('#hook-card-help').hidden = !(platforms[$('#hook-provider').value]?.docs && style === 'card');
  }

  function suggestedName(provider) {
    return TopbaseLocale.t('{name} 通知', {name: TopbaseLocale.t(platforms[provider]?.label || provider)});
  }

  function edit(hook, provider = 'feishu') {
    editing = hook;
    $('#hook-form').reset();
    const selected = hook?.provider || provider;
    const options = Object.entries(platforms).map(([value, info]) => new Option(info.label, value));
    if (!platforms[selected]) options.push(new Option(selected, selected));
    $('#hook-provider').replaceChildren(...options);
    $('#hook-provider').value = selected;
    $('#hook-heading').textContent = hook ? '编辑 Webhook' : '创建 Webhook';
    $('#hook-name').defaultValue = hook?.name || suggestedName(selected);
    $('#hook-name').value = $('#hook-name').defaultValue;
    $('#hook-url').value = '';
    const defaultStyle = !hook && platforms[selected]?.docs ? 'card' : 'text';
    $('#hook-style').value = hook?.template_type || (hook?.template ? 'custom' : defaultStyle);
    $('#hook-title-pattern').value = hook?.template_title || '';
    $('#hook-body-pattern').value = hook?.template_body || '';
    $('#hook-json').value = hook?.template || '';
    $('#hook-content').open = false;
    $('#hook-error').hidden = true;
    syncProviderFields();
    syncMessageFields();
    dialog.open();
    if (!hook) $('#hook-url').focus();
  }

  function fromForm() {
    const provider = $('#hook-provider').value;
    const style = $('#hook-style').value;
    const custom = style === 'custom';
    const title = custom ? '' : $('#hook-title-pattern').value.trim();
    const body = custom ? '' : $('#hook-body-pattern').value.trim();
    let template = '';
    if (custom) {
      template = $('#hook-json').value.trim();
      let parsed;
      try { parsed = JSON.parse(template); } catch { throw new Error('请检查模板 JSON 的格式'); }
      if (!parsed || Array.isArray(parsed) || typeof parsed !== 'object') throw new Error('模板必须是 JSON 对象');
    } else if (editing?.provider === provider && editing.template_type === style && !title && !body
      && !editing.template_title && !editing.template_body) {
      // Earlier releases stored personalized presets as complete JSON payloads.
      template = editing.template || '';
    }
    return {...editing, id: editing?.id || 'hook_' + Array.from(crypto.getRandomValues(new Uint32Array(3)), n => n.toString(36)).join('_'), enabled: editing?.enabled ?? true,
      name: $('#hook-name').value.trim(), url: $('#hook-url').value.trim(), provider,
      secret: $('#hook-secret-field').hidden ? '' : $('#hook-secret').value.trim(),
      template_type: style, template_title: title, template_body: body, template};
  }

  async function saveHooks(next) {
    if (saving) throw new Error('正在保存通道，请稍候再试');
    saving = true;
    try {
      hooks = await api('/api/webhooks', 'PUT', next);
      renderHooks();
      renderSubscriptions();
    } finally {
      saving = false;
    }
  }

  $('#hook-form').onsubmit = async event => {
    event.preventDefault();
    $('#hook-save').disabled = true;
    try {
      const item = fromForm();
      const next = editing ? hooks.map(hook => hook.id === item.id ? item : hook) : [...hooks, item];
      await saveHooks(next);
      dialog.close();
      toast('Webhook 通道已保存');
    } catch (error) {
      $('#hook-error').textContent = error.message;
      $('#hook-error').hidden = false;
    } finally {
      $('#hook-save').disabled = false;
    }
  };
  $('#hook-provider').onchange = () => {
    const name = $('#hook-name');
    if (!editing && name.value === name.defaultValue) {
      name.defaultValue = suggestedName($('#hook-provider').value);
      name.value = name.defaultValue;
    }
    $('#hook-url').value = '';
    $('#hook-secret').value = '';
    syncProviderFields();
    syncMessageFields();
  };
  $('#hook-style').onchange = syncMessageFields;
  $$('[data-close-hook]').forEach(button => button.onclick = () => dialog.close());
  $('#add-hook').onclick = () => edit(null);
  $$('[data-quick-provider]').forEach(button => button.onclick = () => edit(null, button.dataset.quickProvider));

  $('#hooks').onclick = async event => {
    const button = event.target.closest('button');
    if (!button) return;
    const {test, edit: editId, toggle, remove} = button.dataset;
    const hook = hooks.find(item => item.id === (test || editId || toggle || remove));
    if (!hook) return;
    if (editId) return edit(hook);
    button.disabled = true;
    try {
      if (test) {
        await api('/api/webhooks/' + encodeURIComponent(hook.id) + '/test', 'POST');
        toast('平台已接受测试消息，请在目标会话中查看');
      } else if (toggle) {
        await saveHooks(hooks.map(item => item.id === hook.id ? {...item, enabled: !hook.enabled} : item));
        toast(hook.enabled ? '通道已暂停' : '通道已启用');
      } else if (remove && await confirmDialog({title: TopbaseLocale.t('删除“{name}”？', {name:hook.name}), description: '使用该通道的订阅将无法继续投递。', tone: 'danger', confirmText: '删除'})) {
        await saveHooks(hooks.filter(item => item.id !== hook.id));
        toast('通道已删除');
      }
    } catch (error) { toast(error.message); }
    finally { button.disabled = false; }
  };

  async function loadActivity() {
    const [items, notes] = await Promise.all([api('/api/subscriptions'), api('/api/admin/notifications')]);
    subscriptions = items || [];
    renderSubscriptions();
    $('#history').innerHTML = (notes || []).map(note => `<tr><td>${formatDate(note.created_at)}</td><td translate="no">${esc(note.title)}</td><td translate="no">${esc(note.body || '')}</td></tr>`).join('')
      || '<tr><td colspan="3">暂无记录</td></tr>';
  }
  $('#subscriptions').onclick = async event => {
    const button = event.target.closest('button');
    if (!button) return;
    const {run, sub, del} = button.dataset;
    const item = subscriptions.find(item => item.id === (run || sub || del));
    if (!item) return;
    const path = '/api/subscriptions/' + encodeURIComponent(item.id);
    button.disabled = true;
    try {
      if (run) {
        await api(path + '/run', 'POST', {});
        toast('已发起验证');
      } else if (sub) {
        await api(path, 'PUT', {enabled: !item.enabled});
        toast('状态已更新');
      } else if (del) {
        if (!await confirmDialog({title: '删除订阅？', tone: 'danger', confirmText: '删除'})) return;
        await api(path, 'DELETE');
        toast('订阅已删除');
      }
      await loadActivity();
    } catch (error) { toast(error.message); }
    finally { button.disabled = false; }
  };
  $('#sub-filter').onchange = renderSubscriptions;
  $('#sub-search').oninput = renderSubscriptions;
  api('/api/webhooks').then(items => {
    hooks = items || [];
    renderHooks();
    renderSubscriptions();
    $$('#add-hook, [data-quick-provider]').forEach(button => button.disabled = false);
  }).catch(error => toast(error.message));
  loadActivity().catch(error => toast(error.message));
})();
