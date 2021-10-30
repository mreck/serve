package main

import (
	"crypto/sha256"
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

	"github.com/gorilla/mux"
)

var index = `
	<!DOCTYPE html>
	<html lang="en">
	<head>
		<meta charset="UTF-8">
		<meta http-equiv="X-UA-Compatible" content="IE=edge">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>serve</title>
	</head>
	<body>
		<ul>
			{{range .Files}}
			<li>
				<a href="{{.URL}}">{{.Path}}</a>
			</li>
			{{end}}
		</ul>
	</body>
	</html>
	`

type File struct {
	ID   string `json:"id"`
	Path string `json:"path"`
	URL  string `json:"url"`
}

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
			ID:   id,
			Path: abs,
			URL:  fmt.Sprintf("/f/%s", id),
		})

		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	t := time.Now()
	r := mux.NewRouter()

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl, _ := template.New("").Parse(index)
		tmpl.Execute(w, map[string]interface{}{"Files": files})
	})

	r.HandleFunc("/data", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(files)
	})

	r.HandleFunc("/f/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]

		for _, f := range files {
			if f.ID == id {
				fmt.Println(f)
				fp, err := os.Open(f.Path)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				defer fp.Close()

				http.ServeContent(w, r, id, t, fp)
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
