import { useMemo, useState, type CSSProperties } from 'react'
import { scheduleApi, serviceToInput, type Service, type ServiceInput, type WorshipType } from '../api'
import { useCanEdit } from '../auth'
import { KolomField } from '../components/forms'
import { ConfirmDelete, Field, FormShell, Modal, Status } from '../components/ui'
import { kolomLabel, useData } from '../data'
import { fromInputDate, MONTHS, toInputDate } from '../format'
import {
  clock,
  dayKey,
  feedUrl,
  googleCalendarUrl,
  icsUrl,
  serviceDayKey,
  serviceTitle,
  timeRange,
  useSchedule,
  WEEKDAYS,
} from '../schedule'

type View = 'month' | 'agenda'
type Dialog =
  | { kind: 'show'; service: Service }
  | { kind: 'day'; key: string }
  | { kind: 'add'; date: string | null }
  | { kind: 'edit'; service: Service }
  | { kind: 'delete'; service: Service }
  | { kind: 'subscribe' }
  | null

const MAX_PER_CELL = 3
const narrow = () => typeof window !== 'undefined' && window.matchMedia('(max-width: 700px)').matches

/** Google-Calendar-like schedule: a month grid or a schedule list, per kolom. */
export default function JadwalIbadah() {
  const { kolom } = useData()
  const canEdit = useCanEdit()
  const { services, allServices, types, loading, error, reload } = useSchedule()
  const [today] = useState(() => new Date())
  const [cursor, setCursor] = useState(() => ({ y: today.getFullYear(), m: today.getMonth() }))
  const [view, setView] = useState<View>(() => (narrow() ? 'agenda' : 'month'))
  const [hidden, setHidden] = useState<Set<string>>(new Set())
  const [dialog, setDialog] = useState<Dialog>(null)

  const visible = services.filter((s) => !hidden.has(s.typeId))
  const byDay = useMemo(() => {
    const map = new Map<string, Service[]>()
    for (const s of visible) {
      const k = serviceDayKey(s)
      map.set(k, [...(map.get(k) ?? []), s])
    }
    for (const [k, list] of map) map.set(k, list.toSorted((a, b) => a.time.localeCompare(b.time)))
    return map
  }, [visible])
  const todayKey = dayKey(today.getFullYear(), today.getMonth(), today.getDate())

  const move = (delta: number) =>
    setCursor(({ y, m }) => {
      const d = new Date(y, m + delta, 1)
      return { y: d.getFullYear(), m: d.getMonth() }
    })
  const toggleType = (id: string) =>
    setHidden((h) => {
      const next = new Set(h)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })

  return (
    <div className="cal">
      <aside className="cal-side no-print">
        {canEdit && (
          <button className="btn primary cal-create" onClick={() => setDialog({ kind: 'add', date: null })}>
            <span aria-hidden="true">+</span> Buat Jadwal
          </button>
        )}
        <MiniMonth
          cursor={cursor}
          todayKey={todayKey}
          byDay={byDay}
          onPick={(y, m) => setCursor({ y, m })}
          onMove={move}
        />
        <section className="cal-types" aria-labelledby="cal-types-title">
          <h3 id="cal-types-title">Jenis Ibadah</h3>
          <ul>
            {types.map((t) => (
              <li key={t.id}>
                <label className="cal-type">
                  <input
                    type="checkbox"
                    checked={!hidden.has(t.id)}
                    onChange={() => toggleType(t.id)}
                    style={{ '--type-color': t.color } as CSSProperties}
                  />
                  <span>{t.name}</span>
                </label>
              </li>
            ))}
          </ul>
        </section>
        <button className="btn cal-subscribe" onClick={() => setDialog({ kind: 'subscribe' })}>
          📅 Tambahkan ke Kalender Saya
        </button>
      </aside>

      <section className="cal-main">
        <header className="cal-toolbar">
          <button
            className="btn"
            onClick={() => setCursor({ y: today.getFullYear(), m: today.getMonth() })}
          >
            Hari ini
          </button>
          <div className="cal-nav">
            <button className="icon-btn" aria-label="Bulan sebelumnya" onClick={() => move(-1)}>
              ‹
            </button>
            <button className="icon-btn" aria-label="Bulan berikutnya" onClick={() => move(1)}>
              ›
            </button>
          </div>
          <h1 className="cal-title">
            {MONTHS[cursor.m]} {cursor.y}
            <small>Jadwal Ibadah · {kolomLabel(kolom)}</small>
          </h1>
          <fieldset className="segmented">
            <legend className="sr-only">Tampilan</legend>
            <button aria-pressed={view === 'month'} onClick={() => setView('month')}>
              Bulan
            </button>
            <button aria-pressed={view === 'agenda'} onClick={() => setView('agenda')}>
              Jadwal
            </button>
          </fieldset>
        </header>

        <Status loading={loading} error={error} />
        {!loading &&
          !error &&
          (view === 'month' ? (
            <MonthGrid
              cursor={cursor}
              todayKey={todayKey}
              byDay={byDay}
              onShow={(s) => setDialog({ kind: 'show', service: s })}
              onMore={(key) => setDialog({ kind: 'day', key })}
              onAdd={canEdit ? (key) => setDialog({ kind: 'add', date: key }) : undefined}
            />
          ) : (
            <Agenda
              cursor={cursor}
              todayKey={todayKey}
              byDay={byDay}
              onShow={(s) => setDialog({ kind: 'show', service: s })}
            />
          ))}
      </section>

      {dialog?.kind === 'show' && (
        <ServiceDetails
          service={dialog.service}
          canEdit={canEdit}
          onClose={() => setDialog(null)}
          onEdit={() => setDialog({ kind: 'edit', service: dialog.service })}
          onDelete={() => setDialog({ kind: 'delete', service: dialog.service })}
        />
      )}
      {dialog?.kind === 'day' && (
        <DayList
          dayKeyStr={dialog.key}
          services={byDay.get(dialog.key) ?? []}
          onClose={() => setDialog(null)}
          onShow={(s) => setDialog({ kind: 'show', service: s })}
        />
      )}
      {(dialog?.kind === 'add' || dialog?.kind === 'edit') && (
        <ServiceForm
          service={dialog.kind === 'edit' ? dialog.service : undefined}
          date={dialog.kind === 'add' ? dialog.date : null}
          types={types}
          people={[...new Set(allServices.flatMap((x) => [x.leader, x.mc]).filter(Boolean))].toSorted()}
          onClose={() => setDialog(null)}
          onSaved={reload}
        />
      )}
      {dialog?.kind === 'delete' && (
        <ConfirmDelete
          title="Hapus Jadwal"
          onClose={() => setDialog(null)}
          message={
            <>
              Hapus jadwal <b>Ibadah {dialog.service.typeName}</b> {serviceTitle(dialog.service)} pada{' '}
              {longDay(serviceDayKey(dialog.service))}?
            </>
          }
          onConfirm={async () => {
            await scheduleApi.deleteService(dialog.service.id)
            await reload()
            setDialog(null)
          }}
        />
      )}
      {dialog?.kind === 'subscribe' && <Subscribe kolom={kolom} onClose={() => setDialog(null)} />}
    </div>
  )
}

