package preview

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"
)

type Server struct {
	dir  string
	addr string
	http *http.Server
}

func New(dir, addr string) *Server {
	return &Server{dir: dir, addr: addr}
}

func (s *Server) Addr() string { return s.addr }

func (s *Server) Start(ctx context.Context) error {
	info, err := os.Stat(s.dir)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("nothing to serve: %s does not exist (build the site first)", s.dir)
	}

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(s.dir)))

	s.http = &http.Server{
		Addr:         s.addr,
		Handler:      mux,
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
