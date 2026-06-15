package datadir

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const bundledManifestName = "bundled-manifest.json"

func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locate home directory: %w", err)
	}
	return filepath.Join(home, ".shadownote"), nil
}

func ThemesDir() (string, error) {
	d, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "themes"), nil
}

func TemplatesDir() (string, error) {
	d, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "templates"), nil
}

func ContentDir() (string, error) {
	d, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "content"), nil
}

func AssetsDir() (string, error) {
	d, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "assets"), nil
}

func PublicDir() (string, error) {
	d, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "public"), nil
}

func LogDir() (string, error) {
	d, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "logs"), nil
}

func Init(themesFS, templatesFS fs.FS) error {
	d, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(d, 0700); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}
	for _, subdir := range []string{"logs", "content", "assets", "public"} {
		if err := os.MkdirAll(filepath.Join(d, subdir), 0700); err != nil {
			return fmt.Errorf("create %s dir: %w", subdir, err)
		}
	}

	manifestPath := filepath.Join(d, bundledManifestName)
	manifest := loadManifest(manifestPath)
	if err := extractFS(themesFS, "themes", filepath.Join(d, "themes"), manifest); err != nil {
		return fmt.Errorf("extract themes: %w", err)
	}
	if err := extractFS(templatesFS, "templates", filepath.Join(d, "templates"), manifest); err != nil {
		return fmt.Errorf("extract templates: %w", err)
	}
	if err := saveManifest(manifestPath, manifest); err != nil {
		return fmt.Errorf("write bundled manifest: %w", err)
	}
	return nil
}

type bundledManifest map[string]string

func loadManifest(path string) bundledManifest {
	data, err := os.ReadFile(path)
	if err != nil {
		return bundledManifest{}
	}
	var m bundledManifest
	if err := json.Unmarshal(data, &m); err != nil || m == nil {
		return bundledManifest{}
	}
	return m
}

func saveManifest(path string, m bundledManifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0600)
}

func extractFS(src fs.FS, srcRoot, dst string, manifest bundledManifest) error {
	return fs.WalkDir(src, srcRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcRoot, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)

		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}

		data, err := fs.ReadFile(src, path)
		if err != nil {
			return err
		}
		newHash := sha256Hex(data)
		key := manifestKey(srcRoot, rel)

		current, statErr := os.ReadFile(target)
		if statErr == nil {
			currentHash := sha256Hex(current)
			if currentHash == newHash {
				manifest[key] = newHash
				return nil
			}
			if oldHash, ok := manifest[key]; !ok || currentHash != oldHash {
				// Existing file was not created by this managed extractor, or it was edited by the user.
				// Preserve it instead of overwriting a custom theme/template.
				return nil
			}
		} else if !os.IsNotExist(statErr) {
			return statErr
		}

		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(target, data, 0644); err != nil {
			return err
		}
		manifest[key] = newHash
		return nil
	})
}

func manifestKey(srcRoot, rel string) string {
	return filepath.ToSlash(filepath.Join(srcRoot, strings.TrimPrefix(rel, string(filepath.Separator))))
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
