/* Chinese source messages are the fallback. Feature catalogs register additional locales. */
(function (global) {
  'use strict';
  const doc = global.document;
  const languages = new Map([['zh-CN', '简体中文'], ['en', 'English']]);
  const catalogs = new Map();
  const attributes = ['placeholder', 'aria-label', 'aria-description', 'aria-valuetext', 'title', 'alt'];
  const ignored = 'script,style,code,pre,template,[translate="no"],[contenteditable]:not([contenteditable="false"])';
  const originals = new WeakMap();
  const live = /^\/(?:auth\/login|setup)(?:\/|$)/.test(global.location?.pathname || '');
  let observer;
  function normalize(value) {
    const base = String(value || '').toLowerCase().split('-')[0];
    return [...languages.keys()].find(code => code.toLowerCase().split('-')[0] === base);
  }
  function savedLanguage() {
    try {
      const cookie = doc.cookie.match(/(?:^|;\s*)topbase_locale=([^;]+)/);
      if (cookie && normalize(cookie[1])) return normalize(cookie[1]);
      return normalize(global.localStorage.getItem('topbase.locale'));
    } catch (_) { return undefined; }
  }
  const path = global.location?.pathname || '';
  const embedded = /^\/(?:public|embed)\/dashboard\//.test(path);
  const override = embedded ? normalize(new URLSearchParams(global.location.search).get('lang')) : '';
  let current = override || savedLanguage() || (global.navigator?.languages || [global.navigator?.language]).map(normalize).find(Boolean) || 'zh-CN';
  function add(messages, language = 'en') {
    if (!catalogs.has(language)) catalogs.set(language, new Map());
    const dictionary = catalogs.get(language);
    for (const [key, value] of Object.entries(messages)) dictionary.set(key, value);
  }
  function t(key, values) {
    if (typeof key !== 'string') return key;
    let message = catalogs.get(current)?.get(key) ?? key;
    if (typeof message === 'object') {
      const form = new Intl.PluralRules(current).select(Number(values?.count));
      message = message[form] ?? message.other ?? key;
    }
    return values ? message.replace(/\{(\w+)\}/g, (token, name) =>
      Object.hasOwn(values, name) ? String(values[name] ?? '') : token) : message;
  }
  function translateString(source) {
    const trimmed = source.trim();
    if (!trimmed) return source;
    const translated = t(trimmed);
    if (translated !== trimmed) return source.replace(trimmed, () => translated);
    return source;
  }
  function apply(node, key, get, set) {
    const value = get();
    if (!value) return;
    let saved = originals.get(node);
    const previous = saved?.get(key);
    const source = previous?.output === value ? previous.source : value;
    const output = translateString(source);
    if (output !== value) {
      if (live) {
        if (!saved) originals.set(node, saved = new Map());
        saved.set(key, {source, output});
      }
      set(output);
    }
  }
  function translateNode(node) {
    if (node.nodeType === 3) {
      if (node.parentElement?.tagName !== 'TEXTAREA') apply(node, 'text', () => node.nodeValue, value => {
        const parent = node.parentElement;
        if (parent?.tagName === 'OPTION' && !parent.hasAttribute('value')) parent.value = parent.text;
        node.nodeValue = value;
      });
    } else if (node.nodeType === 1) {
      for (const key of attributes) if (node.hasAttribute(key)) {
        apply(node, key, () => node.getAttribute(key), value => node.setAttribute(key, value));
      }
    }
  }
  function translateTree(root) {
    const el = root.nodeType === 1 ? root : root.parentElement;
    if (!el || el.closest(ignored)) return;
    translateNode(root);
    if (root.nodeType !== 1) return;
    const walker = doc.createTreeWalker(root, NodeFilter.SHOW_ELEMENT | NodeFilter.SHOW_TEXT, {
      acceptNode: node => node.nodeType === 1 && node.matches(ignored) ? NodeFilter.FILTER_REJECT : NodeFilter.FILTER_ACCEPT
    });
    while (walker.nextNode()) translateNode(walker.currentNode);
  }
  function observe() {
    observer?.disconnect();
    if (current === 'zh-CN' && !live) return;
    observer = new MutationObserver(records => {
      const roots = new Set();
      for (const record of records) {
        if (record.type === 'childList') {
          for (const node of record.addedNodes) roots.add(node);
        } else if (record.type === 'attributes') {
          if (!record.target.closest(ignored)) translateNode(record.target);
        } else {
          roots.add(record.target);
        }
      }
      for (const root of roots) {
        if (root.isConnected && !roots.has(root.parentNode)) translateTree(root);
      }
    });
    observer.observe(doc.documentElement, {subtree:true, childList:true, characterData:true, attributes:true, attributeFilter:attributes});
  }
  function remember(language) {
    try { global.localStorage.setItem('topbase.locale', language); } catch (_) {}
    try { doc.cookie = 'topbase_locale=' + language + '; Path=/; Max-Age=31536000; SameSite=Lax' + (global.location.protocol === 'https:' ? '; Secure' : ''); } catch (_) {}
  }
  function useSaved(language) {
    const next = normalize(language);
    if (!next) return;
    remember(next);
    if (next === current) return;
    if (!live) { global.location.reload(); return; }
    current = next;
    doc.documentElement.lang = next;
    translateTree(doc.documentElement);
    observe();
    global.dispatchEvent(new CustomEvent('topbase:localechange', {detail:next}));
  }
  function start() {
    if (current !== 'zh-CN') translateTree(doc.documentElement);
    observe();
    doc.querySelectorAll('[data-locale-picker]').forEach(select => {
      select.value = current;
      select.onchange = () => {
        if (embedded) {
          const url = new URL(global.location.href);
          url.searchParams.set('lang', select.value);
          global.location.replace(url);
        } else useSaved(select.value);
      };
    });
  }
  function syncAccount(language) {
    const next = normalize(language);
    if (!next || current === next || override) return;
    remember(next);
    // Restricted browsers may deny both local storage and cookies. In this
    // case stay on the page rather than retrying navigation indefinitely.
    if (savedLanguage() === next) global.location.reload();
  }
  global.TopbaseLocale = { add, t, locale: () => current, useSaved, syncAccount };
  if (doc) {
    doc.documentElement.lang = current;
    if (doc.readyState === 'loading') doc.addEventListener('DOMContentLoaded', start, {once:true});
    else start();
  }
})(window);
