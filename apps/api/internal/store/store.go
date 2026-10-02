package store

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrOrderNotFound        = errors.New("order not found")
	ErrOrderStatusConflict  = errors.New("order status changed")
	ErrIdempotencyKeyReused = errors.New("idempotency key was used with another request")
)

type Store struct {
	pool *pgxpool.Pool
}

type MenuItem struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	PricePaise int    `json:"pricePaise"`
	Stock      int    `json:"stock"`
}

type OrderItem struct {
	MenuItemID     int64  `json:"menuItemId"`
	MenuItemName   string `json:"menuItemName"`
	Quantity       int    `json:"quantity"`
	UnitPricePaise int    `json:"unitPricePaise"`
}

type Order struct {
	ID        int64       `json:"id"`
	Status    string      `json:"status"`
	Items     []OrderItem `json:"items"`
	CreatedAt time.Time   `json:"createdAt"`
	UpdatedAt time.Time   `json:"updatedAt"`
}

type RequestedItem struct {
	MenuItemID int64
	Quantity   int
}

type StockIssue struct {
	MenuItemID int64  `json:"menuItemId"`
	Name       string `json:"name"`
	Available  int    `json:"available"`
	Requested  int    `json:"requested"`
}

type OutOfStockError struct {
	Items []StockIssue
}

func (e *OutOfStockError) Error() string {
	return "one or more menu items are out of stock"
}

type queryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) GetMenu(ctx context.Context) ([]MenuItem, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, price_paise, stock
		FROM menu_items
		ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]MenuItem, 0)
	for rows.Next() {
		var item MenuItem
		if err := rows.Scan(&item.ID, &item.Name, &item.PricePaise, &item.Stock); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) GetOrder(ctx context.Context, id int64) (Order, error) {
	return getOrder(ctx, s.pool, id)
}

func (s *Store) ListOrders(ctx context.Context) ([]Order, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id
		FROM orders
		ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := make([]Order, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		order, err := getOrder(ctx, s.pool, id)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, rows.Err()
}

func (s *Store) CreateOrder(ctx context.Context, idempotencyKey, requestHash string, requestedItems []RequestedItem) (Order, []MenuItem, bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Order{}, nil, false, err
	}
	defer tx.Rollback(ctx)

	var existingID int64
	var existingHash string
	err = tx.QueryRow(ctx, `
		SELECT id, request_hash
		FROM orders
		WHERE idempotency_key = $1
		FOR UPDATE`, idempotencyKey).Scan(&existingID, &existingHash)
	if err == nil {
		if existingHash != requestHash {
			return Order{}, nil, false, ErrIdempotencyKeyReused
		}
		order, err := getOrder(ctx, tx, existingID)
		if err != nil {
			return Order{}, nil, false, err
		}
		if err := tx.Commit(ctx); err != nil {
			return Order{}, nil, false, err
		}
		return order, nil, false, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Order{}, nil, false, err
	}

	var orderID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO orders (idempotency_key, request_hash)
		VALUES ($1, $2)
		ON CONFLICT (idempotency_key) DO NOTHING
		RETURNING id`, idempotencyKey, requestHash).Scan(&orderID)
	if errors.Is(err, pgx.ErrNoRows) {
		if err := tx.QueryRow(ctx, `
			SELECT id, request_hash
			FROM orders
			WHERE idempotency_key = $1`, idempotencyKey).Scan(&existingID, &existingHash); err != nil {
			return Order{}, nil, false, err
		}
		if existingHash != requestHash {
			return Order{}, nil, false, ErrIdempotencyKeyReused
		}
		order, err := getOrder(ctx, tx, existingID)
		if err != nil {
			return Order{}, nil, false, err
		}
		if err := tx.Commit(ctx); err != nil {
			return Order{}, nil, false, err
		}
		return order, nil, false, nil
	}
	if err != nil {
		return Order{}, nil, false, err
	}

	menuItems, issues, err := lockedMenuItems(ctx, tx, requestedItems)
	if err != nil {
		return Order{}, nil, false, err
	}
	if len(issues) > 0 {
		return Order{}, nil, false, &OutOfStockError{Items: issues}
	}

	updatedMenuItems := make([]MenuItem, 0, len(requestedItems))
	for _, requested := range requestedItems {
		item := menuItems[requested.MenuItemID]
		result, err := tx.Exec(ctx, `
			UPDATE menu_items
			SET stock = stock - $1, updated_at = NOW()
			WHERE id = $2 AND stock >= $1`, requested.Quantity, requested.MenuItemID)
		if err != nil {
			return Order{}, nil, false, err
		}
		if result.RowsAffected() == 0 {
			return Order{}, nil, false, &OutOfStockError{Items: []StockIssue{{
				MenuItemID: item.ID,
				Name:       item.Name,
				Available:  item.Stock,
				Requested:  requested.Quantity,
			}}}
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO order_items (order_id, menu_item_id, quantity, unit_price_paise)
			VALUES ($1, $2, $3, $4)`, orderID, item.ID, requested.Quantity, item.PricePaise); err != nil {
			return Order{}, nil, false, err
		}
		item.Stock -= requested.Quantity
		updatedMenuItems = append(updatedMenuItems, item)
	}

	order, err := getOrder(ctx, tx, orderID)
	if err != nil {
		return Order{}, nil, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Order{}, nil, false, err
	}
	return order, updatedMenuItems, true, nil
}

