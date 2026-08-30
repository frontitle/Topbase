(function (global) {
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
  function toast(message) {
    var el = $('#toast');
    if (!el) {
      el = document.createElement('div');
      el.id = 'toast';
      document.body.appendChild(el);
    }
    el.textContent = message;
    el.classList.add('show');
    setTimeout(function () { el.classList.remove('show'); }, 2600);
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
        '</div><button class="tb-dialog-close" type="button" aria-label="关闭">×</button></header>' +
        '<form class="tb-dialog-form"><div class="tb-dialog-body">' +
          (options.details && options.details.length ? '<dl class="tb-dialog-details">' + options.details.map(function (item) {
            return '<div><dt>' + esc(item.label) + '</dt><dd>' + esc(item.value) + '</dd></div>';
          }).join('') + '</dl>' : '') +
          fields.map(function (field) {
            var name = esc(field.name);
            var required = field.required ? ' required' : '';
            var label = '<span>' + esc(field.label || '') + (field.required ? ' <em>*</em>' : '') + '</span>';
            var help = field.help ? '<small>' + esc(field.help) + '</small>' : '';
            if (field.type === 'select') {
              return '<label class="tb-dialog-field">' + label + '<select name="' + name + '"' + required + '>' +
                (field.options || []).map(function (option) {
                  return '<option value="' + esc(option.value) + '"' + (String(option.value) === String(field.value ?? '') ? ' selected' : '') + '>' + esc(option.label) + '</option>';
                }).join('') + '</select>' + help + '</label>';
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
            if (field.type === 'textarea') {
              return '<label class="tb-dialog-field">' + label + '<textarea name="' + name + '" placeholder="' + esc(field.placeholder || '') + '"' + required + '>' + esc(field.value || '') + '</textarea>' + help + '</label>';
            }
            return '<label class="tb-dialog-field">' + label + '<input name="' + name + '" type="' + esc(field.type || 'text') + '" value="' + esc(field.value || '') + '" placeholder="' + esc(field.placeholder || '') + '"' + required + ' autocomplete="off">' + help + '</label>';
          }).join('') +
          '<p class="tb-dialog-error" role="alert" hidden></p>' +
        '</div><footer class="tb-dialog-footer">' +
          '<button class="secondary" data-dialog-cancel type="button">' + esc(options.cancelText || '取消') + '</button>' +
          '<button class="' + (options.tone === 'danger' ? 'tb-dialog-danger' : 'primary') + '" data-dialog-confirm type="submit">' + esc(options.confirmText || '确认') + '</button>' +
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
  async function api(path, method, body) {
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
      credentials: 'same-origin'
    });
    if (res.status === 204) return null;
    var data = await res.json().catch(function () { return {}; });
    if (!res.ok) throw Error(data.error || '请求失败');
    return data;
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

    tabs.forEach(function (tab, index) {
      tab.onclick = function () { activate(tabValue(tab), true); };
      tab.onkeydown = function (event) {
        if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return;
        event.preventDefault();
        var enabled = tabs.filter(function (item) { return !item.disabled && !item.hidden; });
        var next = event.key === 'Home' ? enabled[0] : event.key === 'End' ? enabled[enabled.length - 1] : enabled[(enabled.indexOf(tab) + (event.key === 'ArrowRight' ? 1 : -1) + enabled.length) % enabled.length];
        activate(tabValue(next), true);
      };
      tab.setAttribute('role', 'tab');
      if (!tab.id) tab.id = 'tb-tab-' + Date.now() + '-' + index;
    });
    host.setAttribute('role', 'tablist');
    var initial = settings.initial || tabValue(tabs.find(function (tab) { return tab.classList.contains('active'); }) || tabs[0]);
    activate(initial, false, false);
    return { activate: activate, value: function () { var active = tabs.find(function (tab) { return tab.classList.contains('active'); }); return active ? tabValue(active) : ''; } };
  }

  global.$ = $;
  global.$$ = $$;
  global.esc = esc;
  global.toast = toast;
  global.formDialog = formDialog;
  global.promptDialog = promptDialog;
  global.confirmDialog = confirmDialog;
  global.choiceDialog = choiceDialog;
  global.api = api;
  global.emptyHTML = emptyHTML;
  global.cardHTML = cardHTML;
  global.TopbaseTabs = { mount: mountTabs };
})(window);
