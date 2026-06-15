package deploy

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Archive struct{ Dest string }

func (a *Archive) validate() error {
	if strings.TrimSpace(a.Dest) == "" {
		return fmt.Errorf("archive destination is not configured")
	}
	if err := os.MkdirAll(filepath.Dir(a.Dest), 0o755); err != nil {
		return err
	}
	return nil
}

func (a *Archive) Deploy(srcDir string) (string, error) {
	if err := a.validate(); err != nil {
		return "", err
	}
	out, err := os.Create(a.Dest)
	if err != nil {
		return "", err
	}
	defer out.Close()
	gz := gzip.NewWriter(out)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()
	count := 0
	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(filepath.Join("shadownote-public", rel))
		h, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		h.Name = name
		if err := tw.WriteHeader(h); err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(tw, f)
		closeErr := f.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		count++
		return nil
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("exported %d files to %s", count, a.Dest), nil
}
