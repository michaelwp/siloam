# Data Base Jemaat Kolom 9

Web app for managing the congregation data of Jemaat Kolom 9 (2027 – 2031),
replacing the `Data Base Jemaat Kolom 9 2027 - 2031.xlsx` workbook.

- **Backend:** Go 1.25 (`net/http`), MongoDB
- **Frontend:** React 19 + TypeScript (Vite), embedded into the Go binary with `go:embed`
- **Sign-in:** username/password or passkey (WebAuthn), admin-managed accounts
- **Runs:** locally, with Docker (compose), or with Apple `container` on macOS 26+
- **Light / dark theme:** toggle in the sidebar (and on the sign-in page); follows the
  system setting until you pick one, then remembers it per browser

## Features

Each sheet of the workbook is a page, and every page supports create / read / update / delete:

| Sheet                 | Page              | What it manages                                                    |
|-----------------------|-------------------|--------------------------------------------------------------------|
| Daftar Keluarga       | `/keluarga`       | Families: name, keterangan (e.g. Simpatisan), domisili, jiwa count |
| Data Base Keluarga    | `/database`       | Each family with wedding date and its members                      |
| Daftar HUT Jemaat     | `/hut-jemaat`     | Members grouped by birth month, with age (`X Tahun Y Bulan Z Hari`) |
| Daftar HUT Pernikahan | `/hut-pernikahan` | Wedding anniversaries in calendar order, with years married        |
| Daftar Anggota Jemaat | `/anggota`        | All members A–Z, filter by gender                                  |
| Kategorial BIPRA      | `/bipra`          | Members by category, with counts per category                      |

All pages share two MongoDB collections, `families` and `members`, so an edit
on one page shows up everywhere. Values the spreadsheet computed with formulas
are computed by the server:

- **Umur** – years/months/days since birth (same as Excel `DATEDIF`)
- **Umur Pernikahan** – full years since the wedding date
- **Kategori BIPRA** – Anak (ASM) 0–11 · Remaja 12–16 · Pemuda 17–30 ·
  PKB (male) / WKI (female) 31+ or married · Lansia 60+ as an extra flag

## Sign-in & users

- **First run:** when there are no users, the server creates an admin account
  **`admin` / `admin`** (override with `INITIAL_ADMIN_USER` / `INITIAL_ADMIN_PASSWORD`).
  After signing in, the app shows only an *Perbarui Akun* screen until a new
  username and password are set.
- **Passkeys:** under *Akun Saya* each user can add passkeys (Touch ID / Face ID,
  Windows Hello, Android, security keys), rename or delete them, then use
  *Masuk dengan Passkey* on the sign-in page without typing a username.
