package main

import (
	_ "embed"
	"encoding/json"
	"flag"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"time"

	"serve/database"
)

var (
	//go:embed index.html
	index string

	serverAddr string
	logAsJSON  bool
	rootDir    string
	noIndex    bool
)

func init() {
	flag.StringVar(&serverAddr, "addr", "0.0.0.0:8000", "The server address")
	flag.BoolVar(&logAsJSON, "json", false, "Format log messages as JSON")
	flag.StringVar(&rootDir, "root", ".", "The root dir to serve")
	flag.BoolVar(&noIndex, "noindex", false, "Don't create an index page")
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
	fileURLPrefix := "/f/"

	db, err := database.New(rootDir, fileURLPrefix)
	if err != nil {
		slog.Error("loading data failed", "error", err)
		os.Exit(1)
	}

	startTime := time.Now()
	r := http.NewServeMux()

	if !noIndex {
		r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			tmpl, _ := template.New("").Parse(index)
			tmpl.Execute(w, map[string]any{"Files": db.GetAllFiles()})
		})
	}

	r.HandleFunc("/data", func(w http.ResponseWriter, r *http.Request) {
		jsonr(w, db.GetAllFiles(), nil)
	})

	r.HandleFunc("/reload", func(w http.ResponseWriter, r *http.Request) {
		jsonr(w, "ok", db.Reload())
	})

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

func jsonr(w http.ResponseWriter, data any, err error) {
	w.Header().Add("Content-Type", "application/json")

	// @TODO: handle encoding errors
	if err != nil {
		json.NewEncoder(w).Encode(struct {
			Error error `json:"error"`
		}{err})
	} else {
		json.NewEncoder(w).Encode(struct {
			Data any `json:"data"`
		}{data})
	}
}
