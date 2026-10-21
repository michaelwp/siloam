import { useState, type CSSProperties } from 'react'
import { scheduleApi, type WorshipType } from '../api'
import { ConfirmDelete, Field, FormShell, Modal, PageHeader, Status } from '../components/ui'
import { useData } from '../data'
import { TYPE_COLORS, useSchedule } from '../schedule'

type Dialog = { kind: 'add' } | { kind: 'edit' | 'delete'; type: WorshipType } | null

/** Signed-in users manage the kinds of worship services and their colors. */
export default function JenisIbadah() {
  const { types, allServices, loading, error, reload } = useSchedule()
  const { categories } = useData()
  const bipraName = (id: string | null) => categories.find((c) => c.id === id)?.name
  const [dialog, setDialog] = useState<Dialog>(null)
  const used = (id: string) => allServices.filter((s) => s.typeId === id).length

  return (
    <>
      <PageHeader title="Jenis Ibadah" subtitle={`${types.length} jenis · warna dipakai di kalender Jadwal Ibadah`}>
        <button className="btn primary" onClick={() => setDialog({ kind: 'add' })}>
          + Tambah Jenis
        </button>
      </PageHeader>

      <Status loading={loading} error={error} empty={!types.length} />
      {types.length > 0 && (
        <div className="table-wrap">
          <table className="kolom-table">
            <thead>
              <tr>
                <th>Jenis Ibadah</th>
                <th>Kategori BIPRA</th>
                <th className="num">Jadwal</th>
                <th className="actions">Aksi</th>
              </tr>
            </thead>
            <tbody>
              {types.map((t) => {
                const n = used(t.id)
                return (
                  <tr key={t.id}>
                    <td className="strong">
                      <span className="type-swatch" style={{ '--type-color': t.color } as CSSProperties} aria-hidden="true" />
                      {t.name}
                    </td>
                    <td>{bipraName(t.bipraId) ?? <span className="muted">–</span>}</td>
                    <td className="num">{n}</td>
                    <td className="actions">
                      <button className="btn small" onClick={() => setDialog({ kind: 'edit', type: t })}>
                        Ubah
                      </button>
                      <button
                        className="btn small danger"
                        disabled={n > 0}
                        title={n > 0 ? 'Jenis ibadah yang masih dipakai di jadwal tidak dapat dihapus' : undefined}
                        onClick={() => setDialog({ kind: 'delete', type: t })}
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

      {(dialog?.kind === 'add' || dialog?.kind === 'edit') && (
        <TypeForm type={dialog.kind === 'edit' ? dialog.type : undefined} onClose={() => setDialog(null)} onSaved={reload} />
      )}
      {dialog?.kind === 'delete' && (
        <ConfirmDelete
          title="Hapus Jenis Ibadah"
          onClose={() => setDialog(null)}
          message={
            <>
              Hapus jenis ibadah <b>{dialog.type.name}</b>?
            </>
          }
          onConfirm={async () => {
            await scheduleApi.deleteType(dialog.type.id)
            await reload()
            setDialog(null)
          }}
        />
      )}
    </>
  )
}

function TypeForm({ type, onClose, onSaved }: { type?: WorshipType; onClose: () => void; onSaved: () => Promise<void> }) {
  const [name, setName] = useState(type?.name ?? '')
  const [color, setColor] = useState(type?.color ?? TYPE_COLORS[0].hex)
  const [bipraId, setBipraId] = useState(type?.bipraId ?? '')
  const { categories } = useData()
  return (
    <Modal title={type ? 'Ubah Jenis Ibadah' : 'Tambah Jenis Ibadah'} onClose={onClose}>
      <FormShell
        onCancel={onClose}
        onSubmit={async () => {
          if (type) await scheduleApi.updateType(type.id, { name, color, bipraId })
          else await scheduleApi.createType({ name, color, bipraId })
          await onSaved()
          onClose()
        }}
      >
        <Field label="Nama Jenis Ibadah" wide>
          <input required maxLength={80} value={name} onChange={(e) => setName(e.target.value)} placeholder="mis. Pemuda" />
        </Field>
        <Field label="Kategori BIPRA" wide>
          <select value={bipraId} onChange={(e) => setBipraId(e.target.value)}>
            <option value="">— tidak ada (ibadah per kolom) —</option>
            {categories.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
              </option>
            ))}
          </select>
        </Field>
        <p className="wide muted small">
          Ibadah jenis yang terhubung ke kategori BIPRA boleh tanpa kolom (untuk seluruh jemaat), dan dikelola juga oleh
          pengguna “kelola data BIPRA” kategori itu.
        </p>
        <fieldset className="color-picker wide">
          <legend>Warna di kalender</legend>
          {TYPE_COLORS.map((c) => (
            <label key={c.hex} className="color-option" title={c.name}>
              <input type="radio" name="color" value={c.hex} checked={color === c.hex} onChange={() => setColor(c.hex)} />
              <span style={{ '--type-color': c.hex } as CSSProperties} aria-hidden="true" />
              <span className="sr-only">{c.name}</span>
            </label>
          ))}
        </fieldset>
      </FormShell>
    </Modal>
  )
}
