package main

import (
	"context"
	"crypto/tls"
	"crypto/x509/pkix"
	"errors"
	"flag"
	"fmt"
	"git.sr.ht/~sotirisp/go-gemini"
	"git.sr.ht/~sotirisp/go-gemini/certificate"
	"github.com/BurntSushi/toml"
	"log/slog"
	"mime"
	"os"
	"os/signal"
	"time"
)

type Config struct {
	Domain   string `toml:"domain"`
	Duration uint   `toml:"duration"`
	Film     struct {
		Index   string `toml:"index"`
		Display string `toml:"display"`
		Tag     string `toml:"tag"`
	} `toml:"film"`
}

var (
	configFile        = "config.toml"
	certsFolder       = "certs"
	publicFolder      = "public"
	filmsFolder       = "films"
	port         uint = 1965

	cfg Config
)

func init() {
	flag.StringVar(&configFile, "config", configFile, "config file")
	flag.StringVar(&certsFolder, "certs-folder", certsFolder, "certificates folder")
	flag.StringVar(&publicFolder, "public-folder", publicFolder, "public folder")
	flag.StringVar(&filmsFolder, "films-folder", filmsFolder, "films folder")
	flag.UintVar(&port, "port", port, "port to use")

	if err := mime.AddExtensionType(".gmi", "text/gemini"); err != nil {
		panic(err)
	}
}

func main() {
	flag.Parse()
	b, err := os.ReadFile(configFile)
	if err != nil {
		slog.Info(configFile)
		panic(err)
	}
	if err = toml.Unmarshal(b, &cfg); err != nil {
		panic(err)
	}

	createFolder(certsFolder)
	createFolder(publicFolder)
	createFolder(filmsFolder)

	certs := &certificate.Store{}
	certs.CreateCertificate = func(scope string) (tls.Certificate, error) {
		options := certificate.CreateOptions{
			Subject: pkix.Name{
				CommonName: scope,
			},
			DNSNames: []string{scope},
			Duration: time.Duration(cfg.Duration) * 24 * time.Hour,
		}
		slog.Info("Creating certificate", "scope", scope, "duration", cfg.Duration)
		return certificate.Create(options)
	}
	certs.Register(cfg.Domain)
	if err = certs.Load(certsFolder); err != nil {
		panic(err)
	}

	mux := &gemini.Mux{}
	mux.HandleFunc("/films/tag/", handleFilmsTag)
	mux.HandleFunc("/films/", handleFilms)
	mux.HandleFunc("/film", handleFilmsHome)
	mux.Handle("/", gemini.FileServer(os.DirFS(publicFolder)))

	server := &gemini.Server{
		Handler:        gemini.LoggingMiddleware(mux),
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   1 * time.Minute,
		GetCertificate: certs.Get,
		Addr:           fmt.Sprintf(":%d", port),
	}

	// starts the server
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	errCh := make(chan error)
	go func() {
		ctx := context.Background()
		slog.Info("Starting...")
		errCh <- server.ListenAndServe(ctx)
	}()

	select {
	case err = <-errCh:
		slog.Error(err.Error())
	case <-c:
		slog.Info("Shutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err = server.Shutdown(ctx); err != nil {
			panic(err)
		}
	}
}

func createFolder(name string) {
	if err := os.Mkdir(name, 0755); err != nil && !errors.Is(err, os.ErrExist) {
		panic(err)
	}
}
