package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/roy0x01/shadownote/internal/assets"
	"github.com/roy0x01/shadownote/internal/builder"
	"github.com/roy0x01/shadownote/internal/config"
	"github.com/roy0x01/shadownote/internal/content"
	"github.com/roy0x01/shadownote/internal/index"
	"github.com/roy0x01/shadownote/internal/preview"
	"github.com/roy0x01/shadownote/internal/renderer"
	"github.com/roy0x01/shadownote/internal/store"
)

var ErrDocumentNotFound = errors.New("document not found")

type Service struct {
	cfg     *config.Config
	content *content.Store
	store   *store.Store
	render  *renderer.Renderer
	// Preview rendering and public rendering both disable raw HTML by default.
	previewRender *renderer.Renderer
	index         *index.Indexer

	localMu     sync.Mutex
	localCancel context.CancelFunc
	localURL    string

	// cfgMu protects the live config pointer during read/mutate/save cycles.
	cfgMu sync.RWMutex

	// opMu prevents concurrent build/deploy writes to the public directory.
	opMu sync.Mutex
}

var ErrOperationInProgress = errors.New("another build or deploy operation is already running")

func New(cfg *config.Config) (*Service, error) {
	cstore, err := content.NewStore(cfg.Paths.Content)
	if err != nil {
		return nil, fmt.Errorf("content store: %w", err)
	}

	db, err := store.Open(cfg.Paths.DB)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	opts := renderer.DefaultOptions()
	rnd := renderer.New(opts)

	previewOpts := opts
	previewRnd := renderer.New(previewOpts)

	idx := index.New(cstore, db)

	s := &Service{
		cfg:           cfg,
		content:       cstore,
		store:         db,
		render:        rnd,
		previewRender: previewRnd,
		index:         idx,
	}

	slog.Info("service_started", "content", cfg.Paths.Content, "db", cfg.Paths.DB, "theme", cfg.Blog.Theme.Name)
	if err := s.index.Rebuild(); err != nil {
		db.Close()
		return nil, fmt.Errorf("build index: %w", err)
	}

	return s, nil
}

func (s *Service) Close() error {
	s.StopLocal()
	return s.store.Close()
}

func (s *Service) Config() *config.Config {
	s.cfgMu.RLock()
	defer s.cfgMu.RUnlock()
	return s.cfg
}

func (s *Service) ConfigSnapshot() config.Config {
	s.cfgMu.RLock()
	defer s.cfgMu.RUnlock()
	return *s.cfg.Clone()
}

func (s *Service) SaveConfig() error {
	return s.SaveConfigWith(nil)
}

func (s *Service) SaveConfigWith(mutate func(*config.Config) error) error {
	s.cfgMu.Lock()
	defer s.cfgMu.Unlock()
	if mutate != nil {
		if err := mutate(s.cfg); err != nil {
			return err
		}
	}
	return s.cfg.SaveTo(s.store)
}

func (s *Service) List(filter content.Type) ([]*content.Document, error) {
	return s.content.List(filter)
}

func (s *Service) Publishable() ([]*content.Document, error) {
	return s.content.Publishable()
}

func (s *Service) Get(path string) (*content.Document, error) {
	return s.content.Read(path)
}

func (s *Service) Create(title string, t content.Type, tags []string, body string) (*content.Document, error) {
	doc, err := s.content.Create(title, t, tags, body)
	if err != nil {
		return nil, err
	}
	return doc, s.index.Rebuild()
}

func (s *Service) Save(doc *content.Document) error {
	slog.Info("document_save", "slug", doc.Slug, "type", doc.Type)
	if err := s.content.Write(doc); err != nil {
		return err
	}
	return s.index.Rebuild()
}

func (s *Service) ChangeType(doc *content.Document, t content.Type) error {
	if err := s.content.ChangeType(doc, t); err != nil {
		return err
	}
	return s.index.Rebuild()
}

func (s *Service) Delete(doc *content.Document) error {
	slog.Warn("document_delete", "slug", doc.Slug, "path", doc.Path)
	if err := s.content.Delete(doc); err != nil {
		return err
	}
	return s.index.Rebuild()
}

func (s *Service) Publish(doc *content.Document) error {
	// Private document types must never be promoted by the generic publish action.
	switch doc.Type {
	case content.TypeNote, content.TypeResearch:
		return fmt.Errorf("refusing to publish a private %s document; change its type to draft or post in the Editor first", doc.Type)
	}
	if err := s.content.Publish(doc); err != nil {
		return err
	}
	return s.index.Rebuild()
}

func (s *Service) Unpublish(doc *content.Document) error {
	if err := s.content.Unpublish(doc); err != nil {
		return err
	}
	return s.index.Rebuild()
}

func (s *Service) ListVersions(doc *content.Document) ([]content.Version, error) {
	return s.content.ListVersions(doc.Slug)
}

func (s *Service) RestoreVersion(doc *content.Document, id string) (*content.Document, error) {
	restored, err := s.content.RestoreVersion(doc, id)
	if err != nil {
		return nil, err
	}
	return restored, s.index.Rebuild()
}

func (s *Service) Search(query string, limit int) ([]store.SearchResult, error) {
	return s.store.Search(query, limit)
}

func (s *Service) AllMeta() ([]store.DocMeta, error) {
	return s.store.AllMeta()
}

