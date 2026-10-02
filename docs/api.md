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
