package gui

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	assetspkg "github.com/roy0x01/shadownote/internal/assets"

	"github.com/roy0x01/shadownote/internal/content"
	"github.com/roy0x01/shadownote/internal/theme"
)

var errNotFound = errors.New("document not found")

func (s *Server) api(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/")

	switch {
	case path == "documents":
		s.apiDocuments(w, r)
	case strings.HasPrefix(path, "documents/"):
		rest := strings.TrimPrefix(path, "documents/")
		if strings.HasSuffix(rest, "/versions") {
			slug := strings.TrimSuffix(rest, "/versions")
			s.apiDocumentVersions(w, r, slug)
		} else {
			s.apiDocument(w, r, rest)
		}
	case path == "assets":
		s.apiAssets(w, r)
	case strings.HasPrefix(path, "assets/"):
		s.apiAsset(w, r, strings.TrimPrefix(path, "assets/"))
	case strings.HasPrefix(path, "document-templates/"):
		s.apiDocumentTemplate(w, r, strings.TrimPrefix(path, "document-templates/"))
	case path == "search":
		s.apiSearch(w, r)
	case path == "render":
		s.apiRender(w, r)
	default:
		http.NotFound(w, r)
	}
}

const maxDocumentJSONBytes = int64(20 << 20)

// decodeJSON caps request size and rejects unknown fields to catch schema drift.
func decodeJSON(w http.ResponseWriter, r *http.Request, max int64, dst interface{}) error {
	r.Body = http.MaxBytesReader(w, r.Body, max)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Warn("write json response", "error", err)
	}
}

func methodAllowed(w http.ResponseWriter, r *http.Request, methods ...string) bool {
	for _, method := range methods {
		if r.Method == method {
			return true
		}
	}
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	return false
}

func parseContentType(raw string) (content.Type, bool) {
	t := content.Type(strings.TrimSpace(raw))
	if t == "" || !content.IsValidType(t) {
		return "", false
	}
	return t, true
}

type docDTO struct {
	Slug    string   `json:"slug"`
	Title   string   `json:"title"`
	Summary string   `json:"summary"`
	Type    string   `json:"type"`
	Status  string   `json:"status"`
	Tags    []string `json:"tags"`
	Path    string   `json:"path"`
	Body    string   `json:"body,omitempty"`
}

func toDTO(d *content.Document, withBody bool) docDTO {
	dto := docDTO{
		Slug:    d.Slug,
		Title:   d.Title,
		Summary: d.Summary,
		Type:    string(d.Type),
		Status:  string(d.Status),
		Tags:    d.Tags,
		Path:    d.Path,
	}
	if withBody {
		dto.Body = d.Body
	}
	return dto
}

func (s *Server) apiDocuments(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		filterRaw := r.URL.Query().Get("type")
		filter := content.Type("")
		if filterRaw != "" {
			var ok bool
			filter, ok = parseContentType(filterRaw)
			if !ok {
				http.Error(w, "invalid document type", http.StatusBadRequest)
				return
			}
		}
		docs, err := s.svc.List(filter)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		out := make([]docDTO, 0, len(docs))
		for _, d := range docs {
			out = append(out, toDTO(d, false))
		}
		writeJSON(w, out)

	case http.MethodPost:
		var req struct {
			Title   string   `json:"title"`
			Summary string   `json:"summary"`
			Type    string   `json:"type"`
			Tags    []string `json:"tags"`
			Body    string   `json:"body"`
		}
		if err := decodeJSON(w, r, maxDocumentJSONBytes, &req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		t := content.TypeDraft
		if req.Type != "" {
			parsed, ok := parseContentType(req.Type)
			if !ok {
				http.Error(w, "invalid document type", http.StatusBadRequest)
				return
			}
			t = parsed
		}
		doc, err := s.svc.Create(req.Title, t, req.Tags, req.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		doc.Summary = cleanString(req.Summary)
		if doc.Summary != "" {
			if err := s.svc.Save(doc); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		writeJSON(w, toDTO(doc, true))

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) apiDocument(w http.ResponseWriter, r *http.Request, slug string) {
	doc, err := s.findBySlug(slug)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		writeJSON(w, toDTO(doc, true))

	case http.MethodPut:
		var req struct {
			Title   string   `json:"title"`
			Summary string   `json:"summary"`
			Type    string   `json:"type"`
			Tags    []string `json:"tags"`
			Body    string   `json:"body"`
		}
		if err := decodeJSON(w, r, maxDocumentJSONBytes, &req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if req.Title != "" {
			doc.Title = req.Title
		}
		if req.Tags != nil {
			doc.Tags = req.Tags
		}
		doc.Summary = cleanString(req.Summary)
		doc.Body = req.Body

		if req.Type != "" {
			t, ok := parseContentType(req.Type)
			if !ok {
				http.Error(w, "invalid document type", http.StatusBadRequest)
				return
			}
			if t != doc.Type {
				if err := s.svc.ChangeType(doc, t); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				writeJSON(w, toDTO(doc, true))
				return
			}
		}

		if err := s.svc.Save(doc); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, toDTO(doc, true))

	case http.MethodDelete:
		if err := s.svc.Delete(doc); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]bool{"ok": true})

	case http.MethodPost:
		switch r.URL.Query().Get("action") {
		case "publish":
			if err := s.svc.Publish(doc); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, toDTO(doc, true))
		case "unpublish":
			if err := s.svc.Unpublish(doc); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, toDTO(doc, true))
		case "restore_version":
			id := strings.TrimSpace(r.URL.Query().Get("id"))
			restored, err := s.svc.RestoreVersion(doc, id)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			writeJSON(w, toDTO(restored, true))
		default:
			http.Error(w, "unknown document action", http.StatusBadRequest)
		}

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) apiDocumentVersions(w http.ResponseWriter, r *http.Request, slug string) {
	if !methodAllowed(w, r, http.MethodGet) {
		return
	}
	doc, err := s.findBySlug(slug)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	versions, err := s.svc.ListVersions(doc)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, versions)
}

