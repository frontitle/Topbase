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

const chapterButton = document.querySelector('.docs-chapter-bar button[aria-controls="docs-nav"]');
const chapterNav = document.getElementById('docs-nav');

function setChaptersOpen(open) {
  document.body.classList.toggle('nav-open', open);
  chapterButton.setAttribute('aria-expanded', String(open));
}

chapterButton.addEventListener('click', () => {
  setChaptersOpen(chapterButton.getAttribute('aria-expanded') !== 'true');
});
document.addEventListener('keydown', event => {
  if (event.key === 'Escape' && document.body.classList.contains('nav-open')) {
    setChaptersOpen(false);
    chapterButton.focus();
  }
});
document.addEventListener('focusin', event => {
  if (event.target !== chapterButton && !chapterNav.contains(event.target)) setChaptersOpen(false);
});
matchMedia('(max-width: 900px)').addEventListener('change', () => setChaptersOpen(false));
