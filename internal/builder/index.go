package builder

import (
	"bytes"
	"fmt"
)

type Pager struct {
	Current int
	Total   int
	PrevURL string
	NextURL string
}

type listPageData struct {
	baseData
	Heading string
	Posts   []Post
	Pager   *Pager
}

func (b *Builder) renderIndex(out string, posts []Post) (int, error) {
	per := b.cfg.Blog.PostsPerPage
	if per <= 0 {
		per = 10
	}

	total := (len(posts) + per - 1) / per
	if total == 0 {
		total = 1
	}

	written := 0
	for page := 1; page <= total; page++ {
		startIdx := (page - 1) * per
		endIdx := startIdx + per
		if endIdx > len(posts) {
			endIdx = len(posts)
		}

		var slice []Post
		if startIdx < len(posts) {
			slice = posts[startIdx:endIdx]
		}

		pager := &Pager{Current: page, Total: total}
		if page == 2 {
			pager.PrevURL = "."
		} else if page > 2 {
			pager.PrevURL = fmt.Sprintf("page/%d/", page-1)
		}
		if page < total {
			pager.NextURL = fmt.Sprintf("page/%d/", page+1)
		}

		data := listPageData{
			baseData: b.base(b.site.Title, "", page == 1, ""),
			Posts:    slice,
		}
		if total > 1 {
			data.Pager = pager
		}

		var buf bytes.Buffer
		if err := b.tmpl["list"].ExecuteTemplate(&buf, "base", data); err != nil {
			return written, err
		}

		var rel string
		if page == 1 {
			rel = "index.html"
		} else {
			rel = fmt.Sprintf("page/%d/index.html", page)
		}
		if err := writeFile(out, rel, buf.Bytes()); err != nil {
			return written, err
		}
		written++
	}

	return written, nil
}
