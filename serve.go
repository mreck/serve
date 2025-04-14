package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"serve/config"
	"serve/database"

	"github.com/mreck/gotils/httptils"
)

var (
	//go:embed static/*
	staticFS embed.FS

	//go:embed template/*
	templateFS embed.FS
)

func init() {
	config.Load()

	var l *slog.Logger
	if config.Get().LogAsJSON {
		l = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	} else {
		l = slog.New(slog.NewTextHandler(os.Stdout, nil))
	}
	slog.SetDefault(l)
}

func main() {
	ctx := context.Background()
	fileURLPrefix := "/f/"

	db, err := database.New(config.Get().Dirs, fileURLPrefix)
	if err != nil {
		slog.Error("loading data failed", "error", err)
		os.Exit(1)
	}

	startTime := time.Now()

	tfuncs := template.FuncMap{
		"GetQueryStr": func(f database.File) string {
			return strings.ToLower(fmt.Sprintf("dir:%s %s", f.DirName, f.RelPath))
		},
	}

	t, err := template.New("").Funcs(tfuncs).ParseFS(templateFS, "template/*.html")
	if err != nil {
		slog.Error("loading templates failed", "error", err)
		os.Exit(1)
	}

	r := http.NewServeMux()

	if config.Get().WithUI {
		html := httptils.NewHTMLHandler(ctx, t)

		r.Handle("/static/", http.FileServer(http.FS(staticFS)))

		r.HandleFunc("/", html.H(
			func(ctx context.Context, r *http.Request) (int, string, httptils.D, error) {
				data := httptils.D{
					"Files":   db.GetAllFiles(),
					"Styles":  []string{"styles", "index"},
					"Scripts": []string{"index"},
				}
				return http.StatusOK, "index.html", data, nil
			}))
	}

	if config.Get().WithAPI {
		h := httptils.NewJSONHandler(ctx)

		r.HandleFunc("/api/data", h.H(
			func(ctx context.Context, r *http.Request) (int, any, error) {
				return http.StatusOK, db.GetAllFiles(), nil
			}))

		r.HandleFunc("/api/reload", h.H(
			func(ctx context.Context, r *http.Request) (int, any, error) {
				return http.StatusOK, "ok", db.Reload()
			}))

		if config.Get().AllowEdit {
			r.HandleFunc("/api/rename", h.H(
				func(ctx context.Context, r *http.Request) (int, any, error) {
					if r.Method != http.MethodPost {
						code := http.StatusMethodNotAllowed
						return code, nil, errors.New(http.StatusText(code))
					}

					var data struct {
						Hash string `json:"hash"`
						Name string `json:"name"`
					}
					err := json.NewDecoder(r.Body).Decode(&data)
					if err != nil {
						return http.StatusBadRequest, "", err
					}

					f, err := db.RenameFile(data.Hash, data.Name)
					if err != nil {
						return http.StatusInternalServerError, "", err
					}

					return http.StatusOK, f, db.Reload()
				}))
		}
	}

	r.HandleFunc(fileURLPrefix+"{hash}",
		func(w http.ResponseWriter, r *http.Request) {
			hash := r.PathValue("hash")

			f, ok := db.GetFileByHash(hash)
			if !ok {
				http.Error(w, "file not found", http.StatusNotFound)
				return
			}

			fp, err := os.Open(f.FullPath())
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer fp.Close()

			http.ServeContent(w, r, hash, startTime, fp)
		})

	srv := &http.Server{
		Handler:      r,
		Addr:         config.Get().ServerAddr,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	slog.Info("starting server",
		"addr", config.Get().ServerAddr,
		"dirs", config.Get().Dirs,
		"ui", config.Get().WithUI,
		"api", config.Get().WithAPI)

	err = srv.ListenAndServe()
	if err != nil {
		slog.Error("starting server failed", "error", err)
		os.Exit(1)
	}
}
