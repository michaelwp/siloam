import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api, type Kolom as KolomType } from '../api'
import { ConfirmDelete, PageHeader, Status } from '../components/ui'
import { useData } from '../data'
import { useSchedule } from '../schedule'

/** Signed-in users (admin and operators) manage the list of koloms. */
export default function Kolom() {
  const { koloms, allFamilies, allMembers, loading, error, reload, setKolom } = useData()
  const navigate = useNavigate()
  const [number, setNumber] = useState('')
  const [busy, setBusy] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)
  const [deleting, setDeleting] = useState<KolomType | null>(null)

  const { allServices } = useSchedule()
  const familiesIn = (k: number) => allFamilies.filter((f) => f.kolom === k).length
  const servicesIn = (k: number) => allServices.filter((s) => s.kolom === k).length
  const membersIn = (k: number) => allMembers.filter((m) => m.kolom === k).length
  const next = koloms.reduce((max, k) => Math.max(max, k.number), 0) + 1

  return (
    <>
      <PageHeader title="Kolom" subtitle={`${koloms.length} kolom · data jemaat dikelompokkan per kolom`} />

      <form
        className="card kolom-add"
        onSubmit={async (e) => {
          e.preventDefault()
          setBusy(true)
          setFormError(null)
          try {
            await api.createKolom(Number(number))
            await reload()
            setNumber('')
          } catch (err) {
            setFormError((err as Error).message)
          } finally {
            setBusy(false)
          }
        }}
      >
        <label className="field">
          <span>Nomor kolom baru</span>
          <input
            type="number"
            min={1}
            max={999}
            required
            value={number}
            placeholder={String(next)}
            onChange={(e) => {
              setNumber(e.target.value)
              setFormError(null)
            }}
          />
        </label>
        <button type="submit" className="btn primary" disabled={busy}>
          {busy ? 'Menyimpan…' : '+ Tambah Kolom'}
        </button>
        {formError && <p className="form-error">{formError}</p>}
      </form>

      <Status loading={loading} error={error} empty={!koloms.length} />
      {koloms.length > 0 && (
        <div className="table-wrap">
          <table className="kolom-table">
            <thead>
              <tr>
                <th>Kolom</th>
                <th className="num">Keluarga</th>
                <th className="num">Jiwa</th>
                <th className="num">Jadwal</th>
                <th className="actions">Aksi</th>
              </tr>
            </thead>
            <tbody>
              {koloms.map((k) => {
                const fams = familiesIn(k.number)
                const svcs = servicesIn(k.number)
                const inUse = fams > 0 || svcs > 0
                return (
                  <tr key={k.id}>
                    <td className="strong">Kolom {k.number}</td>
                    <td className="num">{fams}</td>
                    <td className="num">{membersIn(k.number)}</td>
                    <td className="num">{svcs}</td>
                    <td className="actions">
                      <button
                        className="btn small"
                        onClick={() => {
                          setKolom(k.number)
                          navigate('/keluarga')
                        }}
                      >
                        Lihat
                      </button>
                      <button
                        className="btn small danger"
                        disabled={inUse}
                        title={inUse ? 'Kolom yang masih memiliki keluarga atau jadwal ibadah tidak dapat dihapus' : undefined}
                        onClick={() => setDeleting(k)}
                      >
                        Hapus
                      </button>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}

      {deleting && (
        <ConfirmDelete
          title="Hapus Kolom"
          onClose={() => setDeleting(null)}
          message={
            <>
              Hapus <b>Kolom {deleting.number}</b> dari daftar kolom?
            </>
          }
          onConfirm={async () => {
            await api.deleteKolom(deleting.id)
            await reload()
            setDeleting(null)
          }}
        />
      )}
    </>
  )
}
