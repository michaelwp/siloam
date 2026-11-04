import { useCallback, useMemo, useState } from 'react'
import { kasApi, type KasBook, type KasEntry, type KasKind, type KasSummary, type Service } from '../api'
import { managesKas, useAuth } from '../auth'
import { KasTile } from '../components/KasStatus'
import { ConfirmDelete, Field, FormShell, Modal, PageHeader, Status } from '../components/ui'
import { kolomLabel, useData } from '../data'
import { fromInputDate, localToday, longDate, MONTHS, rupiah, toInputDate } from '../format'
import { useSchedule } from '../schedule'
import { useLoad } from '../useLoad'
import { xlsx, type Cell } from '../xlsx'

/**
 * Kas Kolom: each kolom's cash book. Admins and the kolom's treasurers
 * (Bendahara Kolom) record cash in (mostly an ibadah's offering) and cash
 * out; who sees the kolom's data may read it.
 */
export default function Kas() {
  const { kolom } = useData()
  return kolom === null ? <KasOverview /> : <KasKolom kolom={kolom} />
}

const summaryOfThisMonth = () => kasApi.summary(localToday().slice(0, 7))

/** "Semua kolom": every kolom's balance (the ones the user may read). */
function KasOverview() {
  const { setKolom } = useData()
  const { data, error, loading } = useLoad<KasSummary[]>('summary', summaryOfThisMonth)
  const total = (data ?? []).reduce((t, s) => ({ in: t.in + s.in, out: t.out + s.out }), { in: 0, out: 0 })
  return (
    <>
      <PageHeader title="Kas Kolom" subtitle="Saldo kas setiap kolom · pilih kolom untuk melihat atau mencatat kas" />
      <Status loading={loading} error={error} empty={!!data && !data.length} />
      {data && data.length > 0 && (
        <>
          <div className="stats">
            <KasTile label="Total kas masuk" value={total.in} />
            <KasTile label="Total kas keluar" value={total.out} />
            <KasTile label="Total saldo" value={total.in - total.out} />
          </div>
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Kolom</th>
                  <th className="num">Kas masuk</th>
                  <th className="num">Kas keluar</th>
                  <th className="num">Saldo</th>
                  <th className="num">Catatan</th>
                  <th>Terakhir</th>
                  <th>
                    <span className="sr-only">Aksi</span>
                  </th>
                </tr>
              </thead>
              <tbody>
                {data.map((s) => (
                  <tr key={s.kolom}>
                    <td className="strong">{kolomLabel(s.kolom)}</td>
                    <td className="num">{rupiah(s.in)}</td>
                    <td className="num">{rupiah(s.out)}</td>
                    <td className={`num strong${s.in - s.out < 0 ? ' negative' : ''}`}>{rupiah(s.in - s.out)}</td>
                    <td className="num">{s.count}</td>
                    <td>{s.last ? longDate(s.last) : '–'}</td>
                    <td className="actions">
                      <button className="btn small" onClick={() => setKolom(s.kolom)}>
                        Buka
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}
    </>
  )
}

/** "" = the whole book, else a month "2026-11". */
type Period = string

function periodLabel(p: Period) {
  if (!p) return 'sepanjang waktu'
  const [y, m] = p.split('-').map(Number)
  return `${MONTHS[m - 1]} ${y}`
}

type Dialog = { kind: 'add'; entryKind: KasKind } | { kind: 'edit' | 'delete'; entry: KasEntry } | null

/** One kolom's cash book for a month (or all of it), with the running balance. */
function KasKolom({ kolom }: { kolom: number }) {
  const { user } = useAuth()
  const { allServices, loading: scheduleLoading, error: scheduleError } = useSchedule()
  const loadBook = useCallback(() => kasApi.book(kolom), [kolom])
  const { data: book, error, status, loading, reload } = useLoad<KasBook>(String(kolom), loadBook)
  // without the schedule, a linked ibadah cannot be told from a deleted one
  const scheduleKnown = !scheduleLoading && !scheduleError
  const [period, setPeriod] = useState<Period>(() => localToday().slice(0, 7))
  const [dialog, setDialog] = useState<Dialog>(null)
  const canEdit = !!book?.canEdit && managesKas(user, kolom)
  const services = useMemo(() => new Map(allServices.map((s) => [s.id, s])), [allServices])

  const view = useMemo(() => (book ? bookView(book.entries, period) : null), [book, period])
  const done = async () => {
    reload()
    setDialog(null)
  }

  return (
    <>
      <PageHeader
        title={`Kas ${kolomLabel(kolom)}`}
        subtitle={
          canEdit ? 'Catat kas masuk (mis. persembahan ibadah) dan kas keluar beserta keterangannya' : 'Buku kas kolom'
        }
      >
        {canEdit && (
          <>
            <button className="btn primary" onClick={() => setDialog({ kind: 'add', entryKind: 'in' })}>
              + Kas Masuk
            </button>
            <button className="btn" onClick={() => setDialog({ kind: 'add', entryKind: 'out' })}>
              + Kas Keluar
            </button>
          </>
        )}
      </PageHeader>

      <fieldset className="filters no-print">
        <label className="date-filter">
          <span>Bulan</span>
          <input
            type="month"
            value={period}
            disabled={!period}
            onChange={(e) => setPeriod(e.target.value || localToday().slice(0, 7))}
          />
        </label>
        <label className="check">
          <input
            type="checkbox"
            checked={!period}
            onChange={(e) => setPeriod(e.target.checked ? '' : localToday().slice(0, 7))}
          />
          Semua bulan
        </label>
        {book && view && (
          <>
            <button type="button" className="link" onClick={() => downloadXlsx(kolom, period, view, services)}>
              Unduh Excel
            </button>
            <button type="button" className="link" onClick={() => downloadCsv(kolom, period, view, services)}>
              Unduh CSV
            </button>
          </>
        )}
      </fieldset>

      {status === 403 ? (
        <p className="status">Anda tidak dapat melihat kas {kolomLabel(kolom)}. Pilih kolom Anda di menu samping.</p>
      ) : (
        <Status loading={loading} error={error} />
      )}
      {book && view && (
        <>
          <div className="stats">
            {period && <KasTile label="Saldo awal" value={view.opening} note={`sebelum ${periodLabel(period)}`} />}
            <KasTile label="Kas masuk" value={view.in} note={periodLabel(period)} />
            <KasTile label="Kas keluar" value={view.out} note={periodLabel(period)} />
            <KasTile
              label={period ? 'Saldo akhir' : 'Saldo'}
              value={view.closing}
              note={period ? `akhir ${periodLabel(period)}` : undefined}
            />
          </div>
          {view.rows.length ? (
            <div className="table-wrap">
              <table className="kas-table">
                <thead>
                  <tr>
                    <th>Tanggal</th>
                    <th>Keterangan</th>
                    <th className="num">Masuk</th>
                    <th className="num">Keluar</th>
                    <th className="num">Saldo</th>
                    {canEdit && (
                      <th className="no-print">
                        <span className="sr-only">Aksi</span>
                      </th>
                    )}
                  </tr>
                </thead>
                <tbody>
                  {view.rows.map(({ entry: e, balance }) => (
                    <tr key={e.id}>
                      <td className="nowrap">{longDate(e.date)}</td>
                      <td>
                        <EntryText
                          entry={e}
                          service={e.serviceId ? services.get(e.serviceId) : undefined}
                          scheduleKnown={scheduleKnown}
                        />
                      </td>
                      <td className="num">{e.kind === 'in' ? rupiah(e.amount) : ''}</td>
                      <td className="num">{e.kind === 'out' ? rupiah(e.amount) : ''}</td>
                      <td className={`num strong${balance < 0 ? ' negative' : ''}`}>{rupiah(balance)}</td>
                      {canEdit && (
                        <td className="actions no-print">
                          <button className="link" onClick={() => setDialog({ kind: 'edit', entry: e })}>
                            Ubah
                          </button>
                          <button className="link danger" onClick={() => setDialog({ kind: 'delete', entry: e })}>
                            Hapus
                          </button>
                        </td>
                      )}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <p className="status">Belum ada catatan kas {period ? `pada ${periodLabel(period)}` : ''}.</p>
          )}
        </>
      )}

      {dialog?.kind === 'add' && (
        <KasForm kolom={kolom} kind={dialog.entryKind} onClose={() => setDialog(null)} onSaved={done} />
      )}
      {dialog?.kind === 'edit' && (
        <KasForm
          kolom={kolom}
          kind={dialog.entry.kind}
          entry={dialog.entry}
          onClose={() => setDialog(null)}
          onSaved={done}
        />
      )}
      {dialog?.kind === 'delete' && (
        <ConfirmDelete
          title="Hapus Catatan Kas"
          onClose={() => setDialog(null)}
          message={
            <>
              Hapus {dialog.entry.kind === 'in' ? 'kas masuk' : 'kas keluar'} <b>{rupiah(dialog.entry.amount)}</b>{' '}
              tanggal {longDate(dialog.entry.date)}?
            </>
          }
          onConfirm={async () => {
            await kasApi.remove(dialog.entry.id)
            await done()
          }}
        />
      )}
    </>
  )
}

/** The ibadah (its current name, or the one saved if it was deleted), the description and who recorded it. */
function EntryText({
  entry: e,
  service,
  scheduleKnown,
}: {
  entry: KasEntry
  service?: Service
  scheduleKnown: boolean
}) {
  const ibadah = service ? serviceName(service) : e.serviceLabel
  return (
    <>
      {ibadah && (
        <span className="strong block">
          {e.kind === 'in' ? 'Dari ' : 'Untuk '}
          {ibadah}
          {!service && e.serviceId && scheduleKnown && <span className="muted"> (jadwal sudah dihapus)</span>}
        </span>
      )}
      {e.description && <span className="block">{e.description}</span>}
      <span className="muted small block">
        dicatat {e.createdBy}
        {e.updatedBy && ` · diubah ${e.updatedBy}`}
      </span>
    </>
  )
}

/** "Ibadah Kolom · Kel. Umboh · 3 November 2026" */
function serviceName(s: Service) {
  return [`Ibadah ${s.typeName}`, s.host && `Kel. ${s.host}`, longDate(s.date)].filter(Boolean).join(' · ')
}

type BookView = {
  opening: number
  in: number
  out: number
  closing: number
  rows: { entry: KasEntry; balance: number }[]
}

/** The entries of the period with their running balance, and the period's totals. */
function bookView(entries: KasEntry[], period: Period): BookView {
  const v: BookView = { opening: 0, in: 0, out: 0, closing: 0, rows: [] }
  let balance = 0
  for (const e of entries) {
    const month = e.date.slice(0, 7)
    if (period && month > period) break // oldest first: the rest is later
    balance += e.kind === 'in' ? e.amount : -e.amount
    if (period && month < period) {
      v.opening = balance
      continue
    }
    if (e.kind === 'in') v.in += e.amount
    else v.out += e.amount
    v.rows.push({ entry: e, balance })
  }
  v.closing = balance
  return v
}

/** One exported line: an entry with its running balance, or the opening balance (entry null). */
type ExportRow = {
  date: string | null // YYYY-MM-DD
  kind: string
  ibadah: string
  description: string
  in: number | null
  out: number | null
  balance: number
  by: string
}

/** The lines of the book as shown (the period, with its opening balance). */
function exportRows(period: Period, view: BookView, services: Map<string, Service>): ExportRow[] {
  const rows: ExportRow[] = []
  if (period) {
    rows.push({
      date: null,
      kind: '',
      ibadah: '',
      description: 'Saldo awal',
      in: null,
      out: null,
      balance: view.opening,
      by: '',
    })
  }
  for (const { entry: e, balance } of view.rows) {
    const s = e.serviceId ? services.get(e.serviceId) : undefined
    rows.push({
      date: toInputDate(e.date),
      kind: e.kind === 'in' ? 'Masuk' : 'Keluar',
      ibadah: s ? serviceName(s) : e.serviceLabel,
      description: e.description,
      in: e.kind === 'in' ? e.amount : null,
      out: e.kind === 'out' ? e.amount : null,
      balance,
      by: e.createdBy,
    })
  }
  return rows
}

const EXPORT_HEADER = ['Tanggal', 'Jenis', 'Ibadah', 'Keterangan', 'Masuk', 'Keluar', 'Saldo', 'Dicatat oleh']

function exportName(kolom: number, period: Period) {
  return `kas-kolom-${kolom}${period ? `-${period}` : ''}`
}

/** Saves blob as a download named name. */
function save(blob: Blob, name: string) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = name
  a.click()
  // later: revoking at once can cancel the download in some browsers
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}

/**
 * A CSV cell, quoted. Text that a spreadsheet would run as a formula
 * (=, +, -, @, tab, CR first) gets a leading apostrophe: a description is
 * typed by users, the file is opened by others.
 */
function cell(v: string | number | null) {
  const s = typeof v === 'string' && /^[=+\-@\t\r]/.test(v) ? `'${v}` : String(v ?? '')
  return `"${s.replaceAll('"', '""')}"`
}

function downloadCsv(kolom: number, period: Period, view: BookView, services: Map<string, Service>) {
  const lines = [EXPORT_HEADER.map(cell).join(';')]
  for (const r of exportRows(period, view, services)) {
    lines.push([r.date, r.kind, r.ibadah, r.description, r.in, r.out, r.balance, r.by].map(cell).join(';'))
  }
  // a BOM so spreadsheet apps read it as UTF-8
  save(
    new Blob(['\uFEFF' + lines.join('\r\n')], { type: 'text/csv;charset=utf-8' }),
    `${exportName(kolom, period)}.csv`,
  )
}

const money = (v: number | null): Cell => (v === null ? null : { v, style: 'money' })
const total = (v: number): Cell => ({ v, style: 'moneyBold' })

/**
 * The book as an Excel file: a title, the header, the lines (real dates and
 * amounts, so they can be summed and sorted) and the period's totals.
 */
function downloadXlsx(kolom: number, period: Period, view: BookView, services: Map<string, Service>) {
  const rows: Cell[][] = [
    [{ v: `Kas ${kolomLabel(kolom)} — ${periodLabel(period)}`, style: 'bold' }],
    [],
    EXPORT_HEADER.map((h) => ({ v: h, style: 'bold' })),
    ...exportRows(period, view, services).map((r): Cell[] => [
      r.date ? new Date(`${r.date}T00:00:00Z`) : null,
      r.kind,
      r.ibadah,
      r.date ? r.description : { v: r.description, style: 'bold' },
      money(r.in),
      money(r.out),
      r.date ? money(r.balance) : total(r.balance),
      r.by,
    ]),
    [],
    [null, null, null, { v: 'Jumlah', style: 'bold' }, total(view.in), total(view.out), total(view.closing)],
  ]
  save(xlsx(`Kas ${kolomLabel(kolom)}`, rows, [14, 10, 42, 42, 15, 15, 16, 16]), `${exportName(kolom, period)}.xlsx`)
}

/** Adds or changes an entry: cash in (from an ibadah or another source) or cash out. */
function KasForm({
  kolom,
  kind: kind0,
  entry,
  onClose,
  onSaved,
}: {
  kolom: number
  kind: KasKind
  entry?: KasEntry
  onClose: () => void
  onSaved: () => Promise<void>
}) {
  const { allServices, loading: scheduleLoading, error: scheduleError } = useSchedule()
  const [kind, setKind] = useState<KasKind>(kind0)
  const [date, setDate] = useState(entry ? toInputDate(entry.date) : localToday())
  const [amount, setAmount] = useState(entry ? String(entry.amount) : '')
  const [fromService, setFromService] = useState(entry ? !!entry.serviceId : kind0 === 'in')
  const [serviceId, setServiceId] = useState(entry?.serviceId ?? '')
  const [description, setDescription] = useState(entry?.description ?? '')

  // the kolom's ibadah, the latest first (the ones still to come after those already held)
  const today = localToday()
  const choices = useMemo(() => {
    const own = allServices.filter((s) => s.kolom === kolom)
    const held = own.filter((s) => toInputDate(s.date) <= today).toSorted((a, b) => b.date.localeCompare(a.date))
    const coming = own.filter((s) => toInputDate(s.date) > today).toSorted((a, b) => a.date.localeCompare(b.date))
    return { held, coming }
  }, [allServices, kolom, today])
  // linked to an ibadah not in the schedule: deleted (or the schedule is not loaded: keep it selectable)
  const linkedMissing = !!entry?.serviceId && !allServices.some((s) => s.id === entry.serviceId)
  const missingNote = scheduleLoading || scheduleError ? '' : ' (jadwal sudah dihapus)'

  const what = kind === 'in' ? 'Sumber kas' : 'Digunakan untuk'
  return (
    <Modal title={entry ? 'Ubah Catatan Kas' : kind === 'in' ? 'Kas Masuk' : 'Kas Keluar'} onClose={onClose}>
      <FormShell
        onCancel={onClose}
        onSubmit={async () => {
          const n = Number(amount)
          if (!n) throw new Error('Isi jumlah lebih dari 0')
          if (fromService && !serviceId) throw new Error('Pilih ibadahnya')
          if (!fromService && !description.trim())
            throw new Error(kind === 'in' ? 'Tulis keterangan sumber kas' : 'Tulis keterangan penggunaan kas')
          const input = {
            kolom,
            kind,
            date: fromInputDate(date) ?? '',
            amount: n,
            serviceId: fromService ? serviceId : '',
            description,
          }
          if (entry) await kasApi.update(entry.id, input)
          else await kasApi.create(input)
          await onSaved()
        }}
      >
        <fieldset className="wide segmented" aria-label="Jenis">
          <button type="button" aria-pressed={kind === 'in'} onClick={() => setKind('in')}>
            Kas masuk
          </button>
          <button type="button" aria-pressed={kind === 'out'} onClick={() => setKind('out')}>
            Kas keluar
          </button>
        </fieldset>
        <Field label="Tanggal">
          <input type="date" required value={date} onChange={(e) => setDate(e.target.value)} />
        </Field>
        <Field label="Jumlah (Rp)">
          <input
            inputMode="numeric"
            required
            placeholder="0"
            value={amount ? Number(amount).toLocaleString('id-ID') : ''}
            onChange={(e) => setAmount(e.target.value.replace(/\D/g, '').replace(/^0+/, '').slice(0, 13))}
          />
        </Field>
        <Field label={what} wide>
          <select
            value={fromService ? 'ibadah' : 'other'}
            onChange={(e) => setFromService(e.target.value === 'ibadah')}
          >
            <option value="ibadah">
              {kind === 'in' ? 'Ibadah (mis. persembahan)' : 'Ibadah (mis. keperluan ibadah)'}
            </option>
            <option value="other">Lainnya (tulis di keterangan)</option>
          </select>
        </Field>
        {fromService && (
          <Field label="Ibadah" wide>
            <select
              required
              value={serviceId}
              onChange={(e) => {
                setServiceId(e.target.value)
                // the cash is usually counted on the day of the ibadah
                const s = allServices.find((x) => x.id === e.target.value)
                if (s && !entry) setDate(toInputDate(s.date))
              }}
            >
              <option value="">— pilih ibadah {kolomLabel(kolom)} —</option>
              {linkedMissing && (
                <option value={entry.serviceId!}>
                  {entry.serviceLabel}
                  {missingNote}
                </option>
              )}
              {choices.held.length > 0 && (
                <optgroup label="Sudah berlangsung">
                  {choices.held.map((s) => (
                    <option key={s.id} value={s.id}>
                      {serviceName(s)}
                    </option>
                  ))}
                </optgroup>
              )}
              {choices.coming.length > 0 && (
                <optgroup label="Akan datang">
                  {choices.coming.map((s) => (
                    <option key={s.id} value={s.id}>
                      {serviceName(s)}
                    </option>
                  ))}
                </optgroup>
              )}
            </select>
          </Field>
        )}
        <Field label={fromService ? 'Keterangan (opsional)' : 'Keterangan'} wide>
          <textarea
            rows={2}
            maxLength={300}
            required={!fromService}
            value={description}
            placeholder={
              kind === 'in'
                ? 'mis. Persembahan syukur, sumbangan Kel. …, iuran …'
                : 'mis. Konsumsi ibadah, bantuan duka Kel. …, alat tulis …'
            }
            onChange={(e) => setDescription(e.target.value)}
          />
        </Field>
      </FormShell>
    </Modal>
  )
}
