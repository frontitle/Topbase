// The inspector and drag handles edit the same grid coordinates.
const TopbaseDashboardLayout = (() => {
  const labels = { x: '横向位置', y: '纵向位置', w: '宽度', h: '高度' };
  const defaults = { x: 0, y: 0, w: 4, h: 4 };
  const alignments = { left: '靠左', center: '居中', right: '靠右' };

  function coordinates(layout) {
    return Object.fromEntries(Object.entries(defaults).map(([key, fallback]) => [
      key, Number.isFinite(layout?.[key]) ? layout[key] : fallback
    ]));
  }

  function limitsFor(key, box, limits) {
    switch (key) {
      case 'x': return [0, Math.max(0, limits.columns - box.w)];
      case 'y': return [0, Infinity];
      case 'w': {
        const available = Math.max(0, limits.columns - box.x);
        return [Math.min(limits.minW, available), available];
      }
      case 'h': return [limits.minH, limits.maxH];
    }
  }

  function section(layout, limits) {
    const box = coordinates(layout);
    const fields = Object.entries(labels).map(([key, label]) => {
      const [min, max] = limitsFor(key, box, limits);
      const upper = Number.isFinite(max) ? `max="${max}"` : '';
      return `<label><span>${label} <small>${key.toUpperCase()}</small></span>
        <input class="tb-input" type="number" inputmode="decimal" data-layout-field="${key}" min="${min}" ${upper}
          step="${limits.snap ? 1 : 0.25}" value="${box[key]}"></label>`;
    }).join('');
    const buttons = Object.entries(alignments).map(([edge, label]) =>
      `<button type="button" class="tb-btn tb-btn-secondary" data-align-to="${edge}" title="${globalThis.TopbaseLocale?.t('相对画布' + label) ?? ('相对画布' + label)}">${label}</button>`
    ).join('');
    return `<section class="card-style-section layout-fields"><b>位置与尺寸 <small>网格单位</small></b>
      <div class="layout-field-grid">${fields}</div>
      <div class="layout-align" role="group" aria-label="相对画布对齐">
        <span>画布对齐</span>${buttons}
      </div></section>`;
  }

  function sync(host, layout, limits) {
    if (!host) return;
    const box = coordinates(layout);
    for (const input of host.querySelectorAll('[data-layout-field]')) {
      const key = input.dataset.layoutField;
      const [min, max] = limitsFor(key, box, limits);
      input.value = String(box[key]);
      input.min = String(min);
      input.max = Number.isFinite(max) ? String(max) : '';
      input.step = limits.snap ? '1' : '0.25';
    }
  }

  function bind(host, getCard, getLimits, commit) {
    function update(key, compute) {
      const card = getCard();
      if (!card) return;
      const box = coordinates(card.layout);
      const limits = getLimits(card);
      const proposed = compute(box, limits);
      if (Number.isFinite(proposed)) {
        const [min, max] = limitsFor(key, box, limits);
        const next = Math.min(max, Math.max(min, proposed));
        if (next !== box[key]) {
          box[key] = next;
          commit(card, box);
        }
      }
      sync(host, box, limits);
    }
    for (const input of host.querySelectorAll('[data-layout-field]')) {
      const key = input.dataset.layoutField;
      const commitInput = () => update(key, (box, limits) => {
        const text = input.value.trim();
        if (!text) return NaN;
        const value = Number(text);
        if (value === box[key]) return value; // Exact center can be between grid lines.
        const units = Math.round(value * (limits.snap ? 1 : 4));
        return Number.isSafeInteger(units) ? units / (limits.snap ? 1 : 4) : NaN;
      });
      input.onchange = commitInput;
      input.onblur = commitInput;
      input.onkeydown = event => {
        if (event.key === 'Enter') {
          event.preventDefault();
          commitInput();
          input.blur();
        }
      };
    }
    for (const button of host.querySelectorAll('[data-align-to]')) {
      button.onclick = () => update('x', (box, limits) => {
        const space = limits.columns - box.w;
        if (button.dataset.alignTo === 'center') return space / 2;
        return button.dataset.alignTo === 'right' ? space : 0;
      });
    }
  }

  function intersects(a, b) {
    return a.x < b.x + b.w && a.x + a.w > b.x && a.y < b.y + b.h && a.y + a.h > b.y;
  }

  function resolveOverlap(anchor, cards) {
    if (!cards.some(card => card.id !== anchor.id && card.layout && intersects(card.layout, anchor.layout))) return;
    const others = cards.filter(card => card.id !== anchor.id && card.layout)
      .sort((a, b) => a.layout.y - b.layout.y || a.layout.x - b.layout.x);
    const moved = [anchor.layout];
    const placed = [anchor.layout];
    for (const card of others) {
      const box = card.layout;
      if (moved.some(blocker => intersects(box, blocker))) {
        // Earlier cards have priority. Walk them in final Y order so landing
        // below one occupied row cannot introduce a collision on the next.
        for (const blocker of placed) {
          if (intersects(box, blocker)) box.y = blocker.y + blocker.h;
        }
        moved.push(box);
      }
      const after = placed.findIndex(blocker => blocker.y > box.y);
      placed.splice(after < 0 ? placed.length : after, 0, box);
    }
  }

  return { section, bind, sync, resolveOverlap };
})();