func (s *Service) FindBySlug(slug string) (*content.Document, error) {
	meta, ok, err := s.store.MetaBySlug(slug)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrDocumentNotFound
	}
	return s.content.Read(meta.Path)
}

func (s *Service) Reindex() error {
	return s.index.Rebuild()
}

func (s *Service) RenderHTML(doc *content.Document) (string, error) {
	return s.render.RenderString(doc.Body)
}

// RenderPreviewHTML uses the same safe raw-HTML policy as published rendering.
func (s *Service) RenderPreviewHTML(body string) (string, error) {
	return s.previewRender.RenderString(body)
}

func (s *Service) TableOfContents(doc *content.Document) renderer.TOC {
	return s.render.TableOfContents([]byte(doc.Body), 2, 4)
}

func (s *Service) assetStore() (*assets.Store, error) {
	s.cfgMu.RLock()
	dir := s.cfg.Assets.Directory
	s.cfgMu.RUnlock()
	return assets.New(dir)
}

func (s *Service) ListAssets() ([]assets.Asset, error) {
	st, err := s.assetStore()
	if err != nil {
		return nil, err
	}
	return st.List()
}

func (s *Service) ListAssetsWithUsage() ([]assets.Asset, error) {
	list, err := s.ListAssets()
	if err != nil {
		return nil, err
	}
	usage, err := s.ScanAssetUsage()
	if err != nil {
		usage = nil
	}
	for i := range list {
		if refs := usage[list[i].Path]; len(refs) > 0 {
			list[i].Used = true
			list[i].UsageCount = len(refs)
			list[i].UsedBy = refs
		}
	}
	return list, nil
}

func (s *Service) ScanAssetUsage() (map[string][]string, error) {
	st, err := s.assetStore()
	if err != nil {
		return nil, err
	}
	list, err := st.List()
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return map[string][]string{}, nil
	}

	s.cfgMu.RLock()
	contentRoot := s.cfg.Paths.Content
	s.cfgMu.RUnlock()
	usage := make(map[string][]string, len(list))
	err = filepath.WalkDir(contentRoot, func(p string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() || !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			return nil
		}
		data, readErr := os.ReadFile(p)
		if readErr != nil {
			return nil
		}
		text := string(data)
		rel, _ := filepath.Rel(contentRoot, p)
		rel = filepath.ToSlash(rel)
		for _, a := range list {
			if strings.Contains(text, "assets/"+a.Path) {
				usage[a.Path] = append(usage[a.Path], rel)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for k := range usage {
		sort.Strings(usage[k])
	}
	return usage, nil
}

func (s *Service) SaveAsset(name string, r io.Reader) (assets.Asset, error) {
	st, err := s.assetStore()
	if err != nil {
		return assets.Asset{}, err
	}
	return st.Save(name, r)
}

func (s *Service) DeleteAsset(name string) error {
	st, err := s.assetStore()
	if err != nil {
		return err
	}
	return st.Delete(name)
}

// BuildSite holds the operation lock for the whole public-dir write.
func (s *Service) BuildSite() (builder.Result, error) {
	if !s.opMu.TryLock() {
		return builder.Result{}, ErrOperationInProgress
	}
	defer s.opMu.Unlock()
	return s.buildSiteLocked()
}

func (s *Service) buildSiteLocked() (builder.Result, error) {
	cfg := s.ConfigSnapshot()
	slog.Info("build_start", "public", cfg.Paths.Public, "theme", cfg.Blog.Theme.Name)
	b, err := builder.New(&cfg, s)
	if err != nil {
		return builder.Result{}, err
	}
	res, err := b.Build()
	if err != nil {
		slog.Error("build_failed", "error", err)
		return builder.Result{}, err
	}
	slog.Info("build_complete", "posts", res.Posts, "pages", res.Pages, "tags", res.Tags, "duration", res.Duration.String())
	return res, nil
}

func (s *Service) startLocalServer(port int) (string, error) {
	s.cfgMu.RLock()
	cfgPort := s.cfg.Deployment.Localhost.Port
	publicDir := s.cfg.Paths.Public
	s.cfgMu.RUnlock()
	if port == 0 {
		port = cfgPort
	}
	if port == 0 {
		port = 8081
	}

	s.localMu.Lock()
	defer s.localMu.Unlock()

	if s.localCancel != nil {
		s.localCancel()
		s.localCancel = nil
	}

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	ctx, cancel := context.WithCancel(context.Background())

	srv := preview.New(publicDir, addr)
	errCh := make(chan error, 1)
	go func() { errCh <- srv.Start(ctx) }()

	select {
	case err := <-errCh:
		cancel()
		return "", fmt.Errorf("local server: %w", err)
	case <-time.After(150 * time.Millisecond):
	}

	s.localCancel = cancel
	s.localURL = fmt.Sprintf("http://localhost:%d", port)
	return s.localURL, nil
}

func (s *Service) StopLocal() {
	s.localMu.Lock()
	defer s.localMu.Unlock()
	if s.localCancel != nil {
		s.localCancel()
		s.localCancel = nil
		s.localURL = ""
	}
}

func (s *Service) LocalRunning() (bool, string) {
	s.localMu.Lock()
	defer s.localMu.Unlock()
	if s.localCancel == nil {
		return false, ""
	}
	return true, s.localURL
}
