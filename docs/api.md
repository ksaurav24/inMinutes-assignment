# API

Base URL: `http://localhost:8080`

CORS is open (`*`) for local development.

## Response format

Every response is JSON in one of two envelopes.

Success:

```json
{ "data": { ... } }
```

Error:

```json
{ "error": { "code": "not_found", "message": "route not found" } }
```

`code` is a stable machine-readable string; `message` is human-readable. Unexpected server errors return `500` with code `internal_error` and no internal details. Unknown routes return `404` with code `not_found`.

Helpers live in `apps/api/internal/httpx` (`httpx.JSON`, `httpx.Error`).

## Logging

The API logs JSON lines to stdout via `log/slog`, including one `request` line per request with `method`, `path`, `status`, and `duration_ms`.

## `GET /health`

Checks that the API is up and can reach Postgres.

- `200 {"data":{"status":"ok"}}`
- `503 {"error":{"code":"db_unavailable","message":"database unavailable"}}`

## `GET /menu`

Returns the menu and current stock counts.

```json
{
  "data": {
    "items": [
      { "id": 1, "name": "Classic Burger", "pricePaise": 24900, "stock": 12 }
    ]
  }
}
```

## `POST /orders`

Places an order. Send an `Idempotency-Key` header with a unique client-generated value; retrying the same request with the same key returns the original order without deducting stock again.

```json
{
  "items": [
    { "menuItemId": 1, "quantity": 2 },
    { "menuItemId": 4, "quantity": 1 }
  ]
}
```

- `201` when a new order is created
- `200` when the request is an idempotent replay
- `409 out_of_stock` when one or more items cannot be fulfilled. `error.details.items` lists each unavailable item with `menuItemId`, `name`, `available`, and `requested`.
- `409 idempotency_key_reused` when the key is used with a different request body

An order response has this shape:

```json
{
  "data": {
    "id": 1,
    "status": "new",
    "items": [
      {
        "menuItemId": 1,
        "menuItemName": "Classic Burger",
        "quantity": 2,
        "unitPricePaise": 24900
      }
    ],
    "createdAt": "2026-10-02T00:00:00Z",
    "updatedAt": "2026-10-02T00:00:00Z"
  }
}
```

## `GET /orders/{orderId}`

Returns one order, or `404 not_found` when it does not exist.

## `GET /kitchen/orders`

Returns all orders, oldest first, for the kitchen board.

```json
{ "data": { "orders": [{ "id": 1, "status": "new", "items": [] }] } }
```

## `PATCH /orders/{orderId}`

Advances an order through `new` → `cooking` → `ready` → `picked_up`. The request must name the next stage. A stale kitchen screen cannot move an order backwards or skip a stage.

```json
{ "status": "cooking" }
```

Returns the updated order, `404 not_found` when it does not exist, or `409 order_status_conflict` when another screen already changed its stage.

## `GET /events`

An SSE stream for live updates. Events are emitted only after their database transaction commits.

- `menu.updated`: a menu item with its latest `stock`, emitted after a successful order
- `order.created`: a new order for the kitchen board
- `order.updated`: an order after a kitchen status change

Clients should load `/menu` or `/kitchen/orders`, keep this stream open, and refetch when it connects or reconnects. Refetching also closes the gap between the initial load and stream subscription.
