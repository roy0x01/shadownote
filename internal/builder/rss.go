package builder

import (
	"encoding/xml"
	"time"

	"github.com/roy0x01/shadownote/internal/meta"
)

type rss struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	Generator   string    `xml:"generator"`
	Items       []rssItem `xml:"item"`
}

type rssItem struct {
	Title   string `xml:"title"`
	Link    string `xml:"link"`
	GUID    string `xml:"guid"`
	PubDate string `xml:"pubDate"`
}

func (b *Builder) writeRSS(out string, posts []Post) error {
	const maxItems = 20
	limit := len(posts)
	if limit > maxItems {
		limit = maxItems
	}

	items := make([]rssItem, 0, limit)
	for _, p := range posts[:limit] {
		link := b.site.BaseURL + p.URL
		items = append(items, rssItem{
			Title:   p.Title,
			Link:    link,
			GUID:    link,
			PubDate: p.Date.Format(time.RFC1123Z),
		})
	}

	feed := rss{
		Version: "2.0",
		Channel: rssChannel{
			Title:       b.site.Title,
			Link:        b.site.BaseURL,
			Description: b.site.Description,
			Generator:   meta.Name,
			Items:       items,
		},
	}

	data, err := xml.MarshalIndent(feed, "", "  ")
	if err != nil {
		return err
	}
	data = append([]byte(xml.Header), data...)
	return writeFile(out, "index.xml", data)
}
