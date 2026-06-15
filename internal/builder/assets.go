package builder

import (
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	managedassets "github.com/roy0x01/shadownote/internal/assets"
)

var imgSrcRe = regexp.MustCompile(`(?i)<img([^>]+?)src=["']([^"']+)["']`)

var assetRefRe = regexp.MustCompile(`(?:["'(]|/)assets/([^"')\s>?#]+)`)

func (b *Builder) copyAssets(out string) (copied, skipped int, err error) {
	keep, err := b.referencedAssets(out)
	if err != nil {
		return 0, 0, err
	}
	// Defense in depth against the SVG/active-content upload policy: even if a
	// pre-existing SVG/HTML asset (placed before the policy, or by hand) is
	// referenced by content, never copy it to the published site, where it
	// would be served inline as same-origin active content.
	for rel := range keep {
		switch strings.ToLower(filepath.Ext(rel)) {
		case ".svg", ".svgz", ".html", ".htm", ".xhtml", ".xml":
			delete(keep, rel)
		}
	}
	return managedassets.CopyDirFiltered(b.cfg.Assets.Directory, filepath.Join(out, "assets"), keep)
}

func (b *Builder) referencedAssets(out string) (map[string]bool, error) {
	keep := map[string]bool{}
	err := filepath.WalkDir(out, func(p string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(p))
		if ext != ".html" && ext != ".xml" {
			return nil
		}
		data, readErr := os.ReadFile(p)
		if readErr != nil {
			return readErr
		}
		for _, m := range assetRefRe.FindAllStringSubmatch(string(data), -1) {
			raw := m[1]
			if dec, decErr := url.PathUnescape(raw); decErr == nil {
				raw = dec
			}
			keep[filepath.ToSlash(raw)] = true
		}
		return nil
	})
	return keep, err
}

func (b *Builder) maybeInlineAssets(html string) string {
	if !b.cfg.Assets.InlineBase64 {
		return html
	}
	st, err := managedassets.New(b.cfg.Assets.Directory)
	if err != nil {
		return html
	}
	max := int64(b.cfg.Assets.MaxInlineKB) * 1024
	if max <= 0 {
		max = 96 * 1024
	}
	return imgSrcRe.ReplaceAllStringFunc(html, func(tag string) string {
		m := imgSrcRe.FindStringSubmatch(tag)
		if len(m) != 3 {
			return tag
		}
		src := m[2]
		rel := ""
		switch {
		case strings.HasPrefix(src, "/assets/"):
			rel = strings.TrimPrefix(src, "/assets/")
		case strings.HasPrefix(src, "assets/"):
			rel = strings.TrimPrefix(src, "assets/")
		default:
			return tag
		}
		if data, ok := st.DataURI(rel, max); ok {
			return strings.Replace(tag, src, data, 1)
		}
		return tag
	})
}
