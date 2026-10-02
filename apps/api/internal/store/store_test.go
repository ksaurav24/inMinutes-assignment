package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"

	"api/internal/migrations"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestConcurrentLastUnitAndStatusTransitions(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is required for the Postgres integration test")
	}

	ctx := context.Background()
	admin, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()

	randomID := make([]byte, 8)
	if _, err := rand.Read(randomID); err != nil {
		t.Fatal(err)
	}
	schema := "test_" + hex.EncodeToString(randomID)
	quotedSchema := pgx.Identifier{schema}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+quotedSchema); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := admin.Exec(ctx, "DROP SCHEMA "+quotedSchema+" CASCADE"); err != nil {
			t.Errorf("drop temporary test schema: %v", err)
		}
	}()

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	for _, name := range []string{"000001_initial_schema.up.sql", "000002_add_order_idempotency.up.sql"} {
		sql, err := migrations.Files.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, string(sql), pgx.QueryExecModeSimpleProtocol); err != nil {
			t.Fatalf("apply %s: %v", name, err)
		}
	}
	var menuItemID int64
	if err := pool.QueryRow(ctx, `INSERT INTO menu_items (name, price_paise, stock) VALUES ('Last Unit', 100, 1) RETURNING id`).Scan(&menuItemID); err != nil {
		t.Fatal(err)
	}

	dataStore := New(pool)
	type result struct {
		order Order
		err   error
	}
	results := make(chan result, 2)
	start := make(chan struct{})
	var workers sync.WaitGroup
	for i := 0; i < 2; i++ {
		workers.Add(1)
		go func(index int) {
			defer workers.Done()
			<-start
			order, _, _, err := dataStore.CreateOrder(ctx, fmt.Sprintf("buyer-%d", index), "same-cart", []RequestedItem{{MenuItemID: menuItemID, Quantity: 1}})
			results <- result{order: order, err: err}
		}(i)
	}
	close(start)
	workers.Wait()
	close(results)

	var winner Order
	var successful, conflicts int
	for result := range results {
		if result.err == nil {
			successful++
			winner = result.order
			continue
		}
		var stockError *OutOfStockError
		if errors.As(result.err, &stockError) {
			conflicts++
			continue
		}
		t.Fatalf("unexpected order error: %v", result.err)
	}
	if successful != 1 || conflicts != 1 {
		t.Fatalf("got %d successful orders and %d stock conflicts, want one each", successful, conflicts)
	}
	menu, err := dataStore.GetMenu(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(menu) != 1 || menu[0].Stock != 0 {
		t.Fatalf("stock after concurrent checkout = %+v, want zero", menu)
	}
	orders, err := dataStore.ListOrders(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(orders) != 1 {
		t.Fatalf("stored orders = %d, want one", len(orders))
	}

	if _, err := dataStore.UpdateOrderStatus(ctx, winner.ID, "new", "cooking"); err != nil {
		t.Fatal(err)
	}
	if _, err := dataStore.UpdateOrderStatus(ctx, winner.ID, "new", "cooking"); !errors.Is(err, ErrOrderStatusConflict) {
		t.Fatalf("stale kitchen update = %v, want status conflict", err)
	}
	if _, err := dataStore.UpdateOrderStatus(ctx, winner.ID, "cooking", "ready"); err != nil {
		t.Fatal(err)
	}
}
