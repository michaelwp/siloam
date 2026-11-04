import { useEffect, useMemo, useState, type CSSProperties } from 'react'
import type { Member, Service } from '../api'
import { Link } from 'react-router-dom'
import { StatsView } from '../components/StatsView'
import { BirthdaysByKolom, ByKolom, PageHeader, Status } from '../components/ui'
import { useAuth, useBipraScope } from '../auth'
import { useData } from '../data'
import { computeStats, statsApi, type Stats } from '../stats'
import { byText, dayMonth, daysUntil, MONTHS, parts } from '../format'
import { clock, serviceDayKey, serviceTitle, shortDay, upcomingServices, useSchedule, WEEK_AHEAD_DAYS } from '../schedule'

/**
 * Homepage. Signed in: today's services and birthdays, the statistics of the
 * viewed data, and this month's birthdays and wedding anniversaries.
 * Visitors: only the whole congregation's statistics summary.
 */
export default function Beranda() {
  const { user } = useAuth()
  return user ? <SignedInBeranda /> : <PublicBeranda />
}

/** Visitors: the statistics summary only (no names, no filters, no notifications). */
function PublicBeranda() {
  const [stats, setStats] = useState<Stats | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [month] = useState(() => new Date().getMonth())
  useEffect(() => {
    statsApi.get().then(setStats, (e: Error) => setError(e.message))
  }, [])
  return (
    <>
      <PageHeader title="Beranda" subtitle="GMIM Siloam · Statistik jemaat" />
      <p className="card public-note">
        Data jemaat hanya dapat dilihat setelah masuk.{' '}
        <Link to="/login" state={{ from: '/' }}>
          Masuk
        </Link>
      </p>
      <Status loading={!stats && !error} error={error} />
      {stats && <StatsView stats={stats} month={month} />}
    </>
  )
}

function SignedInBeranda() {
  const { families, members, categories: allCategories, filterLabel, bipraFilter, loading, error } = useData()
  // a category chosen in the filter, or a BIPRA-only user's (scope): members,
  // families and statistics of those categories only
  const scope = useBipraScope()
  const shown = bipraFilter ? [bipraFilter] : scope
  const categories = shown ?? allCategories
  const [today] = useState(() => new Date())
  const month = today.getMonth()
  const day = today.getDate()

  const stats = useMemo(() => computeStats(families, members, categories), [families, members, categories])

  // month/day are available to visitors; the year (age, years married) only when signed in
  const year = today.getFullYear()
  const birthdays = members
    .filter((m) => m.birthMonth === month + 1)
    .map((m) => ({
      m,
      d: m.birthDay!,
      turning: m.birthDate ? year - parts(m.birthDate).y : null,
    }))
    .toSorted((a, b) => a.d - b.d || byText(a.m.name, b.m.name))
  const anniversaries = families
    .filter((f) => f.weddingMonth === month + 1)
    .map((f) => ({
      f,
      d: f.weddingDay!,
      years: f.weddingDate ? year - parts(f.weddingDate).y : null,
    }))
    .toSorted((a, b) => a.d - b.d)

  return (
    <>
      <PageHeader title="Beranda" subtitle={`GMIM Siloam · ${filterLabel}`} />
      <Status loading={loading} error={error} />

      {!loading && !error && (
        <>
          <div className="today-grid">
            <TodayServices today={today} />
            <TodayBirthdays members={members} today={today} />
          </div>

          <StatsView
            stats={stats}
            month={month}
            links={{
              families: scope ? '/anggota' : '/keluarga',
              members: '/anggota',
              gender: (g) => `/anggota?gender=${encodeURIComponent(g).replaceAll('%20', '+')}`,
              sacrament: (field) => `/anggota?${field}=yes`,
              status: (label) => `/anggota?status=${encodeURIComponent(label)}`,
              // a BIPRA-only user has no BIPRA page; Anggota is already their category
              category: (key) => (scope ? '/anggota' : `/bipra?kategori=${encodeURIComponent(key)}`),
            }}
          />

          <div className="home-grid">
            <section className="card">
              <header className="family-head">
                <h2>Ulang Tahun — {MONTHS[month]}</h2>
                <Link to="/hut-jemaat" className="small">
                  Semua bulan →
                </Link>
              </header>
              {birthdays.length ? (
                <ul className="celebrations">
                  {birthdays.map(({ m, d, turning }) => (
                    <li key={m.id} className={d === day ? 'today' : d < day ? 'past' : ''}>
                      <span className="cal-day">{d}</span>
                      <div className="grow">
                        <strong>{m.name}</strong>
                        <span className="muted small">
                          Kel. {m.familyName}
                          {turning !== null && ` · ${turning} tahun`}
                        </span>
                      </div>
                      {d === day && <span className="badge green">Hari ini</span>}
                    </li>
                  ))}
                </ul>
              ) : (
                <p className="status">Tidak ada ulang tahun bulan ini.</p>
              )}
            </section>

            {!scope && (
              <section className="card">
                <header className="family-head">
                  <h2>HUT Pernikahan — {MONTHS[month]}</h2>
                  <Link to="/hut-pernikahan" className="small">
                    Semua →
                  </Link>
                </header>
                {anniversaries.length ? (
                  <ul className="celebrations">
                    {anniversaries.map(({ f, d, years }) => (
                      <li key={f.id} className={d === day ? 'today' : d < day ? 'past' : ''}>
                        <span className="cal-day">{d}</span>
                        <div className="grow">
                          <strong>Kel. {f.name}</strong>
                          {years !== null && <span className="muted small">{years} tahun pernikahan</span>}
                        </div>
                        {d === day && <span className="badge green">Hari ini</span>}
                      </li>
                    ))}
                  </ul>
                ) : (
                  <p className="status">Tidak ada HUT pernikahan bulan ini.</p>
                )}
              </section>
            )}
          </div>
        </>
      )}
    </>
  )
}

