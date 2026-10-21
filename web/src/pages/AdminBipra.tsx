import { useState, type CSSProperties } from 'react'
import { adminApi, type BipraCategory, type BipraInput } from '../api'
import { BipraBadge, ConfirmDelete, Field, FormShell, Modal, PageHeader, Status } from '../components/ui'
import { useData } from '../data'
import { bipraRule, inBipra, matchesBipra } from '../format'
import { TYPE_COLORS } from '../schedule'

type Dialog = { kind: 'add' } | { kind: 'edit' | 'delete'; category: BipraCategory } | null

/**
 * Admins manage the Kategorial BIPRA categories and their rules. The server
 * puts every member (with a birth date) in the first main category whose rule
 * matches, plus every matching flag (e.g. Lansia).
 */
export default function AdminBipra() {
  const { categories, allMembers, loading, error, reload } = useData()
  const [dialog, setDialog] = useState<Dialog>(null)
  const [moveError, setMoveError] = useState<string | null>(null)
  const count = (c: BipraCategory) => allMembers.filter((m) => inBipra(m, c)).length
  const noAge = allMembers.filter((m) => m.age === null).length
  const none = allMembers.filter((m) => m.age !== null && !m.category).length

  const move = async (i: number, by: -1 | 1) => {
    const ids = categories.map((c) => c.id)
    ;[ids[i], ids[i + by]] = [ids[i + by], ids[i]]
    try {
      await adminApi.reorderBipra(ids)
      setMoveError(null)
      await reload()
    } catch (e) {
      setMoveError((e as Error).message)
    }
  }

  return (
    <>
      <PageHeader title="Kategori BIPRA" subtitle="Kategori jemaat dan aturan yang menentukannya">
        <button className="btn primary" onClick={() => setDialog({ kind: 'add' })}>
          + Tambah Kategori
        </button>
      </PageHeader>

      <section className="card panel">
        <header className="panel-head">
          <h2>Cara kerja</h2>
          <p className="muted">
            Kategori dihitung otomatis dari umur, jenis kelamin dan status pernikahan setiap anggota. Setiap anggota
            masuk ke <b>satu kategori utama</b>: yang pertama cocok menurut urutan di bawah (ubah dengan ↑ ↓). Selain
            itu, setiap <b>tanda tambahan</b> yang cocok (seperti Lansia) ikut ditampilkan. Anggota tanpa tanggal lahir
            tidak berkategori. Perubahan langsung berlaku di web dan aplikasi Android.
          </p>
          {(noAge > 0 || none > 0) && (
            <p className="muted small">
              {noAge > 0 && `${noAge} anggota belum memiliki tanggal lahir. `}
              {none > 0 && `${none} anggota tidak cocok dengan kategori utama mana pun.`}
            </p>
          )}
        </header>
      </section>

      <Status loading={loading} error={error} empty={!categories.length} />
      {moveError && <p className="form-error">{moveError}</p>}
      {categories.length > 0 && (
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Urutan</th>
                <th>Kode</th>
                <th>Nama</th>
                <th>Jenis</th>
                <th>Aturan</th>
                <th className="num">Anggota</th>
                <th>
                  <span className="sr-only">Aksi</span>
                </th>
              </tr>
            </thead>
            <tbody>
              {categories.map((c, i) => (
                <tr key={c.id}>
                  <td className="nowrap">
                    <button
                      className="icon-btn small"
                      disabled={i === 0}
                      onClick={() => move(i, -1)}
                      aria-label={`Naikkan ${c.name}`}
                    >
                      ↑
                    </button>
                    <button
                      className="icon-btn small"
                      disabled={i === categories.length - 1}
                      onClick={() => move(i, 1)}
                      aria-label={`Turunkan ${c.name}`}
                    >
                      ↓
                    </button>
                  </td>
                  <td>
                    <BipraBadge code={c.key} />
                  </td>
                  <td className="strong">{c.name}</td>
                  <td>{c.kind === 'main' ? 'Kategori utama' : 'Tanda tambahan'}</td>
                  <td>{bipraRule(c)}</td>
                  <td className="num">{count(c)}</td>
                  <td className="actions">
                    <button className="link" onClick={() => setDialog({ kind: 'edit', category: c })}>
                      Ubah
                    </button>
                    <button className="link danger" onClick={() => setDialog({ kind: 'delete', category: c })}>
                      Hapus
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {(dialog?.kind === 'add' || dialog?.kind === 'edit') && (
        <BipraForm
          category={dialog.kind === 'edit' ? dialog.category : undefined}
          onClose={() => setDialog(null)}
          onSaved={reload}
        />
      )}
      {dialog?.kind === 'delete' && (
        <ConfirmDelete
          title="Hapus Kategori"
          message={
            <>
              Hapus kategori <b>{dialog.category.name}</b>?{' '}
              {count(dialog.category) > 0 &&
                (dialog.category.kind === 'main'
                  ? `${count(dialog.category)} anggota akan masuk ke kategori berikutnya yang cocok, atau tidak berkategori.`
                  : `Tanda ini hilang dari ${count(dialog.category)} anggota.`)}
            </>
          }
          onConfirm={async () => {
            await adminApi.deleteBipra(dialog.category.id)
            await reload()
            setDialog(null)
          }}
          onClose={() => setDialog(null)}
        />
      )}
    </>
  )
}

const EMPTY: BipraInput = {
  key: '',
  name: '',
  kind: 'main',
  color: TYPE_COLORS[0].hex,
  minAge: null,
  maxAge: null,
  gender: '',
  marital: '',
  orMarried: false,
}

const age = (v: string) => (v === '' ? null : Number(v))

function BipraForm({
  category,
  onClose,
  onSaved,
}: {
  category?: BipraCategory
  onClose: () => void
  onSaved: () => Promise<void>
}) {
  const { allMembers } = useData()
  const [c, setC] = useState<BipraInput>(category ?? EMPTY)
  const set = (patch: Partial<BipraInput>) => setC((prev) => ({ ...prev, ...patch }))
  // preview: who the rule itself matches (for a main category, an earlier one may still win)
  const matching = allMembers.filter((m) => matchesBipra(m, c)).length

  return (
    <Modal title={category ? 'Ubah Kategori BIPRA' : 'Tambah Kategori BIPRA'} onClose={onClose}>
      <FormShell
        onCancel={onClose}
        onSubmit={async () => {
          if (category) await adminApi.updateBipra(category.id, c)
          else await adminApi.createBipra(c)
          await onSaved()
          onClose()
        }}
      >
        <Field label="Kode (tampil pada anggota)">
          <input
            required
            maxLength={20}
            value={c.key}
            onChange={(e) => set({ key: e.target.value })}
            placeholder="mis. Pemudi"
          />
        </Field>
        <Field label="Nama">
          <input
            required
            maxLength={80}
            value={c.name}
            onChange={(e) => set({ name: e.target.value })}
            placeholder="mis. Pemudi Gereja"
          />
        </Field>
        <Field label="Jenis" wide>
          <select value={c.kind} onChange={(e) => set({ kind: e.target.value as BipraInput['kind'] })}>
            <option value="main">Kategori utama — setiap anggota satu, yang pertama cocok</option>
            <option value="flag">Tanda tambahan — seperti Lansia, di samping kategori utama</option>
          </select>
        </Field>
        <Field label="Umur minimal (tahun)">
          <input
            type="number"
            min={0}
            max={150}
            value={c.minAge ?? ''}
            onChange={(e) => set({ minAge: age(e.target.value) })}
            placeholder="tidak ada"
          />
        </Field>
        <Field label="Umur maksimal (tahun)">
          <input
            type="number"
            min={0}
            max={150}
            value={c.maxAge ?? ''}
            onChange={(e) => set({ maxAge: age(e.target.value) })}
            placeholder="tidak ada"
          />
        </Field>
        <Field label="Jenis kelamin">
          <select value={c.gender} onChange={(e) => set({ gender: e.target.value as BipraInput['gender'] })}>
            <option value="">Semua</option>
            <option value="Laki Laki">Laki-laki</option>
            <option value="Perempuan">Perempuan</option>
          </select>
        </Field>
        <Field label="Status pernikahan">
          <select
            value={c.marital}
            onChange={(e) => {
              const marital = e.target.value as BipraInput['marital']
              set({ marital, orMarried: marital === 'unmarried' ? false : c.orMarried })
            }}
          >
            <option value="">Semua</option>
            <option value="married">Sudah menikah (termasuk duda/janda)</option>
            <option value="unmarried">Belum menikah</option>
          </select>
        </Field>
        <label className="check wide">
          <input
            type="checkbox"
            checked={c.orMarried}
            disabled={c.marital === 'unmarried'}
            onChange={(e) => set({ orMarried: e.target.checked })}
          />
          Juga semua yang sudah menikah, berapa pun umurnya (seperti PKB/WKI)
        </label>
        <fieldset className="color-picker wide">
          <legend>Warna</legend>
          {TYPE_COLORS.map((t) => (
            <label key={t.hex} className="color-option" title={t.name}>
              <input
                type="radio"
                name="color"
                value={t.hex}
                checked={c.color === t.hex}
                onChange={() => set({ color: t.hex })}
              />
              <span style={{ '--type-color': t.hex } as CSSProperties} aria-hidden="true" />
              <span className="sr-only">{t.name}</span>
            </label>
          ))}
        </fieldset>
        <p className="wide muted small rule-preview">
          Aturan: {bipraRule(c)}
          . Cocok dengan <b>{matching}</b> anggota saat ini
          {c.kind === 'main' && ' (kategori di atasnya tetap didahulukan)'}.
        </p>
      </FormShell>
    </Modal>
  )
}
