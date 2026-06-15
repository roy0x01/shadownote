(function () {
  'use strict';

  const reduceMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;

  (function navCollapse() {
    const shell = document.getElementById('app-shell');
    const brand = document.getElementById('brand-toggle');
    const side = document.getElementById('side-nav');
    if (!shell || !brand || !side) return;

    const key = 'shadownote.nav.hidden';
    const collapsible = Array.from(side.querySelectorAll('.side-nav, .nav-newdraft'));
    const focusables = Array.from(side.querySelectorAll('.side-nav a, .nav-newdraft'));

    function apply(hidden) {
      shell.classList.toggle('nav-hidden', hidden);
      shell.classList.toggle('nav-open', !hidden);
      brand.setAttribute('aria-expanded', String(!hidden));
      brand.title = hidden ? 'show navigation' : 'hide navigation';
      collapsible.forEach(el => {
        el.setAttribute('aria-hidden', String(hidden));
        el.inert = hidden;
      });
      focusables.forEach(el => {
        if (hidden) el.setAttribute('tabindex', '-1');
        else el.removeAttribute('tabindex');
      });
    }

    let hidden = false;
    try { hidden = localStorage.getItem(key) === '1'; } catch (e) {}
    apply(hidden);

    brand.addEventListener('click', () => {
      const nextHidden = !shell.classList.contains('nav-hidden');
      try { localStorage.setItem(key, nextHidden ? '1' : '0'); } catch (e) {}
      apply(nextHidden);
    });
  })();

  (function themeToggle() {
    const btn = document.getElementById('theme-toggle');
    const key = 'shadownote.theme';
    function readTheme() {
      try {
        const stored = localStorage.getItem(key);
        return stored === 'day' ? 'day' : 'night';
      } catch (e) {
        return document.documentElement.getAttribute('data-theme') === 'day' ? 'day' : 'night';
      }
    }

    function persistTheme(theme) {
      const next = theme === 'day' ? 'day' : 'night';
      document.documentElement.setAttribute('data-theme', next);
      try {
        localStorage.setItem(key, next);
      } catch (e) {}
    }

    function isDay() {
      return document.documentElement.getAttribute('data-theme') === 'day';
    }

    function render() {
      if (!btn) return;
      const label = btn.querySelector('.theme-toggle-label');
      const day = isDay();
      if (label) label.textContent = day ? 'day mode' : 'night mode';
      btn.setAttribute('aria-pressed', String(day));
      btn.setAttribute('aria-label', day ? 'switch to night mode' : 'switch to day mode');
      btn.title = day ? 'switch to night mode' : 'switch to day mode';
    }

    persistTheme(readTheme());
    render();

    if (!btn) return;
    btn.addEventListener('click', () => {
      persistTheme(isDay() ? 'night' : 'day');
      render();
    });
  })();

  function buildModal(opts) {
    const overlay = document.createElement('div');
    overlay.className = 'sn-modal-overlay';
    overlay.setAttribute('role', 'dialog');
    overlay.setAttribute('aria-modal', 'true');
    if (opts.title) overlay.setAttribute('aria-label', opts.title);

    const dialog = document.createElement('div');
    dialog.className = 'sn-modal' + (opts.className ? ' ' + opts.className : '');

    const head = document.createElement('div');
    head.className = 'sn-modal-head';
    const h = document.createElement('strong');
    h.textContent = opts.title || '';
    const close = document.createElement('button');
    close.type = 'button';
    close.className = 'sn-modal-close';
    close.setAttribute('aria-label', 'close');
    close.textContent = '✕';
    head.appendChild(h);
    head.appendChild(close);

    const bodyWrap = document.createElement('div');
    bodyWrap.className = 'sn-modal-body';
    if (opts.body) bodyWrap.appendChild(opts.body);

    dialog.appendChild(head);
    dialog.appendChild(bodyWrap);
    overlay.appendChild(dialog);

    const prevFocus = document.activeElement;
    function destroy() {
      document.removeEventListener('keydown', onKey, true);
      overlay.remove();
      if (prevFocus && prevFocus.focus) prevFocus.focus();
    }
    function onKey(e) {
      if (e.key === 'Escape') { e.stopPropagation(); destroy(); return; }
      if (e.key === 'Tab') {
        const f = dialog.querySelectorAll('a[href],button:not([disabled]),input,select,textarea,[tabindex]:not([tabindex="-1"])');
        if (!f.length) return;
        const first = f[0], last = f[f.length - 1];
        if (e.shiftKey && document.activeElement === first) { e.preventDefault(); last.focus(); }
        else if (!e.shiftKey && document.activeElement === last) { e.preventDefault(); first.focus(); }
      }
    }
    close.addEventListener('click', destroy);
    overlay.addEventListener('mousedown', e => { if (e.target === overlay) destroy(); });
    document.addEventListener('keydown', onKey, true);
    document.body.appendChild(overlay);
    const firstField = dialog.querySelector('input,select,textarea,button');
    if (firstField) firstField.focus();

    return { overlay, dialog, body: bodyWrap, close: destroy };
  }


  function confirmDialog(opts) {
    opts = opts || {};
    if (!document.body) return Promise.resolve(window.confirm(opts.message || 'Continue?'));
    return new Promise(resolve => {
      const body = document.createElement('div');
      body.className = 'sn-confirm';
      const msg = document.createElement('p');
      msg.className = 'sn-confirm-message';
      msg.textContent = opts.message || 'Continue?';
      const actions = document.createElement('div');
      actions.className = 'sn-confirm-actions';
      const cancel = document.createElement('button');
      cancel.type = 'button';
      cancel.textContent = opts.cancelText || 'cancel';
      const ok = document.createElement('button');
      ok.type = 'button';
      ok.className = opts.danger ? 'danger' : 'primary';
      ok.textContent = opts.confirmText || 'continue';
      actions.appendChild(cancel);
      actions.appendChild(ok);
      body.appendChild(msg);
      body.appendChild(actions);
      const modal = buildModal({ title: opts.title || 'confirm action', body: body, className: 'sn-confirm-modal' });
      let settled = false;
      function done(value) {
        if (settled) return;
        settled = true;
        modal.close();
        resolve(value);
      }
      cancel.addEventListener('click', () => done(false));
      ok.addEventListener('click', () => done(true));
      modal.overlay.addEventListener('mousedown', e => { if (e.target === modal.overlay) done(false); });
      document.addEventListener('keydown', function onKey(e) {
        if (settled) { document.removeEventListener('keydown', onKey, true); return; }
        if (e.key === 'Escape') { e.preventDefault(); document.removeEventListener('keydown', onKey, true); done(false); }
      }, true);
      cancel.focus();
    });
  }

  function registerUnsavedGuard(isDirty, options) {
    options = options || {};
    if (typeof isDirty !== 'function') return;
    window.addEventListener('beforeunload', e => {
      if (!isDirty()) return;
      e.preventDefault();
      e.returnValue = '';
    });
    document.addEventListener('click', async e => {
      const link = e.target.closest && e.target.closest('a[href]');
      if (!link || !isDirty()) return;
      if (link.target && link.target !== '_self') return;
      if (link.hasAttribute('download')) return;
      const href = link.getAttribute('href') || '';
      if (!href || href.startsWith('#') || href.startsWith('javascript:')) return;
      const next = new URL(href, window.location.href);
      if (next.origin !== window.location.origin) return;
      e.preventDefault();
      const ok = await confirmDialog({
        title: options.title || 'unsaved changes',
        message: options.message || 'You have unsaved changes. Leave this page anyway?',
        confirmText: options.confirmText || 'leave page',
        cancelText: options.cancelText || 'stay here',
        danger: true
      });
      if (ok) window.location.href = next.href;
    }, true);
  }

  let toastWrap = null;
  function toast(message, opts) {
    opts = opts || {};
    if (!document.body) return;
    if (!toastWrap) {
      toastWrap = document.createElement('div');
      toastWrap.className = 'sn-toast-wrap';
      toastWrap.setAttribute('aria-live', 'polite');
      toastWrap.setAttribute('aria-atomic', 'false');
      document.body.appendChild(toastWrap);
    }
    const type = opts.type || 'info';
    const el = document.createElement('div');
    el.className = 'sn-toast sn-toast-' + type;
    el.setAttribute('role', type === 'error' ? 'alert' : 'status');
    el.textContent = message;
    toastWrap.appendChild(el);
    requestAnimationFrame(() => el.classList.add('in'));
    const timeout = opts.timeout != null ? opts.timeout : (type === 'error' ? 6000 : 3200);
    let timer = null;
    function dismiss() {
      if (!el.parentNode) return;
      el.classList.remove('in');
      const remove = () => el.remove();
      if (reduceMotion) remove(); else setTimeout(remove, 180);
    }
    if (timeout > 0) timer = setTimeout(dismiss, timeout);
    el.addEventListener('click', () => { if (timer) clearTimeout(timer); dismiss(); });
    return dismiss;
  }

  function busy(el, label) {
    if (!el) return function () {};
    if (el.dataset.snBusy === '1') return function () {};
    el.dataset.snBusy = '1';
    const prevDisabled = el.disabled;
    const prevLabel = el.textContent;
    el.disabled = true;
    el.setAttribute('aria-busy', 'true');
    if (label != null) el.textContent = label;
    return function restore() {
      el.disabled = prevDisabled;
      el.removeAttribute('aria-busy');
      if (label != null) el.textContent = prevLabel;
      delete el.dataset.snBusy;
    };
  }

  function assetPicker(onSelect) {
    const wrap = document.createElement('div');
    wrap.className = 'asset-picker';
    wrap.innerHTML =
      '<input type="search" class="asset-picker-search" placeholder="search filename…" aria-label="search assets">' +
      '<div class="asset-picker-grid" role="listbox" aria-label="assets">loading…</div>';
    const search = wrap.querySelector('.asset-picker-search');
    const grid = wrap.querySelector('.asset-picker-grid');
    const modal = buildModal({ title: 'insert asset', body: wrap, className: 'asset-picker-modal' });
    let all = [];

    function isImage(name) { return /\.(png|jpe?g|gif|webp|svg|avif)$/i.test(name); }
    function markdownFor(a) {
      const url = a.url || ('/assets/' + a.name);
      return isImage(a.name) ? `![${a.name}](${url})` : `[${a.name}](${url})`;
    }
    function render(list) {
      if (!list.length) { grid.innerHTML = '<div class="asset-picker-empty">no matching assets</div>'; return; }
      grid.innerHTML = '';
      list.forEach(a => {
        const url = a.url || ('/assets/' + a.name);
        const cell = document.createElement('button');
        cell.type = 'button';
        cell.className = 'asset-picker-cell';
        cell.setAttribute('role', 'option');
        cell.title = a.name || 'asset';
        const thumb = document.createElement('span');
        thumb.className = isImage(a.name) ? 'thumb' : 'thumb thumb-file';
        if (isImage(a.name)) {
          const img = document.createElement('img');
          img.loading = 'lazy';
          img.src = url;
          img.alt = a.name || 'asset';
          thumb.appendChild(img);
        } else {
          thumb.textContent = '▤';
        }
        const cap = document.createElement('span');
        cap.className = 'cap';
        cap.textContent = a.name || 'asset';
        cell.appendChild(thumb);
        cell.appendChild(cap);
        cell.addEventListener('click', () => { onSelect(markdownFor(a)); modal.close(); });
        grid.appendChild(cell);
      });
    }
    search.addEventListener('input', () => {
      const q = search.value.trim().toLowerCase();
      render(q ? all.filter(a => a.name.toLowerCase().includes(q)) : all);
    });
    fetch('/api/assets').then(r => r.ok ? r.json() : []).then(list => {
      all = list || [];
      render(all);
    }).catch(() => { grid.innerHTML = '<div class="asset-picker-empty">could not load assets</div>'; });

    return modal;
  }

  window.SN = { modal: buildModal, confirm: confirmDialog, guardUnsaved: registerUnsavedGuard, assetPicker: assetPicker, toast: toast, busy: busy, reduceMotion: reduceMotion };
})();
