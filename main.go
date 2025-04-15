package main

import (
	"context"
	"log/slog"
	"os"

	"serve/config"
	"serve/database"
	"serve/server"
)

func main() {
	ctx := context.Background()

	config.Load()

	var l *slog.Logger
	if config.Get().LogAsJSON {
		l = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	} else {
		l = slog.New(slog.NewTextHandler(os.Stdout, nil))
	}
	slog.SetDefault(l)

	db, err := database.New(config.Get().Dirs, server.FileURLPrefix)
	if err != nil {
		slog.Error("loading data failed", "error", err)
		os.Exit(1)
	}

	s, err := server.New(ctx, db, config.Get())
	if err != nil {
		slog.Error("creating server failed", "error", err)
		os.Exit(1)
	}

	err = <-s.Run()
	if err != nil {
		slog.Error("running server failed", "error", err)
		os.Exit(1)
	}
	slog.Info("server closed")
}
