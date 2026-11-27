(function (global) {
  const ui = global.TopbaseUI || (global.TopbaseUI = {});
  function $(sel, root) {
    return (root || document).querySelector(sel);
  }
  function $$(sel, root) {
    return [...(root || document).querySelectorAll(sel)];
  }
  function esc(value) {
    return String(value ?? '').replace(/[&<>"']/g, function (c) {
      return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c];
    });
  }
  // Markup helpers accept plain text and native control attributes. Bind
  // business actions with event listeners after inserting the returned HTML.
  const controlAttributes = new Set([
    'id', 'name', 'title', 'tabindex', 'role', 'disabled', 'hidden', 'required', 'readonly',
    'placeholder', 'autocomplete', 'inputmode', 'min', 'max', 'step', 'maxlength', 'minlength', 'multiple', 'rows', 'cols', 'size', 'pattern', 'spellcheck', 'accept'
  ]);
  function attributes(attrs = {}) {
    return Object.entries(attrs).map(([key, value]) => {
      if (value == null || (!controlAttributes.has(key) && !/^(aria|data)-[a-z0-9_-]+$/.test(key))) return '';
      if (['disabled','hidden','required','readonly','multiple'].includes(key)) return value ? ' ' + key : '';
      return ' ' + key + '="' + esc(value) + '"';
    }).join('');
  }

  function icon(name) {
    if (!/^[a-z][a-z0-9-]*$/.test(name || '')) return '';
    return '<svg class="tb-icon" aria-hidden="true" focusable="false" viewBox="0 0 24 24"><use href="/vendor/lucide-nav.svg#' + name + '"></use></svg>';
  }
  function button({ label, variant = 'secondary', icon: name, type = 'button', attrs, className = '' }) {
    const style = ['primary','secondary','ghost','danger'].includes(variant) ? variant : 'secondary';
    return '<button type="' + (type === 'submit' ? 'submit' : 'button') + '" class="tb-btn tb-btn-' + style + (className ? ' ' + esc(className) : '') +
      '"' + attributes(attrs) + '>' + (name ? icon(name) : '') + esc(label) + '</button>';
  }
  function iconButton({ icon: name, label, tone, attrs = {}, className = '' }) {
    if (!label) throw new Error('图标按钮需要可访问名称');
    return '<button type="button" class="tb-icon-trigger' + (tone === 'danger' ? ' tb-icon-danger' : '') +
      (className ? ' ' + esc(className) : '') + '"' + attributes({ title: label, ...attrs, 'aria-label': label }) + '>' + icon(name) + '</button>';
  }

  function field(label, control, help) {
    return label ? '<label class="tb-field"><span>' + esc(label) + '</span>' + control +
      (help ? '<small>' + esc(help) + '</small>' : '') + '</label>' : control;
  }
  function input({ label, value = '', type = 'text', attrs, help }) {
    const types = ['text','number','search','email','url','password','tel','date','time','datetime-local','color','month','week','textarea'];
    const nativeType = types.includes(type) ? type : 'text';
    const html = nativeType === 'textarea'
      ? '<textarea class="tb-input"' + attributes(attrs) + '>' + esc(value) + '</textarea>'
      : '<input class="tb-input" type="' + nativeType + '" value="' + esc(value) + '"' + attributes(attrs) + '>';
    return field(label, html, help);
  }
  function select({ label, options = [], value, attrs = {}, help }) {
    const selected = new Set((Array.isArray(value) ? value : [value]).filter(item => item != null).map(String));
    const html = options.map(option => '<option value="' + esc(option.value) + '"' +
      (selected.has(String(option.value)) ? ' selected' : '') + (option.disabled ? ' disabled' : '') + '>' + esc(option.label) + '</option>').join('');
    return field(label, '<select class="tb-select"' + attributes(attrs) + '>' + html + '</select>', help);
  }
  function checkbox({ label, checked, attrs }) {
    return '<label class="tb-check"><input type="checkbox"' + (checked ? ' checked' : '') + attributes(attrs) + '><span>' + esc(label) + '</span></label>';
  }

  var toastTimer;
  function toast(message) {
    var el = $('#toast');
    if (!el) {
      el = document.createElement('div');
      el.id = 'toast';
      document.body.appendChild(el);
    }
    el.setAttribute('role', 'status');
    el.setAttribute('aria-live', 'polite');
    el.textContent = message;
    el.classList.add('show');
    clearTimeout(toastTimer);
    toastTimer = setTimeout(function () { el.classList.remove('show'); }, 2600);
  }
  var dialogSequence = 0;
  var dialogLayers = [];

  function formDialog(options) {
    options = options || {};
    return new Promise(function (resolve) {
      var previousFocus = document.activeElement;
      var id = 'tb-dialog-' + (++dialogSequence);
      var fields = options.fields || [];
      var layer = document.createElement('div');
      layer.className = 'tb-dialog-layer';
      layer.innerHTML = '<section class="tb-dialog' + (options.size === 'wide' ? ' tb-dialog-wide' : '') + '" role="dialog" aria-modal="true" aria-labelledby="' + id + '-title">' +
        '<header class="tb-dialog-head"><div>' +
          (options.kicker ? '<small>' + esc(options.kicker) + '</small>' : '') +
          '<h2 id="' + id + '-title">' + esc(options.title || '请确认') + '</h2>' +
          (options.description ? '<p>' + esc(options.description) + '</p>' : '') +
        '</div>' + iconButton({ icon: 'x', label: '关闭', className: 'tb-dialog-close' }) + '</header>' +
        '<form class="tb-dialog-form"><div class="tb-dialog-body">' +
          (options.details && options.details.length ? '<dl class="tb-dialog-details">' + options.details.map(function (item) {
            return '<div><dt>' + esc(item.label) + '</dt><dd>' + esc(item.value) + '</dd></div>';
          }).join('') + '</dl>' : '') +
          fields.map(function (field) {
            var name = esc(field.name);
            var fieldAttrs = { name: field.name, required: field.required, placeholder: field.placeholder, autocomplete: 'off' };
            var label = '<span>' + esc(field.label || '') + (field.required ? ' <em>*</em>' : '') + '</span>';
            var help = field.help ? '<small>' + esc(field.help) + '</small>' : '';
            if (field.type === 'select') {
              return '<label class="tb-dialog-field">' + label + select({ value: field.value ?? '', options: field.options, attrs: fieldAttrs }) + help + '</label>';
            }
            if (field.type === 'choice') {
              return '<fieldset class="tb-dialog-field tb-dialog-choice"><legend>' + label + '</legend>' +
                (field.options || []).map(function (option, index) {
                  var checked = String(option.value) === String(field.value ?? '') || (!field.value && index === 0);
                  return '<label><input type="radio" name="' + name + '" value="' + esc(option.value) + '"' + (checked ? ' checked' : '') + '><span><b>' + esc(option.label) + '</b>' + (option.description ? '<small>' + esc(option.description) + '</small>' : '') + '</span></label>';
                }).join('') + help + '</fieldset>';
            }
            if (field.type === 'multichoice') {
              var selected = Array.isArray(field.value) ? field.value.map(String) : [];
              return '<fieldset class="tb-dialog-field tb-dialog-choice tb-dialog-multichoice"><legend>' + label + '</legend>' +
                (field.options || []).map(function (option) {
                  return '<label><input type="checkbox" name="' + name + '" value="' + esc(option.value) + '"' + (selected.includes(String(option.value)) ? ' checked' : '') + '><span><b>' + esc(option.label) + '</b>' + (option.description ? '<small>' + esc(option.description) + '</small>' : '') + '</span></label>';
                }).join('') + help + '</fieldset>';
            }
            return '<label class="tb-dialog-field">' + label + input({ value: field.value, type: field.type, attrs: fieldAttrs }) + help + '</label>';
          }).join('') +
          '<p class="tb-dialog-error" role="alert" hidden></p>' +
        '</div><footer class="tb-dialog-footer">' +
          button({ label: options.cancelText || '取消', attrs: { 'data-dialog-cancel': '' } }) +
          button({ label: options.confirmText || '确认', type: 'submit', variant: options.tone === 'danger' ? 'danger' : 'primary', attrs: { 'data-dialog-confirm': '' } }) +
        '</footer></form></section>';
      document.body.appendChild(layer);
      dialogLayers.push(layer);
      document.body.classList.add('tb-dialog-open');

      var panel = layer.querySelector('.tb-dialog');
      var form = layer.querySelector('form');
      var error = layer.querySelector('.tb-dialog-error');
      var settled = false;
      function finish(value) {
        if (settled) return;
        settled = true;
        document.removeEventListener('keydown', onKeydown, true);
        layer.classList.add('closing');
        dialogLayers = dialogLayers.filter(function (item) { return item !== layer; });
        if (!dialogLayers.length) document.body.classList.remove('tb-dialog-open');
        setTimeout(function () { layer.remove(); }, 140);
        if (previousFocus && previousFocus.focus) previousFocus.focus();
        resolve(value);
      }
      function onKeydown(event) {
        if (dialogLayers[dialogLayers.length - 1] !== layer) return;
        if (event.key === 'Escape') { event.preventDefault(); finish(null); return; }
        if (event.key !== 'Tab') return;
        var focusable = [...panel.querySelectorAll('button,input,select,textarea,[tabindex]:not([tabindex="-1"])')].filter(function (node) { return !node.disabled && !node.hidden; });
        if (!focusable.length) return;
        var first = focusable[0], last = focusable[focusable.length - 1];
        if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last.focus(); }
        else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first.focus(); }
      }
      function values() {
        var output = {};
        fields.forEach(function (field) {
          var node = form.elements[field.name];
          if (field.type === 'multichoice') {
            var nodes = node && typeof node.length === 'number' && !node.tagName ? [...node] : (node ? [node] : []);
            output[field.name] = nodes.filter(function (item) { return item.checked; }).map(function (item) { return String(item.value); });
          } else {
            output[field.name] = node ? String(node.value || '').trim() : '';
          }
        });
        return output;
      }
      form.onsubmit = function (event) {
        event.preventDefault();
        if (!form.reportValidity()) return;
        var output = values();
        var validation = options.validate ? options.validate(output) : '';
        if (validation) {
          error.textContent = validation;
          error.hidden = false;
          return;
        }
        finish(fields.length ? output : true);
      };
      layer.querySelector('[data-dialog-cancel]').onclick = function () { finish(null); };
      layer.querySelector('.tb-dialog-close').onclick = function () { finish(null); };
      layer.onclick = function (event) { if (event.target === layer && options.closeOnBackdrop !== false) finish(null); };
      document.addEventListener('keydown', onKeydown, true);
      requestAnimationFrame(function () {
        layer.classList.add('open');
        var firstField = form.querySelector('input:not([type="radio"]),select,textarea');
        (firstField || layer.querySelector('[data-dialog-confirm]')).focus();
        if (firstField && firstField.select && firstField.tagName === 'INPUT') firstField.select();
      });
    });
  }

  async function promptDialog(options, value) {
    if (typeof options === 'string') options = { title: options, value: value };
    options = options || {};
    var result = await formDialog(Object.assign({}, options, {
      fields: [{ name: 'value', label: options.label || options.title || '名称', value: options.value || '', placeholder: options.placeholder || '', required: options.required !== false }]
    }));
    return result ? result.value : null;
  }

  async function confirmDialog(options, description) {
    if (typeof options === 'string') options = { title: options, description: description };
    return (await formDialog(options || {})) === true;
  }

  async function choiceDialog(options) {
    options = options || {};
    var result = await formDialog(Object.assign({}, options, {
      fields: [{ name: 'value', label: options.label || '', type: 'choice', value: options.value, options: options.options || [], required: true }]
    }));
    return result ? result.value : null;
  }
  async function api(path, method, body, options) {
    options = options || {};
    var requestMethod = (method || 'GET').toUpperCase();
    var headers = body ? { 'Content-Type': 'application/json' } : {};
    if (!['GET', 'HEAD', 'OPTIONS'].includes(requestMethod)) {
      var match = document.cookie.match(/(?:^|; )topbase_csrf=([^;]+)/);
      if (match) headers['X-Topbase-CSRF'] = decodeURIComponent(match[1]);
    }
    var res = await fetch(path, {
      method: requestMethod,
      headers: headers,
      body: body ? JSON.stringify(body) : undefined,
      credentials: 'same-origin',
      signal: options.signal
    });
    if (res.status === 204) return null;
    var data = await res.json().catch(function () { return {}; });
    if (!res.ok) throw Error(data.error || '请求失败');
    return data;
  }
  function queryLimitMessage(meta) {
    meta = meta || {};
    if (!meta.truncated) return '';
    if (meta.truncation_reason === 'result_byte_limit') return '结果体积较大，仅展示已安全加载的部分数据';
    if (meta.truncation_reason === 'cell_byte_limit') return '部分超大字段已截断显示';
    if (meta.truncation_reason === 'row_limit') return '结果已达到 ' + (meta.row_limit || 1000) + ' 行展示上限';
    return '当前展示的是部分查询结果';
  }
  function emptyHTML(opts) {
    opts = opts || {};
    var cta = opts.href
      ? '<a class="primary" href="' + esc(opts.href) + '">' + esc(opts.cta || '去看看') + '</a>'
      : '';
    return '<div class="empty"><div class="empty-mark">' + esc(opts.icon || 'T') + '</div><h2>' +
      esc(opts.title || '暂无内容') + '</h2><p>' + esc(opts.body || '') + '</p>' + cta + '</div>';
  }
  function cardHTML(opts) {
    opts = opts || {};
    // `footer` remains a backwards-compatible alias for action. It is useful
    // for admin status cards whose primary operation belongs at the bottom.
    if (opts.footer && !opts.action) opts.action = opts.footer;
    var inner = '<b>' + esc(opts.title || '') + '</b>' +
      (opts.meta ? '<small>' + esc(opts.meta) + '</small>' : '') +
      (opts.action ? '<div class="card-foot">' + opts.action + '</div>' : '');
    if (opts.href) return '<a class="db-card" href="' + esc(opts.href) + '">' + inner + '</a>';
    return '<article class="db-card">' + inner + '</article>';
  }

  // Shared underline-tab controller. Pages only provide a tab container and
  // matching values on their buttons/panels; visual state and keyboard support
  // stay consistent everywhere.
  var tabSequence = 0;
  function mountTabs(target, options) {
    var host = typeof target === 'string' ? $(target) : target;
    if (!host) return null;
    var settings = options || {};
    var tabSelector = settings.tabSelector || '[data-tab]';
    var panelSelector = settings.panelSelector || '[data-panel]';
    var tabValue = settings.tabValue || function (tab) { return tab.dataset.tab; };
    var panelValue = settings.panelValue || function (panel) { return panel.dataset.panel; };
    var tabs = $$(tabSelector, host);
    var panels = settings.panels ? $$(panelSelector, settings.panels) : $$(panelSelector);
    if (!tabs.length) return null;
    host.classList.add('tb-tabs');

    function activate(value, focus, notify) {
      var selected = tabs.find(function (tab) { return tabValue(tab) === value && !tab.disabled && !tab.hidden; }) || tabs.find(function (tab) { return !tab.disabled && !tab.hidden; });
      if (!selected) return null;
      value = tabValue(selected);
      tabs.forEach(function (tab) {
        var active = tab === selected;
        tab.classList.toggle('active', active);
        tab.setAttribute('role', 'tab');
        tab.setAttribute('aria-selected', active ? 'true' : 'false');
        tab.tabIndex = active ? 0 : -1;
      });
      panels.forEach(function (panel) { panel.hidden = panelValue(panel) !== value; });
      if (focus) selected.focus();
      if (notify !== false) {
        host.dispatchEvent(new CustomEvent('topbase:tabchange', { detail: { value: value, tab: selected } }));
        if (typeof settings.onChange === 'function') settings.onChange(value, selected);
      }
      return value;
    }

    tabs.forEach(function (tab) {
      tab.onclick = function () { activate(tabValue(tab), true); };
      tab.onkeydown = function (event) {
        if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return;
        event.preventDefault();
        var enabled = tabs.filter(function (item) { return !item.disabled && !item.hidden; });
        var next = event.key === 'Home' ? enabled[0] : event.key === 'End' ? enabled[enabled.length - 1] : enabled[(enabled.indexOf(tab) + (event.key === 'ArrowRight' ? 1 : -1) + enabled.length) % enabled.length];
        if (next) activate(tabValue(next), true);
      };
      tab.setAttribute('role', 'tab');
      if (!tab.id) tab.id = 'tb-tab-' + (++tabSequence);
      var linked = panels.filter(function (panel) { return panelValue(panel) === tabValue(tab); });
      if (linked.length === 1) {
        var panel = linked[0];
        if (!panel.id) panel.id = tab.id + '-panel';
        tab.setAttribute('aria-controls', panel.id);
        if (!panel.hasAttribute('role')) panel.setAttribute('role', 'tabpanel');
        if (!panel.hasAttribute('aria-labelledby')) panel.setAttribute('aria-labelledby', tab.id);
      }
    });
    host.setAttribute('role', 'tablist');
    var initial = settings.initial || tabValue(tabs.find(function (tab) { return tab.classList.contains('active'); }) || tabs[0]);
    activate(initial, false, false);
    return { activate: activate, value: function () { var active = tabs.find(function (tab) { return tab.classList.contains('active'); }); return active ? tabValue(active) : ''; } };
  }

  var activePopover = null;
  var popoverSequence = 0;
  function bindPopover(trigger, panel, options = {}) {
    if (!trigger || !panel) return null;
    const doc = trigger.ownerDocument;
    const menu = options.type !== 'dialog';
    const role = menu ? 'menu' : 'dialog';
    const itemSelector = 'button,a[href],[role="menuitem"]';
    const focusSelector = 'button,input:not([type="hidden"]),select,textarea,a[href],[tabindex]';
    let opener = trigger;
    let disposed = false;
    const controller = { open, close, dispose };
    if (!panel.id) panel.id = 'tb-popover-' + (++popoverSequence);
    if (!trigger.id) trigger.id = panel.id + '-trigger';
    panel.setAttribute('role', role);
    if (!panel.hasAttribute('aria-label') && !panel.hasAttribute('aria-labelledby')) panel.setAttribute('aria-labelledby', trigger.id);
    panel.classList.add(menu ? 'tb-menu' : 'tb-popover');
    panel.tabIndex = -1;
    panel.hidden = true;
    setTrigger(trigger, false);
    trigger.addEventListener('click', toggle);
    trigger.addEventListener('keydown', triggerKey);
    panel.addEventListener('click', selectItem, true);
    panel.addEventListener('keydown', navigate);
    return controller;

    function setTrigger(button, expanded) {
      button.setAttribute('aria-controls', panel.id);
      button.setAttribute('aria-haspopup', role);
      button.setAttribute('aria-expanded', String(expanded));
    }
    function available(node) {
      return !node.disabled && node.getAttribute('aria-disabled') !== 'true' &&
        !node.closest('[hidden],[inert]') && node.getClientRects().length > 0;
    }
    function items() { return Array.from(panel.querySelectorAll(itemSelector)).filter(available); }
    function focus(node) { node?.focus({ preventScroll: true }); }
    function open({ from = trigger, last = false, moveFocus = true } = {}) {
      if (disposed || (options.canOpen && !options.canOpen())) return;
      if (activePopover && activePopover !== controller) activePopover.close(false);
      if (activePopover !== controller) {
        activePopover = controller;
        panel.hidden = false;
        opener = from;
        setTrigger(trigger, true);
        if (from !== trigger) setTrigger(from, true);
        options.onOpen?.();
        if (menu) panel.querySelectorAll(itemSelector).forEach(item => {
          item.setAttribute('role', 'menuitem');
          item.classList.add('tb-menu-item');
          item.tabIndex = -1;
        });
        for (const type of ['pointerdown', 'click', 'focusin']) doc.addEventListener(type, outside, true);
        doc.addEventListener('keydown', escape, true);
      }
      if (!moveFocus) return;
      const target = options.initialFocus?.() || (menu ? items().at(last ? -1 : 0) :
        Array.from(panel.querySelectorAll(focusSelector)).find(el => el.tabIndex >= 0 && available(el)));
      focus(target || panel);
    }
    function close(restoreFocus = true) {
      if (activePopover !== controller) return;
      activePopover = null;
      for (const type of ['pointerdown', 'click', 'focusin']) doc.removeEventListener(type, outside, true);
      doc.removeEventListener('keydown', escape, true);
      panel.hidden = true;
      setTrigger(trigger, false);
      if (opener !== trigger) setTrigger(opener, false);
      options.onClose?.();
      if (restoreFocus) focus(opener.isConnected ? opener : trigger);
      opener = trigger;
    }
    function toggle(event) {
      // A summary may be used as a trigger. Keep its native details toggle in sync.
      event.preventDefault();
      if (activePopover === controller) close();
      else open();
    }
    function triggerKey(event) {
      if (!['ArrowDown', 'ArrowUp'].includes(event.key) || event.isComposing) return;
      event.preventDefault();
      open({last: event.key === 'ArrowUp'});
    }
    function outside(event) {
      if (trigger.contains(event.target) || opener.contains(event.target) || panel.contains(event.target)) return;
      if (!options.keepOpenOnOutside?.(event)) close(false);
    }
    function escape(event) {
      if (event.key !== 'Escape' || event.isComposing) return;
      event.preventDefault();
      event.stopPropagation();
      close();
    }
    function selectItem(event) {
      const item = menu && event.target.closest(itemSelector);
      if (item && available(item)) close();
    }
    function navigate(event) {
      if (event.isComposing) return;
      if (event.key === 'Tab') {
        if (menu) {
          // Let the browser advance from the trigger, even when the panel is rendered elsewhere.
          close();
        } else {
          const stops = Array.from(panel.querySelectorAll(focusSelector)).filter(el => el.tabIndex >= 0 && available(el));
          if (event.target === stops.at(event.shiftKey ? 0 : -1) || !stops.length) {
            close();
            if (event.shiftKey) event.preventDefault();
          }
        }
        return;
      }
      if (!menu) return;
      const commands = items();
      if (!commands.length) return;
      const index = commands.indexOf(doc.activeElement);
      let next;
      if (event.key === 'Home') next = 0;
      else if (event.key === 'End') next = commands.length - 1;
      else if (event.key === 'ArrowDown') next = (index + 1) % commands.length;
      else if (event.key === 'ArrowUp') next = (Math.max(0, index) + commands.length - 1) % commands.length;
      else if (event.key === ' ' && event.target.tagName === 'A') {
        event.preventDefault();
        event.target.click();
        return;
      } else return;
      event.preventDefault();
      commands[next].focus();
    }
    function dispose() {
      close(false);
      disposed = true;
      trigger.removeEventListener('click', toggle);
      trigger.removeEventListener('keydown', triggerKey);
      panel.removeEventListener('click', selectItem, true);
      panel.removeEventListener('keydown', navigate);
    }
  }

  global.TopbaseUI.popover = bindPopover;
  global.$ = $;
  global.$$ = $$;
  global.esc = esc;
  global.toast = toast;
  global.formDialog = formDialog;
  global.promptDialog = promptDialog;
  global.confirmDialog = confirmDialog;
  global.choiceDialog = choiceDialog;
  global.api = api;
  global.queryLimitMessage = queryLimitMessage;
  global.emptyHTML = emptyHTML;
  global.cardHTML = cardHTML;
  global.TopbaseTabs = { mount: mountTabs };

  function modal(target, { onClose } = {}) {
    const dialog = typeof target === 'string' ? document.querySelector(target) : target;
    if (!dialog) return null;
    let session = null;
    let backdropPress = false;

    function open({ from = dialog.ownerDocument.activeElement } = {}) {
      if (dialog.open) return;
      dialog.returnValue = '';
      dialog.showModal();
      session = { from };
      backdropPress = false;
    }

    function finish() {
      // A native close event is queued. Its dialog may already be open again.
      if (dialog.open || !session) return;
      const { from } = session;
      session = null;
      backdropPress = false;
      if (from?.isConnected && !from.disabled) from.focus({ preventScroll: true });
      onClose?.(dialog.returnValue);
    }

    function close(value = '') {
      if (!dialog.open) return;
      dialog.close(value);
      finish();
    }

    function onBackdrop(event) {
      if (event.target !== dialog) return false;
      const box = dialog.getBoundingClientRect();
      return event.clientX < box.left || event.clientX >= box.right ||
        event.clientY < box.top || event.clientY >= box.bottom;
    }
    function pointerDown(event) { backdropPress = event.button === 0 && onBackdrop(event); }
    function click(event) {
      const dismiss = backdropPress && onBackdrop(event);
      backdropPress = false;
      if (dismiss) close();
    }
    dialog.addEventListener('pointerdown', pointerDown);
    dialog.addEventListener('click', click);
    dialog.addEventListener('close', finish);
    return { open, close, dispose() {
      close();
      dialog.removeEventListener('pointerdown', pointerDown);
      dialog.removeEventListener('click', click);
      dialog.removeEventListener('close', finish);
    } };
  }

  // Pages load specialist scripts on demand. Read the current implementation
  // when the component is used, regardless of script load order.
  ui.components = {
    get grid() { return global.TopbaseGrid; },
    get viz() { return global.TopbaseViz; },
    get filter() { return global.TopbaseFilter; },
    get code() { return global.TopbaseCode; },
    get queryEditor() { return global.TopbaseQueryEditor; },
    get dashboardWidgets() { return global.TopbaseDashboardWidgets; }
  };
  Object.assign(ui, {
    button, icon, iconButton, input, select, checkbox, modal, tabs: global.TopbaseTabs,
    formDialog, confirmDialog, promptDialog, choiceDialog, toast, cardHTML, emptyHTML
  });
})(window);
