import { useState } from 'react'
import { Link } from 'react-router-dom'
import { FamilyDialogs, type FamilyDialog } from '../components/forms'
import { ActionsHeader, Badge, EditOnly, PageHeader, RowActions, Search, Status } from '../components/ui'
import { useFullView } from '../auth'
import { useData } from '../data'
import { matches } from '../format'

export default function DaftarKeluarga() {
  const full = useFullView()
  const { families, members, kolom, filterLabel, loading, error } = useData()
  const [q, setQ] = useState('')
  const [dialog, setDialog] = useState<FamilyDialog>(null)
  const rows = families.filter((f) => matches(q, f.name, f.domicile, f.note))

  return (
    <>
      <PageHeader
        title="Daftar Keluarga"
        subtitle={`${filterLabel} · ${families.length} keluarga · ${members.length} jiwa`}
      >
        <Search value={q} onChange={setQ} placeholder="Cari keluarga / domisili…" />
        <EditOnly>
          <button className="btn primary" onClick={() => setDialog({ kind: 'add' })}>
            + Tambah Keluarga
          </button>
        </EditOnly>
      </PageHeader>

      <Status loading={loading} error={error} empty={!families.length} />
      {rows.length > 0 && (
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                {kolom === null && <th className="num">Kolom</th>}
                <th className="num">No</th>
                <th>Nama Keluarga</th>
                <th>Keterangan</th>
                {full && <th>Domisili</th>}
                <th className="num">Jiwa</th>
                <ActionsHeader />
              </tr>
            </thead>
            <tbody>
              {rows.map((f) => (
                <tr key={f.id}>
                  {kolom === null && <td className="num">{f.kolom}</td>}
                  <td className="num">{f.no}</td>
                  <td>
                    <Link to={`/database#${f.id}`} className="strong">
                      Kel. {f.name}
                    </Link>
                  </td>
                  <td>{f.note && <Badge tone="amber">{f.note}</Badge>}</td>
                  {full && <td>{f.domicile}</td>}
                  <td className="num">{f.memberCount}</td>
                  <RowActions
                    onEdit={() => setDialog({ kind: 'edit', family: f })}
                    onDelete={() => setDialog({ kind: 'delete', family: f })}
                  />
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      <FamilyDialogs dialog={dialog} onClose={() => setDialog(null)} />
    </>
  )
}
