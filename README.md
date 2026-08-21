# spark

A starter application for the [`forge`](https://github.com/martin3zra/forge)
framework — **Go + Inertia.js + React (SSR) + Tailwind v4 + shadcn/ui**.

Auth-only baseline: users, sessions, authentication, a role-based ACL, and a
styled login + dashboard. No business domains — add your own.

This is the `laravel/laravel` role for forge: scaffold it to start a new project.

---

## Scaffold a new project

Uses [`gonew`](https://pkg.go.dev/golang.org/x/tools/cmd/gonew) to copy the
template and rewrite the module path:

```sh
go install golang.org/x/tools/cmd/gonew@latest

gonew github.com/martin3zra/spark@latest github.com/you/yourapp
cd yourapp
```

spark, forge, and playsql are all public — no `GOPRIVATE` setup needed for
this step. (You'll need it later only if you install [Kiosk](#optional-kiosk-ops-console),
which is private.)

## Requirements

- Go 1.25+
- Node 20+ / npm
- PostgreSQL 14+, MySQL 8+, or SQLite — pick one via `DB_DRIVER`
- [`camel`](https://github.com/martin3zra/camel) for migrations
- forge and playsql are public modules, no extra setup needed to fetch them.
  If you install Kiosk (below), it's a private repo: `go env -w
  GOPRIVATE=github.com/martin3zra/kiosk` and git credentials for it.

## Setup

```sh
cp .env.example .env

# generate a 64-byte base64 APP_KEY and paste it into .env
go run ./cmd/cli generate:key

# set DB_DRIVER (postgres, mysql, or sqlite) and either DB_SOURCE (a full DSN)
# or the discrete DB_HOST/DB_PORT/... fields in .env, then apply the schema
camel migrate

npm install
```

The app shell defaults to a sidebar layout; set `VITE_APP_LAYOUT=topnav` in
`.env` for the top-nav variant instead (build-time only — rebuild or restart
`npm run dev` after changing it).

Query building goes through [`playsql`](https://github.com/martin3zra/playsql),
so the same code runs unmodified against any of the three drivers — switching
is just changing `DB_DRIVER`/`DB_SOURCE` and re-running `camel migrate` against
the new target. Migrations live in `database/*.yaml` (schema, portable across
drivers via camel's DSL) and `database/*.sql` (raw SQL, e.g. the seed data).

### Postgres in a Docker container

If you don't have Postgres locally, run one in a container:

```sh
docker run -d --name spark-pg \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=spark \
  -p 5432:5432 \
  postgres:16
```

Matching `.env`:

```
DB_DRIVER=postgres
DB_HOST=localhost
DB_PORT=5432
DB_NAME=spark
DB_USERNAME=postgres
DB_PASSWORD=postgres
DB_SSLMODE=disable
```

> If port 5432 is already taken, map another (e.g. `-p 5433:5432`) and set
> `DB_PORT=5433`.

### MySQL in a Docker container

```sh
docker run -d --name spark-mysql \
  -e MYSQL_ROOT_PASSWORD=secret \
  -e MYSQL_DATABASE=spark \
  -p 3306:3306 \
  mysql:8
```

Matching `.env`:

```
DB_DRIVER=mysql
DB_SOURCE=root:secret@tcp(localhost:3306)/spark?parseTime=true
```

### SQLite

No container needed — just a file:

```
DB_DRIVER=sqlite
DB_SOURCE=spark.sqlite
```

### Seeding a first user

`database/*_seed_admin_user.sql` seeds one (bcrypt the password however you
like; the framework uses `golang.org/x/crypto/bcrypt`). To add another:

```sql
INSERT INTO users (uuid, name, email, password, role, email_verified_at)
VALUES ('<uuid>', 'You', 'you@example.com', '<bcrypt-hash>', 'admin', CURRENT_TIMESTAMP);
```

## Run — development (live reload)

Frontend hot-reloads with vite; the Go server live-reloads with
[`air`](https://github.com/air-verse/air).

```sh
go install github.com/air-verse/air@latest   # once
```

One command runs both (vite + air), Ctrl-C stops both:

```sh
./dev.sh
```

Or run them in separate terminals:

```sh
npm run dev      # terminal 1 — vite (writes public/hot, hot-reloads React + CSS)
air              # terminal 2 — rebuilds + restarts the Go server on .go changes
```

Open `http://localhost:<APP_PORT>/login`. Editing a `.go` file rebuilds and
restarts automatically; editing a `.tsx`/`.css` file hot-reloads in the browser.

> `air` is configured in `.air.toml` (dev build, no `prod` tag — assets come
> from the vite dev server). It builds to `tmp/` (git-ignored).

## Run — production

```sh
./build.sh       # npm run build + go build -tags prod → bin/spark, bin/cli
./bin/spark
```

`build.sh` builds the client bundle (`public/build`) and the SSR bundle
(`bootstrap/ssr`), then compiles the binaries with the `prod` tag (assets are
embedded).

---

## Project structure

```
main.go                       web entrypoint
cmd/cli/main.go               CLI entrypoint (version, generate:key, kiosk:*)
cmd/cli/kiosk.go              kiosk:install / kiosk:update / kiosk:uninstall
build.sh                     production build
app/
  server.go                  Boot / Start / Shutdown lifecycle
  config.go · driver.go      env config + playsql connection (postgres/mysql/sqlite)
  auth-user.go               AuthUser identity + forge/auth resolver seams
  middleware.go              auth / verified / shared-props / signed
  acl.go                     role → permission map + Can()
  auth-handlers.go           login / logout / create-password
  home-handlers.go           dashboard
  routes.go                  / (welcome) /health /login /home /logout /passwords
  forms.go                   request validation structs
  locales/ + locales.go      i18n (app-owned locale files)
database/                     camel migrations: users, sessions, preview_cache
resources/
  css/app.css                Tailwind v4 + shadcn theme tokens
  js/
    app.tsx · ssr.tsx        Inertia client + SSR bootstrap
    nav-items.ts             sidebar/topnav shared nav item list
    lib/                     cn() helper, useCsrfHeaders()
    hooks/                   use-mobile, use-initials
    components/ui/           shadcn primitives (sidebar, sheet, dropdown, …)
    components/              app shell: app-sidebar, app-topnav, nav-main,
                             nav-user, breadcrumbs, app-shell/content/header
    layouts/                 app-layout (picks sidebar or topnav via
                             VITE_APP_LAYOUT) · auth-layout
    layouts/app/             app-sidebar-layout.tsx · app-topnav-layout.tsx
    Pages/                   Welcome, Auth/Login, Dashboard
  views/                     root.html, error, mail templates
components.json              shadcn config (tsx, @ aliases)
vite.config.js · ssr.config.js
```

## Styling — Tailwind v4 + shadcn/ui

- Tailwind v4 via `@tailwindcss/vite`; theme tokens (CSS variables, light/dark)
  live in `resources/css/app.css`.
- shadcn components are `.tsx` under `resources/js/components/ui`. The `@`
  alias maps to `resources/js` (see `vite.config.js` and `tsconfig.json`).
- Add more primitives with the shadcn CLI:

  ```sh
  npx shadcn@latest add dialog dropdown-menu table
  ```

  (`components.json` is preconfigured: `tsx: true`, `@/components/ui`,
  `@/lib/utils`, lucide icons.)

## Adding a feature

1. Handler: `app/<domain>-handlers.go` (a method on `*Server`).
2. Route: register it in `app/routes.go`; guard with `.Can("action:resource")`.
3. Permissions: add the ability to a role in `app/acl.go`.
4. Page: `resources/js/Pages/<Name>.tsx`; render it with `ctx.Render("<Name>", props)`.
5. Migration: `camel make <name> --format yaml`, then `camel migrate`.

> Open routes (no auth) must be registered in their own `Group` — forge
> accumulates router middleware in place, so a bare global route inherits the
> auth middleware added later (see `/health` in `routes.go`).

## Optional: Kiosk ops console

spark ships the *tooling* to install [Kiosk](https://github.com/martin3zra/kiosk)
— company/plan/addon management, user administration, and impersonation — but
doesn't carry Kiosk itself. A fresh clone has nothing wired in: no route, no
config, no migrations. Installing it is a per-project choice, not a default.

```sh
./cli kiosk:install      # copies Kiosk's migrations + pages into this project,
                          # then prints the routes.go/.env lines to add by hand
./cli kiosk:update        # bump the dependency, re-run install to pick up changes
./cli kiosk:uninstall     # remove what kiosk:install copied in
```

`kiosk:install`/`kiosk:uninstall` only ever touch files Kiosk itself copied in
(tracked in `.kiosk-lock`) — never `routes.go`, never `.env`. See the
[kiosk README](https://github.com/martin3zra/kiosk) for the full endpoint
reference, the `Options` you pass to `kiosk.Register`, and the data model.

## Notes

- `created_at` / `last_activity` on `sessions` are `BIGINT` (Unix-millis), as the
  forge session store expects.
- `AuthenticatedMiddleware` seeds `routing.PermissionKey{}` from the user's role
  so route `.Can(...)` guards resolve.
