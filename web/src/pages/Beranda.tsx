import { useState } from 'react'
import type { Member } from '../api'
import { Link } from 'react-router-dom'
import { ChartCard, ColumnChart, StackedBarChart } from '../components/charts'
import { PageHeader, Status } from '../components/ui'
import { useData } from '../data'
import { byText, CATEGORIES, dayMonth, daysUntil, MONTHS, parts } from '../format'

/** Public homepage: totals plus this month's birthdays and wedding anniversaries. */
export default function Beranda() {
  const { families, members, loading, error } = useData()
  const [today] = useState(() => new Date())
  const month = today.getMonth()
  const day = today.getDate()

  const count = (pred: (m: (typeof members)[number]) => boolean) => members.filter(pred).length
  const tiles = [
    { label: 'Keluarga', value: families.length, to: '/keluarga' },
    { label: 'Jiwa', value: members.length, to: '/anggota' },
    { label: 'Laki-laki', value: count((m) => m.gender === 'Laki Laki'), to: '/anggota' },
    { label: 'Perempuan', value: count((m) => m.gender === 'Perempuan'), to: '/anggota' },
  ]
  const bipra = [
    ...CATEGORIES.map((c) => ({ label: c.label, range: c.range, value: count((m) => m.category === c.key) })),
    { label: 'Lansia', range: '60+', value: count((m) => m.elderly) },
  ]

  // month/day are available to visitors; the year (age, years married) only when signed in
  const year = today.getFullYear()
  const birthdays = members
    .filter((m) => m.birthMonth === month + 1)
    .map((m) => ({ m, d: m.birthDay!, turning: m.birthDate ? year - parts(m.birthDate).y : null }))
    .toSorted((a, b) => a.d - b.d || byText(a.m.name, b.m.name))
  const anniversaries = families
    .filter((f) => f.weddingMonth === month + 1)
    .map((f) => ({ f, d: f.weddingDay!, years: f.weddingDate ? year - parts(f.weddingDate).y : null }))
    .toSorted((a, b) => a.d - b.d)

  return (
    <>
      <PageHeader title="Beranda" subtitle="GMIM Siloam · Jemaat Kolom 9" />
      <Status loading={loading} error={error} />

      {!loading && !error && (
        <>
          <TodayBirthdays members={members} today={today} />

          <div className="stats">
            {tiles.map((t) => (
              <Link key={t.label} to={t.to} className="stat">
                <span className="stat-value">{t.value}</span>
                <span className="stat-label">{t.label}</span>
              </Link>
            ))}
          </div>

          <h2 className="section-title">Kategorial BIPRA</h2>
          <div className="stats">
            {bipra.map((c) => (
              <Link key={c.label} to="/bipra" className="stat">
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
                rows={CATEGORIES.map((c) => {
                  const inCat = members.filter((m) => m.category === c.key)
                  return {
                    label: c.label,
                    values: [
                      inCat.filter((m) => m.gender === 'Laki Laki').length,
                      inCat.filter((m) => m.gender === 'Perempuan').length,
                    ],
                  }
                })}
              />
            </ChartCard>
            <ChartCard title="Ukuran keluarga" subtitle="Jumlah keluarga menurut banyaknya anggota">
              <ColumnChart
                data={Array.from({ length: Math.max(0, ...families.map((f) => f.memberCount)) }, (_, i) => ({
                  label: `${i + 1} jiwa`,
                  short: String(i + 1),
                  value: families.filter((f) => f.memberCount === i + 1).length,
                }))}
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
          </div>
        </>
      )}
    </>
  )
}

const UPCOMING_DAYS = 7

/**
 * The homepage's main information: who has a birthday today. When nobody does,
 * the next birthdays within a week. The age is shown only to signed-in users
 * (visitors don't receive birth years).
 */
function TodayBirthdays({ members, today }: { members: Member[]; today: Date }) {
  const dated = members
    .filter((m) => m.birthMonth && m.birthDay)
    .map((m) => ({
      m,
      days: daysUntil(m.birthMonth!, m.birthDay!, today),
      turning: m.birthDate ? today.getFullYear() - parts(m.birthDate).y : null,
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
        <ul className="today-list">
          {todays.map(({ m, turning }) => (
            <li key={m.id}>
              <strong>{m.name}</strong>
              <span className="muted">
                Kel. {m.familyName}
                {turning !== null && <> · {turning} tahun</>}
              </span>
            </li>
          ))}
        </ul>
      ) : (
        <div className="today-empty">
          <p>Tidak ada anggota jemaat yang berulang tahun hari ini.</p>
          {upcoming.length > 0 && (
            <p className="muted small">
              Berikutnya:{' '}
              {upcoming.map(({ m, days }, i) => (
                <span key={m.id}>
                  {i > 0 && ', '}
                  <strong>{m.name}</strong> ({days === 1 ? 'besok' : dayMonth(m.birthDay, m.birthMonth)})
                </span>
              ))}
            </p>
          )}
        </div>
      )}
    </section>
  )
}
