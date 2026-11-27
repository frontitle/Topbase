const languageLink = document.querySelector('.language-switch a');
const alternateHome = new URL(languageLink.href);
const sections = [...document.querySelectorAll('main > section[id]')];
const footer = document.querySelector('.site-footer');
const header = document.querySelector('.site-header');
const menu = document.querySelector('.mobile-nav');

function updateLanguageLink() {
  const marker = header.offsetHeight + Math.min(innerHeight / 4, 140);
  let current = sections[0];
  for (const section of sections) {
    if (section.getBoundingClientRect().top > marker) break;
    current = section;
  }
  if (scrollY > 0 && Math.ceil(scrollY + innerHeight) >= document.documentElement.scrollHeight) {
    const install = sections.at(-1);
    // Tall screens can show the whole install panel and footer at the same time.
    current = location.hash !== '#footer' && install.getBoundingClientRect().top >= header.offsetHeight
      ? install : footer;
  } else if (footer.getBoundingClientRect().top <= marker) {
    current = footer;
  }
  alternateHome.hash = current && current !== sections[0] ? current.id : '';
  languageLink.href = alternateHome.href;
}

// Update when a link is used, including when opening it in another tab.
for (const event of ['focus', 'pointerenter', 'pointerdown', 'click', 'contextmenu']) {
  languageLink.addEventListener(event, updateLanguageLink);
}
window.addEventListener('pageshow', updateLanguageLink);

document.addEventListener('click', event => {
  if (!menu.contains(event.target) || event.target.closest('a')) menu.open = false;
});
document.addEventListener('keydown', event => {
  if (event.key === 'Escape' && menu.open) {
    menu.open = false;
    menu.querySelector('summary').focus();
  }
});
matchMedia('(max-width: 780px)').addEventListener('change', () => { menu.open = false; });

if ('IntersectionObserver' in window && !matchMedia('(prefers-reduced-motion: reduce)').matches) {
  const observer = new IntersectionObserver(entries => {
    for (const entry of entries) {
      if (!entry.isIntersecting) continue;
      entry.target.animate([{opacity: .4}, {opacity: 1}], {duration: 320});
      observer.unobserve(entry.target);
    }
  });
  document.querySelectorAll('.reveal').forEach(element => observer.observe(element));
}
