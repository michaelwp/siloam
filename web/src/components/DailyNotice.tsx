import { useEffect, useRef, useState, type CSSProperties, type ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { useData } from '../data'
import { byText, daysUntil, MONTHS, parts } from '../format'
import { clock, serviceDayKey, serviceTitle, shortDay, upcomingServices, useSchedule, WEEKDAYS } from '../schedule'

// Remembered per browser (nothing is sent to the server):
//  - "Lewati" hides it for the rest of this tab session (it comes back the next
//    time the app is opened);
//  - "Jangan tampilkan lagi hari ini" hides it until tomorrow.
// (The keys date from when this showed only birthdays.)
const HIDE_TODAY_KEY = 'birthday-toast-hidden-on'
const SKIP_KEY = 'birthday-toast-skipped-on'

/** Services of the next days shown under "Ibadah Minggu Ini" (after today). */
const WEEK_DAYS = 6

function storageKey(d: Date) {
  return `${d.getFullYear()}-${d.getMonth() + 1}-${d.getDate()}`
}

function read(storage: () => Storage, key: string) {
  try {
    return storage().getItem(key)
  } catch {
    return null // storage blocked (private mode, policy): just show it
  }
}

function write(storage: () => Storage, key: string, value: string) {
  try {
    storage().setItem(key, value)
  } catch {
    // storage blocked: the choice lasts until the page is reloaded
  }
}

/**
 * A centered modal shown once the data has loaded, on every page (visitors and
 * signed-in users), for the viewed kolom: today's services, the services of
 * the coming week and today's birthdays. The page behind is blurred and inert
 * (native modal <dialog>) until it is closed. Nothing to tell → not shown.
 */
export function DailyNotice() {
  const { members, loading } = useData()
  const { services, loading: scheduleLoading } = useSchedule()
  const [today] = useState(() => new Date())
  const key = storageKey(today)
  const [hidden, setHidden] = useState(
    () => read(() => localStorage, HIDE_TODAY_KEY) === key || read(() => sessionStorage, SKIP_KEY) === key,
  )
  const [dontShowToday, setDontShowToday] = useState(false)

  const { today: servicesToday, week: servicesWeek } = upcomingServices(services, today, WEEK_DAYS)
  const birthdays = members
    .filter((m) => m.birthMonth && m.birthDay && daysUntil(m.birthMonth, m.birthDay, today) === 0)
    .toSorted((a, b) => byText(a.name, b.name))

  const nothing = servicesToday.length + servicesWeek.length + birthdays.length === 0
  if (hidden || loading || scheduleLoading || nothing) return null

  const close = () => {
    write(() => sessionStorage, SKIP_KEY, key)
    if (dontShowToday) write(() => localStorage, HIDE_TODAY_KEY, key)
    setHidden(true)
  }
  const dateText = `${WEEKDAYS[today.getDay()]}, ${today.getDate()} ${MONTHS[today.getMonth()]} ${today.getFullYear()}`

  return (
    <NoticeDialog onClose={close}>
      <header className="daily-notice-head">
        <span className="today-icon" aria-hidden="true">
          🔔
        </span>
        <div className="grow">
          <h2 id="daily-notice-title">Pengumuman Hari Ini</h2>
          <p className="muted small">{dateText}</p>
        </div>
        <button type="button" className="icon-btn" aria-label="Tutup notifikasi" onClick={close}>
          ×
        </button>
      </header>

      {servicesToday.length > 0 && (
        <section className="notice-section">
          <h3>⛪ Ibadah Hari Ini</h3>
          <ul className="notice-services">
            {servicesToday.map((s) => (
              <li key={s.id} className="today" style={{ '--type-color': s.typeColor } as CSSProperties}>
                <span className="notice-time">{clock(s.time)}</span>
                <span className="notice-text">
                  <strong>
                    Ibadah {s.typeName} · {serviceTitle(s)}
                  </strong>
                  {(s.leader || s.mc) && (
                    <span className="muted small">
                      {s.leader && <>Pemimpin: {s.leader}</>}
                      {s.leader && s.mc && ' · '}
                      {s.mc && <>Pembawa acara: {s.mc}</>}
                    </span>
                  )}
                  {s.location && <span className="muted small">Lokasi: {s.location}</span>}
                </span>
              </li>
            ))}
          </ul>
        </section>
      )}

      {servicesWeek.length > 0 && (
        <section className="notice-section">
          <h3>📅 Ibadah Minggu Ini</h3>
          <ul className="notice-services">
            {servicesWeek.map((s) => (
              <li key={s.id} style={{ '--type-color': s.typeColor } as CSSProperties}>
                <span className="notice-time">
                  {shortDay(serviceDayKey(s))}
                  <br />
                  {clock(s.time)}
                </span>
                <span className="notice-text">
                  <strong>
                    Ibadah {s.typeName} · {serviceTitle(s)}
                  </strong>
                  {s.leader && <span className="muted small">Pemimpin: {s.leader}</span>}
                </span>
              </li>
            ))}
          </ul>
        </section>
      )}

      {birthdays.length > 0 && (
        <section className="notice-section">
          <h3>🎂 Ulang Tahun Hari Ini</h3>
          <ul className="daily-notice-list">
            {birthdays.map((m) => (
              <li key={m.id}>
                <strong>{m.name}</strong>
                <span className="muted small">
                  Kel. {m.familyName}
                  {m.birthDate && <> · {today.getFullYear() - parts(m.birthDate).y} tahun</>}
                </span>
              </li>
            ))}
          </ul>
        </section>
      )}

      <footer className="daily-notice-foot">
        <label className="check small">
          <input type="checkbox" checked={dontShowToday} onChange={(e) => setDontShowToday(e.target.checked)} />
          Jangan tampilkan lagi hari ini
        </label>
        <div className="daily-notice-actions">
          {servicesToday.length + servicesWeek.length > 0 && (
            <Link to="/jadwal" className="btn small" onClick={close}>
              Jadwal Ibadah
            </Link>
          )}
          {birthdays.length > 0 && (
            <Link to="/hut-jemaat" className="btn small" onClick={close}>
              HUT Jemaat
            </Link>
          )}
          <button type="button" className="btn small primary" data-initial-focus onClick={close}>
            Lewati
          </button>
        </div>
      </footer>
    </NoticeDialog>
  )
}

/**
 * Native modal dialog: centered, everything behind it inert (no clicks,
 * focus or scrolling), Escape closes it (as "Lewati"). The blur/dim of the
 * page comes from ::backdrop in CSS.
 */
function NoticeDialog({ onClose, children }: { onClose: () => void; children: ReactNode }) {
  const ref = useRef<HTMLDialogElement>(null)
  useEffect(() => {
    const dialog = ref.current
    if (!dialog || dialog.open) return // StrictMode runs effects twice
    dialog.showModal()
    dialog.querySelector<HTMLElement>('[data-initial-focus]')?.focus()
  }, [])
  return (
    <dialog ref={ref} className="daily-notice no-print" aria-labelledby="daily-notice-title" onClose={onClose}>
      {children}
    </dialog>
  )
}
