# inMinutes Assignment

pnpm monorepo with a Go API, a Next.js web app, and Postgres, all runnable via Docker Compose.

## Structure

```
apps/
  api/   Go HTTP API (pgx, Postgres)
  web/   Next.js app (App Router, TypeScript, Tailwind)
docs/    Project documentation
docker-compose.yml
```

## Prerequisites

- Node 26+ and pnpm 12+
- Go 1.27+
- Docker with Compose

## Quick start (everything in Docker)

```sh
cp .env.example .env   # optional, defaults work without it
pnpm up                # docker compose up --build
```

- Web: http://localhost:3000
- API: http://localhost:8080/health
- Postgres: localhost:5432 (user/pass/db: `app`)

## Local development

Run Postgres in Docker and the apps on the host:

```sh
pnpm install
pnpm db:up
export DATABASE_URL=postgres://app:app@localhost:5432/app?sslmode=disable
pnpm --filter api migrate up
pnpm --filter api seed
pnpm dev               # runs api and web in parallel
```

## Scripts

| Command        | What it does                          |
| -------------- | ------------------------------------- |
| `pnpm dev`     | Run all apps in dev mode              |
| `pnpm dev:web` | Run only the Next.js app              |
| `pnpm dev:api` | Run only the Go API                   |
| `pnpm build`   | Build all apps                        |
| `pnpm lint`    | ESLint (web) and `go vet` (api)       |
| `pnpm db:up`   | Start only Postgres                   |
| `pnpm up`      | Build and start the full stack        |
| `pnpm down`    | Stop the stack                        |

The API build creates `apps/api/bin/migrate` and `apps/api/bin/seed`. With `DATABASE_URL` set, run `./bin/migrate up` to apply schema changes, `./bin/migrate down` to revert them, and `./bin/seed` to insert the initial menu. The seed command exits successfully with a `seed data already exists` log if menu data is present.

See [docs/](docs/README.md) for architecture and API details.

## Ordering and live updates

Orders use a client-supplied `Idempotency-Key`, so a retry returns the original order without reducing stock twice. The API locks every requested menu row in one transaction, checks availability, and deducts stock with a conditional update; Postgres also rejects negative stock. After a successful commit, the SSE endpoint broadcasts menu stock and order events to customer and kitchen screens. See [the API reference](docs/api.md) for request formats and event names.
