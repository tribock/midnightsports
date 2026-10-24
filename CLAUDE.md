# Midnightsports

Web app for "Midnightsports" (open sports nights for young people in Aadorf, Switzerland): event planning,
participant check-in via QR code, team planning with wages, payout with handwritten signatures, reports and
time tracking. Mobile-first, used on phones at the event.

## Read first

- `docs/architektur.md`: system overview, use cases per role, ERD, sequence diagrams (login, QR check-in, payout), performance rules
- `docs/anforderungen.md`: client requirements and their status (done / open / deliberately not done). Update it when requirements change
- `README.md`: features per role (German)
- `docs/lokale-entwicklung.md`: local setup, `.env`, dev login without Google

The client's original documents (`docs/Anpassungen.docx`, `docs/Auszahlung.jpeg`, `docs/Midnightsports_*.pdf`) are
gitignored and may be missing. Their content is captured in `docs/anforderungen.md`.

## Stack and layout

- `api/`: Go (stdlib `net/http` with method patterns, pgx v5, no framework, no ORM). Serves the API and the built frontend
  - `cmd/server/main.go`: startup, config, migrations
  - `internal/server/`: HTTP handlers, one file per area (`events.go`, `checkins.go`, `team.go`, `payout.go`, `users.go`, `emergency.go`, …)
    - `server.go`: all routes with their roles, error handling, JSON helpers, `validUUID`/`pathUUID`
    - `db.go`: DB helpers `execOne`, `loadUser`, `isForeignKeyViolation`, `isUniqueViolation`
    - `static.go`: serves `web/dist`, gzip + immutable caching for `/assets/`
  - `internal/db/migrations/001_init.sql`: the complete schema (see below)
  - `internal/qr`, `internal/wallet`: signed QR tokens, Apple/Google Wallet passes
- `web/`: React 19 + TypeScript + Vite, react-router 7, plain CSS. No UI library, no state library
  - `src/api.ts`: fetch wrapper and all API types (mirror the Go JSON structs, keep in sync)
  - `src/auth.tsx`: current user, role helpers `isAdmin`, `isProjectLead`, `isLeader`, `canViewUsers`
  - `src/format.ts`: labels and formatting (CHF, dates, addresses, `parseDecimal`)
  - `src/pages/`: one page per route (routes in `App.tsx`), `src/components/`: shared pieces
  - `src/styles.css`: design tokens on `:root` with dark mode, all styles, print styles
- `build/`: Dockerfile, Cloud Run demo config. `Makefile`: all commands

## Commands

- `make dev`: Postgres in Docker, backend (`go run`, :8080) and frontend (Vite, :5173) with hot reload
- `make run`: whole app in Docker (:8080), like production
- `make test`: `go vet` + `go test` and `vitest`
- `make db-reset`: delete and recreate the local database (needed after schema changes)
- Frontend only: `cd web && npm run build` (includes `tsc`), `npm test`
- Backend only: `cd api && go vet ./... && go test ./...`

## Conventions

- **Language**: UI texts, error messages, code comments, docs and test names are German, Swiss spelling (`ss`, never `ß`: "gross", "Strasse"). Identifiers are English
- **Schema**: the app is not in production. There is exactly one migration, `001_init.sql`. Change it in place, never add a `002_…` file. The database is always rebuilt from scratch (`make db-reset`); no data migration or legacy compatibility needed
- **Comments**: short German comments explain *why*, at the density of the surrounding code
- **Tests**: keep them simple: table tests for pure functions, no database in unit tests. Extract business rules into pure functions to test them (e.g. `staffEntry`, `validateProfile`, `parseSignature`)
- Run `gofmt`, `go vet`, the tests and `npm run build` before finishing a change

### Backend

- Handler signature `func(w, r, u *User) error`, registered with `s.route("METHOD /api/path", roles, handler)`. Roles: `anyRole`, `leaders` (MA, PL, Admin), `projectLeads` (PL, Admin), `admins`. The server always enforces roles; the frontend only hides things
- Errors: return `errorf(status, "German message")`, `errNotFound`, `errForbidden`. `pgx.ErrNoRows` becomes 404 automatically, anything else 500
- Write path: `s.execOne(...)` (404 if no row affected), then `return noContent(w)`
- **IDs**: compare UUID columns directly (`WHERE id = $1`), never `id::text = $1`, which defeats the index. Validate incoming IDs with `pathUUID` / `validUUID` first
- Load rows with `pgx.CollectRows(rows, pgx.RowToStructByPos[T])`, or a `rows.Next()` loop with `defer rows.Close()`. Several writes in one transaction: `pgx.Batch`
- SQL lives in the handlers as plain strings. Keep it readable rather than clever

