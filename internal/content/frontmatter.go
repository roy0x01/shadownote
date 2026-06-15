package content

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const delimiter = "---"

func parse(raw []byte) (Frontmatter, string, error) {
	var fm Frontmatter

	text := string(raw)
	if !strings.HasPrefix(text, delimiter) {
		return fm, strings.TrimSpace(text), nil
	}

	parts := strings.SplitN(text, delimiter, 3)
	if len(parts) < 3 {
		return fm, strings.TrimSpace(text), nil
	}

	front := []byte(parts[1])
	if err := yaml.Unmarshal(front, &fm); err != nil {
		return fm, "", fmt.Errorf("parse frontmatter: %w", err)
	}
	if err := applyFrontmatterAliases(front, &fm); err != nil {
		return fm, "", err
	}

	return fm, strings.TrimSpace(parts[2]), nil
}

type frontmatterAliases struct {
	Description string      `yaml:"description"`
	Date        interface{} `yaml:"date"`
	Draft       *bool       `yaml:"draft"`
}

func applyFrontmatterAliases(raw []byte, fm *Frontmatter) error {
	var aliases frontmatterAliases
	if err := yaml.Unmarshal(raw, &aliases); err != nil {
		return fmt.Errorf("parse frontmatter aliases: %w", err)
	}

	if fm.Summary == "" && aliases.Description != "" {
		fm.Summary = aliases.Description
	}
	if fm.Created.IsZero() && aliases.Date != nil {
		date, err := parseFrontmatterDate(aliases.Date)
		if err != nil {
			return err
		}
		fm.Created = date
	}
	if aliases.Draft != nil {
		if *aliases.Draft {
			fm.Status = StatusDraft
			fm.PublishTo = PublishNone
		} else {
			fm.Status = StatusLive
			fm.PublishTo = PublishBlog
			if fm.Type == "" {
				fm.Type = TypePost
			}
		}
	}
	return nil
}

func parseFrontmatterDate(v interface{}) (time.Time, error) {
	switch x := v.(type) {
	case time.Time:
		return x, nil
	case string:
		for _, layout := range []string{time.RFC3339, "2006-01-02", "2006-01-02 15:04:05", "2006-01-02 15:04"} {
			if t, err := time.Parse(layout, x); err == nil {
				return t, nil
			}
		}
		return time.Time{}, fmt.Errorf("parse frontmatter date %q: unsupported format", x)
	default:
		return time.Time{}, fmt.Errorf("parse frontmatter date: unsupported type %T", v)
	}
}

func serialize(fm Frontmatter, body string) ([]byte, error) {
	var buf bytes.Buffer

	fmBytes, err := yaml.Marshal(fm)
	if err != nil {
		return nil, fmt.Errorf("marshal frontmatter: %w", err)
	}

	buf.WriteString(delimiter)
	buf.WriteByte('\n')
	buf.Write(fmBytes)
	buf.WriteString(delimiter)
	buf.WriteString("\n\n")
	buf.WriteString(strings.TrimSpace(body))
	buf.WriteByte('\n')

	return buf.Bytes(), nil
}

func newFrontmatter(title string, t Type, tags []string) Frontmatter {
	now := time.Now()

	status := StatusDraft
	publish := PublishNone

	switch t {
	case TypeNote, TypeResearch:
		status = StatusPrivate
	case TypePost:
		status = StatusDraft
		publish = PublishBlog
	}

	return Frontmatter{
		Title:     title,
		Type:      t,
		Status:    status,
		Tags:      tags,
		Created:   now,
		Modified:  now,
		PublishTo: publish,
	}
}
