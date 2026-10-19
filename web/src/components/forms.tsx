import { useState } from 'react'
import { api, familyToInput, memberToInput, type Family, type FamilyInput, type Member, type MemberInput } from '../api'
import { useData } from '../data'
import { fromInputDate, GENDERS, MARITAL_STATUSES, RELATIONSHIPS, toInputDate } from '../format'
import { ConfirmDelete, Field, FormShell, Modal } from './ui'

const emptyFamily: FamilyInput = { name: '', note: '', domicile: '', weddingDate: null }

/** Create (family undefined) or edit a family. */
export function FamilyForm({ family, onClose }: { family?: Family; onClose: () => void }) {
  const { reload } = useData()
  const [f, setF] = useState<FamilyInput>(family ? familyToInput(family) : emptyFamily)
  const set = (patch: Partial<FamilyInput>) => setF((prev) => ({ ...prev, ...patch }))

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
        <Field label="Nama Keluarga" wide>
          <input required value={f.name} onChange={(e) => set({ name: e.target.value })} placeholder="mis. Najoan Ismawi" />
        </Field>
        <Field label="Domisili">
          <input value={f.domicile} onChange={(e) => set({ domicile: e.target.value })} />
        </Field>
        <Field label="Keterangan">
          <input value={f.note} list="note-options" onChange={(e) => set({ note: e.target.value })} />
          <datalist id="note-options">
            <option value="Simpatisan">Simpatisan</option>
          </datalist>
        </Field>
        <Field label="Tanggal Pernikahan">
          <input type="date" value={toInputDate(f.weddingDate)} onChange={(e) => set({ weddingDate: fromInputDate(e.target.value) })} />
        </Field>
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
  const { families, reload } = useData()
  const [m, setM] = useState<MemberInput>(member ? memberToInput(member) : emptyMember(familyId))
  const set = (patch: Partial<MemberInput>) => setM((prev) => ({ ...prev, ...patch }))

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
        <Field label="Nama Lengkap" wide>
          <input required value={m.name} onChange={(e) => set({ name: e.target.value })} />
        </Field>
        <Field label="Keluarga" wide>
          <select required value={m.familyId} onChange={(e) => set({ familyId: e.target.value })}>
            <option value="">— pilih keluarga —</option>
            {families.map((f) => (
              <option key={f.id} value={f.id}>
                {f.no}. Kel. {f.name}
              </option>
            ))}
          </select>
        </Field>
        <Field label="Tempat Lahir">
          <input value={m.birthPlace} onChange={(e) => set({ birthPlace: e.target.value })} />
        </Field>
        <Field label="Tanggal Lahir">
          <input type="date" value={toInputDate(m.birthDate)} onChange={(e) => set({ birthDate: fromInputDate(e.target.value) })} />
        </Field>
        <Field label="Jenis Kelamin">
          <select value={m.gender} onChange={(e) => set({ gender: e.target.value })}>
            <option value="">—</option>
            {GENDERS.map((g) => (
              <option key={g}>{g}</option>
            ))}
          </select>
        </Field>
        <Field label="Status">
          <select value={m.maritalStatus} onChange={(e) => set({ maritalStatus: e.target.value })}>
            <option value="">—</option>
            {MARITAL_STATUSES.map((s) => (
              <option key={s}>{s}</option>
            ))}
          </select>
        </Field>
        <Field label="Hubungan Keluarga">
          <input list="relationship-options" value={m.relationship} onChange={(e) => set({ relationship: e.target.value })} />
          <datalist id="relationship-options">
            {RELATIONSHIPS.map((r) => (
              <option key={r} value={r}>
                {r}
              </option>
            ))}
          </datalist>
        </Field>
        <Field label="Baptis">
          <select value={boolValue(m.baptized)} onChange={(e) => set({ baptized: parseBool(e.target.value) })}>
            <option value="">—</option>
            <option value="true">Baptis</option>
            <option value="false">Belum</option>
          </select>
        </Field>
        <Field label="Sidi">
          <select value={boolValue(m.confirmed)} onChange={(e) => set({ confirmed: parseBool(e.target.value) })}>
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
