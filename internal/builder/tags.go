package builder

import (
	"bytes"
	"sort"

	"github.com/roy0x01/shadownote/internal/content"
)

type TagCount struct {
	Name  string
	Count int
}

type tagsPageData struct {
	baseData
	Tags []TagCount
}

func (b *Builder) renderTags(out string, posts []Post) (int, int, error) {
	byTag := map[string][]Post{}
	for _, p := range posts {
		for _, t := range p.Tags {
			byTag[t] = append(byTag[t], p)
		}
	}

	var tags []TagCount
	for name, ps := range byTag {
		tags = append(tags, TagCount{Name: name, Count: len(ps)})
	}
	sort.Slice(tags, func(i, j int) bool {
		if tags[i].Count != tags[j].Count {
			return tags[i].Count > tags[j].Count
		}
		return tags[i].Name < tags[j].Name
	})

	pagesWritten := 0

	idxData := tagsPageData{
		baseData: b.base("tags", "", false, ""),
		Tags:     tags,
	}
	var idxBuf bytes.Buffer
	if err := b.tmpl["tags"].ExecuteTemplate(&idxBuf, "base", idxData); err != nil {
		return 0, 0, err
	}
	if err := writeFile(out, "tags/index.html", idxBuf.Bytes()); err != nil {
		return 0, 0, err
	}
	pagesWritten++

	for name, ps := range byTag {
		data := listPageData{
			baseData: b.base(name, "", false, ""),
			Heading:  "tag: " + name,
			Posts:    ps,
		}
		var buf bytes.Buffer
		if err := b.tmpl["list"].ExecuteTemplate(&buf, "base", data); err != nil {
			return 0, 0, err
		}
		slug := content.Slugify(name)
		if err := writeFile(out, "tags/"+slug+"/index.html", buf.Bytes()); err != nil {
			return 0, 0, err
		}
		pagesWritten++
	}

	return len(tags), pagesWritten, nil
}
