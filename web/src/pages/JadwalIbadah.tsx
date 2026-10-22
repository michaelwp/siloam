import { useMemo, useState, type CSSProperties } from 'react'
import { scheduleApi, serviceToInput, type Member, type Service, type ServiceInput, type WorshipType } from '../api'
import { managesAll, managesService, useAuth, useCanAddServices } from '../auth'
import { KolomField } from '../components/forms'
import { MonthGrid, MonthNav } from '../components/calendar'
import { ConfirmDelete, Field, FormShell, Modal, Status } from '../components/ui'
import { useData } from '../data'
import { byText, fromInputDate, MONTHS, toInputDate } from '../format'
import {
  clock,
  dayKey,
  longDay,
  monthDays,
  moveCursor,
  type Cursor,
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

const narrow = () => typeof window !== 'undefined' && window.matchMedia('(max-width: 700px)').matches

/** Google-Calendar-like schedule: a month grid or a schedule list, per kolom. */
export default function JadwalIbadah() {
  const { kolom, filterLabel, bipraFilter } = useData()
  const canEdit = useCanAddServices()
  const { user } = useAuth()
  const { services, types, loading, error, reload } = useSchedule()
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
  // services of today and before are held: nobody adds, changes or deletes them (the server agrees)
  const tomorrowKey = dayKey(today.getFullYear(), today.getMonth(), today.getDate() + 1)
  const open = (key: string) => key > todayKey

  const move = (delta: number) => setCursor((c) => moveCursor(c, delta))
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
        <MonthNav cursor={cursor} today={today} subtitle={`Jadwal Ibadah · ${filterLabel}`} onChange={setCursor}>
          <fieldset className="segmented">
            <legend className="sr-only">Tampilan</legend>
            <button aria-pressed={view === 'month'} onClick={() => setView('month')}>
              Bulan
            </button>
            <button aria-pressed={view === 'agenda'} onClick={() => setView('agenda')}>
              Jadwal
            </button>
          </fieldset>
        </MonthNav>

        <Status loading={loading} error={error} />
        {!loading &&
          !error &&
          (view === 'month' ? (
            <MonthGrid
              cursor={cursor}
              todayKey={todayKey}
              byDay={byDay}
              itemKey={(s) => s.id}
              renderItem={(s) => (
                <button
                  className="cal-event"
                  style={{ '--type-color': s.typeColor } as CSSProperties}
                  title={`${clock(s.time)} Ibadah ${s.typeName} · ${serviceTitle(s)}`}
                  onClick={() => setDialog({ kind: 'show', service: s })}
                >
                  <span className="cal-dot" aria-hidden="true" />
                  <span className="cal-event-time">{clock(s.time)}</span>
                  <span className="cal-event-title">{serviceTitle(s)}</span>
                </button>
              )}
              onMore={(key) => setDialog({ kind: 'day', key })}
              onAdd={canEdit ? (key) => setDialog({ kind: 'add', date: key }) : undefined}
              addable={open}
              addLabel="Buat jadwal"
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
          manages={managesService(user, dialog.service, types)}
          locked={!open(serviceDayKey(dialog.service))}
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
          minDate={tomorrowKey}
          types={types}
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
      {dialog?.kind === 'subscribe' && (
        <Subscribe
          kolom={kolom}
          label={filterLabel}
          // a BIPRA category: the jenis linked to it; else the jenis left ticked
          types={
            bipraFilter
              ? types.filter((t) => t.bipraId === bipraFilter.id && !hidden.has(t.id))
              : hidden.size
                ? types.filter((t) => !hidden.has(t.id))
                : null
          }
          onClose={() => setDialog(null)}
        />
      )}
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
                        {s.leader && <> · Khadim: {s.leader}</>}
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
  manages,
  locked,
  onClose,
  onEdit,
  onDelete,
}: {
  service: Service
  /** the user may change this service, unless it is `locked` (today or before) */
  manages: boolean
  locked: boolean
  onClose: () => void
  onEdit: () => void
  onDelete: () => void
}) {
  const rows: [string, string][] = [
    ['Tuan Rumah', s.host ? `Kel. ${s.host}` : ''],
    ['Lokasi', s.location],
    ['Khadim', s.leader],
    ['Pembawa Acara', s.mc],
    ['Kolom', s.kolom ? String(s.kolom) : 'Semua kolom'],
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
      {manages && locked && (
        <p className="muted small locked-note">Jadwal hari ini atau sebelumnya tidak dapat diubah atau dihapus.</p>
      )}
      {manages && !locked && (
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
  minDate,
  types,
  onClose,
  onSaved,
}: {
  service?: Service
  date: string | null
  /** the first day a service may be on ("2026-10-23": tomorrow) */
  minDate: string
  types: WorshipType[]
  onClose: () => void
  onSaved: () => Promise<void>
}) {
  const { kolom, allFamilies, allMembers } = useData()
  const { user } = useAuth()
  // a "kelola data BIPRA" user: only their category's jenis
  const choices = user?.role === 'bipra' ? types.filter((t) => t.bipraId && t.bipraId === user.bipraId) : types
  const [s, setS] = useState<ServiceInput>(
    service
      ? serviceToInput(service)
      : {
          kolom: user?.role === 'bipra' ? 0 : (kolom ?? 0),
          typeId: choices[0]?.id ?? '',
          date: date ? fromInputDate(date) : null,
          time: '19:00',
          host: '',
          location: '',
          leader: '',
          leaderId: null,
          mc: '',
          mcId: null,
          note: '',
        },
  )
  const set = (patch: Partial<ServiceInput>) => setS((prev) => ({ ...prev, ...patch }))
  // a jenis linked to a BIPRA category may be for the whole congregation (no kolom)
  const linked = !!types.find((t) => t.id === s.typeId)?.bipraId
  const off = !s.typeId || (!s.kolom && !linked)
  const hosts = allFamilies.filter((f) => !s.kolom || f.kolom === s.kolom).map((f) => f.name)

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
        <Field label="Jenis Ibadah" wide>
          <select required value={s.typeId} onChange={(e) => set({ typeId: e.target.value })}>
            <option value="">— pilih jenis ibadah —</option>
            {choices.map((t) => (
              <option key={t.id} value={t.id}>
                {t.name}
              </option>
            ))}
          </select>
        </Field>
        {user?.role === 'bipra' && !choices.length && (
          <p className="wide form-error flush">
            Belum ada jenis ibadah yang terhubung ke kategori BIPRA Anda. Minta admin menghubungkannya di Jenis Ibadah.
          </p>
        )}
        {/* "Semua kolom": not for a kolom user (they change their kolom's services only) */}
        <KolomField
          value={s.kolom}
          allowAll={linked && (managesAll(user) || user?.role === 'bipra')}
          onChange={(k) => set({ kolom: k })}
        />
        <Field label="Tanggal">
          <input
            type="date"
            required
            disabled={off}
            min={minDate}
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
        <PersonField
          label="Khadim"
          person={{ id: s.leaderId, name: s.leader }}
          disabled={off}
          kolom={s.kolom}
          members={allMembers}
          onChange={(p) => set({ leaderId: p.id, leader: p.name })}
        />
        <PersonField
          label="Pembawa Acara"
          person={{ id: s.mcId, name: s.mc }}
          disabled={off}
          kolom={s.kolom}
          members={allMembers}
          onChange={(p) => set({ mcId: p.id, mc: p.name })}
        />
        <Field label="Keterangan" wide>
          <textarea disabled={off} rows={2} value={s.note} onChange={(e) => set({ note: e.target.value })} />
        </Field>
      </FormShell>
    </Modal>
  )
}

/** types: the jenis left ticked in the filter, or null when all are shown. */
function Subscribe({
  kolom,
  label,
  types,
  onClose,
}: {
  kolom: number | null
  label: string
  types: WorshipType[] | null
  onClose: () => void
}) {
  const url = feedUrl(kolom, types?.map((t) => t.id))
  const webcal = url.replace(/^https?:/, 'webcal:')
  const [copy, setCopy] = useState<'idle' | 'done' | 'manual'>('idle')
  const local = /^(localhost|127\.|\[::1\])/.test(window.location.hostname)
  return (
    <Modal title="Tambahkan ke Kalender Saya" onClose={onClose}>
      <div className="subscribe">
        <p>
          Berlangganan jadwal ibadah <b>{label}</b>
          {types && types.length > 0 && (
            <>
              {' '}
              (hanya <b>{types.map((t) => t.name).join(', ')}</b>)
            </>
          )}
          : jadwal baru dan perubahan akan muncul otomatis di kalender Anda.
        </p>
        {/* a category without linked jenis: a feed without types would carry every service */}
        {types?.length === 0 ? (
          <p className="form-error flush">Belum ada jenis ibadah untuk pilihan ini, jadi belum ada yang dapat ditambahkan ke kalender.</p>
        ) : (
          <>
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
                <input id="feed-url" readOnly value={url} onFocus={(e) => e.target.select()} />
                <button
                  type="button"
                  className="btn"
                  onClick={async () => {
                    try {
                      // needs HTTPS (or localhost); elsewhere fall back to a selected field
                      await navigator.clipboard.writeText(url)
                      setCopy('done')
                    } catch {
                      document.querySelector<HTMLInputElement>('#feed-url')?.select()
                      setCopy('manual')
                    }
                  }}
                >
                  {copy === 'done' ? 'Disalin ✓' : 'Salin'}
                </button>
              </div>
            </label>
            {copy === 'manual' && (
              <output className="form-error copy-hint">
                Browser tidak mengizinkan menyalin otomatis. Alamat sudah dipilih, tekan Ctrl+C (⌘C di Mac) untuk menyalin.
              </output>
            )}
          </>
        )}
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

const OTHER = '__other__'

/** A member (id) or someone from outside (id null, name only). */
type Person = { id: string | null; name: string }

/**
 * Khadim / pembawa acara: picked from members who are baptized and
 * sidi (this kolom first) and saved as a link to the member, or typed in as
 * another name (e.g. a pastor from outside the congregation). A saved outside
 * name opens in the text box; a linked member who no longer qualifies stays
 * listed so the service keeps them.
 */
function PersonField({
  label,
  person,
  disabled,
  kolom,
  members,
  onChange,
}: {
  label: string
  person: Person
  disabled: boolean
  kolom: number
  members: Member[]
  onChange: (p: Person) => void
}) {
  const eligible = members.filter((m) => m.baptized === true && m.confirmed === true).toSorted((a, b) => byText(a.name, b.name))
  const here = eligible.filter((m) => m.kolom === kolom)
  const elsewhere = eligible.filter((m) => m.kolom !== kolom)
  const linked = person.id ? members.find((m) => m.id === person.id) : undefined
  const current = linked && !eligible.includes(linked) ? linked : undefined
  // 'saved': editing a saved outside name; 'new': the user chose "+ Nama lain"
  const [typing, setTyping] = useState<false | 'saved' | 'new'>(() =>
    person.name !== '' && !linked ? 'saved' : false,
  )
  // same-name members are told apart by their family
  const counts = new Map<string, number>()
  for (const m of eligible) counts.set(m.name, (counts.get(m.name) ?? 0) + 1)
  const optionLabel = (m: Member, withKolom: boolean) =>
    `${m.name}${counts.get(m.name)! > 1 ? ` (Kel. ${m.familyName})` : ''}${withKolom ? ` · Kolom ${m.kolom}` : ''}`

  if (typing) {
    return (
      <Field label={`${label} (nama lain)`}>
        <div className="copy-row">
          <input
            disabled={disabled}
            value={person.name}
            placeholder="mis. Pdt. …"
            // eslint-disable-next-line jsx-a11y/no-autofocus -- the user just asked to type a name
            autoFocus={typing === 'new'}
            onChange={(e) => onChange({ id: null, name: e.target.value })}
          />
          <button
            type="button"
            className="btn"
            disabled={disabled}
            onClick={() => {
              onChange({ id: null, name: '' })
              setTyping(false)
            }}
          >
            Pilih anggota
          </button>
        </div>
      </Field>
    )
  }
  return (
    <Field label={label}>
      <select
        disabled={disabled}
        value={person.id ?? ''}
        onChange={(e) => {
          const v = e.target.value
          if (v === OTHER) {
            onChange({ id: null, name: '' })
            setTyping('new')
            return
          }
          const m = members.find((x) => x.id === v)
          onChange(m ? { id: m.id, name: m.name } : { id: null, name: '' })
        }}
      >
        <option value="">— pilih {label.toLowerCase()} —</option>
        {current && <option value={current.id}>{current.name} (tidak lagi baptis &amp; sidi)</option>}
        {here.length > 0 && (
          <optgroup label={`Anggota Kolom ${kolom} (baptis & sidi)`}>
            {here.map((m) => (
              <option key={m.id} value={m.id}>
                {optionLabel(m, false)}
              </option>
            ))}
          </optgroup>
        )}
        {elsewhere.length > 0 && (
          <optgroup label="Anggota kolom lain (baptis & sidi)">
            {elsewhere.map((m) => (
              <option key={m.id} value={m.id}>
                {optionLabel(m, true)}
              </option>
            ))}
          </optgroup>
        )}
        <option value={OTHER}>+ Nama lain (bukan anggota jemaat)…</option>
      </select>
    </Field>
  )
}
