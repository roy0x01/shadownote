package gui

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/roy0x01/shadownote/internal/app"
	"github.com/roy0x01/shadownote/internal/config"
)

//go:embed templates/*.html
var templateFiles embed.FS

//go:embed static/css/* static/js/*
var staticFiles embed.FS

type Server struct {
	svc      *app.Service
	addr     string
	pages    map[string]*template.Template
	http     *http.Server
	sessions map[string]time.Time
	mu       sync.Mutex
}

func New(svc *app.Service, addr string) (*Server, error) {
	pages, err := parsePages()
	if err != nil {
		return nil, err
	}
	return &Server{svc: svc, addr: addr, pages: pages, sessions: map[string]time.Time{}}, nil
}

func parsePages() (map[string]*template.Template, error) {
	layoutPages := []string{"dashboard", "content", "editor", "settings", "assets", "deploy"}
	standalone := []string{"login"}

	out := make(map[string]*template.Template)

	for _, name := range layoutPages {
		t, err := template.New(name).ParseFS(templateFiles,
			"templates/partials.html", "templates/base.html", "templates/"+name+".html")
		if err != nil {
			return nil, fmt.Errorf("parse page %s: %w", name, err)
		}
		out[name] = t
	}
	for _, name := range standalone {
		t, err := template.New(name).ParseFS(templateFiles, "templates/partials.html", "templates/"+name+".html")
		if err != nil {
			return nil, fmt.Errorf("parse page %s: %w", name, err)
		}
		out[name] = t
	}
	return out, nil
}

func (s *Server) Start(ctx context.Context) error {
	if err := s.svc.Reindex(); err != nil {
		return fmt.Errorf("startup reindex: %w", err)
	}

	mux := http.NewServeMux()
	s.routes(mux)

	s.http = &http.Server{
		Addr:         s.addr,
		Handler:      securityHeaders(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() { errCh <- s.http.ListenAndServe() }()

	select {
	case <-ctx.Done():
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.http.Shutdown(shutCtx)
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
}

const faviconSVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64"><rect width="64" height="64" rx="14" fill="#050807"/><text x="16" y="45" font-family="ui-monospace,monospace" font-size="36" fill="#22c55e">▰</text></svg>`

const contentSecurityPolicy = "default-src 'self'; " +
	"script-src 'self'; " +
	"style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data:; font-src 'self'; connect-src 'self'; " +
	"object-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'"

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "same-origin")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), interest-cohort=()")
		h.Set("Content-Security-Policy", contentSecurityPolicy)
		next.ServeHTTP(w, r)
	})
}

func (s *Server) routes(mux *http.ServeMux) {
	mux.HandleFunc("/favicon.svg", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(faviconSVG))
	})
	mux.Handle("/static/", http.FileServer(http.FS(staticFiles)))
	mux.Handle("/assets/", s.guard(http.StripPrefix("/assets/", http.FileServer(http.Dir(s.svc.Config().Assets.Directory))).ServeHTTP))
	mux.Handle("/theme-assets/", s.guard(http.StripPrefix("/theme-assets/", http.FileServer(http.Dir(s.svc.Config().Paths.ThemesDir))).ServeHTTP))

	mux.HandleFunc("/login", s.handleLogin)
	mux.HandleFunc("/logout", s.handleLogout)

	mux.HandleFunc("/", s.guard(s.pageDashboard))
	mux.HandleFunc("/content", s.guard(s.pageContent))
	mux.HandleFunc("/editor", s.guard(s.pageEditor))
	mux.HandleFunc("/assets", s.guard(s.pageAssets))
	mux.HandleFunc("/assets/upload", s.guard(s.handleAssetUploadFallback))
	mux.HandleFunc("/deploy", s.guard(s.pageDeploy))
	mux.HandleFunc("/settings", s.guard(s.pageSettings))
	mux.HandleFunc("/api/", s.guard(s.api))
}

func (s *Server) guard(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.authed(r) {
			if !s.sameOrigin(r) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next(w, r)
			return
		}
		if isAPIPath(r.URL.Path) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		http.Redirect(w, r, "/login", http.StatusFound)
	}
}

