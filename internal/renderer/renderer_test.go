package renderer

import (
	"strings"
	"testing"
)

func TestDefaultRendererBlocksRawHTML(t *testing.T) {
	r := New(DefaultOptions())
	out, err := r.RenderString(`# Title

<script>alert(1)</script>

<img src=x onerror=alert(1)>
`)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.Contains(out, "<script>") || strings.Contains(out, "onerror=") {
		t.Fatalf("default renderer emitted raw active HTML: %q", out)
	}
}

func TestUnsafeRendererAllowsRawHTMLWhenExplicit(t *testing.T) {
	opts := DefaultOptions()
	opts.Unsafe = true
	r := New(opts)
	out, err := r.RenderString(`<div data-ok="1">raw</div>`)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(out, `<div data-ok="1">raw</div>`) {
		t.Fatalf("unsafe renderer did not preserve raw HTML: %q", out)
	}
}
