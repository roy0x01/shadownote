package builder

import (
	"bytes"
	"html/template"
	"time"

	"github.com/roy0x01/shadownote/internal/content"
	"github.com/roy0x01/shadownote/internal/renderer"
)

type Post struct {
	Title   string
	Slug    string
	Date    time.Time
	Tags    []string
	Summary string
	Author  string
	URL     string
	HTML    template.HTML
	TOC     renderer.TOC
}

func (p Post) DateString() string {
	return p.Date.Format("Jan 02, 2006")
}

func (b *Builder) toPosts(docs []*content.Document) []Post {
	posts := make([]Post, 0, len(docs))
	for _, d := range docs {
		html, err := b.src.RenderHTML(d)
		if err != nil {
			html = ""
		}
		posts = append(posts, Post{
			Title:   d.Title,
			Slug:    d.Slug,
			Date:    d.Created,
			Tags:    d.Tags,
			Summary: d.Summary,
			Author:  d.Author,
			URL:     "posts/" + d.Slug + "/",
			HTML:    template.HTML(b.maybeInlineAssets(html)),
			TOC:     b.src.TableOfContents(d),
		})
	}
	return posts
}

type postPageData struct {
	baseData
	Post Post
	Body template.HTML
	TOC  renderer.TOC
}

func (b *Builder) renderPost(out string, p *Post) error {
	data := postPageData{
		baseData: b.base(p.Title, p.Summary, false, b.mermaidHTML(string(p.HTML))),
		Post:     *p,
		Body:     p.HTML,
		TOC:      p.TOC,
	}

	var buf bytes.Buffer
	if err := b.tmpl["post"].ExecuteTemplate(&buf, "base", data); err != nil {
		return err
	}
	return writeFile(out, "posts/"+p.Slug+"/index.html", buf.Bytes())
}
