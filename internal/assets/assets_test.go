package assets

import (
	"errors"
	"strings"
	"testing"
)

func TestSafeName(t *testing.T) {
	got, err := safeName("hello world!!.png")
	if err != nil {
		t.Fatalf("safeName: %v", err)
	}
	if got != "hello-world--.png" {
		t.Fatalf("safeName = %q", got)
	}
	for _, bad := range []string{"", "../x.png", "a/b.png", `a\\b.png`, "..."} {
		if _, err := safeName(bad); err == nil {
			t.Fatalf("safeName(%q) succeeded", bad)
		}
	}
}

func TestRejectsActiveContentUploads(t *testing.T) {
	st, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	for _, name := range []string{"x.svg", "x.html", "x.xml"} {
		_, err := st.Save(name, strings.NewReader("payload"))
		if !errors.Is(err, ErrDisallowedType) {
			t.Fatalf("Save(%q) error = %v, want ErrDisallowedType", name, err)
		}
	}
}

func TestDataURIHonorsSizeLimit(t *testing.T) {
	st, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	if _, err := st.Save("a.txt", strings.NewReader("hello")); err != nil {
		t.Fatalf("save: %v", err)
	}
	if _, ok := st.DataURI("a.txt", 4); ok {
		t.Fatal("DataURI ignored size limit")
	}
	uri, ok := st.DataURI("a.txt", 5)
	if !ok || !strings.HasPrefix(uri, "data:text/plain") {
		t.Fatalf("DataURI = %q, %v", uri, ok)
	}
}
