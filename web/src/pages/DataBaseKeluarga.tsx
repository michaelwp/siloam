import { useEffect, useMemo, useState } from 'react'
import { useLocation } from 'react-router-dom'
import type { Member } from '../api'
import { FamilyDialogs, MemberDialogs, type FamilyDialog, type MemberDialog } from '../components/forms'
import { ActionsHeader, Badge, EditOnly, PageHeader, RowActions, Search, Status } from '../components/ui'
import { useFullView } from '../auth'
import { kolomLabel, useData } from '../data'
import { birthLine, dayMonth, longDate, matches, triState } from '../format'

export default function DataBaseKeluarga() {
  const full = useFullView()
  const { families, members, kolom, filterLabel, loading, error } = useData()
  const [q, setQ] = useState('')
  const [memberDialog, setMemberDialog] = useState<MemberDialog>(null)
  const [familyDialog, setFamilyDialog] = useState<FamilyDialog>(null)
  const { hash } = useLocation()

  const byFamily = useMemo(() => {
    const map = new Map<string, Member[]>()
    for (const m of members) map.set(m.familyId, [...(map.get(m.familyId) ?? []), m])
    return map
  }, [members])

  // scroll to the family linked from "Daftar Keluarga"
  useEffect(() => {
    if (hash && !loading) document.getElementById(hash.slice(1))?.scrollIntoView({ block: 'start' })
  }, [hash, loading])

  const rows = families.filter(
    (f) => matches(q, f.name) || (byFamily.get(f.id) ?? []).some((m) => matches(q, m.name)),
  )

  return (
    <>
      <PageHeader title="Data Base Keluarga" subtitle={`${filterLabel} · data lengkap setiap keluarga dan anggotanya`}>
        <Search value={q} onChange={setQ} placeholder="Cari keluarga / anggota…" />
        <EditOnly>
          <button className="btn primary" onClick={() => setFamilyDialog({ kind: 'add' })}>
            + Tambah Keluarga
          </button>
        </EditOnly>
      </PageHeader>

      <Status loading={loading} error={error} empty={!families.length} />
      <div className="family-list">
        {rows.map((f) => {
          const ms = byFamily.get(f.id) ?? []
          return (
            <section key={f.id} id={f.id} className={`card family ${hash === `#${f.id}` ? 'target' : ''}`}>
              <header className="family-head">
                <div>
                  <h2>
                    <span className="muted">{f.no}.</span> Kel. {f.name}{' '}
                    {kolom === null && <Badge>{kolomLabel(f.kolom)}</Badge>} {f.note && <Badge tone="amber">{f.note}</Badge>}
                  </h2>
                  <p className="muted">
                    {full ? (
                      <>
                        Tanggal Pernikahan: {longDate(f.weddingDate)}
                        {f.domicile && <> · Domisili: {f.domicile}</>}
                      </>
                    ) : (
                      <>HUT Pernikahan: {dayMonth(f.weddingDay, f.weddingMonth)}</>
                    )}
                  </p>
                </div>
                <EditOnly>
                  <div className="page-actions no-print">
                    <button className="btn small" onClick={() => setMemberDialog({ kind: 'add', familyId: f.id })}>
                      + Anggota
                    </button>
                    <button className="btn small" onClick={() => setFamilyDialog({ kind: 'edit', family: f })}>
                      Ubah
                    </button>
                    <button className="btn small danger" onClick={() => setFamilyDialog({ kind: 'delete', family: f })}>
                      Hapus
                    </button>
                  </div>
                </EditOnly>
              </header>
              {ms.length > 0 ? (
                <div className="table-wrap flat">
                  <table>
                    <thead>
                      <tr>
                        <th className="num">No</th>
                        <th>Nama Lengkap</th>
                        <th>{full ? 'Tempat, Tanggal Lahir' : 'Ulang Tahun'}</th>
                        <th>Jenis Kelamin</th>
                        <th>Status</th>
                        {full && <th>Hubungan Keluarga</th>}
                        <th>Baptis</th>
                        <th>Sidi</th>
                        <ActionsHeader />
                      </tr>
                    </thead>
                    <tbody>
                      {ms.map((m) => (
                        <tr key={m.id}>
                          <td className="num">{m.order}</td>
                          <td className="strong">{m.name}</td>
                          <td>{full ? birthLine(m) : dayMonth(m.birthDay, m.birthMonth)}</td>
                          <td>{m.gender}</td>
                          <td>{m.maritalStatus}</td>
                          {full && <td>{m.relationship}</td>}
                          <td>{triState(m.baptized, 'Baptis')}</td>
                          <td>{triState(m.confirmed, 'Sidi')}</td>
                          <RowActions
                            onEdit={() => setMemberDialog({ kind: 'edit', member: m })}
                            onDelete={() => setMemberDialog({ kind: 'delete', member: m })}
                          />
                        </tr>
                      ))}
                    </tbody>
                    <tfoot>
                      <tr>
                        <td colSpan={9} className="muted">
                          {ms.length} jiwa
                        </td>
                      </tr>
                    </tfoot>
                  </table>
                </div>
              ) : (
                <p className="status">Belum ada anggota.</p>
              )}
            </section>
          )
        })}
      </div>
      <MemberDialogs dialog={memberDialog} onClose={() => setMemberDialog(null)} />
      <FamilyDialogs dialog={familyDialog} onClose={() => setFamilyDialog(null)} />
    </>
  )
}
