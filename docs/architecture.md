# Architecture

## Services

| Service | Path       | Port | Notes                                              |
| ------- | ---------- | ---- | -------------------------------------------------- |
| `db`    | -          | 5432 | Postgres 18, data in the `db-data` volume          |
| `api`   | `apps/api` | 8080 | Go `net/http` server, connects via `pgxpool`       |
| `web`   | `apps/web` | 3000 | Next.js (standalone output), calls the API         |

`api` waits for `db` to pass its healthcheck before starting. `web` depends on `api`.

## Workspace

pnpm workspace over `apps/*`. The Go API has a `package.json` that wraps `go` commands so root scripts (`pnpm dev`, `pnpm build`, `pnpm lint`) cover both apps.

## Configuration

Copy `.env.example` to `.env`. Docker Compose reads it automatically; every value has a default.

| Variable              | Used by      | Default                                                     |
| --------------------- | ------------ | ----------------------------------------------------------- |
| `POSTGRES_USER`       | db, api      | `app`                                                       |
| `POSTGRES_PASSWORD`   | db, api      | `app`                                                       |
| `POSTGRES_DB`         | db, api      | `app`                                                       |
| `POSTGRES_PORT`       | db (host)    | `5432`                                                      |
| `API_PORT`            | api (host)   | `8080`                                                      |
| `WEB_PORT`            | web (host)   | `3000`                                                      |
| `DATABASE_URL`        | api          | built from the Postgres vars in Compose; required on host   |
| `PORT`                | api          | `8080`                                                      |
| `NEXT_PUBLIC_API_URL` | web          | `http://localhost:8080` (inlined at build time)             |

## Docker builds

- `apps/api/Dockerfile`: multi-stage, static binary on Alpine. Build context `apps/api`.
- `apps/web/Dockerfile`: build context is the repo root so it can use `pnpm-lock.yaml`. Runs the Next.js standalone server.
