package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

const (
	MarkStart = "\x02"
	MarkEnd   = "\x03"
)

type DocMeta struct {
	Slug     string
	Title    string
	Type     string
	Status   string
	Tags     string
	Path     string
	Created  string
	Modified string
}

type IndexInput struct {
	DocMeta
	Body string
}

func (s *Store) Reindex(docs []IndexInput) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin reindex: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM documents`); err != nil {
		return fmt.Errorf("clear documents: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM search`); err != nil {
		return fmt.Errorf("clear search: %w", err)
	}

	metaStmt, err := tx.Prepare(`
		INSERT INTO documents (slug, title, type, status, tags, path, created, modified)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare meta: %w", err)
	}
	defer metaStmt.Close()

	searchStmt, err := tx.Prepare(`
		INSERT INTO search (slug, title, body, tags) VALUES (?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare search: %w", err)
	}
	defer searchStmt.Close()

	for _, d := range docs {
		if _, err := metaStmt.Exec(d.Slug, d.Title, d.Type, d.Status,
			d.Tags, d.Path, d.Created, d.Modified); err != nil {
			return fmt.Errorf("index meta %s: %w", d.Slug, err)
		}
		if _, err := searchStmt.Exec(d.Slug, d.Title, d.Body, d.Tags); err != nil {
			return fmt.Errorf("index search %s: %w", d.Slug, err)
		}
	}

	return tx.Commit()
}

type SearchResult struct {
	Slug     string
	Title    string
	Type     string
	Status   string
	Tags     string
	Modified string
	Snippet  string
}

func (s *Store) Search(query string, limit int) ([]SearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 20
	}
	if !s.fts5 {
		return s.searchFallback(query, limit)
	}

	match := ftsQuery(query)
	if match == "" {
		return s.searchFallback(query, limit)
	}

	rows, err := s.db.Query(`
		SELECT search.slug, search.title,
		       COALESCE(documents.type, '')     AS type,
		       COALESCE(documents.status, '')   AS status,
		       COALESCE(documents.tags, '')     AS tags,
		       COALESCE(documents.modified, '') AS modified,
		       snippet(search, 2, char(2), char(3), '…', 24) AS snippet
		FROM search
		LEFT JOIN documents ON documents.slug = search.slug
		WHERE search MATCH ?
		ORDER BY rank
		LIMIT ?`,
		match, limit)
	if err != nil {
		return s.searchFallback(query, limit)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.Slug, &r.Title, &r.Type, &r.Status, &r.Tags, &r.Modified, &r.Snippet); err != nil {
			return nil, fmt.Errorf("scan result: %w", err)
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func (s *Store) searchFallback(query string, limit int) ([]SearchResult, error) {
	terms := strings.Fields(strings.ToLower(query))
	if len(terms) == 0 {
		return nil, nil
	}
	pattern := "%" + strings.Join(terms, "%") + "%"
	rows, err := s.db.Query(`
		SELECT search.slug, search.title, search.body, search.tags,
		       COALESCE(documents.type, '')     AS type,
		       COALESCE(documents.status, '')   AS status,
		       COALESCE(documents.tags, '')     AS dtags,
		       COALESCE(documents.modified, '') AS modified
		FROM search
		LEFT JOIN documents ON documents.slug = search.slug
		WHERE lower(search.title || ' ' || search.body || ' ' || search.tags) LIKE ?
		ORDER BY search.title ASC
		LIMIT ?`, pattern, limit)
	if err != nil {
		return nil, fmt.Errorf("search fallback: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var slug, title, body, tags, typ, status, dtags, modified string
		if err := rows.Scan(&slug, &title, &body, &tags, &typ, &status, &dtags, &modified); err != nil {
			return nil, fmt.Errorf("scan fallback result: %w", err)
		}
		results = append(results, SearchResult{Slug: slug, Title: title, Type: typ, Status: status, Tags: dtags, Modified: modified, Snippet: fallbackSnippet(query, title, body, tags)})
	}
	return results, rows.Err()
}

func fallbackSnippet(query, title, body, tags string) string {
	text := strings.TrimSpace(body)
	if text == "" {
		text = strings.TrimSpace(title + " " + tags)
	}
	if text == "" {
		return ""
	}
	lower := strings.ToLower(text)
	for _, term := range strings.Fields(strings.ToLower(query)) {
		if idx := strings.Index(lower, term); idx >= 0 {
			start := idx - 80
			if start < 0 {
				start = 0
			}
			end := idx + len(term) + 120
			if end > len(text) {
				end = len(text)
			}
			prefix, suffix := "", ""
			if start > 0 {
				prefix = "…"
			}
			if end < len(text) {
				suffix = "…"
			}
			chunk := text[start:end]
			chunkLower := strings.ToLower(chunk)
			mark := strings.Index(chunkLower, term)
			if mark >= 0 {
				chunk = chunk[:mark] + MarkStart + chunk[mark:mark+len(term)] + MarkEnd + chunk[mark+len(term):]
			}
			return prefix + chunk + suffix
		}
	}
	if len(text) > 200 {
		return text[:200] + "…"
	}
	return text
}

func ftsQuery(input string) string {
	input = strings.ReplaceAll(input, `"`, " ")
	fields := strings.Fields(input)
	quoted := make([]string, 0, len(fields))
	for _, f := range fields {
		if !hasAlnum(f) {
			continue
		}
		quoted = append(quoted, `"`+f+`"*`)
	}
	return strings.Join(quoted, " ")
}

func hasAlnum(s string) bool {
	for _, r := range s {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r > 127 {
			return true
		}
	}
	return false
}

func (s *Store) AllMeta() ([]DocMeta, error) {
	rows, err := s.db.Query(`
		SELECT slug, title, type, status, tags, path, created, modified
		FROM documents ORDER BY modified DESC`)
	if err != nil {
		return nil, fmt.Errorf("all meta: %w", err)
	}
	defer rows.Close()

	var metas []DocMeta
	for rows.Next() {
		var m DocMeta
		if err := rows.Scan(&m.Slug, &m.Title, &m.Type, &m.Status,
			&m.Tags, &m.Path, &m.Created, &m.Modified); err != nil {
			return nil, fmt.Errorf("scan meta: %w", err)
		}
		metas = append(metas, m)
	}
	return metas, rows.Err()
}

func (s *Store) MetaBySlug(slug string) (DocMeta, bool, error) {
	var m DocMeta
	err := s.db.QueryRow(`
		SELECT slug, title, type, status, tags, path, created, modified
		FROM documents WHERE slug = ?`, slug).
		Scan(&m.Slug, &m.Title, &m.Type, &m.Status, &m.Tags, &m.Path, &m.Created, &m.Modified)
	if errors.Is(err, sql.ErrNoRows) {
		return DocMeta{}, false, nil
	}
	if err != nil {
		return DocMeta{}, false, fmt.Errorf("meta by slug: %w", err)
	}
	return m, true, nil
}
