package builder

import (
	"fmt"
	"strings"
)

type palette struct {
	bg, surface, border, text, muted, accent string
	codeBG                                   string
	green, yellow, red                       string
}

var fallbackPalette = palette{
	bg:      "#0d1117",
	surface: "#161b22",
	border:  "#30363d",
	text:    "#c9d1d9",
	muted:   "#6e7681",
	accent:  "#58a6ff",
	codeBG:  "#1f2428",
	green:   "#3fb950",
	yellow:  "#d29922",
	red:     "#f85149",
}

func baseThemeCSS() string {
	p := fallbackPalette
	var b strings.Builder
	b.WriteString(":root{\n")
	b.WriteString(fmt.Sprintf("  --bg:%s;\n", p.bg))
	b.WriteString(fmt.Sprintf("  --surface:%s;\n", p.surface))
	b.WriteString(fmt.Sprintf("  --border:%s;\n", p.border))
	b.WriteString(fmt.Sprintf("  --text:%s;\n", p.text))
	b.WriteString(fmt.Sprintf("  --muted:%s;\n", p.muted))
	b.WriteString(fmt.Sprintf("  --accent:%s;\n", p.accent))
	b.WriteString(fmt.Sprintf("  --accent-soft:%s;\n", softColor(p.accent)))
	b.WriteString("  --heading:#e6edf3;\n")
	b.WriteString("  --code-text:#ff7b72;\n")
	b.WriteString(fmt.Sprintf("  --code-bg:%s;\n", p.codeBG))
	b.WriteString(fmt.Sprintf("  --green:%s;\n", p.green))
	b.WriteString(fmt.Sprintf("  --yellow:%s;\n", p.yellow))
	b.WriteString(fmt.Sprintf("  --red:%s;\n", p.red))
	b.WriteString("  --sans:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;\n")
	b.WriteString("  --mono:'JetBrains Mono','Fira Code',monospace;\n")
	b.WriteString("}\n")
	return b.String()
}

func softColor(hex string) string {
	r, g, bl, ok := parseHex(hex)
	if !ok {
		return "rgba(128,128,128,0.12)"
	}
	return fmt.Sprintf("rgba(%d,%d,%d,0.14)", r, g, bl)
}

func parseHex(s string) (r, g, b int, ok bool) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	switch len(s) {
	case 3:
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	case 6:
	default:
		return 0, 0, 0, false
	}
	var vals [3]int
	for i := 0; i < 3; i++ {
		v, err := parseHexByte(s[i*2 : i*2+2])
		if err {
			return 0, 0, 0, false
		}
		vals[i] = v
	}
	return vals[0], vals[1], vals[2], true
}

func parseHexByte(s string) (int, bool) {
	n := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		var d int
		switch {
		case c >= '0' && c <= '9':
			d = int(c - '0')
		case c >= 'a' && c <= 'f':
			d = int(c-'a') + 10
		case c >= 'A' && c <= 'F':
			d = int(c-'A') + 10
		default:
			return 0, true
		}
		n = n*16 + d
	}
	return n, false
}
