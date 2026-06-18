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

# forge + spark are private modules — allow Go to fetch them via git:
go env -w 'GOPRIVATE=github.com/martin3zra/*'

gonew github.com/martin3zra/spark@latest github.com/you/yourapp
cd yourapp
```

## Requirements

- Go 1.23+
- Node 20+ / npm
- PostgreSQL 14+
- `GOPRIVATE=github.com/martin3zra/*` and git credentials (forge is private)

## Setup

```sh
cp .env.example .env

# generate a 64-byte base64 APP_KEY and paste it into .env
go run ./cmd/cli generate:key

# create the database, then apply the base schema
psql "postgres://USER:PASS@localhost:5432/spark" -f db/migrations/0001_init.sql

npm install
```

Seed a first user (bcrypt the password however you like; the framework uses
`golang.org/x/crypto/bcrypt`):

```sql
INSERT INTO users (name, email, password, role, email_verified_at)
VALUES ('You', 'you@example.com', '<bcrypt-hash>', 'admin', NOW());
```

## Run — development (HMR)

Two processes:

```sh
npm run dev      # vite dev server (writes public/hot, hot-reloads React + CSS)
go run .         # web server on APP_PORT
```

Open `http://localhost:<APP_PORT>/login`.

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
main.go · cmd/cli/main.go    web + CLI entrypoints
build.sh                     production build
app/
  server.go                  Boot / Start / Shutdown lifecycle
  config.go · driver.go      env config + Postgres connection
  auth-user.go               AuthUser identity + forge/auth resolver seams
  middleware.go              auth / verified / shared-props / signed
  acl.go                     role → permission map + Can()
  auth-handlers.go           login / logout / create-password
  home-handlers.go           dashboard
  routes.go                  /health /login /home /logout /passwords
  forms.go                   request validation structs
  locales/ + locales.go      i18n (app-owned locale files)
db/migrations/               users, sessions, preview_cache
resources/
  css/app.css                Tailwind v4 + shadcn theme tokens
  js/
    app.jsx · ssr.jsx        Inertia client + SSR bootstrap
    lib/utils.js             cn() helper
    components/ui/           shadcn primitives (button, input, label, card)
    Pages/                   Auth/Login, Dashboard
  views/                     root.html, error, mail templates
components.json              shadcn config (jsx, @ aliases)
vite.config.js · ssr.config.js
```

## Styling — Tailwind v4 + shadcn/ui

- Tailwind v4 via `@tailwindcss/vite`; theme tokens (CSS variables, light/dark)
  live in `resources/css/app.css`.
- shadcn components are plain `.jsx` under `resources/js/components/ui`. The
  `@` alias maps to `resources/js` (see `vite.config.js` and `jsconfig.json`).
- Add more primitives with the shadcn CLI:

  ```sh
  npx shadcn@latest add dialog dropdown-menu table
  ```

  (`components.json` is preconfigured: `tsx: false`, `@/components/ui`,
  `@/lib/utils`, lucide icons.)

## Adding a feature

1. Handler: `app/<domain>-handlers.go` (a method on `*Server`).
2. Route: register it in `app/routes.go`; guard with `.Can("action:resource")`.
3. Permissions: add the ability to a role in `app/acl.go`.
4. Page: `resources/js/Pages/<Name>.jsx`; render it with `ctx.Render("<Name>", props)`.
5. Migration: add a file under `db/migrations/`.

> Open routes (no auth) must be registered in their own `Group` — forge
> accumulates router middleware in place, so a bare global route inherits the
> auth middleware added later (see `/health` in `routes.go`).

## Notes

- `created_at` / `last_activity` on `sessions` are `BIGINT` (Unix-millis), as the
  forge session store expects.
- `AuthenticatedMiddleware` seeds `routing.PermissionKey{}` from the user's role
  so route `.Can(...)` guards resolve.
