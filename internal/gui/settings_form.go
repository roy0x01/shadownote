package gui

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/roy0x01/shadownote/internal/config"
)

func socialHandle(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	v = strings.TrimPrefix(v, "@")
	for _, p := range []string{
		"https://", "http://", "www.",
		"github.com/", "x.com/", "twitter.com/",
		"linkedin.com/in/", "linkedin.com/",
	} {
		v = strings.TrimPrefix(v, p)
	}
	return strings.Trim(v, "/")
}

func bindSettingsForm(r *http.Request, live *config.Config) (*config.Config, string, error) {
	cfg := live.Clone()

	str := func(name string) string { return strings.TrimSpace(r.PostFormValue(name)) }
	boolv := func(name string) bool {
		v := r.PostFormValue(name)
		return v == "on" || v == "true" || v == "1"
	}
	intv := func(name string, cur int) int {
		v := strings.TrimSpace(r.PostFormValue(name))
		if v == "" {
			return cur
		}
		n, err := strconv.Atoi(v)
		if err != nil {
			return cur
		}
		return n
	}
	strKeep := func(name, cur string) string {
		if vs, ok := r.PostForm[name]; ok && len(vs) > 0 {
			return strings.TrimSpace(vs[0])
		}
		return cur
	}

	binders := map[string]func(){
		"site": func() {
			cfg.Blog.Title = str("blog.title")
			cfg.Blog.Description = str("blog.description")
			cfg.Blog.Author = str("blog.author")
			cfg.Blog.BaseURL = str("blog.base_url")
			cfg.Blog.PostsPerPage = intv("blog.posts_per_page", cfg.Blog.PostsPerPage)
			cfg.Branding.Logo = str("branding.logo")
			cfg.Branding.Favicon = str("branding.favicon")
			cfg.Branding.SocialImage = str("branding.social_image")
		},
		"author": func() {
			cfg.Author.Name = str("author.name")
			cfg.Author.Title = str("author.title")
			cfg.Author.Bio = str("author.bio")
			cfg.Author.Avatar = str("author.avatar")
			cfg.Author.Location = str("author.location")
			cfg.Author.Website = str("author.website")
			cfg.Author.Email = str("author.email")
			cfg.Author.GitHub = socialHandle(str("author.github"))
			cfg.Author.LinkedIn = socialHandle(str("author.linkedin"))
			cfg.Author.Twitter = socialHandle(str("author.twitter"))
		},
		"seo": func() {
			cfg.Meta.Title = str("meta.title")
			cfg.Meta.Description = str("meta.description")
			cfg.Meta.Keywords = str("meta.keywords")
			cfg.Meta.Robots = strKeep("meta.robots", cfg.Meta.Robots)
			cfg.Meta.Canonical = str("meta.canonical")
			cfg.Meta.TwitterCard = strKeep("meta.twitter_card", cfg.Meta.TwitterCard)
		},
		"writing": func() {
			cfg.Editor.DefaultTemplate = str("editor.default_template")
			cfg.Blog.Mermaid = boolv("blog.mermaid")
		},
		"publishing": func() {
			if p := intv("deployment.localhost.port", cfg.Deployment.Localhost.Port); p > 0 {
				cfg.Deployment.Localhost.Port = p
			}
		},
		"assets": func() {
			cfg.Assets.Directory = str("assets.directory")
			cfg.Assets.InlineBase64 = boolv("assets.inline_base64")
			cfg.Assets.MaxInlineKB = intv("assets.max_inline_kb", cfg.Assets.MaxInlineKB)
		},
		"advanced": func() {
			cfg.Paths.Content = str("paths.content")
			cfg.Paths.Public = str("paths.public")
			cfg.Logging.Level = strKeep("logging.level", cfg.Logging.Level)
			cfg.Logging.File = str("logging.file")
		},
	}

	section := strings.TrimSpace(r.PostFormValue("section"))
	if section == "" {
		for _, f := range binders {
			f()
		}
	} else if f, ok := binders[section]; ok {
		f()
	} else {
		return cfg, section, fmt.Errorf("unknown settings section %q", section)
	}

	if err := cfg.Validate(); err != nil {
		return cfg, section, err
	}
	return cfg, section, nil
}
