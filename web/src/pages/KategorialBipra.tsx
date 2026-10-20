import { useState } from 'react'
import { MemberDialogs, type MemberDialog } from '../components/forms'
import { ActionsHeader, Badge, EditOnly, PageHeader, RowActions, Search, Status } from '../components/ui'
import { useFullView } from '../auth'
import { useData } from '../data'
import { birthLine, byText, CATEGORIES, dayMonth, matches, triState } from '../format'

const LANSIA = 'Lansia'

export default function KategorialBipra() {
  const full = useFullView()
  const { members, loading, error } = useData()
  const [q, setQ] = useState('')
  const [filter, setFilter] = useState('')
  const [dialog, setDialog] = useState<MemberDialog>(null)

  const count = (key: string) =>
    key === LANSIA ? members.filter((m) => m.elderly).length : members.filter((m) => m.category === key).length
  const unknown = members.filter((m) => !m.category).length

  const rows = members
    .filter((m) => matches(q, m.name, m.familyName))
    .filter((m) => !filter || (filter === LANSIA ? m.elderly : m.category === filter))
    .toSorted((a, b) => byText(a.name, b.name))

  return (
    <>
      <PageHeader title="Kategorial BIPRA" subtitle="Kategori dihitung otomatis dari umur, jenis kelamin dan status">
        <Search value={q} onChange={setQ} />
        <EditOnly>
          <button className="btn primary" onClick={() => setDialog({ kind: 'add' })}>
            + Tambah Anggota
          </button>
        </EditOnly>
      </PageHeader>

      <div className="stats">
        <button className={`stat ${filter === '' ? 'active' : ''}`} onClick={() => setFilter('')}>
          <span className="stat-value">{members.length}</span>
          <span className="stat-label">Semua</span>
        </button>
        {[...CATEGORIES, { key: LANSIA, label: 'Lansia', range: '60+' }].map((c) => (
          <button
            key={c.key}
            className={`stat ${filter === c.key ? 'active' : ''}`}
            onClick={() => setFilter(filter === c.key ? '' : c.key)}
          >
            <span className="stat-value">{count(c.key)}</span>
            <span className="stat-label">{c.label}</span>
            <span className="stat-range">{c.range}</span>
          </button>
        ))}
      </div>
      {unknown > 0 && <p className="muted">{unknown} anggota belum memiliki tanggal lahir sehingga tidak berkategori.</p>}

      <Status loading={loading} error={error} empty={!members.length} />
      {rows.length > 0 && (
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th className="num">No</th>
                <th>Nama Lengkap</th>
                <th>{full ? 'Tempat, Tanggal Lahir' : 'Ulang Tahun'}</th>
                <th>Jenis Kelamin</th>
                <th>Status</th>
                {full && <th className="num">Umur</th>}
                <th>Kategori</th>
                <th>Baptis</th>
                <th>Sidi</th>
                <th>Nama Keluarga</th>
                <ActionsHeader />
              </tr>
            </thead>
            <tbody>
              {rows.map((m, i) => (
                <tr key={m.id}>
                  <td className="num">{i + 1}</td>
                  <td className="strong">{m.name}</td>
                  <td>{full ? birthLine(m) : dayMonth(m.birthDay, m.birthMonth)}</td>
                  <td>{m.gender}</td>
                  <td>{m.maritalStatus}</td>
                  {full && <td className="num">{m.age ?? '–'}</td>}
                  <td className="nowrap">
                    {m.category && <Badge tone={m.category.toLowerCase()}>{m.category}</Badge>}{' '}
                    {m.elderly && <Badge tone="lansia">Lansia</Badge>}
                  </td>
                  <td>{triState(m.baptized, 'Baptis')}</td>
                  <td>{triState(m.confirmed, 'Sidi')}</td>
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
      <MemberDialogs dialog={dialog} onClose={() => setDialog(null)} />
    </>
  )
}
