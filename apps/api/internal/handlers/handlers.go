package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"api/internal/events"
	"api/internal/httpx"
	"api/internal/store"
)

type Handler struct {
	store *store.Store
	hub   *events.Hub
}

type createOrderRequest struct {
	Items []createOrderItem `json:"items"`
}

type createOrderItem struct {
	MenuItemID int64 `json:"menuItemId"`
	Quantity   int   `json:"quantity"`
}

type updateOrderRequest struct {
	Status string `json:"status"`
}

func New(dataStore *store.Store, hub *events.Hub) *Handler {
	return &Handler{store: dataStore, hub: hub}
}

func (h *Handler) GetMenu(w http.ResponseWriter, r *http.Request) {
	items, err := h.store.GetMenu(r.Context())
	if err != nil {
		serverError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey == "" || len(idempotencyKey) > 255 {
		httpx.Error(w, r, httpx.BadRequest("Idempotency-Key must be between 1 and 255 characters"))
		return
	}

	var request createOrderRequest
	if err := decodeJSON(w, r, &request); err != nil {
		httpx.Error(w, r, httpx.BadRequest(err.Error()))
		return
	}
	if len(request.Items) == 0 {
		httpx.Error(w, r, httpx.BadRequest("items is required"))
		return
	}

	requestedItems := make([]store.RequestedItem, len(request.Items))
	for index, item := range request.Items {
		requestedItems[index] = store.RequestedItem{MenuItemID: item.MenuItemID, Quantity: item.Quantity}
	}
	normalizedItems, err := store.NormalizeRequestedItems(requestedItems)
	if err != nil {
		httpx.Error(w, r, httpx.BadRequest(err.Error()))
		return
	}
	requestHash, err := hashRequest(normalizedItems)
	if err != nil {
		serverError(w, r, err)
		return
	}

	order, changedMenuItems, created, err := h.store.CreateOrder(r.Context(), idempotencyKey, requestHash, normalizedItems)
	if err != nil {
		var stockError *store.OutOfStockError
		switch {
		case errors.As(err, &stockError):
			httpx.Error(w, r, httpx.Conflict("out_of_stock", "one or more menu items are out of stock", map[string]any{"items": stockError.Items}))
		case errors.Is(err, store.ErrIdempotencyKeyReused):
			httpx.Error(w, r, httpx.Conflict("idempotency_key_reused", "Idempotency-Key was used with a different order", nil))
		default:
			serverError(w, r, err)
		}
		return
	}

	if created {
		for _, item := range changedMenuItems {
			h.hub.Publish(events.Event{Type: "menu.updated", Data: item})
		}
		h.hub.Publish(events.Event{Type: "order.created", Data: order})
		httpx.JSON(w, http.StatusCreated, order)
		return
	}
	httpx.JSON(w, http.StatusOK, order)
}

func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	orderID, ok := orderID(w, r)
	if !ok {
		return
	}
	order, err := h.store.GetOrder(r.Context(), orderID)
	if err != nil {
		if errors.Is(err, store.ErrOrderNotFound) {
			httpx.Error(w, r, httpx.NotFound("order not found"))
			return
		}
		serverError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, order)
}

func (h *Handler) ListKitchenOrders(w http.ResponseWriter, r *http.Request) {
	orders, err := h.store.ListOrders(r.Context())
	if err != nil {
		serverError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"orders": orders})
}

func (h *Handler) UpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	orderID, ok := orderID(w, r)
	if !ok {
		return
	}

	var request updateOrderRequest
	if err := decodeJSON(w, r, &request); err != nil {
		httpx.Error(w, r, httpx.BadRequest(err.Error()))
		return
	}
	expectedStatus, ok := previousStatus(request.Status)
	if !ok {
		httpx.Error(w, r, httpx.BadRequest("status must be cooking, ready, or picked_up"))
		return
	}

	order, err := h.store.UpdateOrderStatus(r.Context(), orderID, expectedStatus, request.Status)
	if err != nil {
		if errors.Is(err, store.ErrOrderNotFound) {
			httpx.Error(w, r, httpx.NotFound("order not found"))
			return
		}
		if errors.Is(err, store.ErrOrderStatusConflict) {
			httpx.Error(w, r, httpx.Conflict("order_status_conflict", "order status changed; refresh the board", nil))
			return
		}
		serverError(w, r, err)
		return
	}
	h.hub.Publish(events.Event{Type: "order.updated", Data: order})
	httpx.JSON(w, http.StatusOK, order)
}

func (h *Handler) Events(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		httpx.Error(w, r, httpx.NewError(http.StatusInternalServerError, "streaming_unsupported", "streaming is unavailable"))
		return
	}
	eventChannel, unsubscribe := h.hub.Subscribe()
	defer unsubscribe()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	keepAlive := time.NewTicker(25 * time.Second)
	defer keepAlive.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case event, open := <-eventChannel:
			if !open {
				return
			}
			if err := writeEvent(w, event); err != nil {
				return
			}
			flusher.Flush()
		case <-keepAlive.C:
			if _, err := fmt.Fprint(w, ": keep-alive\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("request body must be valid JSON")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("request body must contain one JSON value")
	}
	return nil
}

func hashRequest(items []store.RequestedItem) (string, error) {
	payload, err := json.Marshal(items)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func orderID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("orderId"), 10, 64)
	if err != nil || id <= 0 {
		httpx.Error(w, r, httpx.BadRequest("orderId must be a positive integer"))
		return 0, false
	}
	return id, true
}

func previousStatus(status string) (string, bool) {
	switch status {
	case "cooking":
		return "new", true
	case "ready":
		return "cooking", true
	case "picked_up":
		return "ready", true
	default:
		return "", false
	}
}

func writeEvent(w http.ResponseWriter, event events.Event) error {
	payload, err := json.Marshal(event.Data)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, payload)
	return err
}

func serverError(w http.ResponseWriter, r *http.Request, err error) {
	httpx.Error(w, r, err)
}
