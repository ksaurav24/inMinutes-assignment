# Architecture

## Services

| Service   | Path       | Port | Notes                                                   |
| --------- | ---------- | ---- | ------------------------------------------------------- |
| `db`      | -          | 5432 | Postgres 18, data in the `db-data` volume               |
| `migrate` | `apps/api` | -    | Applies embedded Go migrations, then exits              |
| `seed`    | `apps/api` | -    | Adds initial menu data once, then exits                  |
| `api`     | `apps/api` | 8080 | Go `net/http` server, connects via `pgxpool`            |
| `web`     | `apps/web` | 3000 | Next.js (standalone output), calls the API              |

`migrate` waits for `db` to pass its healthcheck. `seed` waits for a successful migration, and `api` waits for both one-off jobs. `web` depends on `api`.

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
| `DATABASE_URL`        | api, migrate, seed | built from the Postgres vars in Compose; required on host   |
| `PORT`                | api          | `8080`                                                      |
| `NEXT_PUBLIC_API_URL` | web          | `http://localhost:8080` (inlined at build time)             |

## Docker builds

- `apps/api/Dockerfile`: multi-stage, static `api`, `migrate`, and `seed` binaries on Alpine. Build context `apps/api`. Migrations are embedded in the migration binary, so it can run from any working directory.
- `apps/web/Dockerfile`: build context is the repo root so it can use `pnpm-lock.yaml`. Runs the Next.js standalone server.

The Go binaries load `.env` from their working directory or the repository root when it exists. Container environment variables take precedence.
