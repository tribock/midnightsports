# Midnightsports

Web app for "Midnightsports" (open sports nights for young people, Switzerland): event planning,
participant check-in via QR code, team planning with wages, payout with handwritten signatures, reports and
time tracking. Mobile-first, used on phones at the event. Multi-tenant: each project (tenant, e.g.
"Midnightsports Aadorf", "Pleno") has its own events, locations, functions, members, warnings/bans and settings;
people and their profile are global.

## Read first

- `docs/architektur.md`: overview, roles, ERD, rules (tenant isolation, warnings/bans, sharing, audit), sequence diagrams (project link, check-in), open data migration, performance
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
    - `projects.go`: `Project`, `Member`, `loadMember`, project search, join, public project info and logo
    - `members.go`: members of a project (Teilnehmer page), role/status, emergency contact
    - `sanctions.go`: warnings/bans, `effectiveSanction` (deadlines), sanction rules
    - `shares.go`: consent to sharing warnings between projects. `admin.go`: global administration (projects, onboarding, users)
    - `audit.go`: `audit(...)`, action constants, audit list
    - `db.go`: DB helpers `execOne`, `loadUser`, `isForeignKeyViolation`, `isUniqueViolation`
    - `static.go`: serves `web/dist`, gzip + immutable caching for `/assets/`
  - `internal/db/migrations/`: `001_init.sql` is the initial schema, later changes are added as new files (see Schema below)
  - `internal/qr`, `internal/wallet`: signed QR tokens, Apple/Google Wallet passes
- `web/`: React 19 + TypeScript + Vite, react-router 7, plain CSS. No UI library, no state library
  - `src/api.ts`: fetch wrapper and all API types (mirror the Go JSON structs, keep in sync)
  - `src/auth.tsx`: current user (`Me`, with projects), role helpers `isAdmin`, `isProjectLead`, `isLeader`, `canViewMembers` (take the project `Member`), remembered target after login
  - `src/project.tsx`: `ProjectProvider` for all pages under `/p/:slug`, `useProject()` gives `member`, `project`, a scoped `api` (prefix `/p/{slug}`) and `base` for links
  - `src/format.ts`: labels and formatting (CHF, dates, addresses, `parseDecimal`)
  - `src/pages/`: one page per route (routes in `App.tsx`), `src/components/`: shared pieces
  - `src/styles.css`: design tokens on `:root` with dark mode, all styles, print styles
- `build/`: Dockerfile, Cloud Run demo config. `Makefile`: all commands

## Commands

- `make dev`: Postgres in Docker, backend (`go run`, :8080) and frontend (Vite, :5173) with hot reload
- `make run`: whole app in Docker (:8080), like production
- `make test`: `go vet` + `go test` and `vitest`
- `make db-reset`: delete and recreate the **local** database of the current branch (never in production)
- Local Docker containers and the database volume are per branch: compose project `midnightsports` on `main`, `midnightsports-<branch>` elsewhere (`COMPOSE_PROJECT_NAME` in the `Makefile`). Use `make db` / `make psql` / `make containers`, not plain `docker compose`
- Frontend only: `cd web && npm run build` (includes `tsc`), `npm test`
- Backend only: `cd api && go vet ./... && go test ./...`

## Conventions

- **Language**: UI texts, error messages, code comments, docs and test names are German, Swiss spelling (`ss`, never `ß`: "gross", "Strasse"). Identifiers are English
- **Schema**: production (branch `main`) runs on Cloud SQL with real data. There, never change a migration that has already run; every schema change is a new file `00N_short_name.sql` (runs once in a transaction at startup, `db.Migrate`) that works on existing data, and take a Cloud SQL backup before deploying it.
  **Branch `feat/multitenancy`**: the multi-tenant schema is a rewrite of `001_init.sql` for a fresh deployment (`make db-reset` locally). Until the merge, change `001_init.sql` in place on this branch. Migrating the production data is planned after the merge (`docs/architektur.md`, "Migration der bestehenden Daten"); don't write a data migration before that. Keep the ERD in `docs/architektur.md` in sync
