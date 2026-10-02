package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"api/internal/config"
	"api/internal/events"
	"api/internal/handlers"
	"api/internal/httpx"
	"api/internal/store"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := config.LoadDotEnv(); err != nil {
		fatal("load .env", "err", err)
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		fatal("DATABASE_URL is required")
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		fatal("connect db", "err", err)
	}
	defer pool.Close()

	hub := events.NewHub()
	handler := handlers.New(store.New(pool), hub)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		httpx.Error(w, r, httpx.NotFound("route not found"))
	})
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := pool.Ping(ctx); err != nil {
			slog.WarnContext(ctx, "db ping failed", "err", err)
			httpx.Error(w, r, httpx.NewError(http.StatusServiceUnavailable, "db_unavailable", "database unavailable"))
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /menu", handler.GetMenu)
	mux.HandleFunc("POST /orders", handler.CreateOrder)
	mux.HandleFunc("GET /orders/{orderId}", handler.GetOrder)
	mux.HandleFunc("GET /kitchen/orders", handler.ListKitchenOrders)
	mux.HandleFunc("PATCH /orders/{orderId}", handler.UpdateOrderStatus)
	mux.HandleFunc("GET /events", handler.Events)

	slog.Info("api listening", "port", port)
	if err := http.ListenAndServe(":"+port, httpx.LogRequests(withCORS(mux))); err != nil {
		fatal("server stopped", "err", err)
	}
}

func fatal(msg string, args ...any) {
	slog.Error(msg, args...)
	os.Exit(1)
}

// withCORS allows the web app (different origin in local dev) to call the API.
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Idempotency-Key")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
