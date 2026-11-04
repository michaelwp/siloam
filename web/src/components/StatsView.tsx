import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { bipraRange, MONTHS } from '../format'
import type { Stats } from '../stats'
import { ChartCard, ColumnChart, StackedBarChart } from './charts'

/**
 * Where a tile leads (signed in: the filtered list); without links (visitors)
 * the tiles are plain.
 */
export type StatsLinks = {
  families: string
  members: string
  gender: (g: 'Laki Laki' | 'Perempuan') => string
  sacrament: (field: 'baptized' | 'confirmed') => string
  status: (label: string) => string
  category: (key: string) => string
}

function Tile({ to, children }: { to?: string; children: ReactNode }) {
  return to ? (
    <Link to={to} className="stat">
      {children}
    </Link>
  ) : (
    <div className="stat">{children}</div>
  )
}

/** "3 belum · 2 tanpa data" under the baptized / confirmed tiles */
function note(c: Stats['baptized']) {
  return `${c.no} belum` + (c.unknown ? ` · ${c.unknown} tanpa data` : '')
}

/** Totals, statuses, BIPRA counts and the charts, from a statistics summary. */
export function StatsView({ stats, links, month }: { stats: Stats; links?: StatsLinks; month: number }) {
  const tiles: { label: string; value: number; to?: string; note?: string }[] = [
    { label: 'Keluarga', value: stats.families, to: links?.families },
    { label: 'Jiwa', value: stats.members, to: links?.members },
    { label: 'Laki-laki', value: stats.male, to: links?.gender('Laki Laki') },
    { label: 'Perempuan', value: stats.female, to: links?.gender('Perempuan') },
    { label: 'Baptis', value: stats.baptized.yes, to: links?.sacrament('baptized'), note: note(stats.baptized) },
    { label: 'Sidi', value: stats.confirmed.yes, to: links?.sacrament('confirmed'), note: note(stats.confirmed) },
  ]
  return (
    <>
      <div className="stats">
        {tiles.map((t) => (
          <Tile key={t.label} to={t.to}>
            <span className="stat-value">{t.value}</span>
            <span className="stat-label">{t.label}</span>
            {t.note && <span className="stat-range">{t.note}</span>}
          </Tile>
        ))}
      </div>

      <h2 className="section-title">Status</h2>
      <div className="stats">
        {stats.maritalStatuses.map((s) => (
          <Tile key={s.label} to={links?.status(s.label)}>
            <span className="stat-value">{s.value}</span>
            <span className="stat-label">{s.label}</span>
          </Tile>
        ))}
      </div>

      <h2 className="section-title">Kategorial BIPRA</h2>
      <div className="stats">
        {stats.categories.map((c) => (
          <Tile key={c.key} to={links?.category(c.key)}>
            <span className="stat-value">{c.count}</span>
            <span className="stat-label">{c.name}</span>
            <span className="stat-range">{bipraRange(c)}</span>
          </Tile>
        ))}
      </div>

      <h2 className="section-title">Statistik</h2>
      <div className="chart-grid">
        <ChartCard title="Ulang tahun per bulan" subtitle={`Jumlah jiwa yang berulang tahun tiap bulan · ${MONTHS[month]} disorot`}>
          <ColumnChart
            data={MONTHS.map((name, i) => ({
              label: name,
              short: name.slice(0, 3),
              value: stats.birthMonths[i] ?? 0,
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
            rows={stats.categories
              .filter((c) => c.kind === 'main')
              .map((c) => ({ label: c.name, values: [c.male, c.female] }))}
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
                ['Baptis', stats.baptized],
                ['Sidi', stats.confirmed],
              ] as const
            ).map(([label, c]) => ({ label, values: [c.yes, c.no, c.unknown] }))}
          />
        </ChartCard>
        <ChartCard title="Ukuran keluarga" subtitle="Jumlah keluarga menurut banyaknya anggota">
          <ColumnChart
            data={stats.familySizes.map((value, i) => ({ label: `${i + 1} jiwa`, short: String(i + 1), value }))}
            unit="keluarga"
            labelHead="Anggota per keluarga"
          />
        </ChartCard>
      </div>
    </>
  )
}
