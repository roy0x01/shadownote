(function () {
  'use strict';

  const key = 'shadownote.theme';
  const btn = document.getElementById('theme-toggle');

  function currentTheme() {
    return document.documentElement.getAttribute('data-theme') === 'day' ? 'day' : 'night';
  }

  function saveTheme(theme) {
    const next = theme === 'day' ? 'day' : 'night';
    document.documentElement.setAttribute('data-theme', next);
    try { localStorage.setItem(key, next); } catch (e) {}
    render();
  }

  function render() {
    if (!btn) return;
    const day = currentTheme() === 'day';
    const label = btn.querySelector('.theme-toggle-label');
    if (label) label.textContent = day ? 'day mode' : 'night mode';
    btn.setAttribute('aria-pressed', String(day));
    btn.setAttribute('aria-label', day ? 'switch to night mode' : 'switch to day mode');
    btn.title = day ? 'switch to night mode' : 'switch to day mode';
  }

  if (!btn) return;
  render();
  btn.addEventListener('click', function () {
    saveTheme(currentTheme() === 'day' ? 'night' : 'day');
  });
})();
