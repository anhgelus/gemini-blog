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
	Author      string   `toml:"author"`
	Year        int      `toml:"year"`
	Description []string `toml:"description"`
	Tags        []string `toml:"tags"`
	Image       string   `toml:"image"`
	Path        string
}

type Tag struct {
	Name  string
	Films []*FilmConfig
}

type homeData struct {
	Films map[string]*FilmConfig
	Tags  map[string]*Tag
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

	filmsMap map[string]*FilmConfig
	tagsMap  map[string]*Tag
)

func initDir(path string) {
	if filmsMap == nil {
		filmsMap = make(map[string]*FilmConfig)
	}
	var entries []os.DirEntry
	var err error
	if len(path) > 0 && path[0] == '/' {
		entries, err = os.ReadDir(strings.TrimSuffix(filmsFolder, "/") + path)
	} else {
		entries, err = os.ReadDir(strings.TrimSuffix(filmsFolder, "/") + "/" + path)
	}
	if err != nil {
		panic(err)
	}
	for _, entry := range entries {
		p := path + "/" + entry.Name()[:strings.LastIndex(entry.Name(), ".")]
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
	if filmsMap == nil {
		initDir("")
	}
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
	if filmsMap == nil {
		initDir("")
	}
	t, err := loadTemplate(cfg.Film.Index)
	if err != nil {
		slog.Error("Error while parsing template", "err", err, "path", "/films/index.gmi")
		w.WriteHeader(gemini.StatusPermanentFailure, "Internal error")
		return
	}
	err = t.Execute(w, &homeData{filmsMap, tagsMap})
	if err != nil {
		slog.Error("Error while writing response", "err", err, "path", "/films/index.gmi")
		w.WriteHeader(gemini.StatusPermanentFailure, "Internal error")
	}
}

func handleFilmsTag(_ context.Context, w gemini.ResponseWriter, r *gemini.Request) {
	if filmsMap == nil {
		initDir("")
	}
	escaped := r.URL.Path[len("/films/tag/"):]
	tag, ok := tagsMap[escaped]
	if !ok {
		w.WriteHeader(gemini.StatusNotFound, "Not found")
		return
	}
	t, err := loadTemplate(cfg.Film.Tag)
	if err != nil {
		slog.Error("Error while parsing template", "err", err, "path", r.URL.Path)
		w.WriteHeader(gemini.StatusPermanentFailure, "Internal error")
		return
	}
	err = t.Execute(w, tag)
	if err != nil {
		slog.Error("Error while writing response", "err", err, "path", r.URL.Path)
		w.WriteHeader(gemini.StatusPermanentFailure, "Internal error")
	}
}

func loadFilm(path string) (*FilmConfig, error) {
	if tagsMap == nil {
		tagsMap = make(map[string]*Tag)
	}
	b, err := os.ReadFile(strings.TrimSuffix(filmsFolder, "/") + path + ".toml")
	if err != nil {
		return nil, err
	}
	var filmCfg FilmConfig
	if err = toml.Unmarshal(b, &filmCfg); err != nil {
		return nil, err
	}
	filmCfg.Path = path[1:]
	filmsMap[path] = &filmCfg
	for _, t := range filmCfg.Tags {
		tag, ok := tagsMap[escape(t)]
		if !ok {
			tag = &Tag{t, []*FilmConfig{}}
			tagsMap[escape(t)] = tag
		}
		tag.Films = append(tag.Films, &filmCfg)
	}
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
