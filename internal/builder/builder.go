package builder

import (
	"embed"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/roy0x01/shadownote/internal/config"
	"github.com/roy0x01/shadownote/internal/content"
	"github.com/roy0x01/shadownote/internal/renderer"
	"github.com/roy0x01/shadownote/internal/safefs"
	sitetheme "github.com/roy0x01/shadownote/internal/theme"
)

//go:embed templates/*.html templates/*.css
var assets embed.FS

type ContentSource interface {
	Publishable() ([]*content.Document, error)
	RenderHTML(doc *content.Document) (string, error)
	TableOfContents(doc *content.Document) renderer.TOC
}

type Builder struct {
	cfg  *config.Config
	src  ContentSource
	tmpl map[string]*template.Template
	site siteData
}

type baseData struct {
	Site               siteData
	Title              string
	Description        string
	IsHome             bool
	Mermaid            template.HTML
	ThemeName          string
	ThemeColor         string
	AccentColor        string
	Year               int
	BuildTime          string
	CommentsProvider   string
	WebmentionEndpoint string
	Pingback           string
	IndieAuthEndpoint  string
	PlausibleDomain    string
	PlausibleScript    string
	UmamiWebsiteID     string
	UmamiScript        string
	GoatCounterCode    string
	FathomSiteID       string
	Comments           map[string]string
	Webmention         map[string]string
	IndieAuth          map[string]string
	Analytics          map[string]string
	Plausible          map[string]string
	Umami              map[string]string
	GoatCounter        map[string]string
	Fathom             map[string]string
}

type siteData struct {
	Title              string
	Author             string
	Description        string
	Logo               string
	Favicon            string
	BaseURL            string
	URL                string
	Lang               string
	Language           string
	ThemeName          string
	ThemeColor         string
	AccentColor        string
	Year               int
	BuildTime          string
	Copyright          string
	Website            string
	Email              string
	GitHub             string
	LinkedIn           string
	Twitter            string
	RSS                string
	FeedURL            string
	HomeURL            string
	AuthorURL          string
	WebmentionEndpoint string
	Pingback           string
	IndieAuthEndpoint  string
	PlausibleDomain    string
	PlausibleScript    string
	UmamiWebsiteID     string
	UmamiScript        string
	GoatCounterCode    string
	FathomSiteID       string
	CommentsProvider   string
	Webmention         map[string]string
	IndieAuth          map[string]string
	Comments           map[string]string
	Analytics          map[string]string
	Plausible          map[string]string
	Umami              map[string]string
	GoatCounter        map[string]string
	Fathom             map[string]string
	Profile            authorData
	Meta               metaData
}

type metaData struct {
	Title       string
	Description string
	Keywords    string
	Robots      string
	Canonical   string
	Image       string
	TwitterCard string
}

type authorData struct {
	Name     string
	Title    string
	Bio      string
	Avatar   string
	Location string
	Website  string
	Email    string
	GitHub   string
	LinkedIn string
	Twitter  string
}

func themeColor(name string) string {
	switch strings.TrimSpace(strings.ToLower(name)) {
	case "terminal":
		return "#3fb950"
	case "cipher", "plaintext":
		return "#2dd4bf"
	case "daylight":
		return "#16a34a"
	case "mono":
		return "#2563eb"
	default:
		return "#58a6ff"
	}
}

func (b *Builder) base(title, description string, isHome bool, mermaid template.HTML) baseData {
	return baseData{
		Site:               b.site,
		Title:              title,
		Description:        description,
		IsHome:             isHome,
		Mermaid:            mermaid,
		ThemeName:          b.site.ThemeName,
		ThemeColor:         b.site.ThemeColor,
		AccentColor:        b.site.AccentColor,
		Year:               b.site.Year,
		BuildTime:          b.site.BuildTime,
		CommentsProvider:   b.site.CommentsProvider,
		WebmentionEndpoint: b.site.WebmentionEndpoint,
		Pingback:           b.site.Pingback,
		IndieAuthEndpoint:  b.site.IndieAuthEndpoint,
		PlausibleDomain:    b.site.PlausibleDomain,
		PlausibleScript:    b.site.PlausibleScript,
		UmamiWebsiteID:     b.site.UmamiWebsiteID,
		UmamiScript:        b.site.UmamiScript,
		GoatCounterCode:    b.site.GoatCounterCode,
		FathomSiteID:       b.site.FathomSiteID,
		Comments:           b.site.Comments,
		Webmention:         b.site.Webmention,
		IndieAuth:          b.site.IndieAuth,
		Analytics:          b.site.Analytics,
		Plausible:          b.site.Plausible,
		Umami:              b.site.Umami,
		GoatCounter:        b.site.GoatCounter,
		Fathom:             b.site.Fathom,
	}
}

