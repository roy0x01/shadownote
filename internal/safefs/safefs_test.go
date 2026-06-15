package safefs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateManagedDirRejectsDangerousPaths(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for _, p := range []string{"", ".", "..", string(os.PathSeparator), cwd} {
		if _, err := ValidateManagedDir(p); err == nil {
			t.Fatalf("ValidateManagedDir(%q) succeeded", p)
		}
	}
}

func TestResetDirAllowsSafeChild(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "public")
	if err := os.WriteFile(target, []byte("old"), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := ResetDir(target); err != nil {
		t.Fatalf("ResetDir: %v", err)
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("target is not a dir")
	}
}

func TestValidateManagedDirAllowsDefaultPublicDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	publicDir := filepath.Join(home, ".shadownote", "public")
	if _, err := ValidateManagedDir(publicDir); err != nil {
		t.Fatalf("ValidateManagedDir(default public): %v", err)
	}
}

func TestValidateManagedDirRejectsDataRoot(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dataRoot := filepath.Join(home, ".shadownote")
	if _, err := ValidateManagedDir(dataRoot); err == nil {
		t.Fatal("expected data root rejection")
	}
}