/** "Rabu, 21 Januari 2026" from "2026-01-21" */
function longDay(key: string) {
  const [y, m, d] = key.split('-').map(Number)
  const date = new Date(y, m - 1, d)
  return `${WEEKDAYS[date.getDay()]}, ${d} ${MONTHS[m - 1]} ${y}`
}

/** The 42 days (6 weeks, Sunday first) shown for a month. */
function monthDays(y: number, m: number) {
  const first = new Date(y, m, 1)
  const start = new Date(y, m, 1 - first.getDay())
  return Array.from({ length: 42 }, (_, i) => {
    const d = new Date(start.getFullYear(), start.getMonth(), start.getDate() + i)
    return { y: d.getFullYear(), m: d.getMonth(), d: d.getDate(), key: dayKey(d.getFullYear(), d.getMonth(), d.getDate()) }
  })
}

type Cursor = { y: number; m: number }

function MonthGrid({
  cursor,
  todayKey,
  byDay,
  onShow,
  onMore,
  onAdd,
}: {
  cursor: Cursor
  todayKey: string
  byDay: Map<string, Service[]>
  onShow: (s: Service) => void
  onMore: (key: string) => void
  onAdd?: (key: string) => void
}) {
  const days = monthDays(cursor.y, cursor.m)
  // drop a trailing week that lies entirely in the next month
  const weeks = days[35].m !== cursor.m ? 5 : 6
  return (
    <div className="cal-month" style={{ '--weeks': weeks } as CSSProperties}>
      {WEEKDAYS.map((w) => (
        <div key={w} className="cal-weekday" aria-hidden="true">
          {w.slice(0, 3)}
        </div>
      ))}
      {days.slice(0, weeks * 7).map((day) => {
        const list = byDay.get(day.key) ?? []
        const shown = list.length > MAX_PER_CELL ? list.slice(0, MAX_PER_CELL - 1) : list
        const outside = day.m !== cursor.m
        return (
          <div key={day.key} className={`cal-cell${outside ? ' outside' : ''}`}>
            {/* the empty part of a day creates a service there (signed in) */}
            {onAdd && (
              <button className="cal-cell-add" aria-label={`Buat jadwal ${longDay(day.key)}`} onClick={() => onAdd(day.key)} />
            )}
            <span className={`cal-date${day.key === todayKey ? ' today' : ''}`}>
              {day.d === 1 ? `${day.d} ${MONTHS[day.m].slice(0, 3)}` : day.d}
            </span>
            {shown.map((s) => (
              <button
                key={s.id}
                className="cal-event"
                style={{ '--type-color': s.typeColor } as CSSProperties}
                title={`${clock(s.time)} Ibadah ${s.typeName} · ${serviceTitle(s)}`}
                onClick={() => onShow(s)}
              >
                <span className="cal-dot" aria-hidden="true" />
                <span className="cal-event-time">{clock(s.time)}</span>
                <span className="cal-event-title">{serviceTitle(s)}</span>
              </button>
            ))}
            {list.length > shown.length && (
              <button className="cal-more" onClick={() => onMore(day.key)}>
                {list.length - shown.length} lainnya
              </button>
            )}
          </div>
        )
      })}
    </div>
  )
}

