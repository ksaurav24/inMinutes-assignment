package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"api/internal/config"
)

type menuItem struct {
	name       string
	pricePaise int
	stock      int
}

var menuItems = []menuItem{
	{name: "Classic Burger", pricePaise: 24900, stock: 12},
	{name: "Crispy Chicken Burger", pricePaise: 28900, stock: 8},
	{name: "Margherita Pizza", pricePaise: 32900, stock: 6},
	{name: "Loaded Fries", pricePaise: 14900, stock: 10},
	{name: "Caesar Salad", pricePaise: 21900, stock: 5},
	{name: "Chocolate Shake", pricePaise: 12900, stock: 3},
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	cfg, err := config.LoadEnv()
	if err != nil {
		fatal("load environment", "err", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		fatal("connect db", "err", err)
	}
	defer pool.Close()

	tx, err := pool.Begin(ctx)
	if err != nil {
		fatal("begin seed transaction", "err", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, "LOCK TABLE menu_items IN SHARE ROW EXCLUSIVE MODE"); err != nil {
		fatal("lock menu items", "err", err)
	}

	var dataExists bool
	if err := tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM menu_items)").Scan(&dataExists); err != nil {
		fatal("check seed data", "err", err)
	}
	if dataExists {
		slog.Info("seed data already exists")
		return
	}

	rows := make([][]any, len(menuItems))
	for index, item := range menuItems {
		rows[index] = []any{item.name, item.pricePaise, item.stock}
	}
	if _, err := tx.CopyFrom(ctx, pgx.Identifier{"menu_items"}, []string{"name", "price_paise", "stock"}, pgx.CopyFromRows(rows)); err != nil {
		fatal("insert seed data", "err", err)
	}
	if err := tx.Commit(ctx); err != nil {
		fatal("commit seed transaction", "err", err)
	}

	slog.Info("seed data inserted", "menu_items", len(menuItems))
}

func fatal(msg string, args ...any) {
	slog.Error(msg, args...)
	os.Exit(1)
}
