# spark

A starter application for the [`forge`](https://github.com/martin3zra/forge)
framework — **Go + Inertia.js + React (SSR) + Tailwind v4 + shadcn/ui**.

Auth-only baseline: users, sessions, authentication, a role-based ACL, and a
styled login + dashboard. No business domains — add your own.

This is the `laravel/laravel` role for forge: scaffold it to start a new project.

---

## Scaffold a new project

[`spark-installer`](https://github.com/martin3zra/spark-installer) mirrors
`laravel new` — one command, working app:

```sh
go install github.com/martin3zra/spark-installer/cmd/spark@latest

spark new github.com/you/yourapp
cd yourapp
./dev.sh
```

That's `git init` + an initial commit, `.env` with a real generated
`APP_KEY`, `npm install`, and (for the default sqlite driver, since it needs
no server) the first migration — all done, not left for you to do by hand.
`spark new --help` lists the flags (`--db=postgres|mysql`, `--no-git`,
`--no-install`, `--no-migrate`).

Under the hood it's [`gonew`](https://pkg.go.dev/golang.org/x/tools/cmd/gonew)
(Google's own module-template tool) doing the actual copy + module-path
rewrite, plus the setup above. If you'd rather do just the copy step
yourself: `gonew github.com/martin3zra/spark@latest github.com/you/yourapp`.

spark, forge, and playsql are all public — no `GOPRIVATE` setup needed for
this step (or for [Kiosk](#optional-kiosk-ops-console), also public).

## Requirements

- Go 1.25+
- Node 20+ / npm
- PostgreSQL 14+, MySQL 8+, or SQLite — pick one via `DB_DRIVER`
- [`camel`](https://github.com/martin3zra/camel) for migrations
- forge, playsql, and kiosk are all public modules — no `GOPRIVATE` setup
  needed to fetch any of them.

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

## CLI — adding your own commands

`go run ./cmd/cli list` (or no args, or `help`) prints every registered
command. Adding one is two steps, no changes to `main.go`:

1. Write a type implementing `PreBootCommand` (runs immediately — no Config,
   no DB; use this only for something that must work before APP_KEY exists,
   like `generate:key`) or `BootCommand` (runs after the server boots, with
   a `*CommandContext` carrying `Server`/`DB` — what you want for anything
   that touches the database or app config):

   ```go
   type reportCmd struct{}

   func (reportCmd) Name() string        { return "report:weekly" }
   func (reportCmd) Description() string { return "Email the weekly usage report" }

   func (reportCmd) Run(ctx *CommandContext, args []string) error {
       // ctx.DB, ctx.Server available here
       return nil
   }
   ```

2. Add it to the `commands` slice in `cmd/cli/commands.go`.

Return an error rather than calling `os.Exit` — `dispatch` in `main.go` is
the one place that prints it and sets the exit code, so every command fails
the same way. See `cmd/cli/kiosk.go`'s three commands for a complete
worked example (including one, `kiosk:update`, that shells out to `go get`
+ a fresh `go run` subprocess rather than doing everything in-process).

---

## Project structure

```
main.go                       web entrypoint
cmd/cli/main.go               CLI entrypoint — dispatches by name via the registry
cmd/cli/command.go            Command / PreBootCommand / BootCommand interfaces
cmd/cli/commands.go           the command registry, list/help, version, generate:key
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
  theme.go · theme_presets.go  theme customizer: per-user settings, presets, fonts
  theme-handlers.go          PUT /settings/theme, GET /settings/theme/catalog
  locales/ + locales.go      i18n (app-owned locale files)
database/                     camel migrations: users, sessions, preview_cache,
                              users.appearance + users.theme_settings
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

## Theme customizer

The palette button in the header opens a panel where each signed-in user
picks a theme preset, font, color mode (light / dark / system), radius,
content layout, scale, and sidebar variant and mode. Changes show at once
and are saved to the user's account (`users.appearance`,
`users.theme_settings`), so they follow the user to any device.

- The server renders the theme into `<html>` and `<head>` on every page
  (`app/theme.go`, `resources/views/root.html`), so there's no flash of the
  wrong theme. Guests get the defaults with color mode "system".
- Defaults are `defaultThemeSettings` in `app/theme.go`. The "default" preset
  in `app/theme_presets.go` must match `app.css`'s `:root` and `.dark`
  blocks; the other presets and the fonts are fixed catalogs there too.
- Radius, scale and layout values live in `app.css` (`html[data-radius]`,
  `html[data-scale]`). Use `layout-full:` to lift a width cap when the user
  picks the Full content layout, e.g. `max-w-7xl layout-full:max-w-none`.
- `Server.currentTheme` is the one place to layer anything above the user's
  choices; `kiosk:install` adds the company's preset there.

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
go run ./cmd/cli kiosk:install      # copies Kiosk's migrations + pages into this project,
                                     # then wires the allowlist/mail config, the mailer, and
                                     # kiosk.Register into app/config.go, app/server.go,
                                     # app/routes.go, and .env
go run ./cmd/cli kiosk:update        # bump the dependency, re-run install to pick up changes
go run ./cmd/cli kiosk:uninstall     # remove what kiosk:install copied in
```

`kiosk:install` copies Kiosk's migrations/pages (tracked in `.kiosk-lock`,
like before) *and* patches `app/config.go`, `app/server.go`, `app/routes.go`,
and `.env` with the wiring below — it's idempotent, so re-running it (e.g.
via `kiosk:update`) is a no-op on files that are already wired. If one of
those files has diverged too far from a fresh `spark new` project for it to
find where to patch, it prints that one snippet for you to add by hand
instead of guessing. `kiosk:uninstall` only removes the files it copied in
(still tracked via `.kiosk-lock`) — it does not undo the config/server/routes
wiring or run `go mod tidy`; both remain manual cleanup steps. See the
[kiosk README](https://github.com/martin3zra/kiosk) for the full endpoint
reference, the `Options` you pass to `kiosk.Register`, and the data model.

### Mail

Creating a company always needs an owner; when that owner's email doesn't
match an existing user, Kiosk creates an invitation and emails it rather
than failing or creating an account outright — `kiosk.Options.Mailer` is
what actually sends it. `kiosk:install` wires this in automatically; the
code below is what it writes, kept here as reference (e.g. if you're wiring
Kiosk into a non-`spark new` host, or `kiosk:install` couldn't find where to
patch and fell back to printing this).

**`app/config.go`**:

```go
type Config struct {
    // ...
    mail mailer.Config
}

// in LoadConfig():
mail: mailer.Config{
    Driver:      mailDriverOrDefault(os.Getenv("MAIL_DRIVER")),
    Host:        os.Getenv("MAIL_HOST"),
    Port:        os.Getenv("MAIL_PORT"),
    FromAddress: os.Getenv("MAIL_FROM_ADDRESS"),
    FromName:    os.Getenv("MAIL_FROM_NAME"),
    Username:    os.Getenv("MAIL_USERNAME"),
    Password:    os.Getenv("MAIL_PASSWORD"),
    Encryption:  os.Getenv("MAIL_ENCRYPTION"),
},

// mailDriverOrDefault defaults to mailer.Log — logging the rendered email
// (to LOG_FILE, see above) instead of sending it — so a fresh clone works
// with zero mail setup. Set MAIL_DRIVER once you have real credentials.
func mailDriverOrDefault(driver string) mailer.MailDriver {
    if driver == "" {
        return mailer.Log
    }
    return mailer.MailDriver(driver)
}
```

**`app/server.go`** — build the `mailer.Mailer` once, in `NewServer`, from the
same `resources` embed.FS already passed in for views (Kiosk's own mail
template lands in `resources/views/mail/` via `kiosk:install`, so there's
nothing extra to embed):

```go
type Server struct {
    // ...
    mailer mailer.Mailer
}

func NewServer(assets, resources embed.FS) *Server {
    config := LoadConfig()
    return &Server{
        // ...
        mailer: mailer.New(config.mail, resources),
    }
}
```

**`app/routes.go`** — pass both into `kiosk.Register`:

```go
kiosk.Register(s.route, kiosk.Options{
    DB:         s.play,
    Middleware: []routing.Middleware{s.AuthenticatedMiddleware},
    Allowlist:  s.config.kiosk.Developers,
    Mailer:     s.mailer,
    AppURL:     s.config.host,
})
```

Add `MAIL_DRIVER`, `MAIL_HOST`, `MAIL_PORT`, `MAIL_FROM_ADDRESS`, `MAIL_FROM_NAME`,
`MAIL_USERNAME`, `MAIL_PASSWORD`, and `MAIL_ENCRYPTION` to `.env` once you have
real mail credentials. Leave `MAIL_DRIVER` unset until then — invitations
still work, just logged (to `LOG_FILE`) instead of sent.

### Company-aware access control

Kiosk's own routes are gated by the developer allowlist, but nothing stops a
developer from browsing the rest of the app directly — `/home` has no idea
"developer" and "company member" are different identities. `kiosk:install`
wires this in too: a kiosk developer gets redirected to `/kiosks` on login
until they impersonate a company member, and the session carries which
company that member belongs to. This is a real product decision, not just
plumbing — it assumes Kiosk is a login boundary for a multi-tenant app, not
just an admin console layered on top of a single-tenant one. If that's not
what you want, revert this piece by hand (it's ordinary application code at
that point, not something `kiosk:uninstall` tracks or removes): drop
`s.RedirectDeveloperToKiosk` from the authenticated route group in
`app/routes.go`, and remove the `<ImpersonationBanner />` from
`resources/js/layouts/app/app-topnav-layout.tsx` and
`app-sidebar-layout.tsx` if you don't want it either. The rest (the
`isDeveloper`/`isImpersonating` props, `company_id` session tracking) is
inert without the redirect guard, so it's harmless to leave in place.

**`app/config.go`** — the developer allowlist `kiosk.Options.Allowlist` and
the guard below both read. `kiosk:install` wires this in automatically too
(nothing in a fresh clone parses `KIOSK_DEVELOPERS` on its own otherwise).
Kiosk owns both the type and the parsing (`kiosk.AllowlistConfig`,
`kiosk.ParseAllowlist`) — nothing to redefine here:

```go
type Config struct {
    // ...
    kiosk kiosk.AllowlistConfig
}

// in LoadConfig():
kiosk: kiosk.AllowlistConfig{
    Developers: kiosk.ParseAllowlist(os.Getenv("KIOSK_DEVELOPERS")),
},
```

**`app/auth-handlers.go`** — resolve which company the logging-in user
belongs to from Kiosk's `company_users` table and store it in session; clear
it on logout so it doesn't linger stale for the next login on that session:

```go
// companyMembership mirrors just enough of kiosk's company_users pivot to
// read which company (if any) a logging-in user belongs to. Kiosk owns that
// table; importing the kiosk package here would point the module
// relationship backwards, so this is a narrow read-only shadow of its shape.
type companyMembership struct {
    CompanyID int `db:"company_id"`
}

func (companyMembership) TableName() string { return "company_users" }

func resolveCompanyID(ctx *routing.Context, db *playsql.DB, userID int) (int, bool) {
    var row companyMembership
    if err := db.Model(&companyMembership{}).WhereEq("user_id", userID).First(ctx.Request.Context(), &row); err != nil {
        return 0, false
    }
    return row.CompanyID, true
}

// in authHandler(), right after ReGenerate:
if companyID, ok := resolveCompanyID(ctx, s.play, user.GetAuthIdentifier()); ok {
    sess.Put("company_id", companyID)
} else {
    sess.Delete("company_id")
}

// in logoutHandler(), right after Invalidate:
session.GetSession(ctx.Request).Delete("company_id")
```

**`app/middleware.go`** — expose the state to the frontend, and the guard
itself. `isImpersonating` reads `kiosk.ImpersonateSessionKey` (whether an
impersonation is active at all) rather than `company_id` — `isDeveloper`
describes the *current* identity, which is the impersonated target, not the
developer, once impersonation starts, so deriving `isImpersonating` from
`isDeveloper` would be false exactly when it needs to be true:

```go
// in SharedProps' props map, under "auth":
"isDeveloper":     isDeveloperSession(sess, s.config.kiosk.Developers),
"isImpersonating": sess.Get(kiosk.ImpersonateSessionKey) != nil,

// isDeveloperSession reads the session's raw "user" payload directly rather
// than through AuthUserFromContext/auth.ContextUserID{} — SharedProps runs
// before AuthenticatedMiddleware seeds that context value.
func isDeveloperSession(sess *session.Session, developers map[string]bool) bool {
    raw, ok := sess.Get("user").(map[string]any)
    if !ok {
        return false
    }
    email, _ := raw["email"].(string)
    return developers[strings.ToLower(email)]
}

// RedirectDeveloperToKiosk sends a kiosk developer to /kiosks until they're
// impersonating a company member — company_id in session is the signal
// (set/cleared by kiosk's impersonateStart/Stop), not a separate flag:
// developers never get one at login, only by impersonating.
func (s *Server) RedirectDeveloperToKiosk(next routing.HandlerFunc) routing.HandlerFunc {
    return func(ctx *routing.Context) {
        sess := session.GetSession(ctx.Request)
        if isDeveloperSession(sess, s.config.kiosk.Developers) && sess.Get("company_id") == nil {
            ctx.Redirect("/kiosks", http.StatusSeeOther)
            return
        }
        next(ctx)
    }
}
```

**`app/routes.go`** — wire the guard onto the authenticated group, excluded
from `/logout` (a developer stuck with no company context must still be able
to sign out rather than bouncing between `/kiosks` and nowhere else):

```go
s.route.GroupWithMiddleware([]routing.Middleware{s.AuthenticatedMiddleware, Verified, s.RedirectDeveloperToKiosk}, func(route *routing.Router) {
    route.GET("/home", s.homeHandler).Can("view:dashboard")
    route.POST("/logout", s.logoutHandler).
        WithoutMiddleware(s.RedirectDeveloperToKiosk)
    // ...
})
```

**Frontend** — a banner while impersonating, shown on every authenticated
page rather than just inside `/kiosks`, since impersonating someone lands you
on `/home`, not back on the Kiosk dashboard:

```tsx
// resources/js/components/impersonation-banner.tsx
import { usePage } from "@inertiajs/react";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import { useCsrfHeaders } from "@/lib/csrf";
import { type SharedProps } from "@/types";

export function ImpersonationBanner() {
  const { auth } = usePage<SharedProps>().props;
  const headers = useCsrfHeaders();
  const [stopping, setStopping] = useState(false);

  if (!auth.isImpersonating) return null;

  async function stop() {
    setStopping(true);
    await fetch("/kiosks/impersonate", { method: "DELETE", headers });
    // Hard navigation — the session identity just changed back to the
    // developer's, so every prop needs a fresh full-page load.
    window.location.assign("/kiosks");
  }

  return (
    <div className="flex items-center justify-between gap-3 bg-amber-500 px-4 py-2 text-sm font-medium text-amber-950">
      <span>Impersonating <strong>{auth.user?.email}</strong></span>
      <Button type="button" size="xs" variant="outline" disabled={stopping} onClick={stop}>
        {stopping ? "Stopping…" : "Stop impersonating"}
      </Button>
    </div>
  );
}
```

Where it renders has to differ by shell, not live once in `app-layout.tsx`:
`app-topnav-layout.tsx`'s nav is a normal-flow element, so
`<ImpersonationBanner />` as the first child of `<AppShell variant="header">`
(above `<AppTopNav />`) just pushes it down — no overlap. The sidebar shell
is different: `<Sidebar>` is `fixed inset-y-0`, anchored to the viewport
regardless of where it sits in the DOM, so a banner placed above the whole
shell ends up at the same y=0 the sidebar does and gets covered by it rather
than pushing it down. Render it inside `<AppContent variant="sidebar">`
instead (the first child, above `<AppSidebarHeader />`) — that's
`SidebarInset`, a normal-flow column next to the fixed sidebar, not
underneath it. Add `isDeveloper`/`isImpersonating` to the `Auth` type in
`resources/js/types/index.d.ts` either way.

## Notes

- `created_at` / `last_activity` on `sessions` are `BIGINT` (Unix-millis), as the
  forge session store expects.
- `AuthenticatedMiddleware` seeds `routing.PermissionKey{}` from the user's role
  so route `.Can(...)` guards resolve.