func isAPIPath(p string) bool {
	return len(p) >= 4 && p[:4] == "/api"
}

func (s *Server) authed(r *http.Request) bool {
	c, err := r.Cookie("auth")
	if err != nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	expires, ok := s.sessions[c.Value]
	if !ok || time.Now().After(expires) {
		delete(s.sessions, c.Value)
		return false
	}
	return true
}

func (s *Server) sameOrigin(r *http.Request) bool {
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return strings.EqualFold(u.Host, r.Host)
}

type loginData struct {
	Setup bool
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if s.authed(r) {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	if r.Method == http.MethodPost {
		if !s.sameOrigin(r) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if s.passwordConfigured() {
			s.handlePasswordLogin(w, r)
			return
		}
		s.handlePasswordSetup(w, r)
		return
	}
	s.renderLogin(w, "")
}

func (s *Server) passwordConfigured() bool {
	return strings.TrimSpace(s.svc.ConfigSnapshot().Auth.PasswordHash) != ""
}

func (s *Server) renderLogin(w http.ResponseWriter, errText string) {
	setup := !s.passwordConfigured()
	title := "sign in"
	if setup {
		title = "set password"
	}
	s.render(w, "login", pageData{Title: title, Error: errText, Standalone: true, Styles: []string{"pages/login.css"}, Script: "login.js", Data: loginData{Setup: setup}})
}

func (s *Server) handlePasswordSetup(w http.ResponseWriter, r *http.Request) {
	password := r.FormValue("password")
	if len(password) < 8 {
		s.renderLogin(w, "password must be at least 8 characters")
		return
	}
	if password != r.FormValue("confirm_password") {
		s.renderLogin(w, "passwords do not match")
		return
	}
	hash, err := hashPassword(password)
	if err != nil {
		http.Error(w, "could not create password", http.StatusInternalServerError)
		return
	}
	if err := s.svc.SaveConfigWith(func(cfg *config.Config) error {
		if strings.TrimSpace(cfg.Auth.PasswordHash) != "" {
			return fmt.Errorf("dashboard password is already set")
		}
		cfg.Auth.PasswordHash = hash
		return nil
	}); err != nil {
		s.renderLogin(w, err.Error())
		return
	}
	s.startSession(w, r)
}

func (s *Server) handlePasswordLogin(w http.ResponseWriter, r *http.Request) {
	password := r.FormValue("password")
	hash := s.svc.ConfigSnapshot().Auth.PasswordHash
	ok, err := verifyPassword(password, hash)
	if err != nil || !ok {
		s.renderLogin(w, "invalid password")
		return
	}
	s.startSession(w, r)
}

func (s *Server) startSession(w http.ResponseWriter, r *http.Request) {
	session, err := randomSessionToken()
	if err != nil {
		http.Error(w, "could not create session", http.StatusInternalServerError)
		return
	}
	s.mu.Lock()
	if s.sessions == nil {
		s.sessions = map[string]time.Time{}
	}
	s.sessions[session] = time.Now().Add(7 * 24 * time.Hour)
	s.mu.Unlock()
	http.SetCookie(w, &http.Cookie{
		Name:     "auth",
		Value:    session,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   86400 * 7,
	})
	http.Redirect(w, r, "/", http.StatusFound)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie("auth"); err == nil {
		s.mu.Lock()
		delete(s.sessions, c.Value)
		s.mu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{Name: "auth", Value: "", Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: -1})
	http.Redirect(w, r, "/login", http.StatusFound)
}

func randomSessionToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

type pageData struct {
	Title       string
	Nav         string
	Styles      []string
	Script      string
	Error       string
	Standalone  bool
	HeaderClass string
	ActionClass string
	Data        interface{}
}

func (s *Server) render(w http.ResponseWriter, name string, data pageData) {
	t, ok := s.pages[name]
	if !ok {
		http.Error(w, "unknown page: "+name, http.StatusInternalServerError)
		return
	}

	entry := "base"
	if name == "login" {
		entry = "login"
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, entry, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
