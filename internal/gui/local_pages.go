package gui

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/roy0x01/shadownote/internal/config"
	"github.com/roy0x01/shadownote/internal/theme"
)

func (s *Server) pageAssets(w http.ResponseWriter, r *http.Request) {
	cfg := s.svc.Config()
	s.render(w, "assets", pageData{
		Title:  "assets",
		Nav:    "assets",
		Styles: []string{"pages/assets.css"},
		Script: "assets.js",
		Data: struct {
			AssetsRoot  string
			PublicPath  string
			MaxUploadMB int64
			Uploaded    string
			Failed      string
		}{
			AssetsRoot:  cfg.Assets.Directory,
			PublicPath:  "/assets",
			MaxUploadMB: MaxAssetUploadBytes >> 20,
			Uploaded:    r.URL.Query().Get("uploaded"),
			Failed:      r.URL.Query().Get("failed"),
		},
	})
}

func (s *Server) handleAssetUploadFallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, MaxAssetUploadBytes)
	if err := r.ParseMultipartForm(16 << 20); err != nil {
		http.Redirect(w, r, "/assets?uploaded=0&failed=1", http.StatusSeeOther)
		return
	}
	files := r.MultipartForm.File["file"]
	uploaded, failed := 0, 0
	for _, header := range files {
		file, err := header.Open()
		if err != nil {
			failed++
			continue
		}
		_, err = s.svc.SaveAsset(header.Filename, file)
		_ = file.Close()
		if err != nil {
			failed++
			continue
		}
		uploaded++
	}
	http.Redirect(w, r, "/assets?uploaded="+strconv.Itoa(uploaded)+"&failed="+strconv.Itoa(failed), http.StatusSeeOther)
}

func (s *Server) pageSettings(w http.ResponseWriter, r *http.Request) {
	cfg := s.svc.Config()
	active := strings.TrimPrefix(r.URL.Fragment, "grp-")
	if active == "" {
		active = strings.TrimPrefix(r.URL.Query().Get("section"), "grp-")
	}
	if active == "" {
		active = "site"
	}

	var formErr string
	saved := r.URL.Query().Get("saved") == "1"
	viewCfg := cfg

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		next, section, err := bindSettingsForm(r, cfg)
		if section != "" {
			active = section
		}
		if err != nil {
			formErr = err.Error()
			viewCfg = next
		} else if err := s.svc.SaveConfigWith(func(c *config.Config) error { *c = *next; return nil }); err != nil {
			formErr = err.Error()
			viewCfg = next
		} else {
			http.Redirect(w, r, "/settings?saved=1&section="+active, http.StatusSeeOther)
			return
		}
	}

	templates, _ := theme.ListDocumentTemplates(cfg.Paths.TemplatesDir)
	s.render(w, "settings", pageData{
		Title:  "settings",
		Nav:    "settings",
		Styles: []string{"pages/settings.css"},
		Script: "settings.js",
		Data: struct {
			Cfg           *config.Config
			Templates     []theme.DocumentTemplate
			ActiveSection string
			Saved         bool
			FormErr       string
		}{
			Cfg:           viewCfg,
			Templates:     templates,
			ActiveSection: active,
			Saved:         saved,
			FormErr:       formErr,
		},
	})
}