### Frontend

- Data loading: `api.get<T>(path).then(set).catch((err) => setError(errorMessage(err)))` in `useEffect`. Debounced searches must ignore stale responses (see `AdminUsersPage`)
- Role checks only through the helpers in `auth.tsx`
- Use CSS classes from `styles.css` (`stack`, `row`, `card`, `list`, `table`, `actions`, `badge`, `chip`, `icon-button`, `muted`, `small`, `error`) and its color tokens. New colors go into the tokens, light and dark
- Printing: render a `PrintSheet` (`components/PrintSheet.tsx`, hidden on screen, the only thing printed) and call `printDocument(filename)`. "PDF" means the browser's "Save as PDF"
- Pages with heavy libraries are lazy-loaded in `App.tsx`
- Logos: sources are `docs/logo/*.eps`. `make logos` (`build/logo/generate.py`, needs `rsvg-convert`) generates `web/public/icon.svg`, `logo.svg`, `logo-white.svg` (header, dark mode), the PNG icons and the wallet images in `api/internal/wallet/assets/`. Never edit the generated files by hand

## Security

- Every API route is registered with `s.route(...)`, which requires a session. Only health, version, config, the login endpoints and the wallet logo are public (`m.HandleFunc`). `TestRoutesRequireLogin` checks all routes automatically; never register data endpoints with `m.HandleFunc`
- Grant the narrowest role that works; full user profiles (`GET /api/users/{id}`) are admin-only
- `security.go` wraps everything: CSP, `X-Frame-Options`, `nosniff`, HSTS on HTTPS, and `http.CrossOriginProtection` against CSRF. The CSP allows no inline scripts or external hosts; if a feature needs one, extend the policy deliberately
- Google login uses state + PKCE cookies, Apple login state + nonce; sessions are random tokens, stored only as HMAC
- Free text in CSV exports goes through `csvText` (formula injection)
- Keep Go patched (`go` directive in `go.mod`) and check with `go run golang.org/x/vuln/cmd/govulncheck@latest ./...` and `npm audit`

## Domain rules (easy to break)

- **Roles**: Teilnehmer < Mitarbeiter (participant list, check-in, guests) < Projektleiter (+ create/edit/cancel events, team, wages, payout, reports, time tracking) < Admin (+ administration, users). Each role can do everything the role below can
- **Functions** (JC, JSC, SC, PL) carry a wage per event; `salary NULL` means paid through time tracking (no amount at the event, no payout signature)
- **Event staff** (`event_staff`) copies function and wage when someone is assigned, so later wage changes don't alter past events. If a person stays with the same function, the amount and the payout signature are kept (`staffEntry`). The responsible project lead is automatically part of the team (frontend, `EventFormPage`)
- **Payout**: each salaried staff member signs on the device (PNG, max. 512 KB, stored in `event_staff.signature`); signing twice gives 409; the signature can be removed
- **Emergency contacts**: visible to admins always, to roles/functions selected under Administration (default: Projektleiter), never to Teilnehmer. The server strips them in `User.visibleTo`
- **Guests**: normal users with `guest = true`, email optional. Signing in with Google using the same email takes over the account. Only guests' emails are editable (other emails come from Google)
- **Profile status** (`schueler_aadorf`, `gast_extern`, `mitarbeiter`): only admins may set or remove `mitarbeiter` (`checkStatusChange`, enforced on the server). Others see only the first two; an already assigned `mitarbeiter` is shown read-only
- **Deleting users**: check-ins remain with `user_id = NULL` for statistics
- **Event times**: Europe/Zurich; an end time before the start time means the next day (`eventTimes`)
- **Login**: Google OAuth, optionally Sign in with Apple (`apple.go`: `form_post` callback exempt from CSRF, `SameSite=None` state/nonce cookies, client secret JWT from the `.p8` key). Accounts are matched by `google_sub`/`apple_sub`, then email; emails in `ADMIN_EMAIL` always become admin; `DEV_LOGIN=true` allows login by email locally
