package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"serve/assets"
	"serve/config"
	"serve/database"

	"github.com/mreck/gotils/httptils"
)

const (
	FileURLPrefix = "/f/"
)

type Server struct {
	server *http.Server
	cfg    config.Config
}

func New(ctx context.Context, db *database.DB, cfg config.Config) (*Server, error) {
	creationTime := time.Now()

	tfuncs := template.FuncMap{
		"GetQueryStr": func(f database.File) string {
			return strings.ToLower(fmt.Sprintf("dir:%s %s", f.DirName, f.RelPath))
		},
	}

	t, err := template.New("").Funcs(tfuncs).ParseFS(assets.Template, "template/*.html")
	if err != nil {
		slog.Error("loading templates failed", "error", err)
		os.Exit(1)
	}

	r := http.NewServeMux()

	if cfg.WithUI {
		html := httptils.NewHTMLHandler(ctx, t)

		r.Handle("/static/", http.FileServer(http.FS(assets.Static)))

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

	if cfg.WithAPI {
		h := httptils.NewJSONHandler(ctx)

		r.HandleFunc("/api/data", h.H(
			func(ctx context.Context, r *http.Request) (int, any, error) {
				return http.StatusOK, db.GetAllFiles(), nil
			}))

		r.HandleFunc("/api/reload", h.H(
			func(ctx context.Context, r *http.Request) (int, any, error) {
				return http.StatusOK, "ok", db.Reload()
			}))

		if cfg.AllowEdit {
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

	r.HandleFunc(FileURLPrefix+"{hash}",
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

			http.ServeContent(w, r, hash, creationTime, fp)
		})

	srv := &http.Server{
		Handler:      r,
		Addr:         cfg.ServerAddr,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	return &Server{srv, cfg}, nil
}

func (s *Server) Run() <-chan error {
	onClose := make(chan error)

	slog.Info("starting server",
		"addr", s.cfg.ServerAddr,
		"dirs", s.cfg.Dirs,
		"ui", s.cfg.WithUI,
		"api", s.cfg.WithAPI)

	go func() {
		err := s.server.ListenAndServe()
		if err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				onClose <- nil
			} else {
				onClose <- err
			}
		}
	}()

	return onClose
}
