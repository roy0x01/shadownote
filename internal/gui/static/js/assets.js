(function () {
  const grid = document.getElementById('asset-grid');
  if (!grid) return;
  const form = document.getElementById('asset-upload');
  const fileInput = document.getElementById('asset-file');
  const dropzone = document.getElementById('asset-dropzone');
  const picker = document.getElementById('asset-picker');
  const status = document.getElementById('asset-status');
  const count = document.getElementById('asset-count');
  const pager = document.getElementById('asset-pagination');
  const searchEl = document.getElementById('asset-search');
  const resultEl = document.getElementById('asset-upload-result');
  const typePills = Array.from(document.querySelectorAll('.asset-type-filters .filter-pill'));
  const pageSize = 12;
  let currentPage = 1;
  let allAssets = [];
  let currentAssets = [];
  let query = '';
  let typeFilter = 'all';

  function setStatus(msg, err) {
    if (!status) return;
    status.textContent = msg;
    status.className = 'save-state ' + (err ? 'err' : 'ok');
  }
  function notify(msg, type) { if (window.SN && window.SN.toast) window.SN.toast(msg, { type: type || 'info' }); }
  function copy(text, okMsg) {
    if (!navigator.clipboard) { setStatus('clipboard unavailable', true); notify('clipboard unavailable', 'error'); return; }
    navigator.clipboard.writeText(text || '').then(
      () => { setStatus(okMsg); notify(okMsg, 'ok'); },
      () => { setStatus('copy failed', true); notify('copy failed', 'error'); }
    );
  }
  async function readJSON(res) {
    if (!res.ok) throw new Error((await res.text()).trim() || res.statusText);
    const text = await res.text();
    if (!text.trim()) return [];
    return JSON.parse(text);
  }
  function normalizeAssets(data) {
    if (Array.isArray(data)) return data.filter(Boolean);
    if (data && Array.isArray(data.assets)) return data.assets.filter(Boolean);
    return [];
  }
  function safeText(v) { return (v == null ? '' : String(v)); }
  function ext(a) { const m = /\.([^.]+)$/.exec(a.name || ''); return m ? m[1].toLowerCase() : ''; }
  function isPdf(a) { return /\.pdf$/i.test(a.name || '') || (a.mime || '').indexOf('pdf') >= 0; }
  function assetKind(a) { return a.image ? 'images' : (isPdf(a) ? 'pdfs' : 'other'); }
  function friendlyType(a) { return a.image ? (a.mime || 'image') : (isPdf(a) ? 'application/pdf' : (a.mime || 'file')); }
  function mdFor(a) { return a.image ? '![' + safeText(a.name).replace(/\.[^.]+$/, '') + '](' + a.url + ')' : '[' + a.name + '](' + a.url + ')'; }
  function fmtSize(n) { n = Number(n) || 0; return n < 1024 ? n + ' B' : (n < 1048576 ? Math.ceil(n / 1024) + ' KB' : (n / 1048576).toFixed(1) + ' MB'); }
  function fmtDate(d) { if (!d) return ''; const t = new Date(d); return isNaN(t) ? '' : t.toLocaleString(); }
  function usageText(a) {
    if (a.used) { const n = a.usage_count || (a.used_by ? a.used_by.length : 1); return 'Used in ' + n + ' document' + (n === 1 ? '' : 's'); }
    return 'Unused — not published unless referenced';
  }

  function thumbImg(a) {
    const img = document.createElement('img');
    img.loading = 'lazy';
    img.alt = '';
    img.src = a.thumb_url || a.url;
    img.addEventListener('error', () => {
      if (img.dataset.fellBack) { img.replaceWith(fileGlyph(a)); return; }
      img.dataset.fellBack = '1';
      img.src = a.url;
    }, { once: false });
    return img;
  }
  function fileGlyph(a) {
    const span = document.createElement('span');
    span.className = 'asset-file-glyph';
    span.textContent = (ext(a) || 'file').toUpperCase();
    return span;
  }

  function applyFilters() {
    currentAssets = allAssets.filter(a => {
      if (typeFilter === 'unused') { if (a.used) return false; }
      else if (typeFilter !== 'all' && assetKind(a) !== typeFilter) return false;
      if (query) {
        const hay = [a.name, a.mime, ext(a), (a.used_by || []).join(' '), a.used ? 'used' : 'unused'].join(' ').toLowerCase();
        if (hay.indexOf(query) < 0) return false;
      }
      return true;
    });
  }

  function renderPagination() {
    if (!pager) return;
    const pages = Math.ceil(currentAssets.length / pageSize);
    if (pages <= 1) { pager.innerHTML = ''; pager.hidden = true; return; }
    pager.hidden = false;
    currentPage = Math.min(Math.max(1, currentPage), pages);
    const parts = ['<button type="button" data-page="prev"' + (currentPage === 1 ? ' disabled' : '') + '>prev</button>'];
    for (let i = 1; i <= pages; i++) parts.push('<button type="button" data-page="' + i + '"' + (i === currentPage ? ' class="active" aria-current="page"' : '') + '>' + i + '</button>');
    parts.push('<button type="button" data-page="next"' + (currentPage === pages ? ' disabled' : '') + '>next</button>');
    pager.innerHTML = parts.join('');
    pager.querySelectorAll('button[data-page]').forEach(btn => btn.addEventListener('click', () => {
      const action = btn.dataset.page;
      if (action === 'prev') currentPage -= 1;
      else if (action === 'next') currentPage += 1;
      else currentPage = Number(action) || 1;
      renderAssets();
    }));
  }

  function appendDef(dl, term, value, asCode) {
    const dt = document.createElement('dt');
    dt.textContent = term;
    const dd = document.createElement('dd');
    if (asCode) {
      const code = document.createElement('code');
      code.textContent = value || '';
      dd.appendChild(code);
    } else {
      dd.textContent = value || '';
    }
    dl.appendChild(dt);
    dl.appendChild(dd);
  }

  function openPreview(a) {
    if (!(window.SN && window.SN.modal)) return;
    const wrap = document.createElement('div');
    wrap.className = 'asset-preview-dialog';

    const media = document.createElement('div');
    media.className = 'asset-preview-media' + (a.image ? '' : ' asset-preview-file');
    if (a.image) {
      const img = document.createElement('img');
      img.src = a.url || '';
      img.alt = safeText(a.name);
      media.appendChild(img);
    } else {
      media.textContent = (ext(a) || 'file').toUpperCase();
    }
    wrap.appendChild(media);

    const dl = document.createElement('dl');
    dl.className = 'asset-preview-meta';
    appendDef(dl, 'filename', safeText(a.name));
    appendDef(dl, 'url', safeText(a.url), true);
    appendDef(dl, 'markdown', safeText(mdFor(a)), true);
    appendDef(dl, 'size', fmtSize(a.size));
    appendDef(dl, 'type', safeText(friendlyType(a)));
    appendDef(dl, 'modified', fmtDate(a.modified));
    if (a.used_by && a.used_by.length) {
      const dt = document.createElement('dt');
      dt.textContent = 'used by';
      const dd = document.createElement('dd');
      a.used_by.forEach((item, i) => {
        if (i) dd.appendChild(document.createElement('br'));
        dd.appendChild(document.createTextNode(safeText(item)));
      });
      dl.appendChild(dt);
      dl.appendChild(dd);
    } else {
      appendDef(dl, 'usage', usageText(a));
    }
    wrap.appendChild(dl);

    const actions = document.createElement('div');
    actions.className = 'asset-preview-actions';
    const mdBtn = document.createElement('button');
    mdBtn.type = 'button';
    mdBtn.id = 'ap-md';
    mdBtn.className = 'primary';
    mdBtn.textContent = 'copy markdown';
    const urlBtn = document.createElement('button');
    urlBtn.type = 'button';
    urlBtn.id = 'ap-url';
    urlBtn.textContent = 'copy url';
    const open = document.createElement('a');
    open.id = 'ap-open';
    open.className = 'btn';
    open.href = a.url || '#';
    open.target = '_blank';
    open.rel = 'noopener';
    open.textContent = 'open asset';
    const delBtn = document.createElement('button');
    delBtn.type = 'button';
    delBtn.id = 'ap-del';
    delBtn.className = 'danger';
    delBtn.textContent = 'delete';
    actions.append(mdBtn, urlBtn, open, delBtn);
    wrap.appendChild(actions);

    const modal = window.SN.modal({ title: a.name, body: wrap, className: 'asset-preview-modal' });
    mdBtn.addEventListener('click', () => copy(mdFor(a), 'markdown copied ✓'));
    urlBtn.addEventListener('click', () => copy(a.url || '', 'url copied ✓'));
    delBtn.addEventListener('click', async () => { if (await confirmDelete(a)) { modal.close(); } });
  }

  async function confirmDelete(a) {
    const name = a.name || a.path;
    let message = 'Delete ' + name + '?';
    if (a.used && a.used_by && a.used_by.length) {
      message = 'Delete ' + name + '?\n\nThis asset is used in:\n- ' + a.used_by.join('\n- ') + '\n\nDeleting it may break published content.';
    }
    const ok = await (window.SN && window.SN.confirm
      ? window.SN.confirm({ title: 'delete asset', message: message, confirmText: 'delete asset', cancelText: 'keep asset', danger: true })
      : Promise.resolve(window.confirm(message)));
    if (!ok) return false;
    try {
      await readJSON(await fetch('/api/assets/' + encodeURIComponent(a.path || a.name), { method: 'DELETE' }));
      notify('deleted ' + name, 'ok');
      await load();
      return true;
    } catch (e) { setStatus('delete failed: ' + e.message, true); notify('delete failed: ' + e.message, 'error'); return false; }
  }

  function card(a) {
    const div = document.createElement('article');
    div.className = 'asset-card';
    const thumbBtn = document.createElement('button');
    thumbBtn.type = 'button';
    thumbBtn.className = 'asset-thumb';
    thumbBtn.setAttribute('aria-label', 'preview ' + safeText(a.name));
    thumbBtn.appendChild(a.image ? thumbImg(a) : fileGlyph(a));
    thumbBtn.addEventListener('click', () => openPreview(a));

    const name = document.createElement('strong');
    name.className = 'asset-name';
    name.textContent = a.name || 'asset';

    const meta = document.createElement('small');
    meta.className = 'asset-meta';
    meta.textContent = fmtSize(a.size) + ' · ' + friendlyType(a) + (a.modified ? ' · ' + fmtDate(a.modified) : '');

    const usage = document.createElement('span');
    usage.className = 'asset-usage ' + (a.used ? 'is-used' : 'is-unused');
    usage.textContent = usageText(a);

    const actions = document.createElement('div');
    actions.className = 'asset-actions';
    const mk = (label, cls, fn) => { const b = document.createElement('button'); b.type = 'button'; if (cls) b.className = cls; b.textContent = label; b.addEventListener('click', fn); return b; };
    actions.appendChild(mk('preview', '', () => openPreview(a)));
    actions.appendChild(mk('copy md', '', () => copy(mdFor(a), 'markdown copied ✓')));
    actions.appendChild(mk('copy url', '', () => copy(a.url || '', 'url copied ✓')));
    if (!a.image) {
      const open = document.createElement('a');
      open.className = 'btn'; open.textContent = 'open'; open.href = a.url || '#';
      open.target = '_blank'; open.rel = 'noopener';
      actions.appendChild(open);
    }
    actions.appendChild(mk('delete', 'danger', () => confirmDelete(a)));

    div.appendChild(thumbBtn);
    div.appendChild(name);
    div.appendChild(meta);
    div.appendChild(usage);
    div.appendChild(actions);
    return div;
  }

  function renderAssets() {
    applyFilters();
    grid.innerHTML = '';
    grid.classList.toggle('is-empty', !currentAssets.length);
    if (!currentAssets.length) {
      const filtered = query || typeFilter !== 'all';
      grid.innerHTML = '<div class="empty-state asset-empty"><div class="empty-mark">▰</div><strong>' +
        (filtered ? 'No matching assets.' : 'No assets yet.') + '</strong><p>' +
        (filtered ? 'Try changing the search or filter.' : 'Drop images or files above to add your first asset.') + '</p></div>';
      if (pager) pager.hidden = true;
      setStatus(allAssets.length ? '0 of ' + allAssets.length + ' assets' : 'no assets yet');
      return;
    }
    const pages = Math.max(1, Math.ceil(currentAssets.length / pageSize));
    currentPage = Math.min(Math.max(1, currentPage), pages);
    const start = (currentPage - 1) * pageSize;
    currentAssets.slice(start, start + pageSize).forEach(a => grid.appendChild(card(a)));
    renderPagination();
    setStatus(currentAssets.length + (currentAssets.length === allAssets.length ? '' : ' of ' + allAssets.length) + ' assets');
  }

  async function uploadFiles(files) {
    files = Array.from(files || []);
    if (!files.length) return;
    setStatus('uploading ' + files.length + ' file' + (files.length === 1 ? '' : 's') + '…');
    if (resultEl) { resultEl.hidden = false; resultEl.textContent = 'uploading ' + files.length + '…'; }
    let ok = 0; const failed = [];
    for (const file of files) {
      try {
        const fd = new FormData(); fd.append('file', file);
        await readJSON(await fetch('/api/assets', { method: 'POST', body: fd }));
        ok++;
      } catch (e) { failed.push(file.name + ' (' + e.message + ')'); }
    }
    if (fileInput) fileInput.value = '';
    if (picker) picker.textContent = 'or click to choose files';
    currentPage = 1;
    await load();
    if (!failed.length) {
      setStatus('uploaded ' + ok + ' ✓');
      if (resultEl) resultEl.textContent = ok + ' uploaded';
      notify(ok + ' file' + (ok === 1 ? '' : 's') + ' uploaded', 'ok');
    } else {
      const msg = ok + ' uploaded, ' + failed.length + ' failed';
      setStatus(msg, true);
      if (resultEl) resultEl.textContent = msg + ' — ' + failed.join('; ');
      notify(msg, 'error');
    }
  }

  async function load() {
    try {
      allAssets = normalizeAssets(await readJSON(await fetch('/api/assets')));
      allAssets.sort((a, b) => new Date(b.modified || 0) - new Date(a.modified || 0));
      if (count) count.textContent = allAssets.length + ' asset' + (allAssets.length === 1 ? '' : 's');
      renderAssets();
    } catch (e) { setStatus('load failed: ' + e.message, true); notify('load failed: ' + e.message, 'error'); }
  }

  if (searchEl) searchEl.addEventListener('input', () => { query = searchEl.value.trim().toLowerCase(); currentPage = 1; renderAssets(); });
  typePills.forEach(p => p.addEventListener('click', () => {
    typeFilter = p.dataset.assetType;
    typePills.forEach(x => { const on = x === p; x.classList.toggle('active', on); x.setAttribute('aria-pressed', String(on)); });
    currentPage = 1; renderAssets();
  }));

  if (form) form.addEventListener('submit', e => { e.preventDefault(); uploadFiles(fileInput && fileInput.files); });
  if (dropzone) {
    dropzone.addEventListener('click', e => {
      if (e.target.closest('button')) return;
      if (fileInput) fileInput.click();
    });
    dropzone.addEventListener('keydown', e => {
      if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); if (fileInput) fileInput.click(); }
    });
    ['dragenter', 'dragover'].forEach(evt => dropzone.addEventListener(evt, e => { e.preventDefault(); dropzone.classList.add('dragover'); }));
    ['dragleave', 'drop'].forEach(evt => dropzone.addEventListener(evt, e => { e.preventDefault(); dropzone.classList.remove('dragover'); }));
    dropzone.addEventListener('drop', e => uploadFiles(e.dataTransfer.files));
  }
  if (fileInput) fileInput.addEventListener('change', () => { if (picker) picker.textContent = fileInput.files.length ? (fileInput.files.length === 1 ? fileInput.files[0].name : fileInput.files.length + ' files selected') : 'or click to choose files'; });
  load();
})();
