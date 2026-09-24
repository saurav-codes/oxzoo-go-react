# oxzoo-go-react

An official ox deploy example: a Go API built with the standard library only, fronted by a React 18 single-page app built with Vite 5, deployed to a single Ubuntu VPS by the [ox](https://github.com/saurav-codes/vps-ctl) control plane from one `ox.toml` manifest at the repo root. ox runs the install and build steps, starts the compiled `server` binary as a systemd process, and configures nginx to serve the built `dist/` folder statically while proxying only `/api` and `/health` to the Go process.

## Stack

| Layer | Tool | Role |
|---|---|---|
| Frontend | React 18 + Vite 5 | SPA built to `dist/`, served by nginx |
| API | Go stdlib `net/http` | `GET /api/greeting`, `/api/stats`, `/api/visits`, `GET /health`, binds `127.0.0.1:9113` |
| Package manager | npm | lockfile (`package-lock.json`) is committed |
| Deploy | ox | `ox.toml` defines processes, frontend, domain |
| Services | postgres@17, redis@7 | declared in `ox.toml` `[[services]]` |

## Services

Two ox catalog services back the API; their credentials arrive as env keys the deploy injects automatically:

- **postgres** — `DATABASE_URL` (driver: pgx v5). Every `GET /api/greeting` hit inserts one row into `greeting_log`; `GET /api/stats` returns its row count. The schema is applied by `cmd/migrate`, which runs the embedded `migrations/*.sql` files idempotently via a `schema_migrations` ledger (`migrate` hook: `go run ./cmd/migrate`; no golang-migrate CLI to download).
- **redis** — `REDIS_URL` (driver: go-redis v9). `GET /api/visits` atomically increments `oxzoo:visits` and sets a 1-hour TTL on the first hit, so the counter resets itself.

Both are read via `DATABASE_URL`/`REDIS_URL` only, with local-dev fallbacks in `cmd/server/main.go` and `cmd/migrate/main.go`. Never split them into `DATABASE_*`-style keys.

## Environment flow

One variable, two paths:

**`GREETING_TAG`**

- **Runtime path (API):** `cmd/server/main.go` reads `os.Getenv("GREETING_TAG")` on every request to `GET /api/greeting`. A restart with a new value is enough to change it.
- **Build-time path (SPA):** `vite.config.js` sets `envPrefix: ["GREETING_", "VITE_"]`, so any `GREETING_*` variable in the build environment is exposed to `import.meta.env`. `client/src/App.jsx` renders `import.meta.env.GREETING_TAG` inside one template literal, which is baked into the bundle during `npm run build`. No duplicated `VITE_GREETING_TAG` is needed.

**The deploy compiles Go first (`go build -o server ./cmd/server`), then installs and builds the SPA (`npm install`, `npm run build`).** nginx serves `dist/` from the current release and proxies `/api` and `/health` to the Go process on `127.0.0.1:9113`.

**Set `GREETING_TAG` in the ox Environment editor BEFORE the first deploy.** The SPA value is baked during the deploy build step, so changing it later requires a redeploy; the API value updates as soon as the process restarts. `.env.example` documents the variable with a placeholder; real values live in the ox dashboard, never in git.

## Deploy with ox

1. Add the repo in the ox dashboard: paste the clone URL `git@github.com:saurav-codes/oxzoo-go-react.git`.
2. In the Environment editor, set `GREETING_TAG` (for example `v1`).
3. Press **Deploy**. ox runs `go build -o server ./cmd/server` and `npm install`, then `npm run build`, applies the embedded migrations (`go run ./cmd/migrate`, with a pre-migrate database dump), starts `./server -port 9113`, and waits for `http://127.0.0.1:9113/health` to return `ok`.

## Expected output

Visiting the domain shows the project heading plus the two labeled lines:

```
oxzoo-go-react
frontend: hello world oxzoo-go-react_<GREETING_TAG>
backend: hello world oxzoo-go-react_<GREETING_TAG>
```

The `frontend:` line is baked into the SPA at build time; the `backend:` line is fetched live from `GET /api/greeting` at runtime. Both come from the same `GREETING_TAG` set in the ox dashboard.
