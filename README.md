# spark

A minimal starter application built on the [`forge`](https://github.com/martin3zra/forge)
framework — Go + Inertia.js + React (SSR). Auth-only baseline: users, sessions,
authentication, and a role-based ACL. No business domains.

This is the `laravel/laravel` role for forge: clone it to start a new project.

## Start a new project

With [`gonew`](https://pkg.go.dev/golang.org/x/tools/cmd/gonew):

```sh
go install golang.org/x/tools/cmd/gonew@latest
gonew github.com/martin3zra/spark github.com/you/yourapp
cd yourapp
```

`gonew` copies this template and rewrites the module path everywhere.

## What's inside

```
main.go · cmd/cli         web + CLI entrypoints
app/
  server.go               Boot / Start / Shutdown lifecycle
  config.go · driver.go   env config + Postgres connection
  auth-user.go            AuthUser identity + forge/auth resolver seams
  middleware.go           auth / verified / shared-props / signed
  acl.go                  role → permission map + Can()
  auth-handlers.go        login / logout / create-password
  routes.go               /health /login /home /logout /passwords
  forms.go                request validation structs
  locales/                i18n files (app-owned)
db/migrations/            users, sessions, preview_cache
resources/js/             Inertia + React (app.jsx, ssr.jsx, Pages/)
resources/views/          root.html, error, mail templates
```

## Setup

```sh
cp .env.example .env
go run ./cmd/cli generate:key      # prints an APP_KEY; paste into .env
psql "$DATABASE_URL" -f db/migrations/0001_init.sql
npm install
```

## Run (dev)

```sh
npm run dev        # vite dev server (writes public/hot)
go run .           # web server on APP_PORT
```

## Build (prod)

```sh
./build.sh         # npm run build + go build -tags prod (bin/spark, bin/cli)
```

## Notes

- forge is a **private** module; building needs
  `go env -w 'GOPRIVATE=github.com/martin3zra/*'` and git credentials.
- Add your domains as `app/<domain>-handlers.go` + routes in `routes.go`, and
  roles/permissions in `acl.go`.
