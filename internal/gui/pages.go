package gui

import (
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/roy0x01/shadownote/internal/content"
	"github.com/roy0x01/shadownote/internal/store"
	"github.com/roy0x01/shadownote/internal/theme"
)

func typeDir(t string) string {
	switch t {
	case "post":
		return "posts"
	case "note":
		return "notes"
	case "draft":
		return "drafts"
	case "research":
		return "research"
	case "page":
		return "pages"
	default:
		return "notes"
	}
}

func parseMetaTime(v string) (time.Time, bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return time.Time{}, false
	}
	for _, l := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05", "2006-01-02T15:04:05", "2006-01-02"} {
		if t, err := time.Parse(l, v); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func relTime(modified string, now time.Time) string {
	t, ok := parseMetaTime(modified)
	if !ok {
		return ""
	}
	d := now.Sub(t)
	switch {
	case d < 0:
		return "just now"
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m <= 1 {
			return "1m ago"
		}
		return itoa(m) + "m ago"
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h <= 1 {
			return "1h ago"
		}
		return itoa(h) + "h ago"
	case d < 48*time.Hour:
		return "yesterday"
	case d < 7*24*time.Hour:
		return itoa(int(d.Hours())/24) + "d ago"
	case d < 30*24*time.Hour:
		return itoa(int(d.Hours())/(24*7)) + "w ago"
	default:
		return t.Format("2006-01-02")
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

type metaView struct {
	Slug        string
	Title       string
	Type        string
	Status      string
	Tags        []string
	Path        string
	Modified    string
	Snippet     string
	PublishKind string
	Highlight   bool
}

func rowPublishKind(typ, status string) string {
	switch {
	case status == "live" && typ == "post":
		return "unpublish"
	case typ == "post" || typ == "draft":
		return "publish"
	default:
		return ""
	}
}

func metaToView(m store.DocMeta, now time.Time) metaView {
	var tags []string
	for _, t := range strings.Split(m.Tags, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			tags = append(tags, t)
		}
	}
	title := m.Title
	if title == "" {
		title = m.Slug
	}
	return metaView{
		Slug:        m.Slug,
		Title:       title,
		Type:        m.Type,
		Status:      m.Status,
		Tags:        tags,
		Path:        m.Path,
		Modified:    relTime(m.Modified, now),
		PublishKind: rowPublishKind(m.Type, m.Status),
	}
}

func (s *Server) pageDashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	data, err := s.buildDashboard(time.Now())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.render(w, "dashboard", pageData{
		Title:       "dashboard",
		Nav:         "dashboard",
		ActionClass: "dash-actions",
		Styles:      []string{"components/document-list.css", "pages/dashboard.css"},
		Data:        data,
	})
}

type contentFilterView struct {
	Value  string
	Label  string
	Active bool
}

