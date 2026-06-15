package app

import (
	"fmt"
	"path/filepath"

	"github.com/roy0x01/shadownote/internal/config"
	"github.com/roy0x01/shadownote/internal/deploy"
)

func (s *Service) Deploy() (string, error) {
	if !s.opMu.TryLock() {
		return "", ErrOperationInProgress
	}
	defer s.opMu.Unlock()

	res, err := s.buildSiteLocked()
	if err != nil {
		return "", err
	}

	cfg := s.ConfigSnapshot()
	switch cfg.Deployment.Target {
	case "", "localhost":
		port := cfg.Deployment.Localhost.Port
		if port == 0 {
			port = 8081
		}
		url, err := s.startLocalServer(port)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("built %d posts and %d pages; running at %s", res.Posts, res.Pages, url), nil
	case "archive":
		_, summary, err := exportArchiveFromBuilt(cfg, res.Posts, res.Pages)
		return summary, err
	default:
		return "", fmt.Errorf("unsupported deployment target %q", cfg.Deployment.Target)
	}
}

func (s *Service) ExportArchive() (string, string, error) {
	if !s.opMu.TryLock() {
		return "", "", ErrOperationInProgress
	}
	defer s.opMu.Unlock()

	res, err := s.buildSiteLocked()
	if err != nil {
		return "", "", err
	}
	return exportArchiveFromBuilt(s.ConfigSnapshot(), res.Posts, res.Pages)
}

func exportArchiveFromBuilt(cfg config.Config, posts, pages int) (string, string, error) {
	dest := filepath.Clean(cfg.Paths.Public) + ".tar.gz"
	summary, err := (&deploy.Archive{Dest: dest}).Deploy(cfg.Paths.Public)
	if err != nil {
		return "", "", err
	}
	return dest, fmt.Sprintf("built %d posts and %d pages; %s", posts, pages, summary), nil
}