- **Admin → Pengguna:** add users (role *Pengguna* or *Admin*) with a generated
  temporary password, edit username/role, reset a password (optionally removing
  the user's passkeys, e.g. for a lost phone) and delete users. New and reset
  accounts must change their password at the next sign-in. You cannot delete or
  demote yourself, and there is always at least one admin.
- Both roles can manage the jemaat data; only admins manage users.
- **Account lockout:** after **3 wrong passwords in a row** an account is locked
  for **3 minutes** (`MAX_FAILED_LOGINS`, `LOCK_DURATION`; the sign-in page shows
  the current values). During the lock it cannot sign in — not even with the
  correct password or a passkey — and its open sessions are ended; attempts
  during the lock do not extend it. When it expires the user can sign in again
  and the count starts over. Wrong passwords typed into the change-password form
  count too. A failed sign-in always gets the same response (wrong password,
  unknown username and locked account look identical); a user who signs in with
  a valid passkey is told the account is locked and for how long. On *Pengguna*
  a locked account shows **Terkunci s.d. HH.MM**, and **Buka Kunci** unlocks it
  immediately with a temporary password. A successful sign-in (password or
  passkey) resets the counter.
- **API rate limiting:** every `/api` request is limited per client IP
  (`API_RATE_LIMIT`, default 600/minute with bursts), and the sign-in endpoints
  (password login, passkey login, change password) much more strictly
  (`AUTH_RATE_LIMIT`, default 20/minute), whatever the outcome. Over the limit
  the API answers `429` with a `Retry-After` header. Static files and
  `/api/health` are not limited.
- **Locked out as the only admin?** Run, where the server's `.env` is:
  ```sh
  make user-reset NAME=erwin    # unlocks, prints a temporary password
  ```
  This needs access to the database (`MONGODB_URI`), not to the web app.
- **Password rules** (checked live with a strength meter, and enforced by the
  server): at least 10 characters, upper- and lowercase letters, a number, a
  symbol, must not contain the username, and must not be a common password
  (e.g. `Password123!`, `Jemaat2027#`). Generated temporary passwords always
  meet the rules. The rules live in `internal/auth/password.go` and
  `web/src/password.ts` — keep the two in sync.

Security details: bcrypt (cost 12) password hashes; random session tokens in an
`HttpOnly`, `SameSite=Lax` cookie, stored as SHA-256 hashes in MongoDB with a 7-day
TTL; writes from other origins are rejected (CSRF); accounts lock after 3 wrong
passwords for 3 minutes (see above); requests are rate-limited per IP, and 30
failed sign-ins from one IP pause sign-ins from it for 15 minutes; changing or resetting a password
signs the account out of other browsers. User records are only ever changed with
targeted atomic updates, so concurrent actions (a lock, an admin reset, a
passkey change) never overwrite each other, and two admins cannot remove the
last admin at the same moment.

Behind a reverse proxy (nginx, Caddy, a load balancer), set `TRUSTED_PROXIES`
to its address so rate limiting uses the real client IP from `X-Forwarded-For`
instead of the proxy's.

> **Passkeys need a secure context.** They work on `http://localhost` and on HTTPS.
> When serving on a real domain, put the app behind HTTPS and set
> `WEBAUTHN_RP_ID=jemaat.example.org`, `WEBAUTHN_ORIGINS=https://jemaat.example.org`
> and `COOKIE_SECURE=true`. Passkeys are bound to the RP ID, so changing the domain
> later means users must register them again. Opening the app by LAN IP over plain
> HTTP works with passwords only.

## Quick start

Run `make help` to list every command.

### Local (needs Go 1.25+, Node 24+, and a MongoDB)

```sh
make env       # create .env from .env.example, then set MONGODB_URI
make setup     # npm/go dependencies + git pre-commit hook
make import    # load the workbook into MongoDB (replaces existing data)
make run       # build and serve http://localhost:8080
```

Then open http://localhost:8080 and sign in with `admin` / `admin`.

The workbook is read from `~/Downloads/Data Base Jemaat Kolom 9 2027 - 2031.xlsx`
by default; override with `make import XLSX=/path/to/file.xlsx`.

### Docker (compose: app + MongoDB)

```sh
make docker-up       # build image, start the app (+ bundled mongo if MONGODB_URI is localhost) → http://localhost:8080
make docker-import   # load the workbook into the container's MongoDB
make docker-logs     # follow logs
make docker-down     # stop (data kept in the "mongo-data" volume)
make docker-clean    # stop and DELETE the data volume
```

### Apple container (macOS 26+)

```sh
make mac-up          # starts the container service, builds, runs the app (+ bundled mongo if MONGODB_URI is localhost)
make mac-import      # load the workbook into the container's MongoDB
make mac-logs        # follow app logs
make mac-down        # stop (data kept in the "siloam-mongo-data" volume)
make mac-clean       # remove containers, network and DELETE the data volume
```

Apple `container` has no compose and no service-name DNS, so the Makefile
creates a `siloam-net` network and passes MongoDB's IP to the app. If the
MongoDB container is restarted on its own, run `make mac-up` again so the app
gets the new IP.

Use `PORT=9000` with any `*-up` target to change the host port.

## Development

```sh
make dev        # Go API on :8080
make dev-web    # Vite on :5173 with hot reload, proxies /api to :8080
make test       # Go tests (store tests use a throwaway database on MONGODB_TEST_URI,
                #   default mongodb://localhost:27017, and are skipped without one)
```

### Lint

```sh
make lint       # both
make lint-go    # golangci-lint: vet, staticcheck, errcheck, gocritic, gofmt, goimports, …
make lint-web   # oxlint (react, hooks, jsx-a11y, typescript, import, unicorn) + tsc
make lint-fix   # auto-fix what can be fixed
```

Configuration: `.golangci.yml` (Go) and `web/.oxlintrc.json` (React). Warnings
fail the React lint (`--deny-warnings`).

### Git hook

`make hooks` (also part of `make setup`) points git at `.githooks/`. The
`pre-commit` hook runs `lint-go` when Go files are staged and `lint-web` when
files under `web/` are staged; a lint failure blocks the commit. Bypass once
with `git commit --no-verify`. Requires `golangci-lint` (`brew install golangci-lint`).

## Configuration

Settings are environment variables. The app also reads a **`.env`** file from
the working directory (`ENV_FILE` to use another path); variables already set in
the environment win over `.env`. Start from `.env.example` (`make env`). `.env` is
git-ignored and excluded from Docker builds — keep credentials there only.

| Env / flag                       | Default                                   |
|----------------------------------|-------------------------------------------|
| `MONGODB_URI` / `-mongo`         | `mongodb://localhost:27017/siloam` — the database name is the URI path (default `siloam`; `-db` overrides) |
| `ADDR` / `-addr`                 | `:8080`                                   |
| `WEBAUTHN_RP_ID`                 | `localhost` — passkey domain              |
| `WEBAUTHN_ORIGINS`               | `http://localhost:<port>`, `http://localhost:5173` (comma-separated URLs users open) |
| `COOKIE_SECURE`                  | `false` — set `true` behind HTTPS         |
| `INITIAL_ADMIN_USER`             | `admin` — only used when no users exist   |
| `INITIAL_ADMIN_PASSWORD`         | `admin` — must be changed on first sign-in |
| `MAX_FAILED_LOGINS`              | `3` — wrong passwords in a row before an account locks |
| `LOCK_DURATION`                  | `3m` — how long a lock lasts (`90s`, `3m`, `1h`) |
| `API_RATE_LIMIT`                 | `600` — `/api` requests per client IP per minute |
| `AUTH_RATE_LIMIT`                | `20` — sign-in attempts per client IP per minute |
| `TRUSTED_PROXIES`                | none — comma-separated IPs/CIDRs of reverse proxies, e.g. `10.0.0.0/8,127.0.0.1` |
| `-import file.xlsx [-replace]`   | import the workbook and exit              |
| `-copy-from <uri> [-collections families,members] [-replace]` | copy data from another MongoDB and exit |
| `-reset-user <name>`             | unlock an account, print a temporary password and exit |

Example for MongoDB Atlas (URL-encode special characters in the password):

```sh
MONGODB_URI=mongodb+srv://siloam:p%40ssw0rd@cluster0.abcde.mongodb.net/siloam?retryWrites=true&w=majority
```

**Everything uses `MONGODB_URI` from `.env`** — `make run`, `make import`,
`make db-copy`, and the containers (`make docker-up`, `make mac-up`). The one
exception: a `localhost` URI cannot be reached from inside a container, so with
a localhost URI the container targets start the bundled MongoDB container
instead. `make db-which` shows which database each will use. The sign-in
settings from `.env` are passed to the containers too, e.g. `make docker-up PORT=9000`.

### Moving data to another database

```sh
# 1. put the new connection string in .env (MONGODB_URI)
# 2. copy everything from the old database into it
make db-copy                                   # from mongodb://localhost:27017/siloam
make db-copy FROM="mongodb://old-host/siloam"  # from another source
make db-copy REPLACE=1                         # overwrite a target that has data
make db-copy ONLY=families,members             # jemaat data only; keep the target's user accounts
make db-which                                  # show which database each command/container uses
```

To move what is in the bundled container MongoDB (e.g. accounts created while
running `make mac-up` / `make docker-up` with a localhost URI) into the remote
database instead, use `make mac-db-copy` or `make docker-db-copy` (same
`REPLACE=1` option).

Families, members and users (with passwords and passkeys) are copied with
their IDs, so links between them stay intact. Sessions are not copied — everyone
signs in again once. The command refuses a non-empty target unless `REPLACE=1`
(which clears only the collections being copied), and refuses to copy a
database onto itself — detected by writing a marker, so `localhost`,
`127.0.0.1` or a container name for the same server are all caught. The source
is only read.

## API

Everything under `/api` except `/api/health` and the sign-in endpoints requires
a session cookie; `/api/users` requires the admin role. Errors are
`{"error": "...", "code": "..."}`; `code` is `unauthenticated` (401),
`password_change_required` (403), `forbidden` (403), `rate_limited` (429, with
`Retry-After`) or `account_locked` (423, only from passkey sign-in, where the
user has proven who they are).

| Method | Path                                   | Notes                                     |
|--------|----------------------------------------|-------------------------------------------|
| GET    | `/api/health`                          | `200` when MongoDB is reachable (public)  |
| GET    | `/api/auth/config`                     | `{maxFailedLogins, lockSeconds}` for the sign-in page (public) |
| POST   | `/api/auth/login`                      | `{username, password}` (public)           |
| POST   | `/api/auth/logout`                     |                                           |
| GET    | `/api/auth/me`                         | current user with passkeys                |
| POST   | `/api/auth/change-credentials`         | `{currentPassword, username, newPassword}` |
| POST   | `/api/auth/passkey/login/begin`        | WebAuthn assertion options (public)       |
| POST   | `/api/auth/passkey/login/finish`       | assertion response → session (public)     |
| POST   | `/api/auth/passkeys/register/begin`    | WebAuthn creation options                 |
| POST   | `/api/auth/passkeys/register/finish?name=` | attestation response                  |
| PUT    | `/api/auth/passkeys/{id}`              | rename `{name}`                           |
| DELETE | `/api/auth/passkeys/{id}`              |                                           |
| GET    | `/api/users`                           | admin                                     |
| POST   | `/api/users`                           | admin: `{username, password, role}`       |
| PUT    | `/api/users/{id}`                      | admin: `{username, role}`                 |
| POST   | `/api/users/{id}/reset-password`       | admin: `{password, clearPasskeys}`; also unlocks |
| DELETE | `/api/users/{id}`                      | admin                                     |
| GET    | `/api/families`                        | with `memberCount`, `weddingYears`        |
| POST   | `/api/families`                        | `no` is assigned automatically            |
| GET    | `/api/families/{id}`                   | includes `members`                        |
| PUT    | `/api/families/{id}`                   |                                           |
| DELETE | `/api/families/{id}`                   | also deletes its members, renumbers rest  |
| GET    | `/api/members[?familyId=]`             | with `familyName`, `age`, `ageText`, `category`, `elderly` |
| POST   | `/api/members`                         | `order` is assigned automatically         |
| GET    | `/api/members/{id}`                    |                                           |
| PUT    | `/api/members/{id}`                    | moving to another family appends it there |
| DELETE | `/api/members/{id}`                    | renumbers the family                      |

Dates are sent as `YYYY-MM-DDT00:00:00Z`; `baptized`/`confirmed` are `true`,
`false` or `null` (unknown).

## Importing

The importer reads families and members from the **Data Base Keluarga** sheet,
keterangan/domisili from **Daftar Keluarga**, and fills missing wedding dates
from **Daftar HUT Pernikahan**. The other sheets are derived views and are
recomputed by the app. Note the workbook's sheets disagree in places (some name
spellings and birth dates); Data Base Keluarga is treated as the source of truth.

## Project layout

```
main.go                 server, flags, SPA handler
internal/api            REST handlers (families, members)
internal/auth           sign-in, passkeys, sessions, user admin, middleware
internal/httpx          JSON helpers
internal/config         .env loader
internal/store          MongoDB access
internal/model          types (incl. User, Session) + derived values
internal/importer       .xlsx import
web/                    React app (web/embed.go embeds web/dist)
.env.example            settings template (copy to .env)
Dockerfile, compose.yaml, Makefile, .githooks/
```
