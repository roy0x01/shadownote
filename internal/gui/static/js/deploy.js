(function () {
  const form = document.querySelector('[data-deploy-form]');
  if (!form) return;

  const modeCards = Array.from(form.querySelectorAll('[data-mode-card]'));
  const themeCards = Array.from(form.querySelectorAll('.theme-pick'));
  const reviewMode = form.querySelector('[data-review-mode]');
  const reviewTheme = form.querySelector('[data-review-theme]');
  const submitBtn = form.querySelector('[data-submit]');
  const resetBtn = form.querySelector('[data-reset-flow]');

  function cleanThemeName(text) {
    return (text || '')
      .replace(/\s+active\s*$/i, '')
      .replace(/\s+selected\s*$/i, '')
      .replace(/\s+current\s*$/i, '')
      .replace(/\s+invalid\s*$/i, '')
      .trim();
  }

  function selectedMode() {
    return form.querySelector('input[name="action"]:checked');
  }

  function selectedTheme() {
    return form.querySelector('input[name="theme"]:checked');
  }

  function modeLabel(value) {
    return value === 'export' ? 'Local export' : value === 'local' ? 'Local preview' : 'select an action';
  }

  function syncThemeSelection(input) {
    themeCards.forEach((card) => {
      const radio = card.querySelector('input[type="radio"][name="theme"]');
      const selected = radio === input && radio.checked;
      card.classList.toggle('is-selected', selected);
      card.setAttribute('aria-checked', selected ? 'true' : 'false');

      const nameEl = card.querySelector('.theme-name');
      if (!nameEl) return;
      const baseName = cleanThemeName(nameEl.textContent);
      const invalid = radio && radio.disabled;
      nameEl.innerHTML = '';
      nameEl.appendChild(document.createTextNode(baseName));
      if (selected) {
        const badge = document.createElement('span');
        badge.className = 'theme-current';
        badge.textContent = 'selected';
        nameEl.appendChild(document.createTextNode(' '));
        nameEl.appendChild(badge);
      }
      if (invalid) {
        const badge = document.createElement('span');
        badge.className = 'theme-current warn';
        badge.textContent = 'invalid';
        nameEl.appendChild(document.createTextNode(' '));
        nameEl.appendChild(badge);
      }
    });
  }

  function syncModeSelection(input) {
    modeCards.forEach((card) => {
      const radio = card.querySelector('input[type="radio"][name="action"]');
      const selected = radio === input && radio.checked;
      card.classList.toggle('is-selected', selected);
      card.setAttribute('aria-checked', selected ? 'true' : 'false');
    });
  }

  function updateReview() {
    const mode = selectedMode();
    const theme = selectedTheme();
    form.classList.toggle('has-mode', !!mode);
    form.classList.toggle('has-theme', !!theme);
    form.classList.toggle('ready-review', !!mode && !!theme);
    if (reviewMode) reviewMode.textContent = modeLabel(mode && mode.value);
    if (reviewTheme) reviewTheme.textContent = theme ? theme.value : 'select a theme';
    if (submitBtn) submitBtn.textContent = mode && mode.value === 'export' ? 'Export archive' : 'Start local preview';
  }

  modeCards.forEach((card) => {
    const input = card.querySelector('input[type="radio"][name="action"]');
    if (!input) return;
    card.setAttribute('role', 'radio');
    card.setAttribute('tabindex', input.checked ? '0' : '-1');
    card.setAttribute('aria-checked', input.checked ? 'true' : 'false');
    input.addEventListener('change', () => {
      if (!input.checked) return;
      syncModeSelection(input);
      modeCards.forEach((c) => c.setAttribute('tabindex', c === card ? '0' : '-1'));
      updateReview();
    });
    card.addEventListener('keydown', (e) => {
      if (e.key !== 'Enter' && e.key !== ' ') return;
      e.preventDefault();
      input.checked = true;
      input.dispatchEvent(new Event('change', { bubbles: true }));
    });
  });

  themeCards.forEach((card) => {
    const input = card.querySelector('input[type="radio"][name="theme"]');
    if (!input || input.disabled) return;
    card.setAttribute('role', 'radio');
    card.setAttribute('tabindex', input.checked ? '0' : '-1');
    card.setAttribute('aria-checked', input.checked ? 'true' : 'false');
    input.addEventListener('change', () => {
      if (!input.checked) return;
      syncThemeSelection(input);
      themeCards.forEach((c) => c.setAttribute('tabindex', c === card ? '0' : '-1'));
      updateReview();
    });
    card.addEventListener('keydown', (e) => {
      if (e.key !== 'Enter' && e.key !== ' ') return;
      e.preventDefault();
      input.checked = true;
      input.dispatchEvent(new Event('change', { bubbles: true }));
    });
  });

  if (resetBtn) resetBtn.addEventListener('click', () => {
    modeCards.forEach((card) => {
      const radio = card.querySelector('input[type="radio"][name="action"]');
      if (radio) radio.checked = false;
      card.classList.remove('is-selected');
      card.setAttribute('aria-checked', 'false');
      card.setAttribute('tabindex', '-1');
    });
    themeCards.forEach((card) => {
      const radio = card.querySelector('input[type="radio"][name="theme"]');
      if (radio) radio.checked = false;
      card.classList.remove('is-selected');
      card.setAttribute('aria-checked', 'false');
      card.setAttribute('tabindex', '-1');
    });
    if (modeCards[0]) modeCards[0].setAttribute('tabindex', '0');
    if (themeCards[0]) themeCards[0].setAttribute('tabindex', '0');
    updateReview();
  });

  form.addEventListener('submit', (e) => {
    const mode = selectedMode();
    if (!mode) {
      e.preventDefault();
      if (window.SN && window.SN.toast) window.SN.toast('Choose an action first.', { type: 'error' });
      return;
    }
    const theme = selectedTheme();
    if (!theme) {
      e.preventDefault();
      if (window.SN && window.SN.toast) window.SN.toast('Choose a theme first.', { type: 'error' });
      return;
    }
    if (submitBtn && window.SN && window.SN.busy) {
      const restoreBusy = window.SN.busy(submitBtn, mode.value === 'export' ? 'exporting…' : 'starting…');
      if (mode.value === 'export') window.setTimeout(restoreBusy, 4000);
    }
  });

  syncThemeSelection(selectedTheme());
  updateReview();
})();
