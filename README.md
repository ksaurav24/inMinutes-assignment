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
| `pnpm --filter api test` | Run Go tests (set `DATABASE_URL` for the Postgres integration test) |

The API build creates `apps/api/bin/migrate` and `apps/api/bin/seed`. With `DATABASE_URL` set, run `./bin/migrate up` to apply schema changes, `./bin/migrate down` to revert them, and `./bin/seed` to insert the initial menu. The seed command exits successfully with a `seed data already exists` log if menu data is present.

See [docs/](docs/README.md) for architecture and API details.

## Ordering and live updates

Orders use a client-supplied `Idempotency-Key`, so a retry returns the original order without reducing stock twice. The API locks requested menu rows in one transaction, checks availability, and deducts stock with a conditional update; Postgres also rejects negative stock. After commit, SSE events tell screens to refetch current state. They also refetch when the stream connects or reconnects, covering events missed during a disconnect or initial load. Kitchen stage updates succeed only when the order is still at the expected previous stage. See [the API reference](docs/api.md) for request formats and event names.

The Postgres integration test starts two orders for the last unit at the same time and asserts one success, one stock conflict, one stored order, and zero remaining stock. It also checks that a stale kitchen update is rejected. It creates and removes its own temporary schema, leaving the app's menu and orders untouched:

```sh
DATABASE_URL='postgres://app:app@localhost:5432/app?sslmode=disable' pnpm --filter api test
```

## Packages and choices

- Go `net/http` serves the API; `pgx` connects to Postgres and runs the locking transactions; `golang-migrate` applies schema changes; `godotenv` supports local development.
- Next.js and React provide the two screens; TanStack Query caches API data and refetches after SSE events; Tailwind CSS and shadcn/Radix components style the UI; Lucide supplies icons and Framer Motion supplies transitions.
- Postgres persists stock and orders. SSE handles one-way server updates while normal HTTP requests place orders and advance stages.

## Assumptions

- Payment is simulated; placing an order makes no charge.
- Stock is deducted when an order is placed and is not replenished when it is picked up.
- The kitchen follows `new` → `cooking` → `ready` → `picked_up` in that order.
- A customer tracks the latest order in their current browser tab. There are no customer accounts.
- The deployment uses one API instance; the in-process SSE hub does not broadcast across separate API instances.

## Submission demo

[Watch the 10-second demo recording](docs/demo.mp4): two customers try the final unit; one succeeds, one sees the stock conflict, and the kitchen moves the order through all stages. Include this video link with the repository link in the submission email.
