package main

import (
	"context"
	"log/slog"
	"os"

	"serve/config"
	"serve/database"
	"serve/server"
)

func init() {
	config.Load()

	var l *slog.Logger
	if config.Get().LogAsJSON {
		l = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	} else {
		l = slog.New(slog.NewTextHandler(os.Stdout, nil))
	}
	slog.SetDefault(l)
}

func main() {
	ctx := context.Background()

	db, err := database.New(config.Get().Dirs, server.FileURLPrefix)
	if err != nil {
		slog.Error("loading data failed", "error", err)
		os.Exit(1)
	}

	server.Run(ctx, db)
}
