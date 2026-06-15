(function () {
  const shell = document.querySelector('.settings-edit');
  if (!shell) return;

  shell.classList.add('settings-enhanced');

  const buttons = Array.from(document.querySelectorAll('[data-open-setting]'));
  const groups = Array.from(document.querySelectorAll('[data-setting-group]'));

  function showGroup(name) {
    let matched = false;
    groups.forEach((g) => {
      const on = g.dataset.settingGroup === name;
      if (on) matched = true;
      g.classList.toggle('active', on);
    });
    buttons.forEach((b) => {
      const on = b.dataset.openSetting === name;
      b.classList.toggle('active', on);
      b.setAttribute('aria-current', on ? 'true' : 'false');
    });
    return matched;
  }

  buttons.forEach((b) =>
    b.addEventListener('click', () => {
      showGroup(b.dataset.openSetting);
      if (history.replaceState) history.replaceState(null, '', '#grp-' + b.dataset.openSetting);
    })
  );

  buttons.forEach((b, i) => {
    b.addEventListener('keydown', (e) => {
      let next = -1;
      if (e.key === 'ArrowDown' || e.key === 'ArrowRight') next = (i + 1) % buttons.length;
      else if (e.key === 'ArrowUp' || e.key === 'ArrowLeft') next = (i - 1 + buttons.length) % buttons.length;
      if (next >= 0) { e.preventDefault(); buttons[next].focus(); buttons[next].click(); }
    });
  });

  const initial =
    (shell.dataset.activeSection || '').trim() ||
    (location.hash || '').replace('#grp-', '') ||
    (groups[0] && groups[0].dataset.settingGroup);
  if (!initial || !showGroup(initial)) {
    if (groups[0]) showGroup(groups[0].dataset.settingGroup);
  }

  function el(name) { return document.querySelector('[name="' + name + '"]'); }
  function val(name) { const e = el(name); return e ? (e.value || '').trim() : ''; }
  function checked(name) { const e = el(name); return !!(e && e.checked); }
  function setFlow(selector, visible, root) {
    (root || document).querySelectorAll(selector).forEach((e) => { e.hidden = !visible; });
  }

  function syncFlows() {
    setFlow('[data-flow="assets-inline"]', checked('assets.inline_base64'));
  }

  document.addEventListener('change', (e) => {
    if (!e.target.name) return;
    if (/^(assets\.inline_base64)/.test(e.target.name)) syncFlows();
  });
  syncFlows();

  const seo = {
    title: document.querySelector('[data-seo-title]'),
    desc: document.querySelector('[data-seo-desc]'),
    url: document.querySelector('[data-seo-url]'),
  };
  function syncSeo() {
    if (seo.title) seo.title.textContent = val('meta.title') || val('blog.title') || 'site title';
    if (seo.desc) seo.desc.textContent = val('meta.description') || val('blog.description') || 'site description';
    if (seo.url) seo.url.textContent = val('meta.canonical') || val('blog.base_url') || 'https://example.com';
  }
  ['meta.title', 'meta.description', 'meta.canonical'].forEach((n) => {
    const e = el(n);
    if (e) e.addEventListener('input', syncSeo);
  });
  syncSeo();


})();
