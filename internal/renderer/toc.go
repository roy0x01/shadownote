package renderer

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

type Heading struct {
	Level int
	Text  string
	ID    string
}

type TOC []Heading

func (r *Renderer) TableOfContents(source []byte, start, end int) TOC {
	doc, reader := r.parse(source)

	var toc TOC
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		h, ok := n.(*ast.Heading)
		if !ok {
			return ast.WalkContinue, nil
		}
		if h.Level < start || h.Level > end {
			return ast.WalkContinue, nil
		}

		toc = append(toc, Heading{
			Level: h.Level,
			Text:  headingText(h, reader),
			ID:    headingID(h),
		})
		return ast.WalkContinue, nil
	})

	return toc
}

func headingText(h *ast.Heading, reader text.Reader) string {
	var out []byte
	for c := h.FirstChild(); c != nil; c = c.NextSibling() {
		if t, ok := c.(*ast.Text); ok {
			out = append(out, t.Segment.Value(reader.Source())...)
		}
	}
	return string(out)
}

func headingID(h *ast.Heading) string {
	if id, ok := h.AttributeString("id"); ok {
		if b, ok := id.([]byte); ok {
			return string(b)
		}
	}
	return ""
}

func (t TOC) Empty() bool {
	return len(t) == 0
}