- **Docs**: short and to the point, no emojis
- **Comments**: short German comments explain *why*, at the density of the surrounding code
- **Tests**: keep them simple: table tests for pure functions, no database in unit tests. Extract business rules into pure functions to test them (e.g. `staffEntry`, `validateProfile`, `parseSignature`)
- Run `gofmt`, `go vet`, the tests and `npm run build` before finishing a change

### Backend

- Three kinds of routes in `server.go`. The server always enforces roles; the frontend only hides things:
  - `s.route("METHOD /api/path", h)`: logged in, no project (`func(w, r, u *User) error`)
  - `s.adminRoute(...)`: global admins only (`/api/admin/...`)
  - project routes `p("METHOD /path", roles, h)` → `/api/p/{project}/path`, handler `func(w, r, m *Member) error`. `Member` embeds `*User` and has `Project`, `Role`, `Status`, `EmergencyAccess`. Roles per project: `anyRole`, `leaders` (MA, PL, Admin), `projectLeads` (PL, Admin), `admins` (project admin). Global admins are admin in every project
- **Tenant isolation**: every query in a project handler filters by `m.Project.ID` (or joins through an event of the project). Load events with `loadEvent(ctx, m, id)` or check them with `projectEvent(r, m)`, members with `projectMemberUser(r, m)`; data of other projects must come out as 404. Locations and functions have composite foreign keys `(project_id, id)` so the database rejects cross-project references
- **Audit**: changes and accesses to personal data write `s.audit(ctx, tx, auditEntry{...})` in the same transaction as the change (no entry, no change). New audit-relevant actions get a constant in `audit.go` and a label in `format.ts` (`auditLabels`)
- Errors: return `errorf(status, "German message")`, `errNotFound`, `errForbidden`. `pgx.ErrNoRows` becomes 404 automatically, anything else 500
- Write path: `s.execOne(...)` (404 if no row affected), then `return noContent(w)`
- **IDs**: compare UUID columns directly (`WHERE id = $1`), never `id::text = $1`, which defeats the index. Validate incoming IDs with `pathUUID` / `validUUID` first
- Load rows with `pgx.CollectRows(rows, pgx.RowToStructByPos[T])`, or a `rows.Next()` loop with `defer rows.Close()`. Several writes in one transaction: `pgx.Batch`
- SQL lives in the handlers as plain strings. Keep it readable rather than clever

### Frontend

- Data loading: `api.get<T>(path).then(set).catch((err) => setError(errorMessage(err)))` in `useEffect`. In project pages use `const { api, base, member } = useProject()` (scoped API, links as `${base}/events/...`), elsewhere the global `api`. Debounced searches must ignore stale responses (see `MembersPage`)
- Role checks only through the helpers in `auth.tsx`, with the project `member` (not the global user)
- Use CSS classes from `styles.css` (`stack`, `row`, `card`, `list`, `table`, `actions`, `badge`, `chip`, `icon-button`, `muted`, `small`, `error`) and its color tokens. New colors go into the tokens, light and dark
- Printing: render a `PrintSheet` (`components/PrintSheet.tsx`, hidden on screen, the only thing printed) and call `printDocument(filename)`. "PDF" means the browser's "Save as PDF"
- Pages with heavy libraries are lazy-loaded in `App.tsx`
- Project logos: uploaded logo (`/api/public/projects/{slug}/logo`) or the default logo, always through `projectLogo(...)` in `project.tsx`
- Logos: sources are `docs/logo/*.eps`. `make logos` (`build/logo/generate.py`, needs `rsvg-convert`) generates `web/public/icon.svg`, `logo.svg`, `logo-white.svg` (header, dark mode), the PNG icons and the wallet images in `api/internal/wallet/assets/`. Never edit the generated files by hand

## Security

