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

const kolomMonth = () => kasApi.summary(localToday().slice(0, 7))
const bipraMonth = () => kasApi.summary(localToday().slice(0, 7), true)

/**
 * Beranda's kas status: the kolom books (the viewed kolom, or all the koloms
 * whose kas the user may read) and the BIPRA books (the picked category, or
 * all the readable ones), each with the balance and this month's cash in and
 * out. Nothing for books the user may not read.
 */
export function KasStatus() {
  const { kolom, setKolom, bipraFilter, setBipra } = useData()
  const { data: koloms } = useLoad<KasSummary[]>('kas-kolom', kolomMonth)
  const { data: bipras } = useLoad<KasSummary[]>('kas-bipra', bipraMonth)
  const kolomBooks = (koloms ?? []).filter((s) => kolom === null || s.kolom === kolom)
  const bipraBooks = (bipras ?? []).filter((s) => !bipraFilter || s.bipraId === bipraFilter.id)
  return (
    <>
      <KasCard
        id="kas-kolom"
        title={kolom !== null || kolomBooks.length === 1 ? `Kas ${kolomLabel(kolomBooks[0]?.kolom ?? kolom)}` : 'Kas Kolom'}
        what="Kolom"
        link="/kas"
        books={kolomBooks}
        label={(s) => kolomLabel(s.kolom)}
        onOpen={(s) => setKolom(s.kolom)}
      />
      <KasCard
        id="kas-bipra"
        title={bipraFilter || bipraBooks.length === 1 ? `Kas BIPRA ${bipraBooks[0]?.name ?? ''}` : 'Kas BIPRA'}
        what="Kategori"
        link="/kas?buku=bipra"
        books={bipraBooks}
        label={(s) => s.name ?? ''}
        onOpen={(s) => setBipra(s.bipraId ?? null)}
      />
    </>
  )
}

/** One card: the books' total balance and this month's cash, with a line per book when there are several. */
function KasCard({
  id,
  title,
  what,
  link,
  books,
  label,
  onOpen,
}: {
  id: string
  title: string
  what: string
  link: string
  books: KasSummary[]
  label: (s: KasSummary) => string
  onOpen: (s: KasSummary) => void
}) {
  if (!books.length) return null
  const t = books.reduce(
    (a, s) => ({ in: a.in + s.in, out: a.out + s.out, monthIn: a.monthIn + s.monthIn, monthOut: a.monthOut + s.monthOut }),
    { in: 0, out: 0, monthIn: 0, monthOut: 0 },
  )
  const month = MONTHS[Number(books[0].month.slice(5, 7)) - 1]
  return (
    <section className="card kas-status" aria-labelledby={id}>
      <header className="family-head">
        <h2 id={id}>{title.trim()}</h2>
        <Link to={link} className="small">
          Buka Kas →
        </Link>
      </header>
      <div className="stats">
        <KasTile label={books.length > 1 ? 'Total saldo' : 'Saldo'} value={t.in - t.out} />
        <KasTile label="Kas masuk" value={t.monthIn} note={`bulan ${month}`} />
        <KasTile label="Kas keluar" value={t.monthOut} note={`bulan ${month}`} />
      </div>
      {books.length > 1 && (
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>{what}</th>
                <th className="num">Saldo</th>
                <th className="num">Masuk ({month})</th>
                <th className="num">Keluar ({month})</th>
              </tr>
            </thead>
            <tbody>
              {books.map((s) => (
                <tr key={s.bipraId ?? s.kolom}>
                  <td>
                    <button type="button" className="link" onClick={() => onOpen(s)}>
                      {label(s)}
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
