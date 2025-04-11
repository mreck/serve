package database

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	ErrNotFound = errors.New("not found")
)

type File struct {
	DirName  string `json:"dir"`
	PathHash string `json:"hash"`
	RelPath  string `json:"path"`
	URL      string `json:"url"`
	db       *DB
}

func (f File) FullPath() string {
	return filepath.Join(f.db.dirs[f.DirName], f.RelPath)
}

type DB struct {
	dirs      map[string]string
	urlPrefix string
	files     []File
	m         sync.RWMutex
}

func New(dirs map[string]string, urlPrefix string) (*DB, error) {
	db := &DB{
		dirs:      dirs,
		urlPrefix: urlPrefix,
	}

	err := db.Reload()
	if err != nil {
		return nil, err
	}

	return db, nil
}

func (db *DB) newFile(relPath string, dirName string) File {
	relPath = strings.ReplaceAll(relPath, "\\", "/")
	hasher := sha256.New()
	hasher.Write([]byte(relPath))
	hash := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

	return File{
		DirName:  dirName,
		PathHash: hash,
		RelPath:  relPath,
		URL:      fmt.Sprintf("%s%s", db.urlPrefix, hash),
		db:       db,
	}
}

func (db *DB) Reload() error {
	var files []File

	for dirName, dirPath := range db.dirs {
		err := filepath.WalkDir(dirPath, func(path string, dirEntry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if dirEntry.IsDir() {
				return nil
			}

			rel, err := filepath.Rel(dirPath, path)
			if err != nil {
				return err
			}

			files = append(files, db.newFile(rel, dirName))

			return nil
		})
		if err != nil {
			return err
		}
	}

	db.m.Lock()
	defer db.m.Unlock()
	db.files = files

	return nil
}

func (db *DB) GetAllFiles() []File {
	db.m.RLock()
	defer db.m.RUnlock()

	return db.files
}

func (db *DB) GetFileByHash(hash string) (File, bool) {
	db.m.RLock()
	defer db.m.RUnlock()

	for _, f := range db.files {
		if f.PathHash == hash {
			return f, true
		}
	}

	return File{}, false
}

func (db *DB) RenameFile(hash string, fn string) (File, error) {
	for i, f := range db.files {
		if f.PathHash == hash {
			dirPath := db.dirs[f.DirName]

			err := os.Rename(filepath.Join(dirPath, f.RelPath), filepath.Join(dirPath, fn))
			if err != nil {
				return File{}, err
			}

			f = db.newFile(fn, f.DirName)
			db.files[i] = f

			return f, nil
		}
	}

	return File{}, ErrNotFound
}
