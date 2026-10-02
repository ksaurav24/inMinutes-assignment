package routes

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"api/internal/events"
	"api/internal/handlers"
	"api/internal/httpx"
	"api/internal/store"
)

func New(pool *pgxpool.Pool) http.Handler {
	handler := handlers.New(store.New(pool), events.NewHub())
	mux := http.NewServeMux()
	mux.HandleFunc("/", notFound)
	mux.HandleFunc("GET /health", health(pool))
	mux.HandleFunc("GET /menu", handler.GetMenu)
	mux.HandleFunc("POST /orders", handler.CreateOrder)
	mux.HandleFunc("GET /orders/{orderId}", handler.GetOrder)
	mux.HandleFunc("GET /kitchen/orders", handler.ListKitchenOrders)
	mux.HandleFunc("PATCH /orders/{orderId}", handler.UpdateOrderStatus)
	mux.HandleFunc("GET /events", handler.Events)

	return httpx.LogRequests(withCORS(mux))
}

func notFound(w http.ResponseWriter, r *http.Request) {
	httpx.Error(w, r, httpx.NotFound("route not found"))
}

func health(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := pool.Ping(ctx); err != nil {
			slog.WarnContext(ctx, "db ping failed", "err", err)
			httpx.Error(w, r, httpx.NewError(http.StatusServiceUnavailable, "db_unavailable", "database unavailable"))
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

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
