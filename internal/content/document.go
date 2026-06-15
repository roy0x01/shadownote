package content

import (
	"time"
)

type Type string

const (
	TypePost     Type = "post"
	TypeNote     Type = "note"
	TypeDraft    Type = "draft"
	TypeResearch Type = "research"
	TypePage     Type = "page"
)

type Status string

const (
	StatusLive    Status = "live"
	StatusDraft   Status = "draft"
	StatusPrivate Status = "private"
)

type PublishTarget string

const (
	PublishBlog PublishTarget = "blog"
	PublishNone PublishTarget = "none"
)

type Frontmatter struct {
	Title     string        `yaml:"title"`
	Type      Type          `yaml:"type"`
	Status    Status        `yaml:"status"`
	Tags      []string      `yaml:"tags"`
	Created   time.Time     `yaml:"created"`
	Modified  time.Time     `yaml:"modified"`
	PublishTo PublishTarget `yaml:"publish_to"`
	Summary   string        `yaml:"summary,omitempty"`
	Author    string        `yaml:"author,omitempty"`
}

type Document struct {
	Frontmatter
	Body string
	Path string
	Slug string
}

func (d *Document) Publishable() bool {
	return d.Type == TypePost &&
		d.Status == StatusLive &&
		d.PublishTo == PublishBlog
}

func subdir(t Type) string {
	switch t {
	case TypePost:
		return "posts"
	case TypeNote:
		return "notes"
	case TypeDraft:
		return "drafts"
	case TypeResearch:
		return "research"
	case TypePage:
		return "pages"
	default:
		return "notes"
	}
}

func AllTypes() []Type {
	return []Type{TypePost, TypeNote, TypeDraft, TypeResearch, TypePage}
}

func IsValidType(t Type) bool {
	for _, known := range AllTypes() {
		if t == known {
			return true
		}
	}
	return false
}
