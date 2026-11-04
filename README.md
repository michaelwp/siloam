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
calendar app to the user's **personal feed link**
(`/api/calendar.ics?token=…`, optionally `&kolom=N`; calendar apps cannot
sign in, so the secret token is the key — "Buat link baru" replaces it and the
old link stops working), with the services of that user's data, so changes
show up automatically. The kinds of service ("Jenis Ibadah": Kolom, WKI, PKB,
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
**Times.** Every moment the app stores (sign-ins, sessions, locks, the
activity log, created/updated times) is saved in UTC (GMT+0), and the web and
Android apps show it in the viewer's own time zone (the browser's or the
phone's); the admin pages' date filters mean the viewer's local day. Calendar
dates — birth dates, wedding dates, a service's date — are days, not moments,
and are shown as they are. Service times are the church's local time,
`APP_TIMEZONE` (default `Asia/Makassar`, WITA), as entered (calendar apps
convert the `.ics` events to the device's zone). Load a
schedule spreadsheet with
`siloam -import-jadwal jadwal.xlsx [-kolom 11] [-replace]` (the leader's
column may be headed "Pemimpin Ibadah" or "Khadim").

## Android app (`android/`)

A Kotlin (Jetpack Compose) app for the congregation. **Not signed in** it
shows only the whole congregation's statistics summary (`/api/stats`: counts
and charts, no names, no filters) with a **Masuk** button that opens the
sign-in form, and sends no notifications. The app's version is shown at the
foot of Beranda and of the sign-in pages. **Signed in** (username + password, the same accounts as the
web; a new or reset account first sets its own username and password on a
*Perbarui Akun* screen, as on the web, with the same password rules shown
live) it
shows that user's data, with a kolom picker, an offline cache, notifications,
and five tabs. *Pengaturan → Keluar* signs out and deletes the data from the
phone; when the session ends (expired, or an admin used *Keluarkan*) the app
returns to the visitor view. The app follows the user's role like the web: a
*kelola kolom/BIPRA* user or a viewer with choices sees only their koloms'
data plus their categories' members and services in every kolom. As on the
web, the top bar has two separate pickers for what the screens show: **Kolom**
(*Semua Data Saya* and their koloms, every kolom when they have categories;
fixed for a single kolom) and **BIPRA** (every category inside one of their own
koloms, only their own categories — or *Semua BIPRA Saya* — elsewhere). The
**notifications** have their own filter (*Pengaturan → Filter notifikasi*,
limited the same way) and by default cover all of the user's data, whatever
the screens show: e.g. Kolom 11 + Remaja + Pemuda is notified about Kolom 11
and about Remaja and Pemuda in every kolom. A new sign-in resets both to all
of the user's data. A BIPRA-only user has no BIPRA tab and no wedding
anniversaries. The role is read again on every refresh, so an admin's change
applies without signing in again. Sign-ins from the app are counted as *Android* in
the admins' user statistics (header `X-Client: android`).

Older app versions (without sign-in) can no longer load data from this
server; they still get the update notice, so publish the new APK on
*Aplikasi Android* when deploying.

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

The top bar's **Kolom** and **BIPRA** pickers choose what the screens show
(a kolom: its data, its services and the whole congregation's; a category:
its members, the families they are in, and the services of the jenis linked
to it), each on its own so both can apply. The **notifications** follow their
own filter, *Pengaturan → Filter notifikasi* (default: all of the user's
data). Beranda's Pemberitahuan lists what the screens show per kolom, also
the next birthdays ("Kolom 11: Budi Umboh (besok), …").

A WorkManager job checks about every hour, also after a reboot, without the
app being open. The times and
reminders are under **Pengaturan**. The server address is not shown anywhere
in a release build: it is fixed when the app is built (`https://siloam.fly.dev`).
A **debug build** talks to `http://localhost:8080` by default (the local
server, `make run`; run `adb reverse tcp:8080 tcp:8080` so the emulator or a
USB phone reaches your computer's port), and has a **Server (versi debug)**
card on the sign-in page and in *Pengaturan*: pick localhost, `10.0.2.2:8080`
(your computer from the emulator, without adb reverse), the production server,
or type another address. Switching signs out and forgets the device's data.
`./gradlew -PsiloamServer=http://… assembleDebug` changes the debug default
(debug builds allow plain HTTP). A phone that
had another address saved by an older version is signed out once after the
update, with a message, and signs in again on the built-in server.

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

## Visitors (not signed in)

**All jemaat data needs a sign-in.** A visitor sees only **Beranda** with the
whole congregation's statistics summary — totals (families, members, gender,
baptism / sidi), marital statuses, BIPRA counts and the charts — from
`GET /api/stats`, which returns counts only (no names, no member or family
data). Visitors get no kolom / BIPRA picker, no notifications and no other
page: every other page goes to `/login` (**Masuk** in the sidebar) and comes
back after signing in. The site asks search engines not to index it (a
`noindex` meta tag on the page and an `X-Robots-Tag: noindex` header on every
response, including the API); remove both to allow indexing.

For signed-in users, families/members responses still say which view they are
in an `X-Data-View` header, and a limited user (*kelola kolom/BIPRA*, a
viewer with choices) gets the data outside their scope redacted by the
server. When a session ends (expired, signed out elsewhere, or *Keluarkan* by
an admin) the next request answers `401` and the page returns to the visitor
view.

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
The log is kept in files, not in MongoDB: one file per day in `AUDIT_LOG_DIR`
(`audit-2026-10-25.jsonl`, one JSON object per line, by UTC day),
written in the background so a request never waits for it. Day files older
than `AUDIT_RETENTION_DAYS` (default 90, about 3 months) are deleted. On a
host whose disk is wiped on restart (Fly.io without a volume) the log only
goes back to the last start. Imports and
copies from the command line are not in the log. Requests refused before they
reach the app (not signed in, not an admin) are not logged either.

- **First run:** when there are no users, the server creates an admin account
  **`admin` / `admin`** (override with `INITIAL_ADMIN_USER` / `INITIAL_ADMIN_PASSWORD`).
  After signing in, the app shows only an *Perbarui Akun* screen until a new
  username and password are set.
- **Passkeys:** under *Akun Saya* each user can add passkeys (Touch ID / Face ID,
  Windows Hello, Android, security keys), rename or delete them, then use
  *Masuk dengan Passkey* on the sign-in page without typing a username.
- **Admin → Pengguna:** add users (role *Admin*, *Pengguna*, *Pengguna –
  kelola kolom/BIPRA*, *Pengguna – hanya lihat* or *Bendahara*, see below) with a generated
  temporary password, edit username/role, reset a password (optionally removing
  the user's passkeys, e.g. for a lost phone) and delete users. New and reset
  accounts must change their password at the next sign-in. You cannot delete or
  demote yourself, and there is always at least one admin. The list shows each
  user's last sign-in (date, time, and web or Android). **Keluarkan** signs a
  user out on every device (web and Android) and requires a reason, which the
  *Log Aktivitas* keeps ("Keluarkan pengguna", *Alasan: …*); they can sign in
  again (reset the password or delete the account to stop that).
- **Statistik Pengguna** (on *Pengguna*): for a chosen day (default today) how
  many users signed in, how many sign-ins came from the web and from Android,
  the list of sign-ins (time, username, role, source), and who is signed in
  now (per device type, with *Keluarkan*). The day is the viewer's local day,
  and the sign-in log is kept (no expiry).
- **Roles:**

  | Role | Can change | Sees private data of |
  |---|---|---|
  | Admin | all jemaat data, koloms, jenis ibadah, users, Android version, log | every kolom |
  | Pengguna | all jemaat data, koloms, jenis ibadah | every kolom |
  | Pengguna – kelola kolom/BIPRA | **their koloms** (any number): families, members and **every service** there, BIPRA ones too. **Their BIPRA categories** (any number): the services whose jenis ibadah is linked to one of them, **in every kolom** (or "Semua kolom"). Koloms and categories may be combined (they add up); at least one in all | their koloms, and their categories' members in every kolom |
  | Pengguna – hanya lihat | nothing (only their own username, password and passkeys) | with nothing chosen: every kolom; otherwise their koloms and their categories' members in every kolom (combined like above) |
  | Bendahara | **the kas (cash books) of their koloms and/or their BIPRA categories** (at least one in all): cash in and cash out; no other jemaat data | their koloms, and their categories' members in every kolom |

  The server enforces this (`403` otherwise, also for moving data into or out
  of their koloms). In the web app a limited user (*kelola kolom/BIPRA* or a
  viewer with choices) sees only their data: their koloms' families, members
  and services (and the whole congregation's services), plus their
  categories' members and services in every kolom. The notifications
  (services, birthdays) follow the same data. The sidebar offers *Semua Data
  Saya* and their koloms (every kolom when they have categories, to see their
  categories' members there); a user with a single kolom and no categories has
  it fixed. The BIPRA picker offers every category while one of their own
  koloms is picked, and only their own categories in any other kolom or in
  *Semua Data Saya* (e.g. Kolom 11 + Remaja + Pemuda: in Kolom 11 any
  category, elsewhere Remaja or Pemuda). Families and members can be changed while viewing one of their
  koloms (or *Semua Data Saya* without categories); services wherever they
  manage them. Forms offer only their koloms (every kolom for a jenis linked
  to one of their categories), and *Kelola Kolom* and *Jenis Ibadah* are
  hidden. A kolom or category assigned to a user cannot be deleted.

  A user with **only** BIPRA categories (e.g. Pemuda) has no family pages
  (Daftar Keluarga, Data Base Keluarga, Daftar HUT Pernikahan and Kategorial
  BIPRA are hidden), the BIPRA picker offers only their categories, and their
  Jadwal Ibadah form offers only their categories' jenis.

  A *hanya lihat* (viewer) user sees the pages like a signed-in user, but
  without any add, edit or delete buttons; the server refuses every change to
  jemaat data (`403`).

  Users of the former one-kolom (`kolom`) and one-category (`bipra`) roles are
  moved to this role at startup, keeping their kolom or category.
- **Kas** (sidebar), two tabs. **Kas Kolom**: each kolom's cash book; it
  follows the sidebar's kolom picker. **Kas BIPRA**: each BIPRA category's
  cash book, for the whole congregation (e.g. Pemuda's); it follows the
  sidebar's BIPRA picker. With nothing picked, the tab lists every book's cash
  in, cash out and balance; picking one opens its book for a
  month (or *Semua bulan*): opening balance, cash in, cash out, closing
  balance, and every entry with its running balance. *Unduh Excel* saves it as
  an .xlsx file (real dates and amounts, the opening balance and the period's
  totals; written by the page itself, no library) and *Unduh CSV* as CSV.
  An entry is **kas masuk** or **kas keluar**: date, amount (whole rupiah) and
  either the **ibadah** it came from (e.g. its offering) or was spent for — of
  that kolom, or for a BIPRA book of a jenis linked to the category, in any
  kolom — or another source/use told in the **keterangan** (required then).
  The entry keeps the ibadah's name, so it still shows if the schedule entry
  is deleted. **Admins** and the book's **Bendahara** add, change and delete
  entries (each change is in *Log Aktivitas*, category *Kas*). A kolom's kas
  is read by who sees all of that kolom's data (Pengguna, its *kelola kolom*
  users, viewers of it); a category's by Pengguna, viewers of everything, and
  the users who have that category. A kolom or category with kas entries
  cannot be deleted. BIPRA-only users see only the Kas BIPRA tab.
- **Kas on Beranda** (web: a *Kas Kolom* and a *Kas BIPRA* card) and on the
  Android app's Beranda (*💰 Kas*): for each book the user may read, the
  balance and this month's cash in and cash out (the viewer's month) of the
  kolom and category picked, or of all of them with a line per book. The Android app shows these totals only
  (cached for offline use); entries are recorded and listed on the web.
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
| `AUDIT_RETENTION_DAYS`           | `90` — how long *Log Aktivitas* keeps entries |
| `AUDIT_LOG_DIR`                  | `logs/audit` (image: `/home/nonroot/audit`) — where *Log Aktivitas* is kept, one file per day |
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

Only the statistics summary (`GET /api/stats`), the Android update check,
`/api/health`, `/api/version`, sign-in, and the calendar feed (which checks
its own `token`) are public. Every other endpoint — every read of koloms,
families, members, worship types, services and BIPRA categories too —
requires a session cookie, and
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
| GET    | `/api/stats`                           | the whole congregation's summary for visitors: `{families, members, male, female, baptized, confirmed, maritalStatuses, categories (with count, male, female), birthMonths, familySizes}` — counts only (public) |
| GET    | `/api/admin/android`                   | admin: the release in effect, `source` (`database`/`env`/`none`) and the `ANDROID_*` values |
| PUT    | `/api/admin/android`                   | admin: `{latestVersion, downloadUrl}` (both empty = no update notice) |
| DELETE | `/api/admin/android`                   | admin: back to the `ANDROID_*` settings |
| GET    | `/api/admin/audit`                     | admin: log entries newest first; `?username=&category=&result=ok\|failed&from=&to=` (RFC 3339 instants — the page sends the viewer's local day; `to` exclusive — or YYYY-MM-DD in `APP_TIMEZONE`), `before=<id>` for the next 50 |
| GET    | `/api/admin/audit/meta`                | admin: the categories and usernames for the filters |
| GET    | `/api/admin/logins?from=&to=`          | admin: user statistics for a period (RFC 3339 instants, `to` exclusive, at most 48 h — the page sends the viewer's local day; or `?date=YYYY-MM-DD` in `APP_TIMEZONE`, default today): `{from, to, users, web, android, logins: [{username, role, koloms, bipraIds, source, at}], active: [who is signed in now, per source]}`; the sign-in log is kept (no expiry) |
| GET    | `/api/bipra-categories`                | the BIPRA categories and their rules, in order (public) |
| GET    | `/api/kas?kolom=` or `?bipra=`         | a kolom's or BIPRA category's cash book: `{kolom, bipraId, name, in, out, balance, entries (oldest first), canEdit}` — for who may read it |
| GET    | `/api/kas/summary?month=&book=`        | `[{kolom, in, out, count, last, month, monthIn, monthOut}]` for every kolom whose kas the user may read, or with `book=bipra` every BIPRA category's (`bipraId`, `name`); `month` YYYY-MM (the viewer's; default this month in `APP_TIMEZONE`) |
| POST   | `/api/kas`                             | admin or the book's treasurer: `{kolom` or `bipraId, kind (in\|out), date, amount (rupiah > 0), serviceId ("", an ibadah of that kolom, or of a jenis linked to the category), description (required without serviceId)}` |
| PUT    | `/api/kas/{id}`                        | same fields (also the treasurer of the book it was in) |
| DELETE | `/api/kas/{id}`                        | admin or the book's treasurer |
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
| POST   | `/api/users`                           | admin: `{username, password, role, koloms, bipraIds}` — role `admin`, `user`, `scoped` (then `koloms`, kolom numbers, and/or `bipraIds`, BIPRA category ids, at least one in all), `viewer` (`koloms`/`bipraIds` optional; none: sees everything) or `treasurer` (`koloms` and/or `bipraIds`, at least one in all) |
| PUT    | `/api/users/{id}`                      | admin: `{username, role, koloms, bipraIds}` |
| POST   | `/api/users/{id}/reset-password`       | admin: `{password, clearPasskeys}`; also unlocks |
| POST   | `/api/users/{id}/logout`               | admin: `{reason}` (required, 3–200 characters; kept in the activity log) — ends all of the user's sessions, web and Android |
| POST   | `/api/auth/calendar-token`             | `{reset?}` → `{token}`: the secret of the user's calendar feed link (created on first use; `reset` replaces it) |
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
| GET    | `/api/calendar.ics?token=[&kolom=&type=]` | the user's calendar feed: the services of their data (public, but needs the secret `token` of their link) |

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
internal/audit          audit log middleware and its day files (user activities)
internal/httpx          JSON helpers
internal/config         .env loader
internal/store          MongoDB access
internal/model          types (incl. User, Session) + derived values
internal/importer       .xlsx import (members workbook, worship schedule)
internal/ics            iCalendar (.ics) output for the schedule
web/                    React app (web/embed.go embeds web/dist)
android/                Android app (Kotlin, Jetpack Compose; sign-in, data + notifications; statistics for visitors)
.env.example            settings template (copy to .env)
Dockerfile, compose.yaml, Makefile, .githooks/
```
