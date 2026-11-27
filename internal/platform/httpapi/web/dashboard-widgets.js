// The editor consumes this catalog. Saved cards contain their own type, URL,
// config and dimensions so viewers do not need the catalog or its extensions.
(() => {
  const widgets = new Map();
  const groups = { text: '文字与排版', media: '媒体与信息', extension: '嵌入与扩展' };

  function define(id, label, group, symbol, w, h, defaults = {}) {
    widgets.set(id, { id, label, group, symbol, w, h, defaults: { type: id, ...defaults } });
  }

  define('heading', '标题', 'text', 'H', 24, 2, { title: '标题' });
  define('text', '文本', 'text', 'Aa', 24, 2, { body: '说明文字' });
  define('divider', '分隔线', 'text', '—', 24, 1);
  define('image', '图片', 'media', '▧', 12, 8);
  define('time', '时间', 'media', '◷', 6, 4, { title: '时间', config: { time_mode: 'datetime' } });
  define('link', '链接', 'extension', '↗', 12, 4, { title: '链接', body: 'https://' });
  define('iframe', '网页', 'extension', '▤', 12, 10, { title: '嵌入网页', config: { url: 'https://' } });
  define('plugin', '自定义组件', 'extension', '⌘', 12, 10, { title: '自定义组件', config: { url: '', plugin_config: {} } });

  function registerPlugin({ id, label, url, w = 12, h = 10, config = {} }) {
    if (typeof id !== 'string' || !/^[a-z][a-z0-9-]{0,63}$/.test(id) || widgets.has(id)) {
      throw new Error('组件标识须唯一，且仅含小写字母、数字和连字符');
    }
    if (typeof label !== 'string' || !label.trim()) throw new Error('请输入组件名称');
    const entry = new URL(url);
    if (entry.protocol !== 'https:' || entry.username || entry.password) throw new Error('组件地址须使用 HTTPS');
    if (!Number.isInteger(w) || w < 2 || w > 24 || !Number.isInteger(h) || h < 2 || h > 48) {
      throw new Error('组件宽度须为 2–24 格，高度须为 2–48 格');
    }
    if (!config || typeof config !== 'object' || Array.isArray(config)) throw new Error('组件配置须为 JSON 对象');
    define(id, label.trim(), 'extension', '⌘', w, h, {
      type: 'plugin', title: label.trim(), config: { url: entry.href, plugin_config: JSON.parse(JSON.stringify(config)) }
    });
    window.dispatchEvent(new Event('topbase:widgets-changed'));
  }

  function list(query = '') {
    const term = query.trim().toLowerCase();
    const text = label => globalThis.TopbaseLocale?.t(label) ?? label;
    return [...widgets.values()].map(item => ({...item, label: item.defaults.type === 'plugin' && item.id !== 'plugin' ? item.label : text(item.label)}))
      .filter(({ label, group, id }) => (label + ' ' + text(groups[group]) + ' ' + id).toLowerCase().includes(term))
      .map(({ id, label, group, symbol }) => ({ id, label, group: text(groups[group]), symbol }));
  }

  function size(id) {
    const item = widgets.get(id);
    return item && { w: item.w, h: item.h };
  }

  function create(id) {
    const item = widgets.get(id);
    if (!item) return;
    const card = JSON.parse(JSON.stringify(item.defaults));
    if (card.type !== 'plugin' || id === 'plugin') {
      if (card.title) card.title = globalThis.TopbaseLocale?.t(card.title) ?? card.title;
      if (card.body && card.type === 'text') card.body = globalThis.TopbaseLocale?.t(card.body) ?? card.body;
    }
    return card;
  }

  window.TopbaseDashboardWidgets = Object.freeze({ registerPlugin, list, size, create });
})();
