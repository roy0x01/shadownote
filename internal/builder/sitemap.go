package builder

import (
	"encoding/xml"
	"html/template"
	"strings"
)

type sitemap struct {
	XMLName xml.Name     `xml:"urlset"`
	Xmlns   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

type sitemapURL struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

func (b *Builder) writeSitemap(out string, posts []Post) error {
	urls := []sitemapURL{{Loc: b.site.BaseURL}, {Loc: b.site.BaseURL + "author/"}}
	for _, p := range posts {
		urls = append(urls, sitemapURL{
			Loc:     b.site.BaseURL + p.URL,
			LastMod: p.Date.Format("2006-01-02"),
		})
	}

	sm := sitemap{
		Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  urls,
	}

	data, err := xml.MarshalIndent(sm, "", "  ")
	if err != nil {
		return err
	}
	data = append([]byte(xml.Header), data...)
	return writeFile(out, "sitemap.xml", data)
}

func (b *Builder) mermaidHTML(renderedBody string) template.HTML {
	if !b.cfg.Blog.Mermaid {
		return ""
	}
	if !strings.Contains(renderedBody, "language-mermaid") {
		return ""
	}
	const include = `<script type="module">
import mermaid from 'https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.esm.min.mjs';
document.querySelectorAll('pre > code.language-mermaid').forEach(function(el){
  var pre = el.parentElement;
  var div = document.createElement('div');
  div.className = 'mermaid';
  div.textContent = el.textContent;
  pre.replaceWith(div);
});
mermaid.initialize({ startOnLoad: true, theme: 'dark' });
mermaid.run();
</script>`
	return template.HTML(include)
}
