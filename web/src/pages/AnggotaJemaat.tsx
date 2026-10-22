import { useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { MemberDialogs, type MemberDialog } from '../components/forms'
import { ActionsHeader, EditOnly, PageHeader, RowActions, Search, Status } from '../components/ui'
import { useFullView } from '../auth'
import { useData } from '../data'
import { birthLine, byText, dayMonth, GENDERS, MARITAL_STATUSES, matches, triState } from '../format'

type TriFilter = '' | 'yes' | 'no' | 'unknown'

/** Matches a yes/no/unknown field (baptis, sidi) against a filter value. */
function matchTri(v: boolean | null, f: TriFilter) {
  return f === '' || (f === 'yes' && v === true) || (f === 'no' && v === false) || (f === 'unknown' && v === null)
}

const NO_FILTERS = { gender: '', status: '', baptized: '' as TriFilter, confirmed: '' as TriFilter }
type Filters = typeof NO_FILTERS
const TRI: readonly string[] = ['yes', 'no', 'unknown']
const tri = (v: string | null) => (v && TRI.includes(v) ? (v as TriFilter) : '')

export default function AnggotaJemaat() {
  const full = useFullView()
  const { members, kolom, filterLabel, loading, error } = useData()
  const [q, setQ] = useState('')
  // the filters live in the URL (?gender=…&status=…&baptized=yes&confirmed=no), so a
  // Beranda tile can open this page filtered and the back button returns
  const [params, setParams] = useSearchParams()
  const filters: Filters = {
    gender: params.get('gender') ?? '',
    status: params.get('status') ?? '',
    baptized: tri(params.get('baptized')),
    confirmed: tri(params.get('confirmed')),
  }
  const setFilters = (next: Filters) =>
    setParams(Object.fromEntries(Object.entries(next).filter(([, v]) => v)), { replace: true })
  const [dialog, setDialog] = useState<MemberDialog>(null)
  const set = (patch: Partial<Filters>) => setFilters({ ...filters, ...patch })

  const f = filters
  const active = Object.values(f).some(Boolean)
  const rows = members
    .filter(
      (m) =>
        matches(q, m.name, m.familyName, m.birthPlace) &&
        (!f.gender || m.gender === f.gender) &&
        (!f.status || m.maritalStatus === f.status) &&
        matchTri(m.baptized, f.baptized) &&
        matchTri(m.confirmed, f.confirmed),
    )
    .toSorted((a, b) => byText(a.name, b.name))

  return (
    <>
      <PageHeader title="Daftar Anggota Jemaat" subtitle={`${filterLabel} · ${rows.length} dari ${members.length} jiwa, urut abjad`}>
        <Search value={q} onChange={setQ} />
        <EditOnly>
          <button className="btn primary" onClick={() => setDialog({ kind: 'add' })}>
            + Tambah Anggota
          </button>
        </EditOnly>
      </PageHeader>

      <fieldset className="filters">
        <legend className="sr-only">Filter anggota</legend>
        <select aria-label="Jenis kelamin" value={f.gender} onChange={(e) => set({ gender: e.target.value })}>
          <option value="">Semua jenis kelamin</option>
          {GENDERS.map((g) => (
            <option key={g}>{g}</option>
          ))}
        </select>
        <select aria-label="Status" value={f.status} onChange={(e) => set({ status: e.target.value })}>
          <option value="">Semua status</option>
          {MARITAL_STATUSES.map((st) => (
            <option key={st}>{st}</option>
          ))}
          {/* another status from the data, opened from a Beranda tile */}
          {f.status && !MARITAL_STATUSES.includes(f.status) && <option>{f.status}</option>}
        </select>
        <select
          aria-label="Baptis"
          value={f.baptized}
          onChange={(e) => set({ baptized: e.target.value as TriFilter })}
        >
          <option value="">Baptis: semua</option>
          <option value="yes">Sudah baptis</option>
          <option value="no">Belum baptis</option>
          <option value="unknown">Baptis: tidak diketahui</option>
        </select>
        <select
          aria-label="Sidi"
          value={f.confirmed}
          onChange={(e) => set({ confirmed: e.target.value as TriFilter })}
        >
          <option value="">Sidi: semua</option>
          <option value="yes">Sudah sidi</option>
          <option value="no">Belum sidi</option>
          <option value="unknown">Sidi: tidak diketahui</option>
        </select>
        {active && (
          <button type="button" className="link" onClick={() => setFilters(NO_FILTERS)}>
            Reset filter
          </button>
        )}
      </fieldset>

      <Status loading={loading} error={error} empty={!members.length} />
      {!loading && members.length > 0 && rows.length === 0 && (
        <p className="status">Tidak ada anggota yang cocok dengan pencarian/filter.</p>
      )}
      {rows.length > 0 && (
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th className="num">No</th>
                <th>Nama Lengkap</th>
                <th>{full ? 'Tempat, Tanggal Lahir' : 'Ulang Tahun'}</th>
                {full && <th>Umur</th>}
                <th>Jenis Kelamin</th>
                <th>Status</th>
                <th>Baptis</th>
                <th>Sidi</th>
                <th>Nama Keluarga</th>
                {kolom === null && <th className="num">Kolom</th>}
                <ActionsHeader />
              </tr>
            </thead>
            <tbody>
              {rows.map((m, i) => (
                <tr key={m.id}>
                  <td className="num">{i + 1}</td>
                  <td className="strong">{m.name}</td>
                  <td>{full ? birthLine(m) : dayMonth(m.birthDay, m.birthMonth)}</td>
                  {full && <td className="nowrap">{m.ageText || '–'}</td>}
                  <td>{m.gender}</td>
                  <td>{m.maritalStatus}</td>
                  <td>{triState(m.baptized, 'Baptis')}</td>
                  <td>{triState(m.confirmed, 'Sidi')}</td>
                  <td>Kel. {m.familyName}</td>
                  {kolom === null && <td className="num">{m.kolom}</td>}
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
