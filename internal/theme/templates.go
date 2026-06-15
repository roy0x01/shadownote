package theme

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type DocumentTemplate struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Description string `json:"description"`
	Body        string `json:"body,omitempty"`
}

func ListDocumentTemplates(root string) ([]DocumentTemplate, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return []DocumentTemplate{}, nil
		}
		return nil, err
	}
	out := []DocumentTemplate{}
	for _, e := range entries {
		if e.IsDir() {
			p := filepath.Join(root, e.Name(), "template.md")
			if exists(p) {
				out = append(out, docTpl(e.Name(), p))
			}
			continue
		}
		if strings.HasSuffix(e.Name(), ".md") {
			name := strings.TrimSuffix(e.Name(), ".md")
			out = append(out, docTpl(name, filepath.Join(root, e.Name())))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func LoadDocumentTemplate(root, name string) (DocumentTemplate, error) {
	if err := safeName("document template", name); err != nil {
		return DocumentTemplate{Name: name}, err
	}
	candidates := []string{filepath.Join(root, name, "template.md"), filepath.Join(root, name+".md")}
	for _, p := range candidates {
		if exists(p) {
			t := docTpl(name, p)
			b, err := os.ReadFile(p)
			if err != nil {
				return t, err
			}
			t.Body = string(b)
			return t, nil
		}
	}
	return DocumentTemplate{Name: name}, os.ErrNotExist
}

func docTpl(name, path string) DocumentTemplate {
	return DocumentTemplate{Name: name, Path: path, Description: strings.ReplaceAll(name, "-", " ")}
}
