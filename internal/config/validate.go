package config

import (
	"fmt"
	"net/mail"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func validatePort(field string, p int) error {
	if p < 0 || p > 65535 {
		return fmt.Errorf("%s must be between 0 and 65535, got %d", field, p)
	}
	return nil
}

func validatePositive(field string, n int) error {
	if n <= 0 {
		return fmt.Errorf("%s must be greater than 0, got %d", field, n)
	}
	return nil
}

func validateHTTPURL(field, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	u, err := url.Parse(value)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("%s must be an absolute http(s) URL", field)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("%s must use http or https", field)
	}
	return nil
}

func validateOptionalEmail(field, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if _, err := mail.ParseAddress(value); err != nil {
		return fmt.Errorf("%s must be a valid email address", field)
	}
	return nil
}

func validatePathValue(field, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("%s cannot be empty", field)
	}
	clean := filepath.Clean(value)
	if clean == "." || clean == ".." || clean == string(filepath.Separator) {
		return fmt.Errorf("%s cannot be unsafe path %q", field, value)
	}
	if strings.HasPrefix(clean, ".."+string(filepath.Separator)) ||
		strings.Contains(clean, string(filepath.Separator)+".."+string(filepath.Separator)) ||
		strings.HasSuffix(clean, string(filepath.Separator)+"..") {
		return fmt.Errorf("%s cannot contain path traversal: %q", field, value)
	}
	return nil
}

func absClean(path string) (string, error) {
	abs, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}

func sameOrInside(path, parent string) (bool, error) {
	pathAbs, err := absClean(path)
	if err != nil {
		return false, err
	}
	parentAbs, err := absClean(parent)
	if err != nil {
		return false, err
	}
	if pathAbs == parentAbs {
		return true, nil
	}
	rel, err := filepath.Rel(parentAbs, pathAbs)
	if err != nil {
		return false, err
	}
	return rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)), nil
}

func validateNotSameOrInside(field, value, protectedField, protected string) error {
	bad, err := sameOrInside(value, protected)
	if err != nil {
		return fmt.Errorf("validate %s against %s: %w", field, protectedField, err)
	}
	if bad {
		return fmt.Errorf("%s must not be the same as or inside %s", field, protectedField)
	}
	return nil
}

func validatePublishPaths(c *Config) error {
	paths := map[string]string{
		"paths.content":       c.Paths.Content,
		"paths.public":        c.Paths.Public,
		"paths.db":            c.Paths.DB,
		"paths.themes_dir":    c.Paths.ThemesDir,
		"paths.templates_dir": c.Paths.TemplatesDir,
		"assets.directory":    c.Assets.Directory,
	}
	if strings.TrimSpace(c.Logging.File) != "" {
		paths["logging.file"] = c.Logging.File
	}
	for field, value := range paths {
		if err := validatePathValue(field, value); err != nil {
			return err
		}
	}

	public := c.Paths.Public
	protectedDirs := map[string]string{
		"paths.content":       c.Paths.Content,
		"assets.directory":    c.Assets.Directory,
		"paths.themes_dir":    c.Paths.ThemesDir,
		"paths.templates_dir": c.Paths.TemplatesDir,
		".git":                ".git",
		"internal":            "internal",
	}
	for field, protected := range protectedDirs {
		if err := validateNotSameOrInside("paths.public", public, field, protected); err != nil {
			return err
		}
	}

	// Also reject the inverse relationship so source folders do not live inside generated output.
	for field, value := range map[string]string{
		"paths.content":       c.Paths.Content,
		"assets.directory":    c.Assets.Directory,
		"paths.themes_dir":    c.Paths.ThemesDir,
		"paths.templates_dir": c.Paths.TemplatesDir,
	} {
		if err := validateNotSameOrInside(field, value, "paths.public", public); err != nil {
			return err
		}
	}

	managedFiles := map[string]string{
		"paths.db": c.Paths.DB,
	}
	if strings.TrimSpace(c.Logging.File) != "" {
		managedFiles["logging.file"] = c.Logging.File
	}
	for field, value := range managedFiles {
		bad, err := sameOrInside(value, public)
		if err != nil {
			return fmt.Errorf("validate %s against paths.public: %w", field, err)
		}
		if bad {
			return fmt.Errorf("%s must not be inside paths.public", field)
		}
		bad, err = sameOrInside(public, value)
		if err != nil {
			return fmt.Errorf("validate paths.public against %s: %w", field, err)
		}
		if bad {
			return fmt.Errorf("paths.public must not be the same as or inside %s", field)
		}
	}

	cwd, err := os.Getwd()
	if err == nil {
		bad, err := sameOrInside(cwd, public)
		if err == nil && bad {
			return fmt.Errorf("paths.public must not contain the current working directory")
		}
	}
	if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
		publicAbs, publicErr := absClean(public)
		homeAbs, homeErr := absClean(home)
		dataAbs, dataErr := absClean(filepath.Join(home, ".shadownote"))
		if publicErr == nil && homeErr == nil && publicAbs == homeAbs {
			return fmt.Errorf("paths.public must not be the home directory")
		}
		if publicErr == nil && dataErr == nil && publicAbs == dataAbs {
			return fmt.Errorf("paths.public must not be the data directory root")
		}
	}
	return nil
}
