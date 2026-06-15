package datadir

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func testFS(themeData, templateData string) (fs.FS, fs.FS) {
	return fstest.MapFS{
			"themes/default/layouts/base.html": &fstest.MapFile{Data: []byte(themeData)},
		}, fstest.MapFS{
			"templates/blog.md": &fstest.MapFile{Data: []byte(templateData)},
		}
}

func TestInitUpdatesManagedBundledFiles(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	themes, templates := testFS("theme-v1", "template-v1")
	if err := Init(themes, templates); err != nil {
		t.Fatalf("first init: %v", err)
	}

	themes, templates = testFS("theme-v2", "template-v2")
	if err := Init(themes, templates); err != nil {
		t.Fatalf("second init: %v", err)
	}

	themePath := filepath.Join(os.Getenv("HOME"), ".shadownote", "themes", "default", "layouts", "base.html")
	data, err := os.ReadFile(themePath)
	if err != nil {
		t.Fatalf("read theme: %v", err)
	}
	if string(data) != "theme-v2" {
		t.Fatalf("theme was not upgraded, got %q", data)
	}
}

func TestInitPreservesUserEditedBundledFiles(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	themes, templates := testFS("theme-v1", "template-v1")
	if err := Init(themes, templates); err != nil {
		t.Fatalf("first init: %v", err)
	}

	themePath := filepath.Join(os.Getenv("HOME"), ".shadownote", "themes", "default", "layouts", "base.html")
	if err := os.WriteFile(themePath, []byte("user custom theme"), 0644); err != nil {
		t.Fatalf("customize theme: %v", err)
	}

	themes, templates = testFS("theme-v2", "template-v2")
	if err := Init(themes, templates); err != nil {
		t.Fatalf("second init: %v", err)
	}

	data, err := os.ReadFile(themePath)
	if err != nil {
		t.Fatalf("read theme: %v", err)
	}
	if string(data) != "user custom theme" {
		t.Fatalf("custom theme was overwritten, got %q", data)
	}
}

func TestInitCreatesRuntimeContentAssetAndPublicDirs(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	themes, templates := testFS("theme", "template")
	if err := Init(themes, templates); err != nil {
		t.Fatalf("init: %v", err)
	}

	for _, name := range []string{"content", "assets", "public"} {
		path := filepath.Join(home, ".shadownote", name)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat %s: %v", name, err)
		}
		if !info.IsDir() {
			t.Fatalf("%s is not a directory", path)
		}
	}
}
