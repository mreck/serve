package main

import (
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
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
)

func init() {
	flag.StringVar(&serverAddr, "addr", "0.0.0.0:8000", "The server address")
	flag.BoolVar(&logAsJSON, "json", false, "Format log messages as JSON")
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
	var files []File

	err := filepath.WalkDir(".", func(path string, dirEntry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if dirEntry.IsDir() {
			return nil
		}

		abs, _ := filepath.Abs(path)

		hasher := sha256.New()
		hasher.Write([]byte(path))
		id := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

		files = append(files, File{
			PathHash: id,
			FilePath: abs,
			URL:      fmt.Sprintf("/f/%s", id),
		})

		return nil
	})
	if err != nil {
		slog.Error("walking dir failed", "error", err)
		os.Exit(1)
	}

	t := time.Now()
	r := http.NewServeMux()

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl, _ := template.New("").Parse(index)
		tmpl.Execute(w, map[string]any{"Files": files})
	})

	r.HandleFunc("/data", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		json.NewEncoder(w).Encode(files)
	})

	r.HandleFunc("/f/{hash}", func(w http.ResponseWriter, r *http.Request) {
		hash := r.PathValue("hash")

		for _, f := range files {
			if f.PathHash == hash {
				fp, err := os.Open(f.FilePath)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				defer fp.Close()

				http.ServeContent(w, r, hash, t, fp)
				return
			}
		}

		http.Error(w, "file not found", http.StatusNotFound)
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
