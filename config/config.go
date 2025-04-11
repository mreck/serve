package config

import (
	"flag"
	"os"
	"strings"
)

type Config struct {
	ServerAddr string
	LogAsJSON  bool
	RootDir    string
	WithUI     bool
	WithAPI    bool
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
	stringVar(&c.ServerAddr, "addr", "0.0.0.0:8000", "The server address")
	boolVar(&c.LogAsJSON, "json", false, "Format log messages as JSON")
	stringVar(&c.RootDir, "root", ".", "The root dir to serve")
	boolVar(&c.WithUI, "ui", false, "Run with web UI")
	boolVar(&c.WithAPI, "api", true, "Run with API")
	flag.Parse()
}

func Get() Config {
	return c
}
