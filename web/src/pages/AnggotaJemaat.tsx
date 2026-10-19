import { useState } from 'react'
import { MemberDialogs, type MemberDialog } from '../components/forms'
import { PageHeader, RowActions, Search, Status } from '../components/ui'
import { useData } from '../data'
import { byText, GENDERS, longDate, matches, triState } from '../format'

export default function AnggotaJemaat() {
  const { members, loading, error } = useData()
  const [q, setQ] = useState('')
  const [gender, setGender] = useState('')
  const [dialog, setDialog] = useState<MemberDialog>(null)

  const rows = members
    .filter((m) => matches(q, m.name, m.familyName, m.birthPlace) && (!gender || m.gender === gender))
    .toSorted((a, b) => byText(a.name, b.name))

  return (
    <>
      <PageHeader title="Daftar Anggota Jemaat" subtitle={`${rows.length} dari ${members.length} jiwa, urut abjad`}>
        <Search value={q} onChange={setQ} />
        <select value={gender} onChange={(e) => setGender(e.target.value)}>
          <option value="">Semua jenis kelamin</option>
          {GENDERS.map((g) => (
            <option key={g}>{g}</option>
          ))}
        </select>
        <button className="btn primary" onClick={() => setDialog({ kind: 'add' })}>
          + Tambah Anggota
        </button>
      </PageHeader>

      <Status loading={loading} error={error} empty={!members.length} />
      {rows.length > 0 && (
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th className="num">No</th>
                <th>Nama Lengkap</th>
                <th>Tempat, Tanggal Lahir</th>
                <th>Umur</th>
                <th>Jenis Kelamin</th>
                <th>Status</th>
                <th>Baptis</th>
                <th>Sidi</th>
                <th>Nama Keluarga</th>
                <th className="no-print">
                  <span className="sr-only">Aksi</span>
                </th>
              </tr>
            </thead>
            <tbody>
              {rows.map((m, i) => (
                <tr key={m.id}>
                  <td className="num">{i + 1}</td>
                  <td className="strong">{m.name}</td>
                  <td>
                    {m.birthPlace}
                    {m.birthPlace && ', '}
                    {longDate(m.birthDate)}
                  </td>
                  <td className="nowrap">{m.ageText || '–'}</td>
                  <td>{m.gender}</td>
                  <td>{m.maritalStatus}</td>
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
