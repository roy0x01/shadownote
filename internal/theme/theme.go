package theme

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/roy0x01/shadownote/internal/meta"
)

type Theme struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	HasBase     bool   `json:"has_base"`
	HasPost     bool   `json:"has_post"`
	HasList     bool   `json:"has_list"`
	HasTags     bool   `json:"has_tags"`
	HasCSS      bool   `json:"has_css"`
	Valid       bool   `json:"valid"`
	Description string `json:"description"`
	Screenshot  string `json:"screenshot"`
}

func ListThemes(root string) ([]Theme, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return []Theme{}, nil
		}
		return nil, err
	}
	out := make([]Theme, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() || e.Name() == "." || e.Name() == ".." {
			continue
		}
		t := LoadTheme(root, e.Name())
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func LoadTheme(root, name string) Theme {
	if err := safeName("theme", name); err != nil {
		return Theme{Name: name, Path: root, Valid: false, Description: err.Error()}
	}
	p := filepath.Join(root, name)
	t := Theme{Name: name, Path: p}
	t.HasBase = exists(filepath.Join(p, "layouts", "base.html"))
	t.HasPost = exists(filepath.Join(p, "layouts", "post.html"))
	t.HasList = exists(filepath.Join(p, "layouts", "list.html"))
	t.HasTags = exists(filepath.Join(p, "layouts", "tags.html"))
	t.HasCSS = exists(filepath.Join(p, "assets", "blog.css"))
	t.Valid = t.HasBase && t.HasPost && t.HasList
	if t.Valid {
		t.Description = describeTheme(name)
		t.Screenshot = "/theme-assets/" + name + "/screenshot.svg"
		_ = ensureScreenshot(p, name)
	} else {
		t.Description = "missing required layouts"
	}
	return t
}

func ValidateTheme(root, name string) error {
	if name == "" {
		return nil
	}
	if err := safeName("theme", name); err != nil {
		return err
	}
	t := LoadTheme(root, name)
	if !exists(t.Path) {
		return fmt.Errorf("theme %q not found", name)
	}
	if !t.Valid {
		return fmt.Errorf("theme %q must include layouts/base.html, post.html, and list.html", name)
	}
	return nil
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func describeTheme(name string) string {
	switch name {
	case "terminal":
		return "dark terminal publication"
	case "mono":
		return "developer documentation style"
	case "cipher":
		return "minimal monospace security"
	case "plaintext":
		return "light monospace technical"
	case "daylight":
		return "friendly light blog"
	case "default":
		return "balanced professional blog"
	default:
		return strings.ReplaceAll(name, "-", " ")
	}
}

func ensureScreenshot(themePath, name string) error {
	p := filepath.Join(themePath, "screenshot.svg")
	if exists(p) {
		return nil
	}
	bg, fg, muted, title, sub := screenshotPalette(name)
	layout := fmt.Sprintf(`<rect x="76" y="134" width="488" height="66" rx="16" fill="%s" fill-opacity=".10" stroke="%s" stroke-opacity=".28"/><rect x="76" y="222" width="488" height="66" rx="16" fill="%s" fill-opacity=".07" stroke="%s" stroke-opacity=".20"/>`, fg, fg, fg, fg)
	switch name {
	case "terminal":
		layout = fmt.Sprintf(`<text x="88" y="158" fill="%s" font-family="ui-monospace,monospace" font-size="18">$ %s build</text><text x="88" y="198" fill="%s" font-family="ui-monospace,monospace" font-size="15">✓ posts ✓ tags ✓ assets</text>`, fg, meta.Name, muted)
	case "mono":
		layout = fmt.Sprintf(`<rect x="70" y="128" width="170" height="200" fill="%s" fill-opacity=".07"/><rect x="268" y="128" width="302" height="200" fill="%s" fill-opacity=".05"/><line x1="292" y1="166" x2="532" y2="166" stroke="%s" stroke-opacity=".45"/>`, fg, fg, fg)
	}
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="640" height="400" viewBox="0 0 640 400"><rect width="640" height="400" rx="24" fill="%s"/><rect x="34" y="32" width="572" height="336" rx="22" fill="none" stroke="%s" stroke-opacity=".32"/><text x="70" y="86" fill="%s" font-family="ui-monospace,monospace" font-size="21" font-weight="800">▰ %s</text><text x="70" y="112" fill="%s" font-family="ui-sans-serif,system-ui" font-size="13">%s</text>%s</svg>`, bg, fg, fg, title, muted, sub, layout)
	return os.WriteFile(p, []byte(svg), 0644)
}

func screenshotPalette(name string) (bg, fg, muted, title, sub string) {
	switch name {
	case "terminal":
		return "#020503", "#3fb950", "#5f9472", "terminal", "dark terminal publication"
	case "cipher":
		return "#0a0e0d", "#2dd4bf", "#6f7d77", "cipher", "minimal monospace security"
	case "plaintext":
		return "#fbfcfb", "#0d9488", "#6b7770", "plaintext", "light monospace technical"
	case "mono":
		return "#ffffff", "#2563eb", "#8b939e", "mono", "developer documentation style"
	case "daylight":
		return "#ffffff", "#16a34a", "#687269", "daylight", "friendly light blog"
	default:
		return "#0d1117", "#58a6ff", "#6e7681", "default", "balanced professional blog"
	}
}
