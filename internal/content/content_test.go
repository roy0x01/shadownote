package content

import (
	"path/filepath"
	"testing"
)

func TestSlugify(t *testing.T) {
	tests := map[string]string{
		"Hello World":             "hello-world",
		"  Red Team / API Notes ": "red-team-api-notes",
		"!!!":                     "untitled",
		"CVE-2026-0001 write-up":  "cve-2026-0001-write-up",
	}
	for in, want := range tests {
		if got := Slugify(in); got != want {
			t.Fatalf("Slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPrivateTypesAreNotPublishable(t *testing.T) {
	for _, typ := range []Type{TypeNote, TypeResearch} {
		d := &Document{Frontmatter: newFrontmatter("secret", typ, nil)}
		if d.Publishable() {
			t.Fatalf("%s should not be publishable", typ)
		}
	}
}

func TestStoreCreateAndReadDraft(t *testing.T) {
	st, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	doc, err := st.Create("My Draft", TypeDraft, []string{"test"}, "body")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if got, want := filepath.Base(doc.Path), "my-draft.md"; got != want {
		t.Fatalf("path base = %q, want %q", got, want)
	}
	loaded, err := st.Read(doc.Path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if loaded.Title != "My Draft" || loaded.Status != StatusDraft || loaded.PublishTo != PublishNone {
		t.Fatalf("unexpected loaded document: %#v", loaded.Frontmatter)
	}
}