const MaxAssetUploadBytes = int64(32 << 20)

func (s *Server) apiAssets(w http.ResponseWriter, r *http.Request) {
	const maxMemory = int64(16 << 20)
	maxUpload := MaxAssetUploadBytes
	switch r.Method {
	case http.MethodGet:
		assets, err := s.svc.ListAssetsWithUsage()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if assets == nil {
			assets = []assetspkg.Asset{}
		}
		writeJSON(w, assets)
	case http.MethodPost:
		r.Body = http.MaxBytesReader(w, r.Body, maxUpload)
		if err := r.ParseMultipartForm(maxMemory); err != nil {
			http.Error(w, "bad multipart upload or file too large", http.StatusBadRequest)
			return
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "missing file", http.StatusBadRequest)
			return
		}
		defer file.Close()
		asset, err := s.svc.SaveAsset(header.Filename, file)
		if err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, assetspkg.ErrDisallowedType) {
				status = http.StatusUnsupportedMediaType
			}
			http.Error(w, err.Error(), status)
			return
		}
		writeJSON(w, asset)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) apiAsset(w http.ResponseWriter, r *http.Request, name string) {
	if !methodAllowed(w, r, http.MethodDelete) {
		return
	}
	if err := s.svc.DeleteAsset(name); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) apiDocumentTemplate(w http.ResponseWriter, r *http.Request, name string) {
	if !methodAllowed(w, r, http.MethodGet) {
		return
	}
	tpl, err := theme.LoadDocumentTemplate(s.svc.Config().Paths.TemplatesDir, name)
	if err != nil {
		http.Error(w, "template not found", http.StatusNotFound)
		return
	}
	writeJSON(w, tpl)
}

type searchResultDTO struct {
	Slug     string   `json:"slug"`
	Title    string   `json:"title"`
	Type     string   `json:"type"`
	Status   string   `json:"status"`
	Tags     []string `json:"tags"`
	Modified string   `json:"modified"`
	Snippet  string   `json:"snippet"`
}

func (s *Server) apiSearch(w http.ResponseWriter, r *http.Request) {
	if !methodAllowed(w, r, http.MethodGet) {
		return
	}
	q := r.URL.Query().Get("q")
	results, err := s.svc.Search(q, 100)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	out := make([]searchResultDTO, 0, len(results))
	for _, r := range results {
		var tags []string
		for _, t := range strings.Split(r.Tags, ",") {
			if t = strings.TrimSpace(t); t != "" {
				tags = append(tags, t)
			}
		}
		out = append(out, searchResultDTO{Slug: r.Slug, Title: r.Title, Type: r.Type, Status: r.Status, Tags: tags, Modified: r.Modified, Snippet: r.Snippet})
	}
	writeJSON(w, out)
}

func (s *Server) apiRender(w http.ResponseWriter, r *http.Request) {
	if !methodAllowed(w, r, http.MethodPost) {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxDocumentJSONBytes)
	var req struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	html, err := s.svc.RenderPreviewHTML(req.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"html": html})
}

func (s *Server) findBySlug(slug string) (*content.Document, error) {
	doc, err := s.svc.FindBySlug(slug)
	if err != nil {
		return nil, errNotFound
	}
	return doc, nil
}

func cleanString(v string) string { return strings.TrimSpace(v) }
