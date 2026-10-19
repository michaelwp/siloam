import { useState } from 'react'
import { api, familyToInput, type Family } from '../api'
import { ConfirmDelete, Field, FormShell, Modal, PageHeader, RowActions, Status } from '../components/ui'
import { useData } from '../data'
import { fromInputDate, longDate, MONTHS, parts, toInputDate } from '../format'

type Dialog = { kind: 'add' } | { kind: 'edit' | 'delete'; family: Family } | null

export default function HutPernikahan() {
  const { families, loading, error } = useData()
  const [dialog, setDialog] = useState<Dialog>(null)
  const [thisMonth] = useState(() => new Date().getMonth())

  const rows = families
    .filter((f) => f.weddingDate)
    .toSorted((a, b) => {
      const pa = parts(a.weddingDate!)
      const pb = parts(b.weddingDate!)
      return pa.m - pb.m || pa.d - pb.d
    })

  return (
    <>
      <PageHeader
        title="Daftar HUT Pernikahan"
        subtitle={`${rows.length} keluarga dengan tanggal pernikahan, diurutkan per bulan`}
      >
        <button className="btn primary" onClick={() => setDialog({ kind: 'add' })}>
          + Tambah Tanggal Pernikahan
        </button>
      </PageHeader>

      <Status loading={loading} error={error} empty={!rows.length} />
      {rows.length > 0 && (
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th className="num">No</th>
                <th>Nama Keluarga</th>
                <th>Bulan</th>
                <th>Tanggal Pernikahan</th>
                <th>Umur Pernikahan</th>
                <th className="no-print">
                  <span className="sr-only">Aksi</span>
                </th>
              </tr>
            </thead>
            <tbody>
              {rows.map((f, i) => {
                const m = parts(f.weddingDate!).m
                return (
                  <tr key={f.id} className={m === thisMonth ? 'highlight' : ''}>
                    <td className="num">{i + 1}</td>
                    <td className="strong">Kel. {f.name}</td>
                    <td>{MONTHS[m]}</td>
                    <td>{longDate(f.weddingDate)}</td>
                    <td>{f.weddingYears} Tahun</td>
                    <RowActions
                      onEdit={() => setDialog({ kind: 'edit', family: f })}
                      onDelete={() => setDialog({ kind: 'delete', family: f })}
                    />
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}

      {dialog?.kind === 'add' && <WeddingForm onClose={() => setDialog(null)} />}
      {dialog?.kind === 'edit' && <WeddingForm family={dialog.family} onClose={() => setDialog(null)} />}
      {dialog?.kind === 'delete' && <ClearWedding family={dialog.family} onClose={() => setDialog(null)} />}
    </>
  )
}

function WeddingForm({ family, onClose }: { family?: Family; onClose: () => void }) {
  const { families, reload } = useData()
  const [familyId, setFamilyId] = useState(family?.id ?? '')
  const [date, setDate] = useState(toInputDate(family?.weddingDate ?? null))
  const choices = family ? [family] : families.filter((f) => !f.weddingDate)

  return (
    <Modal title={family ? 'Ubah Tanggal Pernikahan' : 'Tambah Tanggal Pernikahan'} onClose={onClose}>
      <FormShell
        onCancel={onClose}
        onSubmit={async () => {
          const f = families.find((x) => x.id === familyId)
          if (!f) throw new Error('Pilih keluarga terlebih dahulu')
          await api.updateFamily(f.id, { ...familyToInput(f), weddingDate: fromInputDate(date) })
          await reload()
          onClose()
        }}
      >
        <Field label="Keluarga" wide>
          <select required disabled={!!family} value={familyId} onChange={(e) => setFamilyId(e.target.value)}>
            <option value="">— pilih keluarga —</option>
            {choices.map((f) => (
              <option key={f.id} value={f.id}>
                {f.no}. Kel. {f.name}
              </option>
            ))}
          </select>
        </Field>
        <Field label="Tanggal Pernikahan" wide>
          <input type="date" required value={date} onChange={(e) => setDate(e.target.value)} />
        </Field>
      </FormShell>
    </Modal>
  )
}

function ClearWedding({ family, onClose }: { family: Family; onClose: () => void }) {
  const { reload } = useData()
  return (
    <ConfirmDelete
      title="Hapus Tanggal Pernikahan"
      onClose={onClose}
      message={
        <>
          Hapus tanggal pernikahan <b>Kel. {family.name}</b> ({longDate(family.weddingDate)})? Data keluarga dan
          anggotanya tetap tersimpan.
        </>
      }
      onConfirm={async () => {
        await api.updateFamily(family.id, { ...familyToInput(family), weddingDate: null })
        await reload()
        onClose()
      }}
    />
  )
}