func New(cfg *config.Config, src ContentSource) (*Builder, error) {
	tmpl, err := parseTemplates(cfg.Paths.ThemesDir, cfg.Blog.Theme.Name)
	if err != nil {
		return nil, err
	}
	base := cfg.Blog.BaseURL
	if base == "" {
		base = "/"
	}
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}
	return &Builder{
		cfg:  cfg,
		src:  src,
		tmpl: tmpl,
		site: siteData{
			Title:       cfg.Blog.Title,
			Author:      cfg.Blog.Author,
			Description: cfg.Blog.Description,
			Logo:        cfg.Branding.Logo,
			Favicon:     cfg.Branding.Favicon,
			BaseURL:     base,
			URL:         base,
			Lang:        "en",
			Language:    "en",
			ThemeName:   cfg.Blog.Theme.Name,
			ThemeColor:  themeColor(cfg.Blog.Theme.Name),
			AccentColor: themeColor(cfg.Blog.Theme.Name),
			Year:        time.Now().Year(),
			BuildTime:   time.Now().Format(time.RFC3339),
			Copyright:   firstNonEmpty(cfg.Blog.Author, cfg.Author.Name, cfg.Blog.Title),
			Website:     cfg.Author.Website,
			Email:       cfg.Author.Email,
			GitHub:      cfg.Author.GitHub,
			LinkedIn:    cfg.Author.LinkedIn,
			Twitter:     cfg.Author.Twitter,
			RSS:         base + "index.xml",
			FeedURL:     base + "index.xml",
			HomeURL:     base,
			AuthorURL:   base + "author/",

			WebmentionEndpoint: "",
			Pingback:           "",
			IndieAuthEndpoint:  "",
			PlausibleDomain:    "",
			PlausibleScript:    "",
			UmamiWebsiteID:     "",
			UmamiScript:        "",
			GoatCounterCode:    "",
			FathomSiteID:       "",
			CommentsProvider:   "",

			Webmention:  map[string]string{},
			IndieAuth:   map[string]string{},
			Comments:    map[string]string{},
			Analytics:   map[string]string{},
			Plausible:   map[string]string{},
			Umami:       map[string]string{},
			GoatCounter: map[string]string{},
			Fathom:      map[string]string{},
			Meta: metaData{
				Title:       firstNonEmpty(cfg.Meta.Title, cfg.Blog.Title),
				Description: firstNonEmpty(cfg.Meta.Description, cfg.Blog.Description),
				Keywords:    cfg.Meta.Keywords,
				Robots:      firstNonEmpty(cfg.Meta.Robots, "index,follow"),
				Canonical:   firstNonEmpty(cfg.Meta.Canonical, cfg.Blog.BaseURL),
				Image:       cfg.Branding.SocialImage,
				TwitterCard: firstNonEmpty(cfg.Meta.TwitterCard, "summary_large_image"),
			},
			Profile: authorData{
				Name:     firstNonEmpty(cfg.Author.Name, cfg.Blog.Author),
				Title:    cfg.Author.Title,
				Bio:      cfg.Author.Bio,
				Avatar:   cfg.Author.Avatar,
				Location: cfg.Author.Location,
				Website:  cfg.Author.Website,
				Email:    cfg.Author.Email,
				GitHub:   cfg.Author.GitHub,
				LinkedIn: cfg.Author.LinkedIn,
				Twitter:  cfg.Author.Twitter,
			},
		},
	}, nil
}

var funcs = template.FuncMap{
	"urlize":    content.Slugify,
	"socialURL": socialURL,
}

func socialURL(kind, v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	if strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://") {
		return v
	}
	u := strings.TrimPrefix(v, "@")
	switch kind {
	case "github":
		return "https://github.com/" + u
	case "twitter":
		return "https://x.com/" + u
	case "linkedin":
		return "https://www.linkedin.com/in/" + u
	default:
		return "https://" + v
	}
}

