package main

import (
	"context"
	"embed"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/roy0x01/shadownote/internal/app"
	"github.com/roy0x01/shadownote/internal/config"
	"github.com/roy0x01/shadownote/internal/datadir"
	"github.com/roy0x01/shadownote/internal/gui"
	"github.com/roy0x01/shadownote/internal/logging"
	"github.com/roy0x01/shadownote/internal/meta"
	"github.com/roy0x01/shadownote/internal/store"
)

//go:embed themes
var themesFS embed.FS

//go:embed templates
var templatesFS embed.FS

func dbPath() string {
	if p := os.Getenv("SHADOWNOTE_DB"); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "shadownote.db"
	}
	return home + "/.shadownote/shadownote.db"
}

func main() {
	if err := datadir.Init(themesFS, templatesFS); err != nil {
		fail("init data directory", err)
	}

	mode := ""
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}

	cfg := loadOrInit()
	cleanup, err := logging.Setup(cfg)
	if err != nil {
		fail("setup logging", err)
	}
	defer cleanup()

	svc, err := app.New(cfg)
	if err != nil {
		fail("start service", err)
	}
	defer svc.Close()

	switch mode {
	case "", "gui":
		runGUI(svc)
	case "build":
		res, err := svc.BuildSite()
		if err != nil {
			fail("build", err)
		}
		fmt.Printf("built %d posts, %d pages, %d tags in %s\n",
			res.Posts, res.Pages, res.Tags, res.Duration)
	case "deploy":
		summary, err := svc.Deploy()
		if err != nil {
			fail("deploy", err)
		}
		fmt.Println(summary)
	case "version":
		fmt.Printf("%s %s\n", meta.Name, meta.Version)
	default:
		fmt.Printf("unknown command %q (use: gui, build, deploy, version)\n", mode)
		os.Exit(1)
	}
}

func isLegacyDefaultPath(value, legacy string) bool {
	return value == "" || value == legacy
}

func loadOrInit() *config.Config {
	db, err := store.Open(dbPath())
	if err != nil {
		fail("open database", err)
	}
	defer db.Close()

	cfg, err := config.LoadOrSeed(db)
	if err != nil {
		fail("load config", err)
	}
	cfg.Paths.DB = dbPath()

	if contentDir, err := datadir.ContentDir(); err == nil && isLegacyDefaultPath(cfg.Paths.Content, "content") {
		cfg.Paths.Content = contentDir
	}
	if publicDir, err := datadir.PublicDir(); err == nil && isLegacyDefaultPath(cfg.Paths.Public, "public") {
		cfg.Paths.Public = publicDir
	}
	if assetsDir, err := datadir.AssetsDir(); err == nil && isLegacyDefaultPath(cfg.Assets.Directory, "assets") {
		cfg.Assets.Directory = assetsDir
	}
	if themesDir, err := datadir.ThemesDir(); err == nil {
		cfg.Paths.ThemesDir = themesDir
	}
	if templatesDir, err := datadir.TemplatesDir(); err == nil {
		cfg.Paths.TemplatesDir = templatesDir
	}
	if logDir, err := datadir.LogDir(); err == nil && cfg.Logging.File != "" {
		if cfg.Logging.File == "shadownote.log" {
			cfg.Logging.File = logDir + "/shadownote.log"
		}
	}

	if err := cfg.Validate(); err != nil {
		fail("validate config", err)
	}
	return cfg
}

func runGUI(svc *app.Service) {
	addr := "127.0.0.1:3000"

	fmt.Print(guiLaunchPanel(addr))

	srv, err := gui.New(svc, addr)
	if err != nil {
		fail("init gui", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := srv.Start(ctx); err != nil {
		fail("gui", err)
	}
}

func guiLaunchPanel(addr string) string {
	green := "\x1b[38;5;46m"
	reset := "\x1b[0m"
	return fmt.Sprintf("%s▰ %s%s  dashboard ready\n\n  url    %shttp://%s%s\n\n", green, meta.Name, reset, green, addr, reset)
}

func fail(what string, err error) {
	fmt.Fprintf(os.Stderr, "error: %s: %v\n", what, err)
	os.Exit(1)
}