func (s *Store) UpdateOrderStatus(ctx context.Context, id int64, expectedStatus, status string) (Order, error) {
	result, err := s.pool.Exec(ctx, `
		UPDATE orders
		SET status = $1, updated_at = NOW()
		WHERE id = $2 AND status = $3`, status, id, expectedStatus)
	if err != nil {
		return Order{}, err
	}
	if result.RowsAffected() == 0 {
		var exists bool
		if err := s.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM orders WHERE id = $1)`, id).Scan(&exists); err != nil {
			return Order{}, err
		}
		if !exists {
			return Order{}, ErrOrderNotFound
		}
		return Order{}, ErrOrderStatusConflict
	}
	return s.GetOrder(ctx, id)
}

func lockedMenuItems(ctx context.Context, tx pgx.Tx, requestedItems []RequestedItem) (map[int64]MenuItem, []StockIssue, error) {
	ids := make([]int64, len(requestedItems))
	for index, requested := range requestedItems {
		ids[index] = requested.MenuItemID
	}

	rows, err := tx.Query(ctx, `
		SELECT id, name, price_paise, stock
		FROM menu_items
		WHERE id = ANY($1)
		ORDER BY id
		FOR UPDATE`, ids)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	menuItems := make(map[int64]MenuItem, len(requestedItems))
	for rows.Next() {
		var item MenuItem
		if err := rows.Scan(&item.ID, &item.Name, &item.PricePaise, &item.Stock); err != nil {
			return nil, nil, err
		}
		menuItems[item.ID] = item
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	issues := make([]StockIssue, 0)
	for _, requested := range requestedItems {
		item, ok := menuItems[requested.MenuItemID]
		if !ok {
			issues = append(issues, StockIssue{MenuItemID: requested.MenuItemID, Available: 0, Requested: requested.Quantity})
			continue
		}
		if item.Stock < requested.Quantity {
			issues = append(issues, StockIssue{MenuItemID: item.ID, Name: item.Name, Available: item.Stock, Requested: requested.Quantity})
		}
	}
	return menuItems, issues, nil
}

func getOrder(ctx context.Context, q queryer, id int64) (Order, error) {
	var order Order
	if err := q.QueryRow(ctx, `
		SELECT id, status, created_at, updated_at
		FROM orders
		WHERE id = $1`, id).Scan(&order.ID, &order.Status, &order.CreatedAt, &order.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Order{}, ErrOrderNotFound
		}
		return Order{}, err
	}

	rows, err := q.Query(ctx, `
		SELECT oi.menu_item_id, mi.name, oi.quantity, oi.unit_price_paise
		FROM order_items oi
		JOIN menu_items mi ON mi.id = oi.menu_item_id
		WHERE oi.order_id = $1
		ORDER BY oi.menu_item_id`, id)
	if err != nil {
		return Order{}, err
	}
	defer rows.Close()

	order.Items = make([]OrderItem, 0)
	for rows.Next() {
		var item OrderItem
		if err := rows.Scan(&item.MenuItemID, &item.MenuItemName, &item.Quantity, &item.UnitPricePaise); err != nil {
			return Order{}, err
		}
		order.Items = append(order.Items, item)
	}
	if err := rows.Err(); err != nil {
		return Order{}, err
	}
	return order, nil
}

func NormalizeRequestedItems(items []RequestedItem) ([]RequestedItem, error) {
	byMenuItemID := make(map[int64]int, len(items))
	for _, item := range items {
		if item.MenuItemID <= 0 || item.Quantity <= 0 {
			return nil, fmt.Errorf("menu item id and quantity must be positive")
		}
		byMenuItemID[item.MenuItemID] += item.Quantity
	}

	normalized := make([]RequestedItem, 0, len(byMenuItemID))
	for menuItemID, quantity := range byMenuItemID {
		normalized = append(normalized, RequestedItem{MenuItemID: menuItemID, Quantity: quantity})
	}
	sort.Slice(normalized, func(i, j int) bool {
		return normalized[i].MenuItemID < normalized[j].MenuItemID
	})
	return normalized, nil
}
