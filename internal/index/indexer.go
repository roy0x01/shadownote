package index

import (
	"strings"

	"github.com/roy0x01/shadownote/internal/content"
	"github.com/roy0x01/shadownote/internal/store"
)

type Indexer struct {
	content *content.Store
	store   *store.Store
}

func New(c *content.Store, s *store.Store) *Indexer {
	return &Indexer{content: c, store: s}
}

func (ix *Indexer) Rebuild() error {
	docs, err := ix.content.List("")
	if err != nil {
		return err
	}

	inputs := make([]store.IndexInput, 0, len(docs))
	for _, d := range docs {
		inputs = append(inputs, store.IndexInput{
			DocMeta: store.DocMeta{
				Slug:     d.Slug,
				Title:    d.Title,
				Type:     string(d.Type),
				Status:   string(d.Status),
				Tags:     strings.Join(d.Tags, ", "),
				Path:     d.Path,
				Created:  d.Created.Format("2006-01-02"),
				Modified: d.Modified.Format("2006-01-02 15:04"),
			},
			Body: d.Body,
		})
	}

	return ix.store.Reindex(inputs)
}
