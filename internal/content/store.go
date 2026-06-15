package content

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Store struct {
	root string
}

func NewStore(root string) (*Store, error) {
	s := &Store{root: root}
	if err := s.scaffold(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) scaffold() error {
	for _, t := range AllTypes() {
		dir := filepath.Join(s.root, subdir(t))
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
	}
	return nil
}

func (s *Store) Create(title string, t Type, tags []string, body string) (*Document, error) {
	if !IsValidType(t) {
		return nil, fmt.Errorf("invalid document type: %s", t)
	}

	slug := uniqueSlugGlobal(s.root, Slugify(title))
	fm := newFrontmatter(title, t, tags)

	if body == "" {
		body = "# " + title + "\n\n"
	}

	doc := &Document{
		Frontmatter: fm,
		Body:        body,
		Slug:        slug,
		Path:        filepath.Join(s.root, subdir(t), slug+".md"),
	}

	if err := s.Write(doc); err != nil {
		return nil, err
	}
	return doc, nil
}

func (s *Store) Read(path string) (*Document, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read document: %w", err)
	}

	fm, body, err := parse(raw)
	if err != nil {
		return nil, err
	}
	if info, statErr := os.Stat(path); statErr == nil {
		normalizeLoadedFrontmatter(path, info.ModTime(), &fm)
	}

	slug := strings.TrimSuffix(filepath.Base(path), ".md")

	return &Document{
		Frontmatter: fm,
		Body:        body,
		Path:        path,
		Slug:        slug,
	}, nil
}

func normalizeLoadedFrontmatter(path string, modTime time.Time, fm *Frontmatter) {
	if fm.Type == "" {
		switch filepath.Base(filepath.Dir(path)) {
		case string(TypePost) + "s":
			fm.Type = TypePost
		case string(TypeDraft) + "s":
			fm.Type = TypeDraft
		case string(TypeNote) + "s":
			fm.Type = TypeNote
		case string(TypeResearch):
			fm.Type = TypeResearch
		case string(TypePage) + "s":
			fm.Type = TypePage
		}
	}

	if fm.Created.IsZero() {
		fm.Created = modTime
	}
	if fm.Modified.IsZero() {
		fm.Modified = fm.Created
	}

	switch fm.Type {
	case TypePost:
		if fm.Status == "" {
			fm.Status = StatusLive
		}
		if fm.PublishTo == "" {
			fm.PublishTo = PublishBlog
		}
	case TypeNote, TypeResearch:
		if fm.Status == "" {
			fm.Status = StatusPrivate
		}
		if fm.PublishTo == "" {
			fm.PublishTo = PublishNone
		}
	default:
		if fm.Status == "" {
			fm.Status = StatusDraft
		}
		if fm.PublishTo == "" {
			fm.PublishTo = PublishNone
		}
	}
}

func (s *Store) Write(doc *Document) error {
	if _, err := os.Stat(doc.Path); err == nil {
		if snapErr := s.snapshot(doc, "save"); snapErr != nil {
			return snapErr
		}
	}
	doc.Modified = time.Now()

	data, err := serialize(doc.Frontmatter, doc.Body)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(doc.Path), 0755); err != nil {
		return fmt.Errorf("ensure dir: %w", err)
	}
	if err := os.WriteFile(doc.Path, data, 0644); err != nil {
		return fmt.Errorf("write document: %w", err)
	}
	return nil
}

func (s *Store) Delete(doc *Document) error {
	if err := s.snapshot(doc, "delete"); err != nil {
		return err
	}
	if err := os.Remove(doc.Path); err != nil {
		return fmt.Errorf("delete document: %w", err)
	}
	return nil
}

func (s *Store) List(filter Type) ([]*Document, error) {
	var docs []*Document

	types := AllTypes()
	if filter != "" {
		types = []Type{filter}
	}

	for _, t := range types {
		dir := filepath.Join(s.root, subdir(t))
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read dir %s: %w", dir, err)
		}

		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			doc, err := s.Read(filepath.Join(dir, e.Name()))
			if err != nil {
				continue
			}
			docs = append(docs, doc)
		}
	}

	return docs, nil
}