const UPCOMING_DAYS = 7

/** One service line in the Beranda panel. */
function ServiceItem({ s, withDay }: { s: Service; withDay: boolean }) {
  return (
    <li style={{ '--type-color': s.typeColor } as CSSProperties}>
      <span className="service-when">
        {withDay && <span className="muted">{shortDay(serviceDayKey(s))} · </span>}
        {clock(s.time)} · Ibadah {s.typeName}
      </span>
      <strong>{serviceTitle(s)}</strong>
      {(s.leader || s.mc) && (
        <span className="muted">
          {s.leader && <>Khadim: {s.leader}</>}
          {s.leader && s.mc && ' · '}
          {s.mc && <>Pembawa acara: {s.mc}</>}
        </span>
      )}
    </li>
  )
}

/** Today's services and the coming week's (of the viewed kolom), per kolom, beside the birthdays. */
function TodayServices({ today }: { today: Date }) {
  const { homeServices: services, loading, error } = useSchedule()
  const { today: todays, week, next } = upcomingServices(services, today, WEEK_AHEAD_DAYS)
  const dateText = `${today.getDate()} ${MONTHS[today.getMonth()]} ${today.getFullYear()}`

  return (
    <section
      className={`card today-card services-card ${todays.length ? 'has-services' : ''}`}
      aria-labelledby="services-title"
    >
      <header className="today-head">
        <span className="today-icon" aria-hidden="true">
          ⛪
        </span>
        <div className="grow">
          <h2 id="services-title">Ibadah Hari Ini & Minggu Ini</h2>
          <p className="muted small">{dateText}</p>
        </div>
        <Link to="/jadwal" className="btn small">
          Jadwal
        </Link>
      </header>

      {loading ? (
        <p className="today-empty muted">Memuat jadwal…</p>
      ) : error ? (
        <p className="today-empty form-error">Jadwal ibadah gagal dimuat: {error}</p>
      ) : (
        <>
          {todays.length === 0 && <p className="today-empty">Tidak ada ibadah hari ini.</p>}
          {/* today's and this week's together, per kolom; today's show the time only */}
          <ByKolom items={[...todays, ...week]} kolomOf={(s) => s.kolom}>
            {(list) => (
              <ul className="service-list">
                {list.map((s) => (
                  <ServiceItem key={s.id} s={s} withDay={!todays.includes(s)} />
                ))}
              </ul>
            )}
          </ByKolom>
          {week.length === 0 && next && (
            <p className="muted small">
              Berikutnya: <strong>{serviceTitle(next)}</strong> ({shortDay(serviceDayKey(next))}, {clock(next.time)})
            </p>
          )}
        </>
      )}
    </section>
  )
}

/**
 * The homepage's main information: who has a birthday today. When nobody does,
 * the next birthdays within a week. Today's are listed per kolom.
 */
function TodayBirthdays({ members, today }: { members: Member[]; today: Date }) {
  const dated = members
    .filter((m) => m.birthMonth && m.birthDay)
    .map((m) => ({
      m,
      days: daysUntil(m.birthMonth!, m.birthDay!, today),
    }))
  const todays = dated.filter((x) => x.days === 0).toSorted((a, b) => byText(a.m.name, b.m.name))
  const upcoming = dated
    .filter((x) => x.days > 0 && x.days <= UPCOMING_DAYS)
    .toSorted((a, b) => a.days - b.days || byText(a.m.name, b.m.name))
  const dateText = `${today.getDate()} ${MONTHS[today.getMonth()]} ${today.getFullYear()}`

  return (
    <section className={`card today-card ${todays.length ? 'has-birthdays' : ''}`} aria-labelledby="today-title">
      <header className="today-head">
        <span className="today-icon" aria-hidden="true">
          🎂
        </span>
        <div>
          <h2 id="today-title">Ulang Tahun Hari Ini</h2>
          <p className="muted small">{dateText}</p>
        </div>
      </header>

      {todays.length > 0 ? (
        <BirthdaysByKolom members={todays.map(({ m }) => m)} />
      ) : (
        <div className="today-empty">
          <p>Tidak ada anggota jemaat yang berulang tahun hari ini.</p>
          {upcoming.length > 0 && (
            <div className="next-birthdays">
              <span className="muted small">Berikutnya:</span>
              {/* per kolom, one line each: "Kolom 11: Budi Umboh (besok), Sari Kowaas (25 Oktober)" */}
              <ByKolom items={upcoming} kolomOf={({ m }) => m.kolom}>
                {(list) => (
                  <p className="kolom-names">
                    {list.map(({ m, days }, i) => (
                      <span key={m.id}>
                        {i > 0 && ', '}
                        {m.name}{' '}
                        <span className="muted small">({days === 1 ? 'besok' : dayMonth(m.birthDay, m.birthMonth)})</span>
                      </span>
                    ))}
                  </p>
                )}
              </ByKolom>
            </div>
          )}
        </div>
      )}
    </section>
  )
}
