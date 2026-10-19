import { useState } from 'react'
import { MemberDialogs, type MemberDialog } from '../components/forms'
import { PageHeader, RowActions, Search, Status } from '../components/ui'
import { useData } from '../data'
import { longDate, matches, MONTHS, parts } from '../format'

export default function HutJemaat() {
  const { members, loading, error } = useData()
  const [q, setQ] = useState('')
  const [month, setMonth] = useState<number | 'all'>('all')
  const [dialog, setDialog] = useState<MemberDialog>(null)
  const [thisMonth] = useState(() => new Date().getMonth())

  const withDate = members.filter((m) => m.birthDate && matches(q, m.name, m.familyName))
  const noDate = members.filter((m) => !m.birthDate)
  const months = MONTHS.map((name, i) => ({
    name,
    i,
    rows: withDate
      .filter((m) => parts(m.birthDate!).m === i)
      .toSorted((a, b) => parts(a.birthDate!).d - parts(b.birthDate!).d),
  })).filter((g) => month === 'all' || g.i === month)

  return (
    <>
      <PageHeader title="Daftar HUT Jemaat" subtitle={`${members.length} jiwa, dikelompokkan per bulan kelahiran`}>
        <Search value={q} onChange={setQ} />
        <select value={month} onChange={(e) => setMonth(e.target.value === 'all' ? 'all' : Number(e.target.value))}>
          <option value="all">Semua bulan</option>
          {MONTHS.map((m, i) => (
            <option key={m} value={i}>
              {m}
            </option>
          ))}
        </select>
        <button className="btn primary" onClick={() => setDialog({ kind: 'add' })}>
          + Tambah Anggota
        </button>
      </PageHeader>

      <Status loading={loading} error={error} empty={!members.length} />
      {!loading &&
        months.map((g) => (
          <section key={g.name} className={`card month ${g.i === thisMonth ? 'current' : ''}`}>
            <header className="family-head">
              <h2>
                {g.name.toUpperCase()} {g.i === thisMonth && <span className="badge green">Bulan ini</span>}
              </h2>
              <span className="muted">{g.rows.length} orang</span>
            </header>
            {g.rows.length > 0 && (
              <div className="table-wrap flat">
                <table>
                  <thead>
                    <tr>
                      <th className="num">No</th>
                      <th>Nama Lengkap</th>
                      <th>Tempat, Tanggal Lahir</th>
                      <th>Umur</th>
                      <th>Nama Keluarga</th>
                      <th className="no-print">
                  <span className="sr-only">Aksi</span>
                </th>
                    </tr>
                  </thead>
                  <tbody>
                    {g.rows.map((m, i) => (
                      <tr key={m.id}>
                        <td className="num">{i + 1}</td>
                        <td className="strong">{m.name}</td>
                        <td>
                          {m.birthPlace}
                          {m.birthPlace && ', '}
                          {longDate(m.birthDate)}
                        </td>
                        <td className="nowrap">{m.ageText}</td>
                        <td>Kel. {m.familyName}</td>
                        <RowActions
                          onEdit={() => setDialog({ kind: 'edit', member: m })}
                          onDelete={() => setDialog({ kind: 'delete', member: m })}
                        />
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </section>
        ))}

      {!loading && noDate.length > 0 && (
        <section className="card">
          <header className="family-head">
            <h2>Tanggal lahir belum diisi</h2>
          </header>
          <ul className="plain-list">
            {noDate.map((m) => (
              <li key={m.id}>
                {m.name} <span className="muted">(Kel. {m.familyName})</span>{' '}
                <button className="link" onClick={() => setDialog({ kind: 'edit', member: m })}>
                  Lengkapi
                </button>
              </li>
            ))}
          </ul>
        </section>
      )}
      <MemberDialogs dialog={dialog} onClose={() => setDialog(null)} />
    </>
  )
}
