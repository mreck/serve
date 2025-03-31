package database

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io/fs"
	"path/filepath"
	"sync"
)

type File struct {
	PathHash string `json:"hash"`
	FilePath string `json:"path"`
	URL      string `json:"url"`
}

type DB struct {
	root  string
	files []File
	m     sync.RWMutex
}

func New(root string) (*DB, error) {
	db := &DB{root: root}

	err := db.Reload()
	if err != nil {
		return nil, err
	}

	return db, nil
}

func (db *DB) Reload() error {
	var files []File

	err := filepath.WalkDir(db.root, func(path string, dirEntry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if dirEntry.IsDir() {
			return nil
		}

		abs, _ := filepath.Abs(path)

		hasher := sha256.New()
		hasher.Write([]byte(path))
		hash := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

		files = append(files, File{
			PathHash: hash,
			FilePath: abs,
			URL:      fmt.Sprintf("/f/%s", hash),
		})

		return nil
	})
	if err != nil {
		return err
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
