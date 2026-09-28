package main

import (
	"context"
	"embed"
	"flag"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/Thiagojm/crv-go/internal/app"
	"github.com/Thiagojm/crv-go/internal/store"
)

//go:embed all:dist
var distEmbed embed.FS

func main() {
	dataDirFlag := flag.String("data-dir", "", "diretório de dados isolado (desenvolvimento/testes)")
	catalogDirFlag := flag.String("catalog-dir", "", "caminho do banco farsight (desenvolvimento/testes)")
	noBrowser := flag.Bool("no-browser", false, "não abrir o navegador automaticamente")
	flag.Parse()

	exe, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	exeDir := filepath.Dir(exe)

	dataDir := *dataDirFlag
	if dataDir == "" {
		dataDir, err = app.DefaultDataDir()
		if err != nil {
			log.Fatal(err)
		}
	}
	catalogDir := *catalogDirFlag
	if catalogDir == "" {
		catalogDir = filepath.Join(exeDir, "farsight")
	}

	lock, err := app.AcquireInstance(dataDir)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = lock.Release() }()

	st, err := store.Open(dataDir)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = st.Close() }()

	sec, err := app.NewSecurity()
	if err != nil {
		log.Fatal(err)
	}

	ui, err := fs.Sub(distEmbed, "dist")
	if err != nil {
		log.Fatal(err)
	}

	report, ready, initErr := app.EnsureCatalog(st, catalogDir)
	srv := &app.Server{
		Store:      st,
		Security:   sec,
		UI:         ui,
		CatalogDir: catalogDir,
		Report:     report,
		Ready:      ready,
		InitError:  initErr,
		OnListen: func(port int, url string) {
			_ = lock.WriteMeta(port, url)
		},
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := srv.ListenAndServe(ctx, !*noBrowser); err != nil && err != context.Canceled {
		log.Fatal(err)
	}
}
