package main_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"serve/config"
	"serve/database"
	"serve/server"

	"github.com/mreck/gotils/httptils"
	"github.com/stretchr/testify/assert"
)

const (
	testDir    = "./testdata"
	serverAddr = "127.0.0.1:8081"
)

func checkSetupFail(task string, err error) {
	if err != nil {
		log.Printf("setup failed: %s: %v", task, err)
		os.Exit(1)
	}
}

func TestMain(m *testing.M) {
	ctx := context.Background()

	err := os.RemoveAll(testDir)
	checkSetupFail("create test dir", err)

	err = os.MkdirAll(testDir, os.ModePerm)
	checkSetupFail("create test dir", err)

	err = os.WriteFile(filepath.Join(testDir, "foo"), []byte("bar"), os.ModePerm)
	checkSetupFail("create test dir", err)

	dirs := map[string]string{"test": testDir}

	db, err := database.New(dirs, server.FileURLPrefix)
	checkSetupFail("create DB", err)

	s, err := server.New(ctx, db, config.Config{
		ServerAddr: serverAddr,
		LogAsJSON:  false,
		Dirs:       dirs,
		WithUI:     true,
		WithAPI:    true,
		AllowEdit:  true,
	})
	checkSetupFail("create server", err)

	s.Run()
	time.Sleep(1 * time.Second)

	os.Exit(m.Run())
}

func fetchJSON(path string, data any, body ...any) error {
	var req *http.Request
	var err error

	url := "http://" + serverAddr + path

	if len(body) > 0 {
		b, err := json.Marshal(body[0])
		if err != nil {
			return err
		}

		req, err = http.NewRequest("POST", url, bytes.NewBuffer(b))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest("GET", url, nil)
		if err != nil {
			return err
		}
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(data)
	if err != nil {
		return err
	}

	return nil
}

func makeFile(relPath string, dirName string) database.File {
	hash := database.HashRelPath(relPath)

	return database.File{
		DirName:  dirName,
		PathHash: hash,
		RelPath:  relPath,
		URL:      fmt.Sprintf("/f/%s", hash),
	}
}

func TestAPIHealth(t *testing.T) {
	var data httptils.JSONResponse[string]
	err := fetchJSON("/api/health", &data)
	assert.Nil(t, err)
	expected := httptils.JSONResponse[string]{Data: "ok", Error: ""}
	assert.Equal(t, expected, data)
}

func TestAPIData(t *testing.T) {
	var data httptils.JSONResponse[[]database.File]
	err := fetchJSON("/api/data", &data)
	assert.Nil(t, err)
	expected := httptils.JSONResponse[[]database.File]{
		Data: []database.File{
			makeFile("foo", "test"),
		},
		Error: "",
	}
	assert.Equal(t, expected, data)
}

func TestAPIRename(t *testing.T) {
	{
		var data httptils.JSONResponse[database.File]
		body := server.RenameRequest{
			Hash: "LCa0a2j_xo_5m0U8HTBBNBNCLXBkg7-g-YpeiGJm564=",
			Name: "baz",
		}
		err := fetchJSON("/api/rename", &data, body)
		assert.Nil(t, err)
		expected := httptils.JSONResponse[database.File]{
			Data:  makeFile("baz", "test"),
			Error: "",
		}
		assert.Equal(t, expected, data)
	}
	{
		var data httptils.JSONResponse[[]database.File]
		err := fetchJSON("/api/data", &data)
		assert.Nil(t, err)
		expected := httptils.JSONResponse[[]database.File]{
			Data: []database.File{
				makeFile("baz", "test"),
			},
			Error: "",
		}
		assert.Equal(t, expected, data)
	}
	{
		ents, err := os.ReadDir(testDir)
		assert.Nil(t, err)
		assert.Equal(t, 1, len(ents))
		assert.Equal(t, "baz", ents[0].Name())

		b, err := os.ReadFile(filepath.Join(testDir, "baz"))
		assert.Nil(t, err)
		assert.Equal(t, b, []byte("bar"))
	}
}
