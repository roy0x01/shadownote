package theme

import (
	"fmt"
	"path/filepath"
	"strings"
)

func safeName(kind, name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("%s name is required", kind)
	}
	if filepath.IsAbs(name) || strings.Contains(name, "\\") {
		return fmt.Errorf("invalid %s name %q", kind, name)
	}
	clean := filepath.Clean(name)
	if clean == "." || clean != name || strings.HasPrefix(clean, "..") || strings.Contains(clean, string(filepath.Separator)+".."+string(filepath.Separator)) || strings.Contains(clean, "..") {
		return fmt.Errorf("invalid %s name %q", kind, name)
	}
	return nil
}
