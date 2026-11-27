const current = location.pathname.split('/docs/')[1] || 'index.html';
document.querySelectorAll('[data-route]').forEach((link) => {
  if (link.dataset.route === current) link.classList.add('active');
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

document.querySelector('.docs-header button')?.addEventListener('click', () => document.body.classList.toggle('nav-open'));