/** Schedule view: every day of the month that has a service, like Google's "Jadwal". */
function Agenda({
  cursor,
  todayKey,
  byDay,
  onShow,
}: {
  cursor: Cursor
  todayKey: string
  byDay: Map<string, Service[]>
  onShow: (s: Service) => void
}) {
  const prefix = `${cursor.y}-${String(cursor.m + 1).padStart(2, '0')}-`
  const keys = [...byDay.keys()].filter((k) => k.startsWith(prefix)).toSorted()
  if (keys.length === 0) return <p className="status">Tidak ada jadwal ibadah di bulan ini.</p>
  return (
    <ol className="cal-agenda">
      {keys.map((k) => {
        const [y, m, d] = k.split('-').map(Number)
        const wd = WEEKDAYS[new Date(y, m - 1, d).getDay()]
        return (
          <li key={k} className={`cal-agenda-day${k === todayKey ? ' today' : ''}${k < todayKey ? ' past' : ''}`}>
            <div className="cal-agenda-date">
              <span className="cal-agenda-num">{d}</span>
              <span className="cal-agenda-wd">
                {MONTHS[m - 1].slice(0, 3)}, {wd.slice(0, 3)}
              </span>
            </div>
            <ul>
              {byDay.get(k)!.map((s) => (
                <li key={s.id}>
                  <button className="cal-agenda-item" style={{ '--type-color': s.typeColor } as CSSProperties} onClick={() => onShow(s)}>
                    <span className="cal-dot" aria-hidden="true" />
                    <span className="cal-agenda-time">{timeRange(s)}</span>
                    <span className="cal-agenda-text">
                      <strong>{serviceTitle(s)}</strong>
                      <span className="muted">
                        Ibadah {s.typeName}
                        {s.leader && <> · Pemimpin: {s.leader}</>}
                      </span>
                    </span>
                  </button>
                </li>
              ))}
            </ul>
          </li>
        )
      })}
    </ol>
  )
}

function MiniMonth({
  cursor,
  todayKey,
  byDay,
  onPick,
  onMove,
}: {
  cursor: Cursor
  todayKey: string
  byDay: Map<string, Service[]>
  onPick: (y: number, m: number) => void
  onMove: (delta: number) => void
}) {
  return (
    <section className="mini-month" aria-label="Kalender kecil">
      <header>
        <strong>
          {MONTHS[cursor.m]} {cursor.y}
        </strong>
        <button className="icon-btn" aria-label="Bulan sebelumnya" onClick={() => onMove(-1)}>
          ‹
        </button>
        <button className="icon-btn" aria-label="Bulan berikutnya" onClick={() => onMove(1)}>
          ›
        </button>
      </header>
      <div className="mini-grid">
        {WEEKDAYS.map((w) => (
          <span key={w} className="mini-wd" aria-hidden="true">
            {w[0]}
          </span>
        ))}
        {monthDays(cursor.y, cursor.m).map((day) => (
          <button
            key={day.key}
            className={`mini-day${day.m !== cursor.m ? ' outside' : ''}${day.key === todayKey ? ' today' : ''}${byDay.has(day.key) ? ' has' : ''}`}
            onClick={() => onPick(day.y, day.m)}
            aria-label={`${longDay(day.key)}${byDay.has(day.key) ? ', ada ibadah' : ''}`}
          >
            {day.d}
          </button>
        ))}
      </div>
    </section>
  )
}

