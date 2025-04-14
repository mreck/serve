package config

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	ServerAddr string
	LogAsJSON  bool
	Dirs       map[string]string
	WithUI     bool
	WithAPI    bool
	AllowEdit  bool
}

var c Config

func stringVar(p *string, name string, value string, usage string) {
	s := os.Getenv("SERVE_" + strings.ToUpper(name))
	if len(s) > 0 {
		value = s
	}
	flag.StringVar(p, name, value, usage)
}

func boolVar(p *bool, name string, value bool, usage string) {
	s := os.Getenv("SERVE_" + strings.ToUpper(name))
	if len(s) > 0 {
		s = strings.ToLower(s)
		if s == "false" || s == "0" {
			value = false
		} else {
			value = true
		}
	}
	flag.BoolVar(p, name, value, usage)
}

func Load() {
	// @TODO: support a config file
	var dirs string
	stringVar(&c.ServerAddr, "addr", "0.0.0.0:8000", "The server address")
	boolVar(&c.LogAsJSON, "json", false, "Format log messages as JSON")
	stringVar(&dirs, "dirs", ".", "The dirs that will be served")
	boolVar(&c.WithUI, "ui", false, "Run with web UI")
	boolVar(&c.WithAPI, "api", true, "Run with JSON API")
	boolVar(&c.AllowEdit, "edit", true, "Allow file editing")
	flag.Parse()

	c.Dirs = map[string]string{}

	for i, dir := range strings.Split(dirs, ";") {
		name := fmt.Sprintf("%d:%s", i, filepath.Base(dir))
		path := dir

		i := strings.Index(dir, "=")
		if i > -1 {
			name = dir[0:i]
			path = dir[i+1:]
		}

		c.Dirs[name] = filepath.Clean(path)
	}
}

func Get() Config {
	return c
}