func parseTemplates(themesDir, themeName string) (map[string]*template.Template, error) {
	pages := []string{"post", "list", "tags", "author"}
	out := make(map[string]*template.Template)

	if themeName != "" {
		if err := sitetheme.ValidateTheme(themesDir, themeName); err == nil {
			root := filepath.Join(themesDir, themeName, "layouts")
			for _, name := range pages {
				base := filepath.Join(root, "base.html")
				page := filepath.Join(root, name+".html")
				if _, err := os.Stat(page); err != nil && (name == "tags" || name == "author") {
					page = ""
				}
				var t *template.Template
				var err error
				if page == "" {
					t, err = template.New(name).Funcs(funcs).ParseFS(assets, "templates/base.html", "templates/"+name+".html")
				} else {
					t, err = template.New(name).Funcs(funcs).ParseFiles(base, page)
				}
				if err != nil {
					return nil, fmt.Errorf("parse %s theme template: %w", name, err)
				}
				out[name] = t
			}
			return out, nil
		}
	}

	for _, name := range pages {
		t, err := template.New(name).Funcs(funcs).ParseFS(assets,
			"templates/base.html", "templates/"+name+".html")
		if err != nil {
			return nil, fmt.Errorf("parse %s template: %w", name, err)
		}
		out[name] = t
	}
	return out, nil
}

type Result struct {
	Posts         int
	Pages         int
	Tags          int
	AssetsCopied  int
	AssetsSkipped int
	Duration      time.Duration
}

func (b *Builder) Build() (Result, error) {
	start := time.Now()
	out := b.cfg.Paths.Public

	if err := resetDir(
		out,
		b.cfg.Paths.Content,
		b.cfg.Assets.Directory,
		b.cfg.Paths.ThemesDir,
		b.cfg.Paths.TemplatesDir,
		b.cfg.Paths.DB,
		b.cfg.Logging.File,
	); err != nil {
		return Result{}, err
	}

	if err := b.writeCSS(out); err != nil {
		return Result{}, err
	}

	docs, err := b.src.Publishable()
	if err != nil {
		return Result{}, fmt.Errorf("gather posts: %w", err)
	}
	posts := b.toPosts(docs)
	sort.Slice(posts, func(i, j int) bool {
		return posts[i].Date.After(posts[j].Date)
	})

	res := Result{Posts: len(posts)}

	for i := range posts {
		if err := b.renderPost(out, &posts[i]); err != nil {
			return Result{}, err
		}
		res.Pages++
	}

	n, err := b.renderIndex(out, posts)
	if err != nil {
		return Result{}, err
	}
	res.Pages += n

	tagCount, tagPages, err := b.renderTags(out, posts)
	if err != nil {
		return Result{}, err
	}
	res.Tags = tagCount
	res.Pages += tagPages

	if err := b.renderAuthor(out); err != nil {
		return Result{}, err
	}
	res.Pages++

	if err := b.writeRSS(out, posts); err != nil {
		return Result{}, err
	}
	if err := b.writeSitemap(out, posts); err != nil {
		return Result{}, err
	}

	copied, skipped, err := b.copyAssets(out)
	if err != nil {
		return Result{}, err
	}
	res.AssetsCopied = copied
	res.AssetsSkipped = skipped

	res.Duration = time.Since(start)
	return res, nil
}

func (b *Builder) writeCSS(out string) error {
	css, err := os.ReadFile(filepath.Join(b.cfg.Paths.ThemesDir, b.cfg.Blog.Theme.Name, "assets", "blog.css"))
	if err != nil {
		css, err = assets.ReadFile("templates/blog.css")
		if err != nil {
			return fmt.Errorf("read css: %w", err)
		}
	}
	dir := filepath.Join(out, "css")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	full := baseThemeCSS() + "\n" + string(css) + "\n"
	return os.WriteFile(filepath.Join(dir, "blog.css"), []byte(full), 0644)
}

// resetDir validates targets before deleting generated output.
func resetDir(dir string, protected ...string) error {
	if err := safefs.ResetDir(dir, protected...); err != nil {
		return fmt.Errorf("reset output directory: %w", err)
	}
	return nil
}

func writeFile(out, relpath string, data []byte) error {
	full := filepath.Join(out, relpath)
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		return err
	}
	return os.WriteFile(full, data, 0644)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
