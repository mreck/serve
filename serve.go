package main

import (
	"context"
	"embed"
	"html/template"
	"log/slog"
	"net/http"
	"os"
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

	db, err := database.New(config.Get().RootDir, fileURLPrefix)
	if err != nil {
		slog.Error("loading data failed", "error", err)
		os.Exit(1)
	}

	startTime := time.Now()

	t, err := template.ParseFS(templateFS, "template/*.html")
	if err != nil {
		slog.Error("loading templates failed", "error", err)
		os.Exit(1)
	}

	r := http.NewServeMux()

	if config.Get().WithUI {
		html := httptils.NewHTMLHandler(ctx, t)

		r.Handle("/static/", http.FileServer(http.FS(staticFS)))

		r.HandleFunc("/", html.H(func(ctx context.Context, r *http.Request) (int, string, httptils.D, error) {
			data := httptils.D{
				"Files":   db.GetAllFiles(),
				"Styles":  []string{"styles", "index"},
				"Scripts": []string{"index"},
			}
			return http.StatusOK, "index.html", data, nil
		}))
	}

	if config.Get().WithAPI {
		json := httptils.NewJSONHandler(ctx)

		r.HandleFunc("/api/data", json.H(func(ctx context.Context, r *http.Request) (int, any, error) {
			return http.StatusOK, db.GetAllFiles(), nil
		}))

		r.HandleFunc("/api/reload", json.H(func(ctx context.Context, r *http.Request) (int, any, error) {
			return http.StatusOK, "ok", db.Reload()
		}))
	}

	r.HandleFunc(fileURLPrefix+"{hash}", func(w http.ResponseWriter, r *http.Request) {
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
		"root", config.Get().RootDir,
		"ui", config.Get().WithUI,
		"api", config.Get().WithAPI)

	err = srv.ListenAndServe()
	if err != nil {
		slog.Error("starting server failed", "error", err)
		os.Exit(1)
	}
}
