# API

Base URL: `http://localhost:8080`

CORS is open (`*`) for local development.

## `GET /health`

Checks that the API is up and can reach Postgres.

- `200 {"status":"ok"}`
- `503 {"status":"db unavailable"}`
