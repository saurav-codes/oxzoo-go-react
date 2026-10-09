# oxzoo-go-react

Deployed with [ox](https://deploywithox.com): deploy a repo to your own server with one command, no Docker. [Docs](https://deploywithox.com/docs) · [Stack guides](https://deploywithox.com/docs/guides)

An [ox](https://deploywithox.com) deploy example: a Go API built with the standard library, with a React 18 SPA built by Vite 5, deployed to your own Ubuntu server. ox compiles the Go binary, builds the SPA, runs the migrations, starts the binary under systemd, and Caddy serves `dist/` while sending only `/api` and `/health` to the Go process.

## Stack

| Layer | Tool | Role |
|---|---|---|
| Frontend | React 18 + Vite 5 | SPA built to `dist/` |
| API | Go stdlib `net/http` (Go 1.25 from `go.mod`) | `/api/greeting`, `/api/stats`, `/api/visits`, `/health` |
| Database | pgx v5, go-redis v9 | |
| Services | PostgreSQL 18, Redis 8 | provided by ox from `[services]` |

## ox.toml

```toml
# Go API with postgres + redis, a migration command, and a React SPA.

[app]
start  = "./server -port $PORT"
health = "/health"

[static]
dir = "dist"
spa = true
api = ["/api", "/health"]

[build]
commands = ["go build -o server ./cmd/server", "npm run build"]
migrate  = "go run ./cmd/migrate"

[services]
postgres = {}
redis    = {}

[tools]
node = "24"
```

The repo has two languages, so `[build] commands` names both builds. `[build] migrate` runs before traffic switches, and ox snapshots the database first.

## Services

- **postgres:** `DATABASE_URL`. Every `GET /api/greeting` inserts one row into `greeting_log`; `GET /api/stats` returns its row count. `cmd/migrate` applies the embedded `migrations/*.sql` files once each, tracked in a `schema_migrations` table.
- **redis:** `REDIS_URL`. `GET /api/visits` increments `oxzoo:visits` and sets a one-hour TTL on the first hit.

## Environment flow

- **Run time (API):** `cmd/server/main.go` reads `GREETING_TAG` on every request to `/api/greeting`.
- **Build time (SPA):** `vite.config.js` sets `envPrefix: ["GREETING_", "VITE_"]`, so `client/src/App.jsx` gets `import.meta.env.GREETING_TAG`, baked into the bundle by `npm run build`. ox sets your variables before the build, and changing one with `ox vars set` redeploys, which rebuilds the SPA.

## Deploy with ox

```sh
curl -fsSL https://deploywithox.com/install.sh | sh
ox login
ox new https://github.com/saurav-codes/oxzoo-go-react
printf 'GREETING_TAG=demo\n' | ox review oxzoo-go-react --from-file - --wait
```

The plan, offline:

```console
$ ox check .
ox check . (manifest: ox.toml)

  app.start                  ./server -port $PORT                                 declared
  app.health                 /health                                              declared
  static.dir                 dist                                                 declared
  static.spa                 true                                                 declared
  static.api                 /api, /health                                        declared
  build.install              npm ci                                               detected:package-lock.json
  build.commands[0]          go build -o server ./cmd/server                      declared
  build.commands[1]          npm run build                                        declared
  build.migrate              go run ./cmd/migrate                                 declared
  tools.go                   1.25.0                                               detected:go.mod
  tools.node                 24                                                   declared
  services.postgres          postgres 18 (shared)                                 default
  services.redis             redis 8 (only for this project)                      default

  Provided by ox: PORT, HOST, OX_ENV, OX_PROJECT, OX_RELEASE, OX_DATA_DIR, PUBLIC_URL, PUBLIC_HOST, DATABASE_URL, REDIS_URL
  Set on the dashboard before the first deploy: GREETING_TAG

Ready to deploy.
```

## Expected output

```
oxzoo-go-react
frontend: hello world oxzoo-go-react_<GREETING_TAG>
backend: hello world oxzoo-go-react_<GREETING_TAG>
```

The `frontend:` line is baked into the SPA; the `backend:` line comes from `GET /api/greeting`.

## Local development

```sh
npm install && GREETING_TAG=dev npm run build
go run ./cmd/migrate                    # needs a local PostgreSQL, or export DATABASE_URL
GREETING_TAG=dev go run ./cmd/server -port 9113
```
