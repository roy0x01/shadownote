(function () {
  const root = document.querySelector('.editor');
  if (!root) return;

  let slug = root.dataset.slug;

  const titleEl = document.getElementById('ed-title');
  const summaryEl = document.getElementById('ed-summary');
  const tagsEl = document.getElementById('ed-tags');
  const typeEl = document.getElementById('ed-type');
  const textEl = document.getElementById('ed-text');
  const previewEl = document.getElementById('ed-preview');
  const wcEl = document.getElementById('ed-wc');
  const statusEl = document.getElementById('ed-status');
  const readEl = document.getElementById('ed-read');
  const posEl = document.getElementById('ed-pos');
  const pathEl = document.getElementById('ed-path');
  const bodyEl = document.querySelector('.ed-body');
  const typeStatusEl = document.getElementById('ed-type-status');
  const saveBtn = document.getElementById('ed-save');
  const publishBtn = document.getElementById('ed-publish');
  const unpublishBtn = document.getElementById('ed-unpublish');
  const markedMsg = document.getElementById('ed-marked-msg');
  const deleteBtn = document.getElementById('ed-delete');
  const templateEl = document.getElementById('ed-template');
  const focusBtn = document.getElementById('ed-focus');

  let previewTimer = null;
  let dirty = false;

  function escapeHtml(s) {
    return (s || '').replace(/[&<>\"]/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]));
  }

  async function readJSON(res) {
    if (!res.ok) {
      const msg = (await res.text()).trim() || res.statusText || 'request failed';
      throw new Error(msg);
    }
    const type = res.headers.get('Content-Type') || '';
    if (!type.includes('application/json')) return {};
    return res.json();
  }

  async function api(path, opts = {}) {
    opts.headers = Object.assign({ 'Content-Type': 'application/json' }, opts.headers || {});
    return readJSON(await fetch('/api/' + path, opts));
  }

  function updateWordCount() {
    const t = textEl.value.trim();
    const words = t ? t.split(/\s+/).length : 0;
    wcEl.textContent = words + ' words';
    if (readEl) readEl.textContent = Math.max(1, Math.ceil(words / 220)) + ' min read';
  }

  function updateCursor() {
    if (!posEl) return;
    const before = textEl.value.slice(0, textEl.selectionStart);
    const lines = before.split('\n');
    posEl.textContent = 'ln ' + lines.length + ', col ' + lines[lines.length - 1].length;
  }

  function setSaveState(text, cls) {
    if (!statusEl) return;
    statusEl.textContent = text || '';
    statusEl.className = 'save-state' + (cls ? ' ' + cls : '');
  }

  function markDirty() {
    dirty = true;
    setSaveState('unsaved changes', 'dirty');
    if (focusBtn) focusBtn.classList.add('has-unsaved');
  }

  function typeDir(type) {
    return ({ post: 'posts', note: 'notes', draft: 'drafts', research: 'research', page: 'pages' })[type] || 'notes';
  }

  function updatePath() {
    if (pathEl) pathEl.textContent = 'content/' + typeDir(typeEl.value) + '/' + (slug || 'new-document') + '.md';
    if (typeStatusEl) typeStatusEl.textContent = typeEl.value;
    if (publishBtn) {
      const t = typeEl.value;
      const blocked = t === 'note' || t === 'research';
      publishBtn.disabled = blocked;
      if (blocked) {
        publishBtn.title = 'notes and research are private — change the type to draft, post, or page to publish';
      } else {
        publishBtn.removeAttribute('title');
      }
      publishBtn.textContent = t === 'page' ? 'publish page' : 'publish as post';
    }
  }

  function looksLikeShell(text) {
    const lines = (text || '').split('\n').map(l => l.trim()).filter(Boolean);
    if (!lines.length) return false;
    const shell = /^(?:\$\s+|#\s+)?(?:sudo\s+|env\s+|time\s+|xargs\s+|watch\s+)?(?:\.\/|\.\s+|source\s+|export\s+|unset\s+|alias\s+|cd\s+|ls\b|pwd\b|cat\s+|less\s+|tail\s+|head\s+|grep\s+|rg\s+|awk\s+|sed\s+|find\s+|chmod\s+|chown\s+|mkdir\s+|touch\s+|cp\s+|mv\s+|rm\s+|ln\s+|tar\s+|gzip\s+|gunzip\s+|xz\s+|zip\s+|unzip\s+|curl\s+|wget\s+|git\s+|go\s+|cargo\s+|rustc\s+|python3?\s+|pip3?\s+|node\s+|npm\s+|pnpm\s+|yarn\s+|docker\s+|docker-compose\s+|kubectl\s+|helm\s+|terraform\s+|make\s+|cmake\s+|nmap\s+|httpx\s+|dnsx\s+|subfinder\s+|katana\s+|nuclei\s+|naabu\s+|\.\/)/;
    return lines.filter(l => shell.test(l) || /^\w+=/.test(l)).length >= Math.max(1, Math.ceil(lines.length * 0.6));
  }

  function shellTokenHTML(token, index) {
    const esc = escapeHtml(token);
    if (!token) return '';
    if (/^#/.test(token)) return '<span class="shell-comment">' + esc + '</span>';
    if (/^[|&;<>()]+$/.test(token)) return '<span class="shell-op">' + esc + '</span>';
    if (/^\$?[A-Z_][A-Z0-9_]*=/.test(token) || /^\$[{(]?[A-Za-z_][A-Za-z0-9_]*[})]?$/.test(token)) return '<span class="shell-env">' + esc + '</span>';
    if (/^--?[A-Za-z0-9][A-Za-z0-9_-]*(=.*)?$/.test(token)) return '<span class="shell-flag">' + esc + '</span>';
    if (/^['"].*['"]$/.test(token)) return '<span class="shell-string">' + esc + '</span>';
    if (/^(?:\.\/|\.\.\/|\/|~\/|[A-Za-z0-9_.-]+\/)/.test(token)) return '<span class="shell-path">' + esc + '</span>';
    if (index === 0 || (index === 1 && /^(?:sudo|env|time|watch|xargs)$/.test(token))) {
      if (/^(?:cd|ls|pwd|cat|less|tail|head|grep|rg|awk|sed|find|chmod|chown|mkdir|touch|cp|mv|rm|ln|tar|gzip|gunzip|xz|zip|unzip|export|unset|alias|source)$/.test(token)) return '<span class="shell-builtin">' + esc + '</span>';
      return '<span class="shell-command">' + esc + '</span>';
    }
    return esc;
  }

  function highlightShellLine(line) {
    const commentAt = line.search(/(^|\s)#/);
    const code = commentAt >= 0 ? line.slice(0, commentAt) : line;
    const comment = commentAt >= 0 ? line.slice(commentAt) : '';
    const parts = code.match(/'[^']*'|"[^"]*"|\S+|\s+/g) || [];
    let tokenIndex = 0;
    const html = parts.map(part => {
      if (/^\s+$/.test(part)) return part.replace(/ /g, '&nbsp;');
      const out = shellTokenHTML(part, tokenIndex);
      tokenIndex++;
      return out;
    }).join('');
    return html + (comment ? '<span class="shell-comment">' + escapeHtml(comment) + '</span>' : '');
  }

  function enhanceShellBlocks() {
    previewEl.querySelectorAll('pre').forEach(pre => {
      if (pre.dataset.shellEnhanced === 'true' || pre.closest('.md-mermaid-fallback')) return;
      const code = pre.querySelector('code') || pre;
      const classText = (code.className || '') + ' ' + (pre.className || '');
      const langShell = /language-(?:bash|sh|shell|zsh|console|terminal)/i.test(classText) || /\b(?:bash|shell|console|terminal)\b/i.test(code.getAttribute('data-lang') || '');
      const text = code.textContent || '';
      if (!langShell && !looksLikeShell(text)) return;
      pre.dataset.shellEnhanced = 'true';
      code.classList.add('shell-code');
      code.innerHTML = text.split('\n').map(line => '<span class="shell-line">' + highlightShellLine(line) + '</span>').join('');
    });
  }

  function postProcessPreview() {
    if (!previewEl) return;

    enhanceShellBlocks();

    previewEl.querySelectorAll('table').forEach(table => {
      if (table.closest('.md-table-wrap') || table.closest('.chroma') || table.closest('.highlight')) return;
      const wrap = document.createElement('div');
      wrap.className = 'md-table-wrap';
      table.parentNode.insertBefore(wrap, table);
      wrap.appendChild(table);
    });

    previewEl.querySelectorAll('pre > code[class*="language-mermaid"], pre > code[class*="language-mmd"]').forEach(code => {
      const pre = code.closest('pre');
      if (!pre || pre.dataset.processedMermaid === 'true') return;
      pre.dataset.processedMermaid = 'true';
      const box = document.createElement('div');
      box.className = 'md-mermaid-fallback';
      const label = document.createElement('div');
      label.className = 'md-mermaid-label';
      label.textContent = 'mermaid diagram';
      const body = document.createElement('pre');
      body.textContent = code.textContent || '';
      box.appendChild(label);
      box.appendChild(body);
      pre.replaceWith(box);
    });

    previewEl.querySelectorAll('img').forEach(img => {
      img.loading = 'lazy';
      img.decoding = 'async';
      img.addEventListener('error', () => {
        const fallback = document.createElement('div');
        fallback.className = 'md-image-fallback';
        fallback.textContent = 'image unavailable: ' + (img.getAttribute('alt') || img.getAttribute('src') || 'asset');
        img.replaceWith(fallback);
      }, { once: true });
    });
  }

  let previewSeq = 0;
  let previewAbort = null;

  async function renderPreview() {
    const seq = ++previewSeq;
    if (previewAbort) previewAbort.abort();
    const ctrl = ('AbortController' in window) ? new AbortController() : null;
    previewAbort = ctrl;
    try {
      const res = await fetch('/api/render', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ body: textEl.value }),
        signal: ctrl ? ctrl.signal : undefined,
      });
      const { html } = await readJSON(res);
      if (seq !== previewSeq) return;
      previewEl.innerHTML = html || '';
      postProcessPreview();
    } catch (e) {
      if (e && e.name === 'AbortError') return;
      if (seq !== previewSeq) return;
      previewEl.innerHTML = '<p>preview failed: ' + escapeHtml(e.message) + '</p>';
    }
  }

  function previewActive() {
    const v = root.dataset.view;
    return v === 'split' || v === 'preview';
  }

  function schedulePreview() {
    updateWordCount();
    if (!previewActive()) return;
    clearTimeout(previewTimer);
    previewTimer = setTimeout(renderPreview, 250);
  }

  function setStatus(s, isError) {
    setSaveState(s, isError ? 'err' : 'ok');
    if (!isError && !dirty && focusBtn) focusBtn.classList.remove('has-unsaved');
    setTimeout(() => {
      if (!dirty && statusEl && statusEl.textContent === s) setSaveState('ready', '');
    }, 3000);
  }

  function notify(msg, type) {
    if (window.SN && window.SN.toast) window.SN.toast(msg, { type: type || 'info' });
  }


  function setBusy(on, label) {
    if (saveBtn) saveBtn.disabled = on;
    if (publishBtn) publishBtn.disabled = on || typeEl.value === 'note' || typeEl.value === 'research';
    if (on) setSaveState(label || 'working…', 'saving');
  }

  async function askConfirm(message, opts = {}) {
    if (window.SN && window.SN.confirm) {
      return window.SN.confirm(Object.assign({ message: message }, opts));
    }
    return window.confirm(message);
  }

  function tags() {
    return tagsEl.value.split(',').map(s => s.trim()).filter(Boolean);
  }

  function applyDoc(doc) {
    if (!doc) return;
    if (doc.type && typeEl.value !== doc.type) typeEl.value = doc.type;
    if (doc.title && titleEl.value !== doc.title) titleEl.value = doc.title;
    if (summaryEl && doc.summary !== undefined && summaryEl.value !== doc.summary) summaryEl.value = doc.summary || '';
    if (doc.slug) {
      slug = doc.slug;
      history.replaceState(null, '', '/editor?slug=' + encodeURIComponent(slug));
      root.dataset.slug = slug;
      root.dataset.new = 'false';
    }
    updatePath();
  }

  async function save() {
    setBusy(true, slug ? 'saving…' : 'creating…');
    if (markedMsg) markedMsg.hidden = true;
    try {
      const payload = { title: titleEl.value, summary: summaryEl ? summaryEl.value : '', type: typeEl.value, tags: tags(), body: textEl.value };
      if (!slug) {
        const doc = await api('documents', { method: 'POST', body: JSON.stringify(payload) });
        applyDoc(doc);
        dirty = false;
        setStatus('created ✓', false);
        notify('document created', 'ok');
        return doc;
      }
      const doc = await api('documents/' + encodeURIComponent(slug), { method: 'PUT', body: JSON.stringify(payload) });
      applyDoc(doc);
      dirty = false;
      setStatus('saved ✓', false);
      notify('saved', 'ok');
      return doc;
    } finally {
      setBusy(false);
    }
  }


  async function deleteDocument() {
    if (!slug) return;
    const name = titleEl.value.trim() || slug;
    if (!(await askConfirm('Delete "' + name + '" permanently? This removes the markdown file.', { title: 'delete document', confirmText: 'delete document', cancelText: 'keep document', danger: true }))) return;
    setBusy(true, 'deleting…');
    try {
      await api('documents/' + encodeURIComponent(slug), { method: 'DELETE' });
      dirty = false;
      window.location.href = closeTarget();
    } finally {
      setBusy(false);
    }
  }

  async function publish() {
    await save();
    if (!slug) return;
    setBusy(true, 'publishing…');
    try {
      const doc = await api('documents/' + encodeURIComponent(slug) + '?action=publish', { method: 'POST' });
      applyDoc(doc);
      dirty = false;
      setStatus('published ✓', false);
      notify('published — review from Preview & Export', 'ok');
      reflectLiveState('live');
      if (markedMsg) markedMsg.hidden = false;
    } finally {
      setBusy(false);
    }
  }

  async function unpublish() {
    if (!slug) return;
    setBusy(true, 'moving to draft…');
    try {
      const doc = await api('documents/' + encodeURIComponent(slug) + '?action=unpublish', { method: 'POST' });
      applyDoc(doc);
      dirty = false;
      setStatus('moved to draft ✓', false);
      notify('moved to draft', 'ok');
      reflectLiveState('draft');
      if (markedMsg) markedMsg.hidden = true;
    } finally {
      setBusy(false);
    }
  }

  function reflectLiveState(status) {
    const live = status === 'live';
    if (unpublishBtn) unpublishBtn.hidden = !live;
    if (publishBtn) publishBtn.hidden = live;
  }

  textEl.addEventListener('input', () => { markDirty(); schedulePreview(); });
  textEl.addEventListener('keyup', updateCursor);
  textEl.addEventListener('click', updateCursor);
  titleEl.addEventListener('input', markDirty);
  if (summaryEl) summaryEl.addEventListener('input', markDirty);
  tagsEl.addEventListener('input', markDirty);
  typeEl.addEventListener('change', () => { markDirty(); updatePath(); });
  if (templateEl) {
    templateEl.addEventListener('change', () => applyTemplate(templateEl.value, false));
    if (root.dataset.new === 'true' && !textEl.value.trim() && templateEl.value) applyTemplate(templateEl.value, true);
  }

  function setMode(mode) {
    bodyEl.classList.remove('write-mode', 'split-mode', 'preview-mode');
    bodyEl.classList.add(mode + '-mode');
    root.dataset.view = mode;
    document.querySelectorAll('.mode').forEach(b => {
      const on = b.id === 'mode-' + mode;
      b.classList.toggle('active', on);
      b.setAttribute('aria-pressed', on ? 'true' : 'false');
    });
    if (mode !== 'write') renderPreview();
  }

  function setFocus(on) {
    root.classList.toggle('editor-focus', on);
    document.body.classList.toggle('editor-focus-open', on);
    if (focusBtn) {
      focusBtn.setAttribute('aria-pressed', on ? 'true' : 'false');
      focusBtn.setAttribute('title', on ? 'Exit focus mode (Esc)' : 'Focus mode (⌘⇧F)');
      focusBtn.setAttribute('aria-label', on ? 'Exit focus mode' : 'Enter focus mode');
    }
    if (on && root.dataset.view !== 'write') renderPreview();
    requestAnimationFrame(() => { textEl.focus(); updateCursor(); });
  }

  function toggleFocus() {
    setFocus(!root.classList.contains('editor-focus'));
  }
  ['write', 'split', 'preview'].forEach(mode => {
    const btn = document.getElementById('mode-' + mode);
    if (btn) btn.addEventListener('click', () => setMode(mode));
  });
  if (focusBtn) focusBtn.addEventListener('click', toggleFocus);


  async function applyTemplate(name, force) {
    if (!name) return;
    if (!force && textEl.value.trim() && !(await askConfirm('Replace current draft with this template?', { title: 'replace draft', confirmText: 'replace draft', cancelText: 'keep current draft' }))) return;
    try {
      const tpl = await api('document-templates/' + encodeURIComponent(name), { method: 'GET', headers: {} });
      if (tpl.body) {
        textEl.value = tpl.body;
        markDirty();
        schedulePreview();
        updateCursor();
      }
    } catch (e) {
      setStatus('template failed: ' + e.message, true);
    }
  }


  function insertAsset() {
    if (window.SN && window.SN.assetPicker) {
      window.SN.assetPicker(md => wrapSelection('', '', md));
    } else {
      setStatus('asset picker unavailable', true);
    }
  }

  function wrapSelection(prefix, suffix, fallback) {
    const s = textEl.selectionStart, e = textEl.selectionEnd;
    const selected = textEl.value.slice(s, e) || fallback;
    const next = prefix + selected + suffix;
    textEl.setRangeText(next, s, e, 'end');
    textEl.focus();
    markDirty();
    schedulePreview();
  }
  const assetBtn = document.getElementById('asset-insert');
  if (assetBtn) assetBtn.addEventListener('click', insertAsset);

  document.querySelectorAll('.ed-toolbar [data-md]').forEach(btn => {
    btn.addEventListener('click', () => {
      const kind = btn.dataset.md;
      if (kind === 'bold') wrapSelection('**', '**', 'bold text');
      if (kind === 'italic') wrapSelection('_', '_', 'italic text');
      if (kind === 'code') wrapSelection('`', '`', 'code');
      if (kind === 'link') wrapSelection('[', '](https://example.com)', 'link');
      if (kind === 'h2') wrapSelection('\n## ', '\n', 'heading');
      if (kind === 'list') wrapSelection('\n- ', '', 'list item');
      if (kind === 'quote') wrapSelection('\n> ', '', 'quote');
    });
  });
  saveBtn.addEventListener('click', async () => {
    try { await save(); }
    catch (e) { setStatus('save failed: ' + e.message, true); notify('save failed: ' + e.message, 'error'); }
  });
  publishBtn.addEventListener('click', async () => {
    try { await publish(); }
    catch (e) { setStatus('publish failed: ' + e.message, true); notify('publish failed: ' + e.message, 'error'); }
  });
  if (unpublishBtn) unpublishBtn.addEventListener('click', async () => {
    try { await unpublish(); }
    catch (e) { setStatus('move to draft failed: ' + e.message, true); notify('move to draft failed: ' + e.message, 'error'); }
  });
  if (deleteBtn) deleteBtn.addEventListener('click', async () => {
    try { await deleteDocument(); }
    catch (e) { setStatus('delete failed: ' + e.message, true); notify('delete failed: ' + e.message, 'error'); }
  });
  document.addEventListener('keydown', async e => {
    if ((e.metaKey || e.ctrlKey) && e.key === 's') {
      e.preventDefault();
      try { await save(); }
      catch (err) { setStatus('save failed: ' + err.message, true); }
    }
    if ((e.metaKey || e.ctrlKey) && e.shiftKey && e.key.toLowerCase() === 'f') {
      e.preventDefault();
      toggleFocus();
    }
    if (e.key === 'Escape' && root.classList.contains('editor-focus')) {
      setFocus(false);
    }
  });

  if (window.SN && window.SN.guardUnsaved) {
    window.SN.guardUnsaved(() => dirty, {
      title: 'unsaved editor changes',
      message: 'This document has unsaved changes. Leave the editor anyway?',
      confirmText: 'leave editor',
      cancelText: 'continue editing'
    });
  } else {
    window.addEventListener('beforeunload', e => {
      if (!dirty) return;
      e.preventDefault();
      e.returnValue = '';
    });
  }
  const closeBtn = document.getElementById('ed-close');
  function closeTarget() {
    const from = root.dataset.from;
    return (from && from.indexOf('/content') === 0) ? from : '/content';
  }
  if (closeBtn) closeBtn.setAttribute('href', closeTarget());
  window.addEventListener('pagehide', () => setFocus(false));
  if (closeBtn) closeBtn.addEventListener('click', async e => {
    if (!dirty) return;
    e.preventDefault();
    const ok = await askConfirm('Discard unsaved changes and close editor?', { title: 'close editor', confirmText: 'discard and close', cancelText: 'continue editing', danger: true });
    if (ok) window.location.href = closeTarget();
  });

  if (unpublishBtn && !unpublishBtn.hidden) reflectLiveState('live');
  updateWordCount();
  updateCursor();
  updatePath();
  setMode('write');
})();

(function(){
  const box = document.getElementById('version-list');
  if (!box) return;
  const slug = box.dataset.slug;
  function escapeHtml(s) {
    return (s || '').replace(/[&<>"]/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]));
  }
  async function json(res){ if(!res.ok) throw new Error((await res.text()).trim() || res.statusText); return res.json(); }
  async function loadVersions(){
    try {
      const versions = await json(await fetch('/api/documents/'+encodeURIComponent(slug)+'/versions'));
      if(!versions.length){ box.innerHTML = '<div class="empty small">no versions yet</div>'; return; }
      box.innerHTML = '';
      versions.slice(0,12).forEach(v => {
        const row = document.createElement('div'); row.className = 'version-row';
        const when = new Date(v.created).toLocaleString();
        row.innerHTML = '<span><strong>'+escapeHtml(v.reason || '')+'</strong><small>'+escapeHtml(when)+'</small></span><button type="button">restore</button>';
        row.querySelector('button').onclick = async () => {
          if(!(await (window.SN && window.SN.confirm ? window.SN.confirm({ title: 'restore version', message: 'Restore this version? Current content will be snapshotted first.', confirmText: 'restore version', cancelText: 'cancel' }) : Promise.resolve(window.confirm('Restore this version? Current content will be snapshotted first.'))))) return;
          await json(await fetch('/api/documents/'+encodeURIComponent(slug)+'?action=restore_version&id='+encodeURIComponent(v.id), {method:'POST'}));
          location.reload();
        };
        box.appendChild(row);
      });
    } catch(e){ box.textContent = 'version load failed: ' + e.message; }
  }
  loadVersions();
})();
