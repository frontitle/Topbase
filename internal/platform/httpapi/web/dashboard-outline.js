// The outline lists cards already on the current canvas in reading order.
function componentName(card) {
  return card.title || (card.type === 'text' && card.body?.split('\n')[0]) ||
    (card.type === 'question' && cardTitle(card)) || inspectorType(card);
}

function renderOutline() {
  const panel = $('#component-outline');
  if (!panel) return;
  const cards = tabCards();
  $('#outline-count').textContent = cards.length;
  if (panel.hidden) return;
  const term = $('#outline-search').value.trim().toLowerCase();
  const items = cards.map(card => ({ card, name: componentName(card), type: inspectorType(card) }))
    .filter(item => (item.name + ' ' + item.type).toLowerCase().includes(term))
    .sort((a, b) => (a.card.layout.y - b.card.layout.y) || (a.card.layout.x - b.card.layout.x));
  $('#outline-items').innerHTML = items.map(({card, name, type}) =>
    `<button type="button" data-outline-card="${esc(card.id)}" aria-current="${card.id === selectedCard}">
       <span translate="no">${esc(name)}</span>${type !== name ? `<small>${esc(type)}</small>` : ''}
     </button>`).join('') || `<p>${term ? '没有匹配的组件' : '画布上还没有组件'}</p>`;
}

let outlinePopover;
function closeOutline(restoreFocus = false) { outlinePopover?.close(restoreFocus); }

function bindOutline() {
  const panel = $('#component-outline');
  const search = $('#outline-search');
  outlinePopover = TopbaseUI.popover($('#outline-toggle'), panel, {
    type: 'dialog',
    initialFocus: () => search,
    onOpen: () => { search.value = ''; renderOutline(); }
  });
  search.oninput = renderOutline;
  panel.onclick = event => {
    const item = event.target.closest('[data-outline-card]');
    if (!item) return;
    selectCard(item.dataset.outlineCard);
    closeOutline();
    const card = document.querySelector(`[data-card="${CSS.escape(item.dataset.outlineCard)}"]`);
    card?.scrollIntoView({ block: 'nearest', inline: 'nearest' });
    card?.querySelector('[data-card-style]')?.focus({ preventScroll: true });
  };
  panel.onkeydown = event => {
    if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
      const items = [...panel.querySelectorAll('[data-outline-card]')];
      if (!items.length) return;
      event.preventDefault();
      const index = items.indexOf(document.activeElement);
      const next = index < 0 && event.key === 'ArrowUp' ? items.length - 1 :
        (index + (event.key === 'ArrowDown' ? 1 : -1) + items.length) % items.length;
      items[next].focus();
    }
  };
}
