import { useState, type CSSProperties } from 'react'
import type { Member, Service } from '../api'
import { Link } from 'react-router-dom'
import { ChartCard, ColumnChart, StackedBarChart } from '../components/charts'
import { BirthdaysByKolom, ByKolom, PageHeader, Status } from '../components/ui'
import { useBipraScope } from '../auth'
import { useData } from '../data'
import { bipraRange, byText, dayMonth, daysUntil, inBipra, MARITAL_STATUSES, MONTHS, parts } from '../format'
import { clock, serviceDayKey, serviceTitle, shortDay, upcomingServices, useSchedule, WEEK_AHEAD_DAYS } from '../schedule'

/** Public homepage: totals plus this month's birthdays and wedding anniversaries. */
export default function Beranda() {
  const { families, members, categories: allCategories, filterLabel, bipraFilter, loading, error } = useData()
  // a "kelola data BIPRA" user (scope), or a category chosen in the filter:
  // members, families and statistics of that category only
  const scope = useBipraScope()
  const shown = scope ?? bipraFilter
  const categories = shown ? [shown] : allCategories
  const [today] = useState(() => new Date())
  const month = today.getMonth()
  const day = today.getDate()

  const count = (pred: (m: (typeof members)[number]) => boolean) => members.filter(pred).length
  // baptized / confirmed (sidi): those who are, with the not-yet and unknown below
  const sacrament = (label: string, field: 'baptized' | 'confirmed') => {
    const unknown = count((m) => m[field] === null)
    return {
      label,
      value: count((m) => m[field] === true),
      to: `/anggota?${field}=yes`,
      note: `${count((m) => m[field] === false)} belum` + (unknown ? ` · ${unknown} tanpa data` : ''),
    }
  }
  const tiles: { label: string; value: number; to: string; note?: string }[] = [
    { label: 'Keluarga', value: families.length, to: scope ? '/anggota' : '/keluarga' },
    { label: 'Jiwa', value: members.length, to: '/anggota' },
    {
      label: 'Laki-laki',
      value: count((m) => m.gender === 'Laki Laki'),
      to: '/anggota?gender=Laki+Laki',
    },
    {
      label: 'Perempuan',
      value: count((m) => m.gender === 'Perempuan'),
      to: '/anggota?gender=Perempuan',
    },
    sacrament('Baptis', 'baptized'),
    sacrament('Sidi', 'confirmed'),
  ]
  // marital statuses: the known ones, then any other the data has
  const statuses = [
    ...MARITAL_STATUSES,
    ...new Set(members.map((m) => m.maritalStatus).filter((s) => s && !MARITAL_STATUSES.includes(s))),
  ].map((s) => ({ label: s, value: count((m) => m.maritalStatus === s) }))
  const bipra = categories.map((c) => ({
    key: c.key,
    label: c.name,
    range: bipraRange(c),
    value: count((m) => inBipra(m, c)),
  }))

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

          <div className="stats">
            {tiles.map((t) => (
              <Link key={t.label} to={t.to} className="stat">
                <span className="stat-value">{t.value}</span>
                <span className="stat-label">{t.label}</span>
                {t.note && <span className="stat-range">{t.note}</span>}
              </Link>
            ))}
          </div>

          <h2 className="section-title">Status</h2>
          <div className="stats">
            {statuses.map((s) => (
              <Link key={s.label} to={`/anggota?status=${encodeURIComponent(s.label)}`} className="stat">
                <span className="stat-value">{s.value}</span>
                <span className="stat-label">{s.label}</span>
              </Link>
            ))}
          </div>

          <h2 className="section-title">Kategorial BIPRA</h2>
          <div className="stats">
            {bipra.map((c) => (
              <Link
                key={c.label}
                // a "kelola data BIPRA" user has no BIPRA page; Anggota is already their category
                to={scope ? '/anggota' : `/bipra?kategori=${encodeURIComponent(c.key)}`}
                className="stat"
              >
                <span className="stat-value">{c.value}</span>
                <span className="stat-label">{c.label}</span>
                <span className="stat-range">{c.range}</span>
              </Link>
            ))}
          </div>

          <h2 className="section-title">Statistik</h2>
          <div className="chart-grid">
            <ChartCard
              title="Ulang tahun per bulan"
              subtitle={`Jumlah jiwa yang berulang tahun tiap bulan · ${MONTHS[month]} disorot`}
            >
              <ColumnChart
                data={MONTHS.map((name, i) => ({
                  label: name,
                  short: name.slice(0, 3),
                  value: members.filter((m) => m.birthMonth === i + 1).length,
                  emphasis: i === month,
                }))}
                unit="jiwa"
                labelHead="Bulan"
                emphasize
              />
            </ChartCard>
            <ChartCard title="Kategorial BIPRA per jenis kelamin" subtitle="Jumlah jiwa per kategori">
              <StackedBarChart
                rowHead="Kategori"
                series={[
                  { name: 'Laki-laki', cls: 'viz-s1' },
                  { name: 'Perempuan', cls: 'viz-s2' },
                ]}
                rows={categories
                  .filter((c) => c.kind === 'main')
                  .map((c) => {
                    const inCat = members.filter((m) => m.category === c.key)
                    return {
                      label: c.name,
                      values: [
                        inCat.filter((m) => m.gender === 'Laki Laki').length,
                        inCat.filter((m) => m.gender === 'Perempuan').length,
                      ],
                    }
                  })}
              />
            </ChartCard>
            <ChartCard title="Baptis & Sidi" subtitle="Jumlah jiwa yang sudah, belum, atau tanpa data">
              <StackedBarChart
                rowHead="Sakramen"
                series={[
                  { name: 'Sudah', cls: 'viz-s1' },
                  { name: 'Belum', cls: 'viz-s2' },
                  // unknown: recessive gray, not a third hue
                  { name: 'Tanpa data', cls: 'viz-deemph' },
                ]}
                rows={(
                  [
                    ['Baptis', 'baptized'],
                    ['Sidi', 'confirmed'],
                  ] as const
                ).map(([label, field]) => ({
                  label,
                  values: [true, false, null].map((v) => count((m) => m[field] === v)),
                }))}
              />
            </ChartCard>
            <ChartCard title="Ukuran keluarga" subtitle="Jumlah keluarga menurut banyaknya anggota">
              <ColumnChart
                data={Array.from(
                  {
                    length: Math.max(0, ...families.map((f) => f.memberCount)),
                  },
                  (_, i) => ({
                    label: `${i + 1} jiwa`,
                    short: String(i + 1),
                    value: families.filter((f) => f.memberCount === i + 1).length,
                  }),
                )}
                unit="keluarga"
                labelHead="Anggota per keluarga"
              />
            </ChartCard>
          </div>

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
