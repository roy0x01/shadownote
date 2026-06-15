package gui

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/roy0x01/shadownote/internal/config"
	"github.com/roy0x01/shadownote/internal/theme"
)

const deployWriteDeadline = 10 * time.Minute

func extendWriteDeadline(w http.ResponseWriter, d time.Duration) {
	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(time.Now().Add(d))
}

func truncateMsg(s string) string {
	const max = 300
	s = strings.TrimSpace(s)
	if len(s) > max {
		return s[:max] + "…"
	}
	return s
}

type themeView struct {
	Name        string
	Description string
	Screenshot  string
	Selected    bool
	Valid       bool
}

func (s *Server) pageDeploy(w http.ResponseWriter, r *http.Request) {
	cfg := s.svc.Config()

	var result, resultErr string

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		extendWriteDeadline(w, deployWriteDeadline)

		themeName := strings.TrimSpace(r.PostFormValue("theme"))
		action := strings.TrimSpace(r.PostFormValue("action"))

		var target string
		var okMsg, errMsg string
		switch action {
		case "local":
			target = "localhost"
		case "export":
			target = "archive"
		default:
			errMsg = "choose local preview or local export"
		}
		if errMsg == "" && themeName == "" {
			errMsg = "choose a theme"
		}
		if errMsg == "" && theme.ValidateTheme(cfg.Paths.ThemesDir, themeName) != nil {
			errMsg = "unknown or invalid theme: " + themeName
		}

		if errMsg == "" {
			saveErr := s.svc.SaveConfigWith(func(c *config.Config) error {
				c.Deployment.Target = target
				if themeName != "" {
					c.Blog.Theme.Name = themeName
				}
				return nil
			})
			if saveErr != nil {
				errMsg = saveErr.Error()
			} else if action == "export" {
				archivePath, _, derr := s.svc.ExportArchive()
				if derr != nil {
					errMsg = derr.Error()
				} else {
					w.Header().Set("Content-Type", "application/gzip")
					w.Header().Set("Content-Disposition", `attachment; filename="public.tar.gz"`)
					w.Header().Set("X-Content-Type-Options", "nosniff")
					http.ServeFile(w, r, archivePath)
					return
				}
			} else if summary, derr := s.svc.Deploy(); derr != nil {
				errMsg = derr.Error()
			} else {
				okMsg = summary
			}
		}

		q := url.Values{}
		if errMsg != "" {
			q.Set("err", truncateMsg(errMsg))
		} else {
			q.Set("ok", truncateMsg(okMsg))
		}
		http.Redirect(w, r, "/deploy?"+q.Encode(), http.StatusSeeOther)
		return
	}

	result = r.URL.Query().Get("ok")
	resultErr = r.URL.Query().Get("err")

	themes, _ := theme.ListThemes(cfg.Paths.ThemesDir)
	themeViews := make([]themeView, 0, len(themes))
	for _, t := range themes {
		themeViews = append(themeViews, themeView{
			Name:        t.Name,
			Description: t.Description,
			Screenshot:  t.Screenshot,
			Selected:    t.Name == cfg.Blog.Theme.Name,
			Valid:       t.Valid,
		})
	}

	s.render(w, "deploy", pageData{
		Title:       "preview & export",
		Nav:         "deploy",
		HeaderClass: "deploy-head",
		Styles:      []string{"pages/deploy.css"},
		Script:      "deploy.js",
		Data: struct {
			Themes    []themeView
			Result    string
			ResultErr string
		}{
			Themes:    themeViews,
			Result:    result,
			ResultErr: resultErr,
		},
	})
}
