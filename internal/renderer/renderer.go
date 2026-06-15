package renderer

import (
	"bytes"
	"fmt"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
)

type Renderer struct {
	md goldmark.Markdown
}

type Options struct {
	HighlightStyle string
	LineNumbers    bool
	Unsafe         bool
}

// DefaultOptions keeps raw HTML disabled so preview and published output share the same safe default.
func DefaultOptions() Options {
	return Options{
		HighlightStyle: "github",
		LineNumbers:    false,
		Unsafe:         false,
	}
}

func New(opts Options) *Renderer {
	chromaOpts := []chromahtml.Option{
		chromahtml.WithClasses(true),
	}
	if opts.LineNumbers {
		chromaOpts = append(chromaOpts,
			chromahtml.WithLineNumbers(true),
			chromahtml.LineNumbersInTable(true),
		)
	}

	gmOpts := []goldmark.Option{
		goldmark.WithExtensions(
			extension.GFM,
			extension.Footnote,
			extension.DefinitionList,
			highlighting.NewHighlighting(
				highlighting.WithStyle(opts.HighlightStyle),
				highlighting.WithFormatOptions(chromaOpts...),
			),
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
	}

	htmlOpts := []renderer.Option{
		html.WithHardWraps(),
	}
	if opts.Unsafe {
		htmlOpts = append(htmlOpts, html.WithUnsafe())
	}
	gmOpts = append(gmOpts, goldmark.WithRendererOptions(htmlOpts...))

	return &Renderer{md: goldmark.New(gmOpts...)}
}

func (r *Renderer) Render(source []byte) ([]byte, error) {
	var buf bytes.Buffer
	if err := r.md.Convert(source, &buf); err != nil {
		return nil, fmt.Errorf("render markdown: %w", err)
	}
	return buf.Bytes(), nil
}

func (r *Renderer) RenderString(source string) (string, error) {
	out, err := r.Render([]byte(source))
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (r *Renderer) parse(source []byte) (ast.Node, text.Reader) {
	reader := text.NewReader(source)
	doc := r.md.Parser().Parse(reader)
	return doc, reader
}
