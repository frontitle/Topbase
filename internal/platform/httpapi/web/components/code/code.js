(function (global) {
  'use strict';

  function resolve(target) {
    return typeof target === 'string' ? document.querySelector(target) : target;
  }

  function copyText(value) {
    if (navigator.clipboard && navigator.clipboard.writeText) return navigator.clipboard.writeText(value);
    const input = document.createElement('textarea');
    input.value = value;
    input.setAttribute('readonly', '');
    input.style.position = 'fixed';
    input.style.opacity = '0';
    document.body.appendChild(input);
    input.select();
    try {
      if (!document.execCommand('copy')) throw Error('浏览器拒绝了复制操作');
      return Promise.resolve();
    } catch (error) {
      return Promise.reject(error);
    } finally {
      input.remove();
    }
  }

  function copyButton(code, label) {
    const button = document.createElement('button');
    button.type = 'button';
    button.className = 'tb-code-copy';
    button.textContent = '复制代码';
    button.setAttribute('aria-label', '复制' + (label || '代码'));
    button.onclick = async () => {
      if (!code()) return;
      const original = button.textContent;
      try {
        await copyText(code());
        button.textContent = '已复制';
        button.classList.add('is-copied');
        if (typeof global.toast === 'function') global.toast('代码已复制');
      } catch (error) {
        if (typeof global.toast === 'function') global.toast('复制失败：' + error.message);
      } finally {
        setTimeout(() => {
          button.textContent = original;
          button.classList.remove('is-copied');
        }, 1600);
      }
    };
    return button;
  }

  function createHeader(language, label, code) {
    const header = document.createElement('div');
    header.className = 'tb-code-head';
    const meta = document.createElement('div');
    const name = document.createElement('strong');
    name.textContent = label || '代码';
    const languageName = document.createElement('span');
    languageName.textContent = String(language || 'text').toUpperCase();
    meta.append(name, languageName);
    header.append(meta, copyButton(code, label));
    return header;
  }

  function mountBlock(target, options) {
    const host = resolve(target);
    if (!host) return null;
    const settings = Object.assign({ language: 'text', label: '代码', code: '' }, options || {});
    host.classList.add('tb-code');
    host.innerHTML = '';
    const pre = document.createElement('pre');
    pre.className = 'tb-code-pre';
    const code = document.createElement('code');
    code.textContent = settings.code || '';
    pre.appendChild(code);
    host.append(createHeader(settings.language, settings.label, () => code.textContent), pre);
    host._topbaseCode = { type: 'block', code, settings };
    return {
      set(value, next) {
        code.textContent = value || '';
        if (next) Object.assign(settings, next);
      },
      value() { return code.textContent; }
    };
  }

  function setCode(target, value, options) {
    const host = resolve(target);
    if (!host) return null;
    if (!host._topbaseCode || host._topbaseCode.type !== 'block') return mountBlock(host, Object.assign({}, options, { code: value }));
    host._topbaseCode.code.textContent = value || '';
    return host._topbaseCode;
  }

  function mountEditor(target, options) {
    const host = resolve(target);
    if (!host) return null;
    const settings = Object.assign({ language: 'sql', label: 'SQL 编辑器', value: '', placeholder: '' }, options || {});
    host.classList.add('tb-code', 'tb-code-editor');
    host.innerHTML = '';
    const textarea = document.createElement('textarea');
    textarea.className = 'tb-code-input';
    textarea.value = settings.value || '';
    textarea.placeholder = settings.placeholder || '';
    textarea.spellcheck = false;
    textarea.setAttribute('aria-label', settings.label);
    textarea.addEventListener('input', () => {
      if (typeof settings.onChange === 'function') settings.onChange(textarea.value);
    });
    textarea.addEventListener('keydown', event => {
      if (event.key !== 'Tab') return;
      event.preventDefault();
      const start = textarea.selectionStart;
      const end = textarea.selectionEnd;
      textarea.setRangeText('  ', start, end, 'end');
      textarea.dispatchEvent(new Event('input', { bubbles: true }));
    });
    host.append(createHeader(settings.language, settings.label, () => textarea.value), textarea);
    host._topbaseCode = { type: 'editor', textarea, settings };
    return {
      element: textarea,
      value() { return textarea.value; },
      set(value, notify) {
        textarea.value = value || '';
        if (notify && typeof settings.onChange === 'function') settings.onChange(textarea.value);
      },
      focus() { textarea.focus(); }
    };
  }

  global.TopbaseCode = { mountBlock, mountEditor, setCode, copyText };
})(window);
