import { useState } from 'react'
import { api, familyToInput, memberToInput, type Family, type FamilyInput, type Member, type MemberInput } from '../api'
import { managesKolom, useAuth } from '../auth'
import { familyOption, useData } from '../data'
import { fromInputDate, GENDERS, MARITAL_STATUSES, RELATIONSHIPS, toInputDate } from '../format'
import { ConfirmDelete, Field, FormShell, Modal } from './ui'

/**
 * Kolom picker; forms ask for it first and keep the other fields off until
 * chosen. It offers only the koloms the user may change (a "kelola data
 * kolom" user: their own).
 */
export function KolomField({
  value,
  onChange,
  allowAll,
}: {
  value: number
  onChange: (k: number) => void
  /** offer "Semua kolom" (0): a service of a jenis linked to a BIPRA category */
  allowAll?: boolean
}) {
  const { user } = useAuth()
  // a "kelola data BIPRA" user's services may be in any kolom
  const koloms = useData().koloms.filter((k) => user?.role === 'bipra' || managesKolom(user, k.number))
  return (
    <Field label="Kolom" wide>
      <select required value={allowAll ? String(value) : value || ''} onChange={(e) => onChange(Number(e.target.value))}>
        {allowAll ? (
          <option value="0">Semua kolom (seluruh jemaat)</option>
        ) : (
          <option value="">— pilih kolom terlebih dahulu —</option>
        )}
        {koloms.map((k) => (
          <option key={k.id} value={k.number}>
            Kolom {k.number}
          </option>
        ))}
      </select>
    </Field>
  )
}

/** Create (family undefined) or edit a family. A new family starts in the viewed kolom, if any. */
export function FamilyForm({ family, onClose }: { family?: Family; onClose: () => void }) {
  const { reload, kolom } = useData()
  const [f, setF] = useState<FamilyInput>(
    family ? familyToInput(family) : { kolom: kolom ?? 0, name: '', note: '', domicile: '', weddingDate: null },
  )
  const set = (patch: Partial<FamilyInput>) => setF((prev) => ({ ...prev, ...patch }))
  const off = !f.kolom

  return (
    <Modal title={family ? 'Ubah Keluarga' : 'Tambah Keluarga'} onClose={onClose}>
      <FormShell
        onCancel={onClose}
        onSubmit={async () => {
          if (family) await api.updateFamily(family.id, f)
          else await api.createFamily(f)
          await reload()
          onClose()
        }}
      >
        <KolomField value={f.kolom} onChange={(k) => set({ kolom: k })} />
        <Field label="Nama Keluarga" wide>
          <input
            required
            disabled={off}
            value={f.name}
            onChange={(e) => set({ name: e.target.value })}
            placeholder="mis. Najoan Ismawi"
          />
        </Field>
        <Field label="Domisili">
          <input disabled={off} value={f.domicile} onChange={(e) => set({ domicile: e.target.value })} />
        </Field>
        <Field label="Keterangan">
          <input disabled={off} value={f.note} list="note-options" onChange={(e) => set({ note: e.target.value })} />
          <datalist id="note-options">
            <option value="Simpatisan">Simpatisan</option>
          </datalist>
        </Field>
        <Field label="Tanggal Pernikahan">
          <input
            type="date"
            disabled={off}
            value={toInputDate(f.weddingDate)}
            onChange={(e) => set({ weddingDate: fromInputDate(e.target.value) })}
          />
        </Field>
        {family && f.kolom !== family.kolom && (
          <p className="muted small wide">Keluarga ini akan dipindahkan ke Kolom {f.kolom} dan diberi nomor terakhir di sana.</p>
        )}
      </FormShell>
    </Modal>
  )
}

export function DeleteFamily({ family, onClose }: { family: Family; onClose: () => void }) {
  const { reload } = useData()
  return (
    <ConfirmDelete
      title="Hapus Keluarga"
      onClose={onClose}
      message={
        <>
          Hapus <b>Kel. {family.name}</b> beserta <b>{family.memberCount} anggota</b>-nya? Tindakan ini tidak dapat
          dibatalkan.
        </>
      }
      onConfirm={async () => {
        await api.deleteFamily(family.id)
        await reload()
        onClose()
      }}
    />
  )
}

function emptyMember(familyId: string): MemberInput {
  return {
    familyId,
    name: '',
    birthPlace: '',
    birthDate: null,
    gender: '',
    maritalStatus: 'Belum Menikah',
    relationship: '',
    baptized: null,
    confirmed: null,
  }
}

function boolValue(v: boolean | null) {
  return v === null ? '' : String(v)
}

function parseBool(v: string) {
  return v === '' ? null : v === 'true'
}

