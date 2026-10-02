package main

import (
	"errors"
	"log/slog"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"api/internal/config"
	"api/internal/migrations"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	cfg, err := config.LoadEnv()
	if err != nil {
		fatal("load environment", "err", err)
	}

	if len(os.Args) != 2 || (os.Args[1] != "up" && os.Args[1] != "down") {
		slog.Error("usage: migrate <up|down>")
		os.Exit(2)
	}

	source, err := iofs.New(migrations.Files, ".")
	if err != nil {
		fatal("load migrations", "err", err)
	}

	migration, err := migrate.NewWithSourceInstance("iofs", source, cfg.DatabaseURL)
	if err != nil {
		fatal("initialize migrations", "err", err)
	}
	defer migration.Close()

	if os.Args[1] == "up" {
		err = migration.Up()
	} else {
		err = migration.Down()
	}
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		fatal("run migrations", "direction", os.Args[1], "err", err)
	}

	if errors.Is(err, migrate.ErrNoChange) {
		slog.Info("database schema already current", "direction", os.Args[1])
		return
	}
	slog.Info("database migrations complete", "direction", os.Args[1])
}

func fatal(msg string, args ...any) {
	slog.Error(msg, args...)
	os.Exit(1)
}