function ServiceDetails({
  service: s,
  canEdit,
  onClose,
  onEdit,
  onDelete,
}: {
  service: Service
  canEdit: boolean
  onClose: () => void
  onEdit: () => void
  onDelete: () => void
}) {
  const rows: [string, string][] = [
    ['Tuan Rumah', s.host ? `Kel. ${s.host}` : ''],
    ['Lokasi', s.location],
    ['Pemimpin Ibadah', s.leader],
    ['Pembawa Acara', s.mc],
    ['Kolom', String(s.kolom)],
    ['Keterangan', s.note],
  ]
  return (
    <Modal title={`Ibadah ${s.typeName}`} onClose={onClose}>
      <div className="event-details" style={{ '--type-color': s.typeColor } as CSSProperties}>
        <div className="event-head">
          <span className="event-swatch" aria-hidden="true" />
          <div>
            <h3>{serviceTitle(s)}</h3>
            <p className="muted">
              {longDay(serviceDayKey(s))} · {timeRange(s)} WITA
            </p>
          </div>
        </div>
        <dl className="event-rows">
          {rows
            .filter(([, v]) => v)
            .map(([k, v]) => (
              <div key={k}>
                <dt>{k}</dt>
                <dd>{v}</dd>
              </div>
            ))}
        </dl>
        <div className="event-add">
          <span className="muted small">Tambahkan ke kalender:</span>
          <a className="btn small" href={googleCalendarUrl(s)} target="_blank" rel="noopener noreferrer">
            Google Calendar
          </a>
          <a className="btn small" href={icsUrl(s)} download>
            Apple Calendar (.ics)
          </a>
        </div>
      </div>
      {canEdit && (
        <footer className="modal-foot">
          <button className="btn danger" onClick={onDelete}>
            Hapus
          </button>
          <button className="btn primary" onClick={onEdit}>
            Ubah
          </button>
        </footer>
      )}
    </Modal>
  )
}

function DayList({
  dayKeyStr,
  services,
  onClose,
  onShow,
}: {
  dayKeyStr: string
  services: Service[]
  onClose: () => void
  onShow: (s: Service) => void
}) {
  return (
    <Modal title={longDay(dayKeyStr)} onClose={onClose}>
      <ul className="day-list">
        {services.map((s) => (
          <li key={s.id}>
            <button className="cal-event" style={{ '--type-color': s.typeColor } as CSSProperties} onClick={() => onShow(s)}>
              <span className="cal-dot" aria-hidden="true" />
              <span className="cal-event-time">{clock(s.time)}</span>
              <span className="cal-event-title">
                Ibadah {s.typeName} · {serviceTitle(s)}
              </span>
            </button>
          </li>
        ))}
      </ul>
    </Modal>
  )
}

