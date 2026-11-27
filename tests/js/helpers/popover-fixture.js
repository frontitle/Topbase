const fs = require('node:fs');
const vm = require('node:vm');

// A small event tree for exercising the shared controller with the Node test runner.
function eventTarget(target) {
  const handlers = new Map();
  target.addEventListener = (type, fn, capture = false) => {
    if (!handlers.has(type)) handlers.set(type, []);
    handlers.get(type).push({fn, capture});
  };
  target.removeEventListener = (type, fn, capture = false) => {
    handlers.set(type, (handlers.get(type) || []).filter(item => item.fn !== fn || item.capture !== capture));
  };
  target.run = (event, capture) => {
    for (const item of [...(handlers.get(event.type) || [])]) {
      if (item.capture === capture) item.fn(event);
    }
    if (!capture) target['on' + event.type]?.(event);
  };
  target.send = (type, data = {}) => {
    const event = {type, target, ...data, preventDefault() { this.defaultPrevented = true; }, stopPropagation() { this.stopped = true; }};
    const path = [];
    for (let p = target; p; p = p.parent) path.push(p);
    for (const p of [...path].reverse()) { p.run(event, true); if (event.stopped) return event; }
    for (const p of path) { p.run(event, false); if (event.stopped) return event; }
    return event;
  };
  return target;
}

function surface() {
  const doc = eventTarget({ activeElement: null });
  const views = new Map();
  function element(tag = 'button', parent = doc) {
    const classes = new Set();
    const el = eventTarget({
      parent, ownerDocument: doc, tagName: tag.toUpperCase(), children: [], attrs: {}, isConnected: true,
      classList: {add: c => classes.add(c), remove: c => classes.delete(c), contains: c => classes.has(c)},
      getAttribute(key) { return this.attrs[key] ?? null; },
      hasAttribute(key) { return key in this.attrs; },
      setAttribute(key, value) { this.attrs[key] = String(value); },
      tabIndex: 0, hidden: false, disabled: false, getClientRects() { return [1]; },
      contains(other) { return other === this || this.children.some(c => c.contains(other)); },
      matches(selector) {
        if (selector === '[hidden]') return this.hidden;
        if (selector === 'input:not([type="hidden"])') return tag === 'input' && this.attrs.type !== 'hidden';
        if (selector === 'a[href]') return tag === 'a' && this.hasAttribute('href');
        if (selector.startsWith('[')) {
          const [name, value] = selector.slice(1, -1).replaceAll('"', '').split('=');
          return value === undefined ? this.hasAttribute(name) : this.getAttribute(name) === value;
        }
        return tag === selector;
      },
      closest(selector) {
        if (selector.split(',').some(s => this.matches(s))) return this;
        return this.parent?.closest?.(selector) || null;
      },
      querySelectorAll(selector) {
        return this.children.flatMap(c => [...(selector.split(',').some(s => c.matches(s)) ? [c] : []), ...c.querySelectorAll(selector)]);
      },
      focus() { doc.activeElement = this; this.send('focusin'); },
      click() { this.send('click'); }
    });
    parent.children?.push(el);
    return el;
  }
  doc.querySelector = sel => views.get(sel);
  doc.querySelectorAll = () => [];
  return {doc, element, views};
}
function loadUI(context) {
  vm.runInContext(fs.readFileSync('internal/platform/httpapi/web/ui.js', 'utf8'), context);
}
module.exports = {surface, loadUI};
