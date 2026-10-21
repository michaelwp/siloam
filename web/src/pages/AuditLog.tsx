import { useCallback, useEffect, useState } from 'react'
import { adminApi, type AuditEntry, type AuditFilter } from '../api'
import { Badge, PageHeader, Status } from '../components/ui'
import { clockTime, longDate } from '../format'

const PAGE = 50
const NO_FILTER: AuditFilter = { username: '', category: '', result: '', from: '', to: '' }

/** Admins see who did what: changes, sign-ins and sign-outs, newest first. */
export default function AuditLog() {
  const [filter, setFilter] = useState<AuditFilter>(NO_FILTER)
  const [entries, setEntries] = useState<AuditEntry[]>([])
  const [more, setMore] = useState(false)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [meta, setMeta] = useState<{ categories: { key: string; label: string }[]; usernames: string[] }>({
    categories: [],
    usernames: [],
  })
  // a new filter shows "loading" until its first page arrives
  const changeFilter = (next: (f: AuditFilter) => AuditFilter) => {
    setLoading(true)
    setFilter(next)
  }
  const set = (patch: Partial<AuditFilter>) => changeFilter((f) => ({ ...f, ...patch }))

  useEffect(() => {
    adminApi.auditMeta().then(setMeta, () => {})
  }, [])

  // a new filter starts from the newest entry; "Muat lebih banyak" continues after the last one
  const load = useCallback(
    (before?: string) => {
      adminApi.audit({ ...filter, before }).then(
        (page) => {
          setEntries((old) => (before ? [...old, ...page] : page))
          setMore(page.length === PAGE)
          setError(null)
          setLoading(false)
        },
        (e: Error) => {
          setError(e.message)
          setLoading(false)
        },
      )
    },
    [filter],
  )
  useEffect(() => load(), [load])

  const label = (key: string) => meta.categories.find((c) => c.key === key)?.label ?? key
  const active = Object.values(filter).some(Boolean)

  return (
    <>
      <PageHeader title="Log Aktivitas" subtitle="Perubahan data, masuk dan keluar oleh pengguna, terbaru di atas" />

      <fieldset className="filters">
        <legend className="sr-only">Filter log</legend>
        <select aria-label="Pengguna" value={filter.username} onChange={(e) => set({ username: e.target.value })}>
          <option value="">Semua pengguna</option>
          {meta.usernames.map((u) => (
            <option key={u}>{u}</option>
          ))}
        </select>
        <select aria-label="Aktivitas" value={filter.category} onChange={(e) => set({ category: e.target.value })}>
          <option value="">Semua aktivitas</option>
          {meta.categories.map((c) => (
            <option key={c.key} value={c.key}>
              {c.label}
            </option>
          ))}
        </select>
        <select
          aria-label="Hasil"
          value={filter.result}
          onChange={(e) => set({ result: e.target.value as AuditFilter['result'] })}
        >
          <option value="">Berhasil & gagal</option>
          <option value="ok">Berhasil</option>
          <option value="failed">Gagal / ditolak</option>
        </select>
        <label className="date-filter">
          <span>Dari</span>
          <input type="date" value={filter.from} onChange={(e) => set({ from: e.target.value })} />
        </label>
        <label className="date-filter">
          <span>Sampai</span>
          <input type="date" value={filter.to} onChange={(e) => set({ to: e.target.value })} />
        </label>
        {active && (
          <button type="button" className="link" onClick={() => changeFilter(() => NO_FILTER)}>
            Hapus filter
          </button>
        )}
      </fieldset>

      <Status loading={loading && !entries.length} error={error} empty={!loading && !entries.length} />
      {entries.length > 0 && (
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Waktu</th>
                <th>Pengguna</th>
                <th>Aktivitas</th>
                <th>Data</th>
                <th>Hasil</th>
                <th>IP</th>
              </tr>
            </thead>
            <tbody>
              {entries.map((e) => (
                <tr key={e.id}>
                  <td className="nowrap">
                    {longDate(e.at)} <span className="muted">{clockTime(e.at)}</span>
                  </td>
                  <td className="strong">{e.username || <span className="muted">(tidak dikenal)</span>}</td>
                  <td>
                    {e.action}
                    <span className="muted small block">{label(e.category)}</span>
                  </td>
                  <td>{e.target || <span className="muted">–</span>}</td>
                  <td>
                    {e.status < 400 ? (
                      <Badge tone="green">Berhasil</Badge>
                    ) : (
                      <Badge tone="red">{e.status === 401 || e.status === 403 ? 'Ditolak' : 'Gagal'}</Badge>
                    )}
                  </td>
                  <td className="muted small">{e.ip}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      {more && (
        <p className="load-more">
          <button
            className="btn"
            disabled={loading}
            onClick={() => {
              setLoading(true)
              load(entries.at(-1)?.id)
            }}
          >
            {loading ? 'Memuat…' : 'Muat lebih banyak'}
          </button>
        </p>
      )}
    </>
  )
}
