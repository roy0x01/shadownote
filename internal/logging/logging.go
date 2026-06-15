package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/roy0x01/shadownote/internal/config"
)

func Setup(cfg *config.Config) (func() error, error) {
	level := slog.LevelInfo
	switch strings.ToLower(strings.TrimSpace(cfg.Logging.Level)) {
	case "debug":
		level = slog.LevelDebug
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	var w io.Writer = os.Stderr
	cleanup := func() error { return nil }
	if strings.TrimSpace(cfg.Logging.File) != "" {
		f, err := os.OpenFile(cfg.Logging.File, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return cleanup, err
		}
		w = f
		cleanup = f.Close
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level})))
	return cleanup, nil
}
