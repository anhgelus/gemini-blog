package main

import (
	"context"
	"errors"
	"git.sr.ht/~sotirisp/go-gemini"
	"github.com/BurntSushi/toml"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"text/template"
)

type FilmConfig struct {
	Title       string   `toml:"title"`
	Description []string `toml:"description"`
	Tags        []string `toml:"tags"`
	Image       string   `toml:"image"`
}

var (
	escapeChars = map[rune]string{
		'é': "e",
		'è': "e",
		'ê': "e",
		'ë': "e",
		'ô': "o",
		'ç': "c",
		'ù': "u",
		'û': "u",
		'ï': "i",
		'î': "i",
	}

	filmsMap = map[string]*FilmConfig{}
)

func init() {
	initDir(strings.TrimSuffix(filmsFolder, "/"))
}

func initDir(path string) {
	entries, err := os.ReadDir(path)
	if err != nil {
		panic(err)
	}
	for _, entry := range entries {
		p := path + "/" + entry.Name()
		if entry.IsDir() {
			initDir(p)
		} else {
			_, err = loadFilm(p)
			if err != nil {
				panic(err)
			}
		}
	}
}

func handleFilms(ctx context.Context, w gemini.ResponseWriter, r *gemini.Request) {
	path := r.URL.Path[len("/films"):]
	if path == "/" || path == "/index.gmi" {
		handleFilmsHome(ctx, w, r)
		return
	}
	filmCfg, ok := filmsMap[path]
	var err error
	if !ok {
		filmCfg, err = loadFilm(path)
	}
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			w.WriteHeader(gemini.StatusNotFound, "Not found")
			return
		} else {
			slog.Error("Error while reading file", "err", err, "path", r.URL.Path)
			w.WriteHeader(gemini.StatusPermanentFailure, "Internal error")
			return
		}
	}
	t, err := loadTemplate(cfg.Film.Display)
	if err != nil {
		slog.Error("Error while parsing template", "err", err, "path", r.URL.Path)
		w.WriteHeader(gemini.StatusPermanentFailure, "Internal error")
		return
	}

	if err = t.Execute(w, filmCfg); err != nil {
		slog.Error("Error while rendering", "err", err, "path", r.URL.Path)
		w.WriteHeader(gemini.StatusPermanentFailure, "Internal error")
		return
	}
}

func handleFilmsHome(_ context.Context, w gemini.ResponseWriter, _ *gemini.Request) {
	t, err := loadTemplate(cfg.Film.Index)
	if err != nil {
		slog.Error("Error while parsing template", "err", err, "path", "/films/index.gmi")
		w.WriteHeader(gemini.StatusPermanentFailure, "Internal error")
		return
	}
	err = t.Execute(w, nil)
	if err != nil {
		slog.Error("Error while writing response", "err", err, "path", "/films/index.gmi")
		w.WriteHeader(gemini.StatusPermanentFailure, "Internal error")
	}
}

func loadFilm(path string) (*FilmConfig, error) {
	b, err := os.ReadFile(strings.TrimSuffix(filmsFolder, "/") + path + ".toml")
	if err != nil {
		return nil, err
	}
	var filmCfg FilmConfig
	if err = toml.Unmarshal(b, &filmCfg); err != nil {
		return nil, err
	}
	filmsMap[path] = &filmCfg
	return &filmCfg, nil
}

func loadTemplate(raw string) (*template.Template, error) {
	return template.New("film").Funcs(template.FuncMap{
		"escape": escape,
	}).Parse(strings.TrimSpace(raw))
}

func escape(s string) string {
	s = strings.ToLower(s)
	for _, r := range []rune(s) {
		if v, ok := escapeChars[r]; ok {
			s = strings.ReplaceAll(s, string(r), v)
		}
	}
	s = strings.ReplaceAll(s, ` `, `-`)
	reg := regexp.MustCompile("[^a-z0-9-]+")
	return reg.ReplaceAllString(s, "")
}
