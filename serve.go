package main

import (
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type File struct {
	Hash string `json:"hash"`
	Path string `json:"path"`
	URL  string `json:"url"`
}

var (
	//go:embed index.html
	index string
)

func main() {
	var files []File

	err := filepath.WalkDir(".", func(path string, dentry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if dentry.IsDir() {
			return nil
		}

		abs, _ := filepath.Abs(path)

		hasher := sha256.New()
		hasher.Write([]byte(path))
		id := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

		files = append(files, File{
			Hash: id,
			Path: abs,
			URL:  fmt.Sprintf("/f/%s", id),
		})

		return nil
	})
	if err != nil {
		log.Fatal(err)
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
			if f.Hash == hash {
				fmt.Println(f)
				fp, err := os.Open(f.Path)
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
		Addr:         "0.0.0.0:8000",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}
