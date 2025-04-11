package main

import (
	"context"
	"embed"
	"flag"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"time"

	"serve/database"

	"github.com/mreck/gotils/httptils"
)

var (
	//go:embed static/*
	staticFS embed.FS

	//go:embed template/*
	templateFS embed.FS

	serverAddr string
	logAsJSON  bool
	rootDir    string
	withUI     bool
	withAPI    bool
)

func init() {
	flag.StringVar(&serverAddr, "addr", "0.0.0.0:8000", "The server address")
	flag.BoolVar(&logAsJSON, "json", false, "Format log messages as JSON")
	flag.StringVar(&rootDir, "root", ".", "The root dir to serve")
	flag.BoolVar(&withUI, "ui", false, "Run with web UI")
	flag.BoolVar(&withAPI, "api", true, "Run with API")
	flag.Parse()

	var l *slog.Logger
	if logAsJSON {
		l = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	} else {
		l = slog.New(slog.NewTextHandler(os.Stdout, nil))
	}
	slog.SetDefault(l)
}

func main() {
	ctx := context.Background()
	fileURLPrefix := "/f/"

	db, err := database.New(rootDir, fileURLPrefix)
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

	if withUI {
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

	if withAPI {
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
		Addr:         serverAddr,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	slog.Info("starting server", "addr", serverAddr)

	err = srv.ListenAndServe()
	if err != nil {
		slog.Error("starting server failed", "error", err)
		os.Exit(1)
	}
}