/** Create (service undefined, optionally on a date) or edit a service; the kolom is picked first. */
function ServiceForm({
  service,
  date,
  types,
  people,
  onClose,
  onSaved,
}: {
  service?: Service
  date: string | null
  types: WorshipType[]
  /** names already used as leader / MC, offered as suggestions */
  people: string[]
  onClose: () => void
  onSaved: () => Promise<void>
}) {
  const { kolom, allFamilies } = useData()
  const [s, setS] = useState<ServiceInput>(
    service
      ? serviceToInput(service)
      : {
          kolom: kolom ?? 0,
          typeId: types[0]?.id ?? '',
          date: date ? fromInputDate(date) : null,
          time: '19:00',
          host: '',
          location: '',
          leader: '',
          mc: '',
          note: '',
        },
  )
  const set = (patch: Partial<ServiceInput>) => setS((prev) => ({ ...prev, ...patch }))
  const off = !s.kolom
  const hosts = allFamilies.filter((f) => f.kolom === s.kolom).map((f) => f.name)

  return (
    <Modal title={service ? 'Ubah Jadwal Ibadah' : 'Buat Jadwal Ibadah'} onClose={onClose}>
      <FormShell
        onCancel={onClose}
        onSubmit={async () => {
          if (service) await scheduleApi.updateService(service.id, s)
          else await scheduleApi.createService(s)
          await onSaved()
          onClose()
        }}
      >
        <KolomField value={s.kolom} onChange={(k) => set({ kolom: k })} />
        <Field label="Jenis Ibadah" wide>
          <select required disabled={off} value={s.typeId} onChange={(e) => set({ typeId: e.target.value })}>
            <option value="">— pilih jenis ibadah —</option>
            {types.map((t) => (
              <option key={t.id} value={t.id}>
                {t.name}
              </option>
            ))}
          </select>
        </Field>
        <Field label="Tanggal">
          <input
            type="date"
            required
            disabled={off}
            value={toInputDate(s.date)}
            onChange={(e) => set({ date: fromInputDate(e.target.value) })}
          />
        </Field>
        <Field label="Jam (WITA)">
          <input type="time" required disabled={off} value={s.time} onChange={(e) => set({ time: e.target.value })} />
        </Field>
        <Field label="Tuan Rumah (Kel.)" wide>
          <input
            disabled={off}
            list="host-options"
            value={s.host}
            placeholder="mis. Umboh Riedel"
            onChange={(e) => set({ host: e.target.value })}
          />
          <datalist id="host-options">
            {hosts.map((h) => (
              <option key={h} value={h}>
                {h}
              </option>
            ))}
          </datalist>
        </Field>
        <Field label="Lokasi (jika bukan di rumah tuan rumah)" wide>
          <input disabled={off} value={s.location} onChange={(e) => set({ location: e.target.value })} />
        </Field>
        <Field label="Pemimpin Ibadah">
          <input disabled={off} list="people-options" value={s.leader} onChange={(e) => set({ leader: e.target.value })} />
        </Field>
        <Field label="Pembawa Acara">
          <input disabled={off} list="people-options" value={s.mc} onChange={(e) => set({ mc: e.target.value })} />
          <datalist id="people-options">
            {people.map((p) => (
              <option key={p} value={p}>
                {p}
              </option>
            ))}
          </datalist>
        </Field>
        <Field label="Keterangan" wide>
          <textarea disabled={off} rows={2} value={s.note} onChange={(e) => set({ note: e.target.value })} />
        </Field>
      </FormShell>
    </Modal>
  )
}

function Subscribe({ kolom, onClose }: { kolom: number | null; onClose: () => void }) {
  const url = feedUrl(kolom)
  const webcal = url.replace(/^https?:/, 'webcal:')
  const [copied, setCopied] = useState(false)
  const local = /^(localhost|127\.|\[::1\])/.test(window.location.hostname)
  return (
    <Modal title="Tambahkan ke Kalender Saya" onClose={onClose}>
      <div className="subscribe">
        <p>
          Berlangganan jadwal ibadah <b>{kolomLabel(kolom)}</b>: jadwal baru dan perubahan akan muncul otomatis di
          kalender Anda.
        </p>
        <div className="subscribe-actions">
          <a className="btn primary" href={`https://calendar.google.com/calendar/r?cid=${encodeURIComponent(webcal)}`} target="_blank" rel="noopener noreferrer">
            Google Calendar
          </a>
          <a className="btn primary" href={webcal}>
            Apple Calendar
          </a>
        </div>
        <label className="field">
          <span>Alamat kalender (untuk aplikasi lain)</span>
          <div className="copy-row">
            <input readOnly value={url} onFocus={(e) => e.target.select()} />
            <button
              type="button"
              className="btn"
              onClick={async () => {
                try {
                  await navigator.clipboard.writeText(url)
                  setCopied(true)
                } catch {
                  setCopied(false)
                }
              }}
            >
              {copied ? 'Disalin ✓' : 'Salin'}
            </button>
          </div>
        </label>
        {local && (
          <p className="muted small">
            Catatan: aplikasi ini sedang dibuka di alamat lokal. Google Calendar hanya dapat berlangganan dari alamat
            publik (setelah aplikasi di-deploy).
          </p>
        )}
        <p className="muted small">Ingin menambahkan satu ibadah saja? Klik ibadahnya di kalender.</p>
      </div>
    </Modal>
  )
}