func (s *Server) pageContent(w http.ResponseWriter, r *http.Request) {
	active := r.URL.Query().Get("type")
	if active != "" && !content.IsValidType(content.Type(active)) {
		http.Error(w, "invalid document type", http.StatusBadRequest)
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	sortOrder := r.URL.Query().Get("sort")
	if sortOrder == "" {
		if query != "" {
			sortOrder = "relevance"
		} else {
			sortOrder = "modified-desc"
		}
	}

	typeFilters := []contentFilterView{
		{"", "all", active == ""},
		{"post", "posts", active == "post"},
		{"draft", "drafts", active == "draft"},
		{"note", "notes", active == "note"},
		{"research", "research", active == "research"},
		{"page", "pages", active == "page"},
	}
	now := time.Now()
	allMetas, err := s.svc.AllMeta()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var stats struct{ All, LivePosts, Drafts, Private int }
	stats.All = len(allMetas)
	for _, m := range allMetas {
		switch {
		case m.Type == "post" && m.Status == "live":
			stats.LivePosts++
		}
		if m.Status == "draft" {
			stats.Drafts++
		}
		if m.Status == "private" {
			stats.Private++
		}
	}

	var views []metaView
	searching := query != ""
	if searching {
		results, serr := s.svc.Search(query, 200)
		if serr != nil {
			http.Error(w, serr.Error(), http.StatusInternalServerError)
			return
		}
		for _, rr := range results {
			if active != "" && rr.Type != active {
				continue
			}
			var tags []string
			for _, t := range strings.Split(rr.Tags, ",") {
				if t = strings.TrimSpace(t); t != "" {
					tags = append(tags, t)
				}
			}
			views = append(views, metaView{
				Slug: rr.Slug, Title: orSlug(rr.Title, rr.Slug), Type: rr.Type, Status: rr.Status,
				Tags: tags, Modified: relTime(rr.Modified, now), Snippet: rr.Snippet,
				PublishKind: rowPublishKind(rr.Type, rr.Status),
			})
		}
	} else {
		for _, m := range allMetas {
			if active != "" && m.Type != active {
				continue
			}
			v := metaToView(m, now)
			views = append(views, v)
		}
	}
	sortContentViews(views, sortOrder, searching)
	if len(views) > 0 && !searching && sortOrder == "modified-desc" {
		views[0].Highlight = true
	}

	s.render(w, "content", pageData{
		Title:  "content",
		Nav:    "content",
		Styles: []string{"components/document-list.css", "components/empty-state.css", "pages/content.css"},
		Script: "content.js",
		Data: struct {
			TypeFilters []contentFilterView
			Docs        []metaView
			ActiveType  string
			InitialQ    string
			Sort        string
			Searching   bool
			Stats       struct{ All, LivePosts, Drafts, Private int }
		}{typeFilters, views, active, query, sortOrder, searching, stats},
	})
}

func orSlug(title, slug string) string {
	if strings.TrimSpace(title) == "" {
		return slug
	}
	return title
}

func sortContentViews(v []metaView, order string, searching bool) {
	switch order {
	case "title-asc":
		sort.SliceStable(v, func(i, j int) bool { return strings.ToLower(v[i].Title) < strings.ToLower(v[j].Title) })
	case "title-desc":
		sort.SliceStable(v, func(i, j int) bool { return strings.ToLower(v[i].Title) > strings.ToLower(v[j].Title) })
	case "modified-asc":
		if !searching {
			for i, j := 0, len(v)-1; i < j; i, j = i+1, j-1 {
				v[i], v[j] = v[j], v[i]
			}
		}
	case "modified-desc", "relevance", "":
	}
}

type editorData struct {
	Slug            string
	Title           string
	Summary         string
	Body            string
	TagString       string
	DocType         string
	DocDir          string
	IsNew           bool
	IsLive          bool
	Stale           bool
	From            string
	Types           []string
	Templates       []theme.DocumentTemplate
	DefaultTemplate string
}

func (s *Server) pageEditor(w http.ResponseWriter, r *http.Request) {
	types := []string{"draft", "note", "post", "research", "page"}
	slug := r.URL.Query().Get("slug")
	from := r.URL.Query().Get("from")

	cfg := s.svc.Config()
	templates, _ := theme.ListDocumentTemplates(cfg.Paths.TemplatesDir)
	data := editorData{
		Types:           types,
		DocType:         "draft",
		DocDir:          typeDir("draft"),
		IsNew:           true,
		From:            from,
		Templates:       templates,
		DefaultTemplate: cfg.Editor.DefaultTemplate,
	}

	if slug != "" {
		doc, err := s.findBySlug(slug)
		if err != nil {
			http.Error(w, "document not found", http.StatusNotFound)
			return
		}
		isLive := string(doc.Status) == "live"
		stale := false
		data = editorData{
			Slug:            doc.Slug,
			Title:           doc.Title,
			Summary:         doc.Summary,
			Body:            doc.Body,
			TagString:       strings.Join(doc.Tags, ", "),
			DocType:         string(doc.Type),
			DocDir:          typeDir(string(doc.Type)),
			IsNew:           false,
			IsLive:          isLive,
			Stale:           stale,
			From:            from,
			Types:           types,
			Templates:       templates,
			DefaultTemplate: s.svc.Config().Editor.DefaultTemplate,
		}
	}

	s.render(w, "editor", pageData{
		Title:       "editor",
		Nav:         "",
		HeaderClass: "editor-head",
		Styles:      []string{"pages/editor.css"},
		Script:      "editor.js",
		Data:        data,
	})
}