- Every API route is registered with `s.route`, `s.adminRoute` or a project route, which require a session. Only health, version, config, the login endpoints, the wallet logo and the public project info for the login page (`/api/public/projects/{project}` and its logo, active projects only) are public (`m.HandleFunc`). `TestRoutesRequireLogin` checks all routes automatically; never register data endpoints with `m.HandleFunc`
- Grant the narrowest role that works; full profiles, editing and deleting people are global-admin only (`/api/admin/users`)
- Uploaded logos: PNG, JPEG or WebP only (`parseLogo` checks the bytes), never SVG (scripts)
- Pending projects (sharing not yet consented) are only reachable by global admins (`loadMember`), hidden from search, public info and the public logo; the frontend doesn't open them
- **Strict tenant isolation**: never load data before the membership/project check passed. Check membership first, then load the person (`checkInUser`). Error messages must not reveal data of other projects (names, whether an email exists)
- `security.go` wraps everything: CSP, `X-Frame-Options`, `nosniff`, HSTS on HTTPS, and `http.CrossOriginProtection` against CSRF. The CSP allows no inline scripts or external hosts; if a feature needs one, extend the policy deliberately
- Google login uses state + PKCE cookies, Apple login state + nonce; sessions are random tokens, stored only as HMAC
- Free text in CSV exports goes through `csvText` (formula injection)
- Keep Go patched (`go` directive in `go.mod`) and check with `go run golang.org/x/vuln/cmd/govulncheck@latest ./...` and `npm audit`

## Domain rules (easy to break)

- **Roles** (per project, `project_members.role`): Teilnehmer < Mitarbeiter (participant list, check-in, guests) < Projektleiter (+ create/edit/cancel events, team, wages, payout, reports, time tracking, warnings/bans) < Admin = project admin (+ administration, members' role and status, sharing consent). Each role can do everything the role below can. **Global admins** (`users.global_admin`, from `ADMIN_EMAIL`) are admin in every project and alone may create projects, edit or delete people, and read the audit log
- **Joining a project**: only by the person themself, via the start page (search) or the project link `/?projekt=<slug>` (remembered across the login, `rememberAfterLogin`). Non-members can't be checked in. Guests created at check-in become members
- **Functions** (JC, JSC, SC, PL) carry a wage per event; `salary NULL` means paid through time tracking (no amount at the event, no payout signature)
- **Event staff** (`event_staff`) copies function and wage when someone is assigned, so later wage changes don't alter past events. If a person stays with the same function, the amount and the payout signature are kept (`staffEntry`). The responsible project lead is automatically part of the team (frontend, `EventFormPage`)
- **Payout**: each salaried staff member signs on the device (PNG, max. 512 KB, stored in `event_staff.signature`); signing twice gives 409; the signature can be removed
- **Emergency contacts**: visible to project admins and global admins always, to roles/functions selected under the project's Administration (default: Projektleiter), never to Teilnehmer. The server strips them in `User.visibleTo`; every explicit lookup is audited
- **Guests**: normal users with `guest = true`, email optional. Signing in with Google using the same email takes over the account. Only guests' emails are editable (other emails come from Google)
- **Status** per project (`project_members.status`: `schueler` = "Schüler-/in aus <project region>", `gast_extern`, `mitarbeiter`): chosen when joining; only project admins may set or remove `mitarbeiter` (`checkStatusChange`)
- **Warnings and bans** (`sanctions`, one per person and project): warned people are checked in with a yellow notice, banned people are not checked in and can't register for events. Deadlines per project are applied when reading (`effectiveSanction`), using the rules of the issuing project. Visible only in the issuing project and in partner projects with consented sharing; only the issuing project can change them
- **Sharing** (`project_shares`): requested by the global admin when creating a project; the new project stays pending until every partner's project admin consents
- **Deleting users** (global admin or the person): check-ins remain with `user_id = NULL` for statistics; names in the audit log become "Gelöschtes Konto"
- **Event times**: Europe/Zurich; an end time before the start time means the next day (`eventTimes`)
- **Login**: Google OAuth, optionally Sign in with Apple (`apple.go`: `form_post` callback exempt from CSRF, `SameSite=None` state/nonce cookies, client secret JWT from the `.p8` key). Accounts are matched by `google_sub`/`apple_sub`, then email; emails in `ADMIN_EMAIL` always become global admins; a project admin created by email during onboarding takes over that account on first login; `DEV_LOGIN=true` allows login by email locally
