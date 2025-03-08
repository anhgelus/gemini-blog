package main

import (
	"context"
	"errors"
	"flag"
	"git.sr.ht/~sotirisp/go-gemini"
	"git.sr.ht/~sotirisp/go-gemini/certificate"
	"github.com/BurntSushi/toml"
	"log/slog"
	"os"
	"os/signal"
	"time"
)

type Config struct {
	Domain string `toml:"domain"`
}

var (
	configFile   = "config.toml"
	certsFolder  = "certs"
	publicFolder = "public"
)

func init() {
	flag.StringVar(&configFile, "config", configFile, "config file")
	flag.StringVar(&certsFolder, "certs-folder", certsFolder, "certificates folder")
	flag.StringVar(&publicFolder, "public-folder", publicFolder, "public folder")
}

func main() {
	flag.Parse()
	b, err := os.ReadFile(configFile)
	if err != nil {
		slog.Info(configFile)
		panic(err)
	}
	var cfg Config
	if err = toml.Unmarshal(b, &cfg); err != nil {
		panic(err)
	}

	if err = os.Mkdir(certsFolder, 0755); err != nil && !errors.Is(err, os.ErrExist) {
		panic(err)
	}
	if err = os.Mkdir(publicFolder, 0755); err != nil && !errors.Is(err, os.ErrExist) {
		panic(err)
	}

	certificates := &certificate.Store{}
	certificates.Register(cfg.Domain)
	if err = certificates.Load(certsFolder); err != nil {
		panic(err)
	}

	mux := &gemini.Mux{}
	mux.Handle("/", gemini.FileServer(os.DirFS(publicFolder)))

	server := &gemini.Server{
		Handler:        gemini.LoggingMiddleware(mux),
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   1 * time.Minute,
		GetCertificate: certificates.Get,
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
