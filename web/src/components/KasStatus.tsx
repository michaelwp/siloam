import { Link } from 'react-router-dom'
import { kasApi, type KasSummary } from '../api'
import { kolomLabel, useData } from '../data'
import { localToday, MONTHS, rupiah } from '../format'
import { useLoad } from '../useLoad'

/** An amount tile ("Rp 1.250.000", red below zero). */
export function KasTile({ label, value, note }: { label: string; value: number; note?: string }) {
  return (
    <div className="stat static">
      <span className={`stat-value${value < 0 ? ' negative' : ''}`}>{rupiah(value)}</span>
      <span className="stat-label">{label}</span>
      {note && <span className="stat-range">{note}</span>}
    </div>
  )
}

const thisMonth = () => kasApi.summary(localToday().slice(0, 7))

/**
 * Beranda's kas status: the balance and this month's cash in and out of the
 * viewed kolom, or of all the koloms whose kas the user may read (with a line
 * per kolom). Nothing for a user who may read no kolom's kas.
 */
export function KasStatus() {
  const { kolom, setKolom } = useData()
  const { data } = useLoad<KasSummary[]>('kas-status', thisMonth)
  if (!data?.length) return null
  const shown = kolom === null ? data : data.filter((s) => s.kolom === kolom)
  if (!shown.length) return null
  const t = shown.reduce(
    (a, s) => ({ in: a.in + s.in, out: a.out + s.out, monthIn: a.monthIn + s.monthIn, monthOut: a.monthOut + s.monthOut }),
    { in: 0, out: 0, monthIn: 0, monthOut: 0 },
  )
  const month = MONTHS[Number(shown[0].month.slice(5, 7)) - 1]
  return (
    <section className="card kas-status" aria-labelledby="kas-title">
      <header className="family-head">
        <h2 id="kas-title">Kas {kolom === null ? (shown.length > 1 ? 'Kolom' : kolomLabel(shown[0].kolom)) : kolomLabel(kolom)}</h2>
        <Link to="/kas" className="small">
          Buka Kas Kolom →
        </Link>
      </header>
      <div className="stats">
        <KasTile label={shown.length > 1 ? 'Total saldo' : 'Saldo'} value={t.in - t.out} />
        <KasTile label="Kas masuk" value={t.monthIn} note={`bulan ${month}`} />
        <KasTile label="Kas keluar" value={t.monthOut} note={`bulan ${month}`} />
      </div>
      {shown.length > 1 && (
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Kolom</th>
                <th className="num">Saldo</th>
                <th className="num">Masuk ({month})</th>
                <th className="num">Keluar ({month})</th>
              </tr>
            </thead>
            <tbody>
              {shown.map((s) => (
                <tr key={s.kolom}>
                  <td>
                    <button type="button" className="link" onClick={() => setKolom(s.kolom)}>
                      {kolomLabel(s.kolom)}
                    </button>
                  </td>
                  <td className={`num strong${s.in - s.out < 0 ? ' negative' : ''}`}>{rupiah(s.in - s.out)}</td>
                  <td className="num">{rupiah(s.monthIn)}</td>
                  <td className="num">{rupiah(s.monthOut)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  )
}
