# Data Base Jemaat

Web app for managing the congregation data of GMIM Siloam, grouped by kolom,
replacing the `Data Base Jemaat Kolom 9 2027 - 2031.xlsx` workbook.

Families belong to a kolom (a new database starts with Kolom 1–12; signed-in
users add or remove koloms on **Kelola Kolom**). The sidebar picker chooses
what every page shows: all koloms, one kolom, or one BIPRA category (its
members, the families they are in, and the services of the jenis linked to
it), and new families and members must pick their
kolom first. Data from before koloms existed is moved to Kolom 11 at startup,
and `-import` loads the workbook into Kolom 11 unless you pass `-kolom N`.

**Jadwal Ibadah** is a Google-Calendar-style schedule (month grid or list)
of worship services per kolom. Each service can be added to Google Calendar
or Apple Calendar (`.ics`), and "Tambahkan ke Kalender Saya" subscribes a
calendar app to `/api/calendar.ics` (optionally `?kolom=N`), so changes show
up automatically. The kinds of service ("Jenis Ibadah": Kolom, WKI, PKB,
Pemuda, Remaja, ASM, with their colors) are managed on their own page; each
can be linked to a BIPRA category (WKI, PKB, Pemuda, Remaja and ASM are linked
by name once, automatically). A service of a linked jenis may have no kolom
("Semua kolom", the whole congregation): it shows in every kolom, in each
kolom's calendar feed and in the Android app. The
**Khadim** (who leads the service) and the **Pembawa Acara** are picked from
the members who are baptized and sidi (the service's kolom first), or typed in
as another name, e.g. a pastor from outside the congregation. On opening the
site, a popup shows today's and this week's services and today's birthdays
(unless there is nothing to show, or it was hidden for today); Beranda shows
the same. Both list them per kolom, always with the kolom's name ("Semua
kolom:" first, then "Kolom 1:", "Kolom 2:", …): today's and this week's
services together (today's with the time only, the others with the day), and
the birthdays as one line per kolom ("Kolom 11: Budi Umboh, Erwin Umboh").
Service times are in `APP_TIMEZONE` (default `Asia/Makassar`, WITA). Load a
schedule spreadsheet with
`siloam -import-jadwal jadwal.xlsx [-kolom 11] [-replace]` (the leader's
column may be headed "Pemimpin Ibadah" or "Khadim").

## Android app (`android/`)

A Kotlin (Jetpack Compose) app for the congregation: the **public view** only
(no sign-in, the same redacted data visitors see on the web), with a kolom
picker, an offline cache, and five tabs:

| Tab | What it shows |
|---|---|
| Beranda | Two collapsible sections: **Pemberitahuan** (today's and this week's services, today's birthdays and wedding anniversaries) and **Statistik** (totals, BIPRA counts, charts). Tapping a BIPRA count opens that category on the BIPRA tab |
| Jadwal | Services month by month, as a list or a month calendar (swipe to change month); details, "Tambah ke Kalender" and "Bagikan" per service |
| HUT | Birthdays month by month (day and month only) |
| Jemaat | Families and their members, searchable |
| BIPRA | Members per Kategorial BIPRA category, searchable |

It also sends **background notifications**:

- a daily summary (default 06:00): today's services and today's birthdays,
  per kolom ("Kolom 11:" then "- 19.00 · Ibadah Kolom · …"; birthdays as
  "Kolom 11: Budi Umboh, Erwin Umboh");
- a reminder before each of today's services (default 1 hour before).

The **filter data bawaan** (the top bar's picker, also under Pengaturan) is
*Semua Kolom*, a BIPRA category (its members, the families they are in, and
the services of the jenis linked to it, in every kolom) or a kolom (its data,
its services and the whole congregation's). Every screen and the
notifications follow it, and the app opens on it; it starts as the kolom
chosen before the filter existed. Beranda's Pemberitahuan lists the same per
kolom, also the next birthdays ("Kolom 11: Budi Umboh (besok), …").

A WorkManager job checks about every hour, also after a reboot, without the
app being open. The times,
reminders and server address are under **Pengaturan**. The server defaults to
`https://siloam.fly.dev`; debug builds also allow plain HTTP, e.g.
`http://10.0.2.2:8080` (your computer, seen from the Android emulator);
release builds only accept `https://` addresses.

Needs JDK 17+ and the Android SDK (`android/local.properties` →
`sdk.dir=…`, or `ANDROID_HOME`):

```sh
make android-test      # unit tests (dates, birthdays, statistics, calendar grid, JSON parsing)
make android-apk       # debug APK → android/app/build/outputs/apk/debug/app-debug.apk
make android-install   # install it on a phone/emulator via adb ("Siloam Debug")
make android-release   # release APK (signed when android/keystore.properties exists)
make android-install-release   # install the signed release APK ("Jemaat Siloam") beside the debug one
make android-publish VERSION=1.2.0   # set the version, signed APK into downloads/android/ (see below)
```

The debug build is its own app (`id.gmimsiloam.jemaat.debug`, "Siloam Debug",
version `x.y.z-debug`), so it and the release build can be installed together;
each has its own settings and sends its own notifications.

**New versions.** The app asks the server for the newest version
(`GET /api/app/android`) when it loads and in the hourly background check. If
it is newer than the installed one, the app shows a notification (once per
version) and a "Versi baru tersedia · Unduh" banner on Beranda and in
Pengaturan; *Unduh* opens the download link. To publish a version:

1. once: create the release key and `android/keystore.properties` as described
   in `android/keystore.properties.example` (both are git-ignored — keep them
   and their passwords safe and backed up: Android refuses an update signed
   with a different key);
2. `make android-publish VERSION=1.2.0` — sets the version (`versionName` in
   `android/app/build.gradle.kts`; `versionCode` follows from it, 1.2.0 →
   10200, and must grow: a lower version is refused; so the APK matches a
   commit, it also refuses while there are uncommitted changes), builds and signs the
   release APK and adds it as `downloads/android/jemaat-siloam-1.2.0.apk` (the
   only APKs git does not ignore); commit and push it with the changed
   `build.gradle.kts`;
3. an admin enters the version and the link it prints on **Aplikasi Android**
   (web sidebar), e.g. `1.0.1` and
   `https://raw.githubusercontent.com/michaelwp/siloam/main/downloads/android/jemaat-siloam-1.0.1.apk`
   (public: the repository is).

Until an admin saves a version there, the server's `ANDROID_LATEST_VERSION` /
`ANDROID_DOWNLOAD_URL` settings apply (e.g. in `fly.toml` `[env]`); the page
shows which one is in effect and can go back to the server settings. With
neither, the server reports no version and the app shows nothing.

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
- **Kategori BIPRA** – by default Anak (ASM) 0–11 · Remaja 12–16 · Pemuda
  17–30 (unmarried) · PKB (male) / WKI (female) 31+ or married · Lansia 60+ as
  an extra flag. Admins change these on **Kategori BIPRA**: each category has a
  code, a name, a colour and a rule (age from–to, gender, marital status, and
  "also everyone married, whatever their age"). A *kategori utama* is given to
  each member by the first matching rule in the list's order; a *tanda
  tambahan* (like Lansia) is added to everyone it matches. Members without a
  birth date have no category. Categories are computed on every request, so a
  change applies at once on the web and in the Android app (which reads
  `/api/bipra-categories`; older servers without it keep the defaults).

## Public view

The homepage (**Beranda**) and the six data pages can be viewed **without
signing in**, read-only (no add/edit/delete). Visitors only get non-private
information — the server removes the rest from the API responses, it is not
just hidden in the page:

| Visitors see | Only signed-in users see |
|---|---|
| member name, family, gender, BIPRA category (incl. Lansia — an age range) | birth year / full birth date, exact age, birth place |
| birthday as day + month | relationship in the family |
| marital status, baptism / sidi status (on all pages; filters on *Daftar Anggota Jemaat*) | domisili, wedding year, years married |
| family name, note (e.g. Simpatisan), number of members, anniversary as day + month | |

The homepage shows totals, BIPRA counts and this month's birthdays and wedding
anniversaries. **Masuk** (sidebar) opens `/login`; *Akun Saya* and *Pengguna*
require signing in. The site asks search engines not to index it (a
`noindex` meta tag on the page and an `X-Robots-Tag: noindex` header on every
response, including the API); remove both to allow indexing. Public API reads
are rate-limited like everything else.

Every families/members response says which view it is in an `X-Data-View`
header (`full` or `public`). The page shows private columns and edit controls
only when the data it holds is the full view, and reloads all data whenever the
user signs in or out, so redacted data can never be edited and saved back, and
private data never stays on screen after signing out. An expired session on a
public page gets its cookie cleared and the page returns to visitor mode.

## Sign-in & users

Admins also see **Kategori BIPRA** (the categories and their rules, see
Features), **Aplikasi Android** (the update notice above) and **Log
Aktivitas**: every change made in the app (add/edit/delete of koloms,
families, members, jenis ibadah, services, users, passkeys, BIPRA categories, the Android
version) and every sign-in and sign-out, with who, what (the item's name, also
after a delete), success or failure, the IP address and the time, newest first
and filterable by user, kind, result and date. Request contents are never
stored, so passwords cannot end up in the log; a failed sign-in records the
username only if that account exists (what was typed may be a password).
Entries are deleted after `AUDIT_RETENTION_DAYS` (default 365). Imports and
copies from the command line are not in the log. Requests refused before they
reach the app (not signed in, not an admin) are not logged either.

- **First run:** when there are no users, the server creates an admin account
  **`admin` / `admin`** (override with `INITIAL_ADMIN_USER` / `INITIAL_ADMIN_PASSWORD`).
  After signing in, the app shows only an *Perbarui Akun* screen until a new
  username and password are set.
- **Passkeys:** under *Akun Saya* each user can add passkeys (Touch ID / Face ID,
  Windows Hello, Android, security keys), rename or delete them, then use
  *Masuk dengan Passkey* on the sign-in page without typing a username.
- **Admin → Pengguna:** add users (role *Admin*, *Pengguna* or *Pengguna –
  kelola data kolom*, see below) with a generated
  temporary password, edit username/role, reset a password (optionally removing
  the user's passkeys, e.g. for a lost phone) and delete users. New and reset
  accounts must change their password at the next sign-in. You cannot delete or
  demote yourself, and there is always at least one admin.
- **Roles:**

  | Role | Can change | Sees private data of |
  |---|---|---|
  | Admin | all jemaat data, koloms, jenis ibadah, users, Android version, log | every kolom |
  | Pengguna | all jemaat data, koloms, jenis ibadah | every kolom |
  | Pengguna – kelola data kolom | families, members and services of **their one kolom** (chosen when the account is created; required) | their kolom only — the app shows only their kolom |
  | Pengguna – kelola data BIPRA | only **services** whose jenis ibadah is linked to **their one BIPRA category** (chosen when the account is created; required); the kolom is optional | their category's members only |

  The server enforces this (`403` otherwise, also for moving data into or out
  of the kolom); the web app shows a kolom user only their kolom (no kolom
  picker; services for the whole congregation, "Semua kolom", still show),
  offers only that kolom in forms and hides *Kelola Kolom* and *Jenis Ibadah*.
  A kolom assigned to a user cannot be deleted.

  A *kelola data BIPRA* user (e.g. for Pemuda) sees, on Beranda, Daftar HUT
  Jemaat and Daftar Anggota Jemaat, only their category's members (and on
  Beranda only their category's services and statistics); Daftar Keluarga,
  Data Base Keluarga, Daftar HUT Pernikahan and Kategorial BIPRA are hidden.
  Their Jadwal Ibadah form offers only their category's jenis, with
  "Semua kolom" as the default. A category assigned to a user cannot be
  deleted.
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

### Fly.io

`fly.toml` is ready for the app `siloam` (region `sin`, https://siloam.fly.dev).
It builds the Dockerfile, sets the passkey domain, `COOKIE_SECURE` and
`CLIENT_IP_HEADER=Fly-Client-IP` (so rate limiting sees real client IPs), and
health-checks `/api/health`, which fails when MongoDB is unreachable — a broken
deploy stops instead of going live. Machines stop when idle.

```sh
make fly-secrets   # MONGODB_URI (and INITIAL_ADMIN_PASSWORD if set) from .env → Fly secrets
make fly-deploy    # build and deploy
make fly-status    # machines + health checks
make fly-logs
```

**Atlas must allow Fly's IPs.** In Atlas: *Security → Network Access → IP Access
List → Add IP Address*. Either allow `0.0.0.0/0` (simplest; then a strong
database password is what protects it), or give the app fixed outgoing IPs with
`fly ips allocate-egress` (paid) and add exactly those. Without this the app
logs `tls: internal error` and keeps restarting.

Running more than one machine is fine: sessions, lockouts and passkey sign-in
state are all in MongoDB. Rate limits are per machine. With a custom domain,
change `WEBAUTHN_RP_ID` / `WEBAUTHN_ORIGINS` in `fly.toml` (existing passkeys
must then be registered again).

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

`make hooks` (also part of `make setup`) points git at `.githooks/`; every
clone needs it once. The `pre-commit` hook runs `lint-go` **and the Go tests**
(`make test`) when Go files are staged, and `lint-web` (oxlint + type check)
when files under `web/` are staged. It always prints what it did
(`pre-commit: …`); any failure blocks the commit. Tests that need MongoDB use a
local one if it is running and are skipped otherwise. It also works
from Git GUIs/IDEs, which often run hooks with a minimal `PATH`.

If you don't see `pre-commit:` lines when committing, check
`git config core.hooksPath` (should print `.githooks`) and that your Git client
isn't set to skip hooks (`--no-verify`, "bypass hooks" options). CI runs the
same checks regardless.

### Version

The web app shows the server's version at the foot of the sidebar. It is the
git tag the binary was built from (`git describe`: `v3.1.0`, or
`v3.1.0-4-gabc1234` for a build after that tag, `-dirty` with uncommitted
changes), stamped in by `make build`, the Docker / Apple container / Fly
builds (`--build-arg APP_VERSION`) and the CI deploy (the tag's name). A
plain `go build` shows `dev (abc1234)`.

### CI (GitHub Actions) and releases

`.github/workflows/ci.yml` runs **only when a version tag (`v*`) is pushed**, or
when started by hand (*Actions → CI → Run workflow*). Ordinary pushes and pull
requests do not trigger it; the pre-commit hook lints every commit locally.

| Job | What |
|-----|------|
| Go lint | golangci-lint (same version and config as locally) |
| Go tests | `go vet` + `go test -race`, with a MongoDB service so the store tests run |
| Web | `npm ci`, oxlint, TypeScript type check, production build |
| Docker | builds the image for `linux/amd64`, as Fly.io runs it |
| Deploy | after all checks pass, if the tagged commit is on `main`: `flyctl deploy` |

To release, from an up-to-date, clean `main`:

```sh
make ci                       # optional: run the same checks locally first
make release VERSION=v1.0.0   # creates and pushes the tag → CI → deploy
```

`make release` refuses unless you are on `main`, have no uncommitted changes,
are in sync with `origin/main`, and the tag is new.

To enable the deploy job, create a deploy token and store it as a repository
secret (without it the deploy job is skipped with a notice):

```sh
fly tokens create deploy -a siloam | gh secret set FLY_API_TOKEN -R michaelwp/siloam
```

Deploys use the `production` environment, so you can add required reviewers in
the repository's *Settings → Environments* to approve each release.

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
| `CLIENT_IP_HEADER`               | none — client-IP header set by a platform edge proxy (Fly.io: `Fly-Client-IP`) |
| `AUDIT_RETENTION_DAYS`           | `365` — how long *Log Aktivitas* keeps entries |
| `ANDROID_LATEST_VERSION` / `ANDROID_DOWNLOAD_URL` | none — the Android update notice until an admin sets it on *Aplikasi Android* |
| `-import file.xlsx [-replace]`   | import the workbook and exit              |
| `-copy-from <uri> [-collections families,members] [-replace]` | copy data from another MongoDB and exit |
| `-reset-user <name>`             | unlock an account, print a temporary password and exit |


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

Koloms, families, members, worship types, services, BIPRA categories and users (with passwords
and passkeys) are copied with their IDs, so links between them stay intact. Sessions are not copied — everyone
signs in again once. The command refuses a non-empty target unless `REPLACE=1`
(which clears only the collections being copied), and refuses to copy a
database onto itself — detected by writing a marker, so `localhost`,
`127.0.0.1` or a container name for the same server are all caught. The source
is only read.

## API

Reading koloms, families, members, worship types and services (`GET` on
`/api/koloms`, `/api/families…`, `/api/members…`, `/api/worship-types`,
`/api/services…`, `/api/calendar.ics` and `/api/bipra-categories`) and the Android update check are public; families and members come
in the redacted public view unless signed in. Every other
endpoint except `/api/health` and sign-in requires a session cookie, and
`/api/users` and `/api/admin/…` require the admin role. Errors are
`{"error": "...", "code": "..."}`; `code` is `unauthenticated` (401),
`password_change_required` (403), `forbidden` (403), `rate_limited` (429, with
`Retry-After`) or `account_locked` (423, only from passkey sign-in, where the
user has proven who they are).

| Method | Path                                   | Notes                                     |
|--------|----------------------------------------|-------------------------------------------|
| GET    | `/api/health`                          | `200` when MongoDB is reachable (public)  |
| GET    | `/api/version`                         | `{version}`, e.g. `v3.1.0` (public; shown at the foot of the sidebar) |
| GET    | `/api/app/android`                     | `{latestVersion, downloadUrl}` for the app's update check (public; empty when not configured) |
| GET    | `/api/admin/android`                   | admin: the release in effect, `source` (`database`/`env`/`none`) and the `ANDROID_*` values |
| PUT    | `/api/admin/android`                   | admin: `{latestVersion, downloadUrl}` (both empty = no update notice) |
| DELETE | `/api/admin/android`                   | admin: back to the `ANDROID_*` settings |
| GET    | `/api/admin/audit`                     | admin: log entries newest first; `?username=&category=&result=ok\|failed&from=&to=` (YYYY-MM-DD, `APP_TIMEZONE`), `before=<id>` for the next 50 |
| GET    | `/api/admin/audit/meta`                | admin: the categories and usernames for the filters |
| GET    | `/api/bipra-categories`                | the BIPRA categories and their rules, in order (public) |
| POST   | `/api/admin/bipra`                     | admin: `{key, name, kind (main\|flag), color, minAge, maxAge, gender, marital ("", married, unmarried), orMarried}` |
| PUT    | `/api/admin/bipra/{id}`                | admin: same fields                        |
| DELETE | `/api/admin/bipra/{id}`                | admin                                     |
| PUT    | `/api/admin/bipra/order`               | admin: `{ids}` — every category, in the new order (the priority of main categories) |
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
| POST   | `/api/users`                           | admin: `{username, password, role, kolom, bipraId}` — role `admin`, `user`, `kolom` (then `kolom` is required) or `bipra` (then `bipraId`, a BIPRA category's id, is required) |
| PUT    | `/api/users/{id}`                      | admin: `{username, role, kolom, bipraId}` |
| POST   | `/api/users/{id}/reset-password`       | admin: `{password, clearPasskeys}`; also unlocks |
| DELETE | `/api/users/{id}`                      | admin                                     |
| GET    | `/api/koloms`                          | public                                    |
| POST   | `/api/koloms`                          | `{number}`                                |
| DELETE | `/api/koloms/{id}`                     | only when no family or service is in it   |
| GET    | `/api/families`                        | with `memberCount`, `weddingYears`, `weddingMonth`, `weddingDay` (public: redacted) |
| POST   | `/api/families`                        | `no` is assigned automatically            |
| GET    | `/api/families/{id}`                   | includes `members`                        |
| PUT    | `/api/families/{id}`                   |                                           |
| DELETE | `/api/families/{id}`                   | also deletes its members, renumbers rest  |
| GET    | `/api/members[?familyId=]`             | with `familyName`, `age`, `ageText`, `category` (main BIPRA key), `flags` (BIPRA flags), `elderly` (= the Lansia flag), `birthMonth`, `birthDay` (public: redacted) |
| POST   | `/api/members`                         | `order` is assigned automatically         |
| GET    | `/api/members/{id}`                    |                                           |
| PUT    | `/api/members/{id}`                    | moving to another family appends it there |
| DELETE | `/api/members/{id}`                    | renumbers the family                      |
| GET    | `/api/worship-types`                   | jenis ibadah with `color` and `bipraId` (the linked BIPRA category, or null) (public) |
| POST   | `/api/worship-types`                   | `{name, color, bipraId}`                  |
| PUT    | `/api/worship-types/{id}`              | `{name, color, bipraId}`                  |
| DELETE | `/api/worship-types/{id}`              | only when no service uses it              |
| GET    | `/api/services`                        | with `typeName`, `typeColor`, `start`, `end`; `leader` (khadim) and `mc` show the linked member's current name (public) |
| POST   | `/api/services`                        | `leaderId` / `mcId` link a member, or give `leader` / `mc` as a name; `kolom` 0 (the whole congregation) only for a jenis linked to a BIPRA category |
| PUT    | `/api/services/{id}`                   |                                           |
| DELETE | `/api/services/{id}`                   |                                           |
| GET    | `/api/services/{id}/ics`               | one service as an `.ics` file (public)    |
| GET    | `/api/calendar.ics[?kolom=&type=]`     | calendar feed to subscribe to (public)    |

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
internal/api            REST handlers (koloms, families, members, schedule)
internal/auth           sign-in, passkeys, sessions, user admin, middleware
internal/audit          audit log middleware (user activities)
internal/httpx          JSON helpers
internal/config         .env loader
internal/store          MongoDB access
internal/model          types (incl. User, Session) + derived values
internal/importer       .xlsx import (members workbook, worship schedule)
internal/ics            iCalendar (.ics) output for the schedule
web/                    React app (web/embed.go embeds web/dist)
android/                Android app (Kotlin, Jetpack Compose; public view + notifications)
.env.example            settings template (copy to .env)
Dockerfile, compose.yaml, Makefile, .githooks/
```
