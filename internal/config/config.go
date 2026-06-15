package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Blog       Blog       `yaml:"blog"`
	Branding   Branding   `yaml:"branding"`
	Meta       Meta       `yaml:"meta"`
	Author     Author     `yaml:"author"`
	Editor     Editor     `yaml:"editor"`
	Paths      Paths      `yaml:"paths"`
	Deployment Deployment `yaml:"deployment"`
	Assets     Assets     `yaml:"assets"`
	Logging    Logging    `yaml:"logging"`
	Auth       Auth       `yaml:"auth"`
}

type Blog struct {
	Title        string `yaml:"title"`
	Description  string `yaml:"description"`
	Author       string `yaml:"author"`
	BaseURL      string `yaml:"base_url"`
	PostsPerPage int    `yaml:"posts_per_page"`
	Mermaid      bool   `yaml:"mermaid"`
	Theme        Theme  `yaml:"theme"`
}

type Theme struct {
	Name string `yaml:"name"`
}

type Branding struct {
	Logo        string `yaml:"logo"`
	Favicon     string `yaml:"favicon"`
	SocialImage string `yaml:"social_image"`
}

type Meta struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	Keywords    string `yaml:"keywords"`
	Robots      string `yaml:"robots"`
	Canonical   string `yaml:"canonical"`
	TwitterCard string `yaml:"twitter_card"`
}

type Author struct {
	Name     string `yaml:"name"`
	Title    string `yaml:"title"`
	Bio      string `yaml:"bio"`
	Avatar   string `yaml:"avatar"`
	Location string `yaml:"location"`
	Website  string `yaml:"website"`
	Email    string `yaml:"email"`
	GitHub   string `yaml:"github"`
	LinkedIn string `yaml:"linkedin"`
	Twitter  string `yaml:"twitter"`
}

type Editor struct {
	DefaultTemplate string `yaml:"default_template"`
}

type Paths struct {
	Content      string `yaml:"content"`
	Public       string `yaml:"public"`
	DB           string `yaml:"db"`
	ThemesDir    string `yaml:"themes_dir"`
	TemplatesDir string `yaml:"templates_dir"`
}

type Assets struct {
	Directory    string `yaml:"directory"`
	InlineBase64 bool   `yaml:"inline_base64"`
	MaxInlineKB  int    `yaml:"max_inline_kb"`
}

type Logging struct {
	Level string `yaml:"level"`
	File  string `yaml:"file"`
}

type Auth struct {
	PasswordHash string `yaml:"password_hash"`
}

type Deployment struct {
	Target    string      `yaml:"target"`
	Localhost LocalTarget `yaml:"localhost"`
}

type LocalTarget struct {
	Port int `yaml:"port"`
}

func Default() Config {
	var c Config
	applyDefaults(&c)
	return c
}

func defaultDataPath(name string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return name
	}
	return filepath.Join(home, ".shadownote", name)
}

func applyDefaults(c *Config) {
	if c.Blog.PostsPerPage == 0 {
		c.Blog.PostsPerPage = 10
	}
	if c.Meta.Robots == "" {
		c.Meta.Robots = "index,follow"
	}
	if c.Meta.TwitterCard == "" {
		c.Meta.TwitterCard = "summary_large_image"
	}
	if c.Blog.Theme.Name == "" {
		c.Blog.Theme.Name = "default"
	}
	if c.Author.Name == "" {
		c.Author.Name = c.Blog.Author
	}
	if c.Editor.DefaultTemplate == "" {
		c.Editor.DefaultTemplate = "blog"
	}
	if c.Paths.Content == "" {
		c.Paths.Content = defaultDataPath("content")
	}
	if c.Paths.Public == "" {
		c.Paths.Public = defaultDataPath("public")
	}
	if c.Paths.DB == "" {
		c.Paths.DB = defaultDataPath("shadownote.db")
	}
	if c.Paths.ThemesDir == "" {
		c.Paths.ThemesDir = defaultDataPath("themes")
	}
	if c.Paths.TemplatesDir == "" {
		c.Paths.TemplatesDir = defaultDataPath("templates")
	}
	if c.Assets.Directory == "" {
		c.Assets.Directory = defaultDataPath("assets")
	}
	if c.Assets.MaxInlineKB == 0 {
		c.Assets.MaxInlineKB = 96
	}
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	if c.Logging.File == "" {
		c.Logging.File = defaultDataPath(filepath.Join("logs", "shadownote.log"))
	}
	if c.Deployment.Target == "" {
		c.Deployment.Target = "localhost"
	}
	if c.Deployment.Localhost.Port == 0 {
		c.Deployment.Localhost.Port = 8081
	}
}

func (c *Config) Marshal() (string, error) {
	data, err := yaml.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("marshal config: %w", err)
	}
	return string(data), nil
}

func Unmarshal(data string) (*Config, error) {
	var c Config
	if err := yaml.Unmarshal([]byte(data), &c); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	applyDefaults(&c)
	return &c, nil
}

type blobStore interface {
	ConfigBlob() (string, bool, error)
	PutConfigBlob(data string) error
}

func LoadOrSeed(s blobStore) (*Config, error) {
	data, ok, err := s.ConfigBlob()
	if err != nil {
		return nil, err
	}
	if !ok {
		def := Default()
		blob, err := def.Marshal()
		if err != nil {
			return nil, err
		}
		if err := s.PutConfigBlob(blob); err != nil {
			return nil, err
		}
		return &def, nil
	}
	cfg, err := Unmarshal(data)
	if err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("stored config is invalid: %w", err)
	}
	return cfg, nil
}

func (c *Config) SaveTo(s blobStore) error {
	if err := c.Validate(); err != nil {
		return err
	}
	blob, err := c.Marshal()
	if err != nil {
		return err
	}
	return s.PutConfigBlob(blob)
}

func (c *Config) Clone() *Config {
	cp := *c
	return &cp
}

func (c *Config) Validate() error {
	if err := validatePositive("blog.posts_per_page", c.Blog.PostsPerPage); err != nil {
		return err
	}
	if err := validatePositive("assets.max_inline_kb", c.Assets.MaxInlineKB); err != nil {
		return err
	}
	if err := validateHTTPURL("blog.base_url", c.Blog.BaseURL); err != nil {
		return err
	}
	if err := validateHTTPURL("meta.canonical", c.Meta.Canonical); err != nil {
		return err
	}
	if err := validateHTTPURL("author.website", c.Author.Website); err != nil {
		return err
	}
	if err := validateOptionalEmail("author.email", c.Author.Email); err != nil {
		return err
	}
	if err := validatePublishPaths(c); err != nil {
		return err
	}
	switch c.Logging.Level {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("logging.level must be 'debug', 'info', 'warn', or 'error', got %q", c.Logging.Level)
	}
	switch c.Deployment.Target {
	case "", "localhost", "archive":
	default:
		return fmt.Errorf("deployment.target must be localhost or archive, got %q", c.Deployment.Target)
	}
	if err := validatePort("deployment.localhost.port", c.Deployment.Localhost.Port); err != nil {
		return err
	}
	return nil
}
