package content

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

func Slugify(title string) string {
	var b strings.Builder
	lastHyphen := false

	for _, r := range strings.ToLower(title) {
		switch {
		case unicode.IsLetter(r) && r < unicode.MaxASCII:
			b.WriteRune(r)
			lastHyphen = false
		case unicode.IsDigit(r) && r < unicode.MaxASCII:
			b.WriteRune(r)
			lastHyphen = false
		case r == ' ' || r == '-' || r == '_':
			if !lastHyphen && b.Len() > 0 {
				b.WriteByte('-')
				lastHyphen = true
			}
		}
	}

	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		slug = "untitled"
	}
	return slug
}

func uniqueSlugGlobal(root string, base string) string {
	candidate := base
	n := 2
	for {
		if !slugExistsInAnyType(root, candidate) {
			return candidate
		}
		candidate = fmt.Sprintf("%s-%d", base, n)
		n++
	}
}

func slugExistsInAnyType(root, slug string) bool {
	for _, t := range AllTypes() {
		path := filepath.Join(root, subdir(t), slug+".md")
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	return false
}
