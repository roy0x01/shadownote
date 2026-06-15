package builder

import "bytes"

type authorPageData struct {
	baseData
	Profile authorData
}

func (b *Builder) renderAuthor(out string) error {
	data := authorPageData{
		baseData: b.base("author", b.site.Profile.Bio, false, ""),
		Profile:  b.site.Profile,
	}
	var buf bytes.Buffer
	if err := b.tmpl["author"].ExecuteTemplate(&buf, "base", data); err != nil {
		return err
	}
	return writeFile(out, "author/index.html", buf.Bytes())
}
