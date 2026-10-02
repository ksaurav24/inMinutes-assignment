package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"api/internal/config"
	"api/internal/routes"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	cfg, err := config.LoadEnv()
	if err != nil {
		fatal("load environment", "err", err)
	}

	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		fatal("connect db", "err", err)
	}
	defer pool.Close()

	slog.Info("api listening", "port", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, routes.New(pool)); err != nil {
		fatal("server stopped", "err", err)
	}
}

func fatal(msg string, args ...any) {
	slog.Error(msg, args...)
	os.Exit(1)
}