func (s *Store) Publishable() ([]*Document, error) {
	all, err := s.List(TypePost)
	if err != nil {
		return nil, err
	}
	var out []*Document
	for _, d := range all {
		if d.Publishable() {
			out = append(out, d)
		}
	}
	return out, nil
}

func (s *Store) ensureTargetAvailable(oldPath, newPath string) error {
	oldClean := filepath.Clean(oldPath)
	newClean := filepath.Clean(newPath)
	if oldClean == newClean {
		return nil
	}
	if _, err := os.Stat(newPath); err == nil {
		return fmt.Errorf("target document already exists: %s", newPath)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("check target document: %w", err)
	}
	slug := strings.TrimSuffix(filepath.Base(newPath), ".md")
	for _, t := range AllTypes() {
		candidate := filepath.Clean(filepath.Join(s.root, subdir(t), slug+".md"))
		if candidate == oldClean || candidate == newClean {
			continue
		}
		if _, err := os.Stat(candidate); err == nil {
			return fmt.Errorf("slug %q already exists in %s", slug, candidate)
		} else if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("check slug collision: %w", err)
		}
	}
	return nil
}

func applyTypeDefaults(doc *Document, t Type) {
	doc.Type = t
	switch t {
	case TypeNote, TypeResearch:
		doc.Status = StatusPrivate
		doc.PublishTo = PublishNone
	case TypePost:
		doc.Status = StatusDraft
		doc.PublishTo = PublishBlog
	default:
		doc.Status = StatusDraft
		doc.PublishTo = PublishNone
	}
}

func (s *Store) ChangeType(doc *Document, t Type) error {
	if !IsValidType(t) {
		return fmt.Errorf("invalid document type: %s", t)
	}
	if doc.Type == t {
		return s.Write(doc)
	}

	old := doc.Path
	if err := s.snapshot(doc, "move"); err != nil {
		return err
	}
	newPath := filepath.Join(s.root, subdir(t), doc.Slug+".md")
	if err := s.ensureTargetAvailable(old, newPath); err != nil {
		return err
	}

	applyTypeDefaults(doc, t)
	doc.Path = newPath
	if err := s.Write(doc); err != nil {
		doc.Path = old
		return err
	}
	if old != doc.Path {
		if err := os.Remove(old); err != nil {
			return fmt.Errorf("remove old document: %w", err)
		}
	}
	return nil
}

func (s *Store) Publish(doc *Document) error {
	wasElsewhere := doc.Type != TypePost
	if err := s.snapshot(doc, "publish"); err != nil {
		return err
	}

	doc.Type = TypePost
	doc.Status = StatusLive
	doc.PublishTo = PublishBlog

	newPath := filepath.Join(s.root, subdir(TypePost), doc.Slug+".md")
	if err := s.ensureTargetAvailable(doc.Path, newPath); err != nil {
		return err
	}

	if wasElsewhere && doc.Path != newPath {
		old := doc.Path
		doc.Path = newPath
		if err := s.Write(doc); err != nil {
			return err
		}
		return os.Remove(old)
	}

	doc.Path = newPath
	return s.Write(doc)
}

func (s *Store) Unpublish(doc *Document) error {
	if err := s.snapshot(doc, "unpublish"); err != nil {
		return err
	}
	old := doc.Path
	doc.Type = TypeDraft
	doc.Status = StatusDraft
	doc.PublishTo = PublishNone
	newPath := filepath.Join(s.root, subdir(TypeDraft), doc.Slug+".md")
	if err := s.ensureTargetAvailable(old, newPath); err != nil {
		return err
	}
	doc.Path = newPath

	if err := s.Write(doc); err != nil {
		return err
	}
	if old != doc.Path {
		return os.Remove(old)
	}
	return nil
}

func (s *Store) Root() string {
	return s.root
}
