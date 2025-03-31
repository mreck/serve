package main

import (
	_ "embed"
	"encoding/json"
	"flag"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"serve/database"
	"time"
)

type File struct {
	PathHash string `json:"hash"`
	FilePath string `json:"path"`
	URL      string `json:"url"`
}

var (
	//go:embed index.html
	index string

	serverAddr string
	logAsJSON  bool
	rootDir    string
)

func init() {
	flag.StringVar(&serverAddr, "addr", "0.0.0.0:8000", "The server address")
	flag.BoolVar(&logAsJSON, "json", false, "Format log messages as JSON")
	flag.StringVar(&rootDir, "root", ".", "The root dir to serve")
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
	d, err := database.New(rootDir)
	if err != nil {
		slog.Error("loading data failed", "error", err)
		os.Exit(1)
	}

	t := time.Now()
	r := http.NewServeMux()

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl, _ := template.New("").Parse(index)
		tmpl.Execute(w, map[string]any{"Files": d.GetAllFiles()})
	})

	r.HandleFunc("/data", func(w http.ResponseWriter, r *http.Request) {
		jsonr(w, d.GetAllFiles(), nil)
	})

	r.HandleFunc("/reload", func(w http.ResponseWriter, r *http.Request) {
		jsonr(w, "ok", d.Reload())
	})

	r.HandleFunc("/f/{hash}", func(w http.ResponseWriter, r *http.Request) {
		hash := r.PathValue("hash")

		f, ok := d.GetFileByHash(hash)
		if !ok {
			http.Error(w, "file not found", http.StatusNotFound)
			return
		}

		fp, err := os.Open(f.FilePath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer fp.Close()

		http.ServeContent(w, r, hash, t, fp)
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
