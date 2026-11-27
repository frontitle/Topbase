const current = location.pathname.split('/docs/')[1] || 'index.html';
document.querySelectorAll('[data-route]').forEach((link) => {
  if (link.dataset.route === current) {
    link.classList.add('active');
    link.setAttribute('aria-current', 'page');
  }
});

const toc = document.querySelector('#toc');
document.querySelectorAll('.docs-content h2, .docs-content h3').forEach((heading, index) => {
  if (!heading.id) heading.id = `section-${index + 1}`;
  const link = document.createElement('a');
  link.href = `#${heading.id}`;
  link.textContent = heading.textContent;
  link.style.paddingLeft = heading.tagName === 'H3' ? '10px' : '0';
  toc.appendChild(link);
});

const menuButton = document.querySelector('.docs-header button');
menuButton.addEventListener('click', () => {
  menuButton.setAttribute('aria-expanded', String(document.body.classList.toggle('nav-open')));
});
document.addEventListener('keydown', event => {
  if (event.key === 'Escape' && document.body.classList.contains('nav-open')) {
    document.body.classList.remove('nav-open');
    menuButton.setAttribute('aria-expanded', 'false');
    menuButton.focus();
  }
});
