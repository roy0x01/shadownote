package content

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Version struct {
	ID      string    `json:"id"`
	Slug    string    `json:"slug"`
	Type    string    `json:"type"`
	Path    string    `json:"path"`
	Created time.Time `json:"created"`
	Size    int64     `json:"size"`
	Reason  string    `json:"reason"`
}

func (s *Store) versionsRoot() string { return filepath.Join(s.root, ".versions") }

func versionID(t time.Time) string { return t.UTC().Format("20060102T150405.000000000Z") }

func (s *Store) snapshot(doc *Document, reason string) error {
	if doc == nil || doc.Path == "" {
		return nil
	}
	data, err := os.ReadFile(doc.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read snapshot source: %w", err)
	}
	rel, err := filepath.Rel(s.root, doc.Path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return fmt.Errorf("snapshot path outside content root")
	}
	now := time.Now()
	dir := filepath.Join(s.versionsRoot(), doc.Slug)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	name := versionID(now) + "__" + strings.TrimSpace(reason) + "__" + filepath.Base(rel)
	name = strings.ReplaceAll(name, string(os.PathSeparator), "-")
	return os.WriteFile(filepath.Join(dir, name), data, 0644)
}

func (s *Store) ListVersions(slug string) ([]Version, error) {
	slug = Slugify(slug)
	if slug == "" {
		return nil, fmt.Errorf("invalid slug")
	}
	dir := filepath.Join(s.versionsRoot(), slug)
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return []Version{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := make([]Version, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		parts := strings.SplitN(e.Name(), "__", 3)
		reason := "snapshot"
		if len(parts) >= 2 && parts[1] != "" {
			reason = parts[1]
		}
		id := strings.TrimSuffix(e.Name(), ".md")
		if len(parts) >= 1 {
			id = parts[0]
		}
		out = append(out, Version{ID: id, Slug: slug, Path: filepath.ToSlash(filepath.Join(".versions", slug, e.Name())), Created: info.ModTime(), Size: info.Size(), Reason: reason})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Created.After(out[j].Created) })
	return out, nil
}

func (s *Store) RestoreVersion(current *Document, id string) (*Document, error) {
	if current == nil {
		return nil, fmt.Errorf("missing document")
	}
	versions, err := s.ListVersions(current.Slug)
	if err != nil {
		return nil, err
	}
	var picked string
	for _, v := range versions {
		if v.ID == id || strings.HasPrefix(v.Path, filepath.ToSlash(filepath.Join(".versions", current.Slug, id))) {
			picked = filepath.Join(s.root, filepath.FromSlash(v.Path))
			break
		}
	}
	if picked == "" {
		return nil, fmt.Errorf("version not found")
	}
	if err := s.snapshot(current, "before-restore"); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(picked)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(current.Path, data, 0644); err != nil {
		return nil, err
	}
	return s.Read(current.Path)
}
