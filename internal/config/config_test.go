package config

import (
	"strings"
	"testing"
)

func TestDefaultConfigIsValid(t *testing.T) {
	cfg := Default()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("default config invalid: %v", err)
	}
	if cfg.Blog.Theme.Name != "default" {
		t.Fatalf("default theme = %q", cfg.Blog.Theme.Name)
	}
	if cfg.Deployment.Localhost.Port != 8081 {
		t.Fatalf("default local port = %d", cfg.Deployment.Localhost.Port)
	}
}

func TestDefaultRuntimePathsUseDataDir(t *testing.T) {
	cfg := Default()
	for name, value := range map[string]string{
		"content": cfg.Paths.Content,
		"public":  cfg.Paths.Public,
		"assets":  cfg.Assets.Directory,
	} {
		if !strings.Contains(value, ".shadownote") {
			t.Fatalf("%s path %q is not under .shadownote", name, value)
		}
	}
}

func TestValidateRejectsInvalidDeploymentTarget(t *testing.T) {
	cfg := Default()
	cfg.Deployment.Target = "cloudflare"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid deployment target error")
	}
}

func TestValidateRejectsInvalidPort(t *testing.T) {
	cfg := Default()
	cfg.Deployment.Localhost.Port = 70000
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid port error")
	}
}

func TestValidateRejectsPublicInsideAssets(t *testing.T) {
	cfg := Default()
	cfg.Assets.Directory = "media"
	cfg.Paths.Public = "media/public"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected public-inside-assets validation error")
	}
}

func TestValidateRejectsAssetsInsidePublic(t *testing.T) {
	cfg := Default()
	cfg.Paths.Public = "public"
	cfg.Assets.Directory = "public/assets"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected assets-inside-public validation error")
	}
}

func TestValidateRejectsPublicInsideContent(t *testing.T) {
	cfg := Default()
	cfg.Paths.Content = "content"
	cfg.Paths.Public = "content/public"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected public-inside-content validation error")
	}
}

func TestValidateRejectsInvalidPublicSettings(t *testing.T) {
	cases := []struct {
		name string
		mut  func(*Config)
	}{
		{name: "posts per page", mut: func(c *Config) { c.Blog.PostsPerPage = 0 }},
		{name: "inline size", mut: func(c *Config) { c.Assets.MaxInlineKB = 0 }},
		{name: "base url", mut: func(c *Config) { c.Blog.BaseURL = "not a url" }},
		{name: "email", mut: func(c *Config) { c.Author.Email = "not-email" }},
		{name: "traversal path", mut: func(c *Config) { c.Paths.Public = "../public" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Default()
			tc.mut(&cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
