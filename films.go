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
)

func filmsHandler(ctx context.Context, w gemini.ResponseWriter, r *gemini.Request) {
	path := r.URL.Path[len("/films"):]
	b, err := os.ReadFile(strings.TrimSuffix(filmsFolder, "/") + path + ".toml")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			w.WriteHeader(gemini.StatusNotFound, "Not found")
			return
		} else {
			slog.Error("Error while reading file", "err", err, "path", path)
			w.WriteHeader(gemini.StatusPermanentFailure, "Internal error")
			return
		}
	}
	var filmCfg FilmConfig
	if err = toml.Unmarshal(b, &filmCfg); err != nil {
		slog.Error("Error while parsing file", "err", err, "path", path)
		w.WriteHeader(gemini.StatusPermanentFailure, "Internal error")
		return
	}
	//TODO: render file
	t, err := renderFile()
	if err != nil {
		slog.Error("Error while parsing template", "err", err, "path", path)
		w.WriteHeader(gemini.StatusPermanentFailure, "Internal error")
		return
	}

	if err = t.Execute(w, filmCfg); err != nil {
		slog.Error("Error while rendering", "err", err, "path", path)
		w.WriteHeader(gemini.StatusPermanentFailure, "Internal error")
	}
	w.WriteHeader(gemini.StatusSuccess, "Found")
}

func renderFile() (*template.Template, error) {
	return template.New("film").Funcs(template.FuncMap{
		"escape": escape,
	}).Parse(cfg.FilmDisplay)
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
