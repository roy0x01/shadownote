(function () {
  'use strict';
  function notify(msg, type) { if (window.SN && window.SN.toast) window.SN.toast(msg, { type: type || 'info' }); else if (type === 'error') alert(msg); }
  function busy(el, label) { return (window.SN && window.SN.busy) ? window.SN.busy(el, label) : (el && (el.disabled = true), function () { if (el) el.disabled = false; }); }

  const listEl = document.getElementById('content-doc-list');
  if (!listEl) return;
  const searchEl = document.getElementById('content-search');
  const sortEl = document.getElementById('content-sort-select');
  const pager = document.getElementById('content-pagination');
  const typePills = Array.from(document.querySelectorAll('.type-filters .filter-pill'));
  const pageSize = Number(listEl.dataset.pageSize) || 50;

  function esc(s) { return (s || '').replace(/[&<>"]/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c])); }
  function safeSnippet(s) { return esc(s).replace(/\u0002/g, '<mark>').replace(/\u0003/g, '</mark>'); }

  function publishKind(type, status) {
    if (status === 'live' && type === 'post') return 'unpublish';
    if (type === 'post' || type === 'draft') return 'publish';
    return '';
  }

  function parseRows() {
    return Array.from(listEl.querySelectorAll('.doc-row')).map((row, i) => ({
      slug: row.dataset.slug,
      title: row.dataset.title || row.dataset.slug,
      type: row.dataset.type || '',
      status: row.dataset.status || 'draft',
      tags: (row.dataset.tags || '').split(',').map(t => t.trim()).filter(Boolean),
      modified: (row.querySelector('.col-modified') || {}).textContent || '',
      order: i,
      snippet: ''
    }));
  }
  let baseDocs = parseRows();
  const bySlug = {};
  baseDocs.forEach(d => { bySlug[d.slug] = d; });

  const state = {
    query: (searchEl && searchEl.value.trim()) || '',
    type: listEl.dataset.activeType || '',
    sort: listEl.dataset.initSort || 'modified-desc',
    page: 1,
    mode: (listEl.dataset.searching === '1') ? 'search' : 'list',
    searchDocs: []
  };

  function syncURL() {
    const p = new URLSearchParams();
    if (state.query) p.set('q', state.query);
    if (state.type) p.set('type', state.type);
    if (state.sort && !(state.sort === 'modified-desc' && !state.query) && !(state.sort === 'relevance' && state.query)) p.set('sort', state.sort);
    if (state.page > 1) p.set('page', String(state.page));
    const qs = p.toString();
    history.replaceState(null, '', '/content' + (qs ? '?' + qs : ''));
  }

  function rowActionsHTML(d) {
    const kind = publishKind(d.type, d.status);
    let pub;
    if (kind === 'publish') pub = '<button type="button" class="row-act act-publish">publish post</button>';
    else if (kind === 'unpublish') pub = '<button type="button" class="row-act act-unpublish">move to draft</button>';
    else pub = '<span class="row-note" title="open in the Editor to change type and publish">not publishable here</span>';
    return '<a class="row-act act-open" href="/editor?slug=' + encodeURIComponent(d.slug) + '&from=/content">open</a>' +
      pub +
      '<button type="button" class="row-act act-duplicate">duplicate as draft</button>' +
      '<button type="button" class="row-act act-delete">delete</button>';
  }

  function rowHTML(d) {
    const tags = d.tags && d.tags.length ? `<span class="col-tags">${d.tags.map(esc).join(', ')}</span>` : '';
    const snippet = d.snippet ? `<span class="col-snippet">${safeSnippet(d.snippet)}</span>` : '';
    const privateBadge = d.status === 'private' ? '<span class="state-badge private">private</span>' : '';
    const latest = (d.order === 0 && state.mode === 'list' && state.sort === 'modified-desc') ? ' doc-row-latest' : '';
    return `<div class="doc-row${latest}" data-slug="${esc(d.slug)}" data-title="${esc(d.title)}" data-type="${esc(d.type)}" data-status="${esc(d.status)}" data-publish="${publishKind(d.type, d.status)}" data-tags="${esc((d.tags || []).join(','))}">
      <div class="col-main">
        <a class="col-title" href="/editor?slug=${encodeURIComponent(d.slug)}&from=/content">${esc(d.title)}</a>
        ${snippet}
        <span class="col-badges"><span class="type-badge type-${esc(d.type)}">${esc(d.type)}</span><span class="status-pill ${esc(d.status)}">${esc(d.status)}</span>${privateBadge}</span>
        ${tags}
      </div>
      <span class="col-modified">${esc(d.modified)}</span>
      <span class="row-actions">${rowActionsHTML(d)}</span>
    </div>`;
  }

  function applyFilters(docs) {
    return docs.filter(d => {
      return !(state.type && d.type !== state.type);
    });
  }
  function applySort(docs) {
    const s = state.sort, out = docs.slice();
    if (s === 'modified-desc') out.sort((a, b) => a.order - b.order);
    else if (s === 'modified-asc') out.sort((a, b) => b.order - a.order);
    else if (s === 'title-asc') out.sort((a, b) => a.title.localeCompare(b.title));
    else if (s === 'title-desc') out.sort((a, b) => b.title.localeCompare(a.title));
    return out;
  }
  function currentDocs() {
    const src = state.mode === 'search' ? state.searchDocs : baseDocs;
    return applySort(applyFilters(src));
  }

  function render() {
    const docs = currentDocs();
    const pages = Math.max(1, Math.ceil(docs.length / pageSize));
    if (state.page > pages) state.page = pages;
    const start = (state.page - 1) * pageSize;
    const slice = docs.slice(start, start + pageSize);
    if (!docs.length) {
      listEl.innerHTML = `<div class="empty-state"><div class="empty-mark">∅</div><strong>${state.query ? 'No matches.' : 'No documents.'}</strong><span>${state.query ? 'Try a different title, tag, or phrase.' : 'Create a new draft to get started.'}</span></div>`;
    } else {
      listEl.innerHTML = slice.map(rowHTML).join('');
    }
    renderPagination(pages);
    wireRowActions();
  }

  function renderPagination(pages) {
    if (!pager) return;
    if (pages <= 1) { pager.innerHTML = ''; pager.hidden = true; return; }
    pager.hidden = false;
    const parts = ['<button type="button" data-page="prev"' + (state.page === 1 ? ' disabled' : '') + '>prev</button>'];
    for (let i = 1; i <= pages; i++) parts.push(`<button type="button" data-page="${i}"${i === state.page ? ' class="active" aria-current="page"' : ''}>${i}</button>`);
    parts.push('<button type="button" data-page="next"' + (state.page === pages ? ' disabled' : '') + '>next</button>');
    pager.innerHTML = parts.join('');
    pager.querySelectorAll('button[data-page]').forEach(b => b.addEventListener('click', () => {
      const a = b.dataset.page;
      if (a === 'prev') state.page--; else if (a === 'next') state.page++; else state.page = Number(a) || 1;
      syncURL(); render();
    }));
  }

  function applyStatus(slug, dto) {
    const d = bySlug[slug];
    if (!d) return;
    if (dto.status) d.status = dto.status;
    if (dto.type) d.type = dto.type;
    render();
  }
  function wireRowActions() {
    listEl.querySelectorAll('.doc-row').forEach(row => {
      const slug = row.dataset.slug;
      const pubBtn = row.querySelector('.act-publish');
      const unpubBtn = row.querySelector('.act-unpublish');
      const delBtn = row.querySelector('.act-delete');
      const dupBtn = row.querySelector('.act-duplicate');

      async function doAction(btn, action, okMsg) {
        const restore = busy(btn);
        try {
          const res = await fetch(`/api/documents/${encodeURIComponent(slug)}?action=${action}`, { method: 'POST' });
          if (!res.ok) throw new Error((await res.text()).trim() || 'failed');
          applyStatus(slug, await res.json());
          notify(okMsg, 'ok');
        } catch (e) { restore(); notify('Failed: ' + e.message, 'error'); }
      }
      if (pubBtn) pubBtn.addEventListener('click', () => doAction(pubBtn, 'publish', 'published — review from Preview & Export'));
      if (unpubBtn) unpubBtn.addEventListener('click', () => doAction(unpubBtn, 'unpublish', 'moved to draft'));

      if (delBtn) delBtn.addEventListener('click', async () => {
        const title = row.dataset.title;
        const meta = (row.dataset.type || '') + ' · ' + (row.dataset.status || '');
        const message = `Delete “${title}”?\n\n${meta}\n\nThis removes the markdown file from the active library. A version snapshot is kept first if the document already existed.`;
        if (!(await (window.SN && window.SN.confirm ? window.SN.confirm({ title: 'delete document', message: message, confirmText: 'delete document', cancelText: 'keep document', danger: true }) : Promise.resolve(window.confirm(message))))) return;
        const restore = busy(delBtn);
        try {
          const res = await fetch('/api/documents/' + encodeURIComponent(slug), { method: 'DELETE' });
          if (!res.ok) throw new Error((await res.text()).trim() || 'delete failed');
          baseDocs = baseDocs.filter(d => d.slug !== slug);
          state.searchDocs = state.searchDocs.filter(d => d.slug !== slug);
          notify('deleted', 'ok');
          render();
        } catch (e) { restore(); notify('Delete failed: ' + e.message, 'error'); }
      });

      if (dupBtn) dupBtn.addEventListener('click', async () => {
        const restore = busy(dupBtn);
        try {
          const src = await (await fetch('/api/documents/' + encodeURIComponent(slug))).json();
          const dupType = (src.type === 'note' || src.type === 'research') ? src.type : 'draft';
          const res = await fetch('/api/documents', {
            method: 'POST', headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ title: 'Copy of ' + src.title, type: dupType, tags: src.tags, body: src.body, summary: src.summary })
          });
          if (!res.ok) throw new Error((await res.text()).trim() || 'duplicate failed');
          const dto = await res.json();
          window.location.href = '/editor?slug=' + encodeURIComponent(dto.slug) + '&from=/content';
        } catch (e) { restore(); notify('Duplicate failed: ' + e.message, 'error'); }
      });
    });
  }

  function updateRelevanceOption() {
    if (!sortEl) return;
    const rel = sortEl.querySelector('option[value="relevance"]');
    if (rel) rel.disabled = !state.query;
  }
  let timer = null;
  async function runSearch(q) {
    if (!q.trim()) {
      state.mode = 'list'; state.page = 1;
      if (state.sort === 'relevance') { state.sort = 'modified-desc'; if (sortEl) sortEl.value = state.sort; }
      updateRelevanceOption(); syncURL(); render();
      return;
    }
    try {
      const res = await fetch('/api/search?q=' + encodeURIComponent(q));
      if (!res.ok) throw new Error((await res.text()).trim() || res.statusText);
      const results = (await res.json()) || [];
      state.searchDocs = results.map((r, i) => {
        const base = bySlug[r.slug] || {};
        return {
          slug: r.slug, title: r.title || base.title || r.slug,
          type: r.type || base.type || '', status: r.status || base.status || 'draft',
          tags: (r.tags && r.tags.length ? r.tags : (base.tags || [])),
          modified: base.modified || '', order: i, snippet: r.snippet || ''
        };
      });
      state.mode = 'search'; state.page = 1;
      if (state.sort === 'modified-desc') { state.sort = 'relevance'; if (sortEl) sortEl.value = 'relevance'; }
      updateRelevanceOption(); syncURL(); render();
    } catch (e) { notify('Search failed: ' + e.message, 'error'); }
  }
  if (searchEl) searchEl.addEventListener('input', () => {
    state.query = searchEl.value.trim();
    clearTimeout(timer);
    timer = setTimeout(() => runSearch(searchEl.value), 220);
  });

  typePills.forEach(p => p.addEventListener('click', (e) => {
    e.preventDefault();
    state.type = p.dataset.type || '';
    typePills.forEach(x => { const on = x === p; x.classList.toggle('active', on); x.setAttribute('aria-pressed', String(on)); });
    state.page = 1; syncURL(); render();
  }));
  if (sortEl) sortEl.addEventListener('change', () => { state.sort = sortEl.value; state.page = 1; syncURL(); render(); });

  updateRelevanceOption();
  if (state.query && state.mode !== 'search') { runSearch(state.query); } else { render(); }
})();