/** Create (member undefined) or edit a member. familyId preselects the family on create. */
export function MemberForm({
  member,
  familyId = '',
  onClose,
}: {
  member?: Member
  familyId?: string
  onClose: () => void
}) {
  const { allFamilies, kolom: viewed, reload } = useData()
  const [m, setM] = useState<MemberInput>(member ? memberToInput(member) : emptyMember(familyId))
  const set = (patch: Partial<MemberInput>) => setM((prev) => ({ ...prev, ...patch }))
  // the kolom is picked first; it narrows the families to choose from
  const [kolom, setKolom] = useState(
    () => allFamilies.find((f) => f.id === (member?.familyId ?? familyId))?.kolom ?? viewed ?? 0,
  )
  const choices = allFamilies.filter((f) => f.kolom === kolom)
  const off = !kolom

  return (
    <Modal title={member ? 'Ubah Anggota' : 'Tambah Anggota'} onClose={onClose}>
      <FormShell
        onCancel={onClose}
        onSubmit={async () => {
          if (member) await api.updateMember(member.id, m)
          else await api.createMember(m)
          await reload()
          onClose()
        }}
      >
        <KolomField
          value={kolom}
          onChange={(k) => {
            setKolom(k)
            if (!allFamilies.some((f) => f.id === m.familyId && f.kolom === k)) set({ familyId: '' })
          }}
        />
        <Field label="Keluarga" wide>
          <select required disabled={off} value={m.familyId} onChange={(e) => set({ familyId: e.target.value })}>
            <option value="">{choices.length || off ? '— pilih keluarga —' : '— belum ada keluarga di kolom ini —'}</option>
            {choices.map((f) => (
              <option key={f.id} value={f.id}>
                {familyOption(f, false)}
              </option>
            ))}
          </select>
        </Field>
        <Field label="Nama Lengkap" wide>
          <input required disabled={off} value={m.name} onChange={(e) => set({ name: e.target.value })} />
        </Field>
        <Field label="Tempat Lahir">
          <input disabled={off} value={m.birthPlace} onChange={(e) => set({ birthPlace: e.target.value })} />
        </Field>
        <Field label="Tanggal Lahir">
          <input
            type="date"
            disabled={off}
            value={toInputDate(m.birthDate)}
            onChange={(e) => set({ birthDate: fromInputDate(e.target.value) })}
          />
        </Field>
        <Field label="Jenis Kelamin">
          <select disabled={off} value={m.gender} onChange={(e) => set({ gender: e.target.value })}>
            <option value="">—</option>
            {GENDERS.map((g) => (
              <option key={g}>{g}</option>
            ))}
          </select>
        </Field>
        <Field label="Status">
          <select disabled={off} value={m.maritalStatus} onChange={(e) => set({ maritalStatus: e.target.value })}>
            <option value="">—</option>
            {MARITAL_STATUSES.map((s) => (
              <option key={s}>{s}</option>
            ))}
          </select>
        </Field>
        <Field label="Hubungan Keluarga">
          <input disabled={off} list="relationship-options" value={m.relationship} onChange={(e) => set({ relationship: e.target.value })} />
          <datalist id="relationship-options">
            {RELATIONSHIPS.map((r) => (
              <option key={r} value={r}>
                {r}
              </option>
            ))}
          </datalist>
        </Field>
        <Field label="Baptis">
          <select disabled={off} value={boolValue(m.baptized)} onChange={(e) => set({ baptized: parseBool(e.target.value) })}>
            <option value="">—</option>
            <option value="true">Baptis</option>
            <option value="false">Belum</option>
          </select>
        </Field>
        <Field label="Sidi">
          <select disabled={off} value={boolValue(m.confirmed)} onChange={(e) => set({ confirmed: parseBool(e.target.value) })}>
            <option value="">—</option>
            <option value="true">Sidi</option>
            <option value="false">Belum</option>
          </select>
        </Field>
        {member && (
          <Field label="No. Urut di Keluarga">
            <input type="number" min={1} value={m.order ?? ''} onChange={(e) => set({ order: Number(e.target.value) })} />
          </Field>
        )}
      </FormShell>
    </Modal>
  )
}

export function DeleteMember({ member, onClose }: { member: Member; onClose: () => void }) {
  const { reload } = useData()
  return (
    <ConfirmDelete
      title="Hapus Anggota"
      onClose={onClose}
      message={
        <>
          Hapus <b>{member.name}</b> dari Kel. {member.familyName}?
        </>
      }
      onConfirm={async () => {
        await api.deleteMember(member.id)
        await reload()
        onClose()
      }}
    />
  )
}

/** State helper for pages that open add/edit/delete dialogs for members. */
export type MemberDialog = { kind: 'add'; familyId?: string } | { kind: 'edit' | 'delete'; member: Member } | null

export function MemberDialogs({ dialog, onClose }: { dialog: MemberDialog; onClose: () => void }) {
  if (!dialog) return null
  if (dialog.kind === 'add') return <MemberForm familyId={dialog.familyId} onClose={onClose} />
  if (dialog.kind === 'edit') return <MemberForm member={dialog.member} onClose={onClose} />
  return <DeleteMember member={dialog.member} onClose={onClose} />
}

export type FamilyDialog = { kind: 'add' } | { kind: 'edit' | 'delete'; family: Family } | null

export function FamilyDialogs({ dialog, onClose }: { dialog: FamilyDialog; onClose: () => void }) {
  if (!dialog) return null
  if (dialog.kind === 'add') return <FamilyForm onClose={onClose} />
  if (dialog.kind === 'edit') return <FamilyForm family={dialog.family} onClose={onClose} />
  return <DeleteFamily family={dialog.family} onClose={onClose} />
}
