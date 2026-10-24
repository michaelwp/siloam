import { useEffect, useRef, useState, type CSSProperties, type ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { useData } from '../data'
import { byText, daysUntil, MONTHS } from '../format'
import {
  clock,
  dayKey,
  serviceDayKey,
  serviceTitle,
  shortDay,
  upcomingServices,
  useSchedule,
  WEEKDAYS,
} from '../schedule'
import { BirthdaysByKolom, ByKolom } from './ui'

// Remembered per browser (nothing is sent to the server):
//  - "Lewati" hides it for the rest of this tab session (it comes back the next
//    time the app is opened);
//  - "Jangan tampilkan lagi hari ini" hides it until tomorrow.
// (The keys date from when this showed only birthdays.)
const HIDE_TODAY_KEY = 'birthday-toast-hidden-on'
const SKIP_KEY = 'birthday-toast-skipped-on'

/** Services and birthdays are announced every day for a week before the day (and on it). */
const NOTICE_AHEAD_DAYS = 7

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

/** "Hari ini", "Besok", else "Senin, 26 Okt": the day `days` after `today`. */
function dayLabel(today: Date, days: number) {
  if (days === 0) return 'Hari ini'
  if (days === 1) return 'Besok'
  const d = new Date(today.getFullYear(), today.getMonth(), today.getDate() + days)
  return shortDay(dayKey(d.getFullYear(), d.getMonth(), d.getDate()))
}

/**
 * A centered modal shown once the data has loaded, on every page (visitors and
 * signed-in users), for the viewed kolom: the services and birthdays of today
 * and of the next NOTICE_AHEAD_DAYS days, per kolom ("Kolom 11:"). The page behind is blurred and inert
 * (native modal <dialog>) until it is closed. Nothing to tell → not shown.
 */
export function DailyNotice() {
  const { members, loading } = useData()
  const { homeServices: services, loading: scheduleLoading } = useSchedule()
  const [today] = useState(() => new Date())
  const key = storageKey(today)
  const [hidden, setHidden] = useState(
    () => read(() => localStorage, HIDE_TODAY_KEY) === key || read(() => sessionStorage, SKIP_KEY) === key,
  )
  const [dontShowToday, setDontShowToday] = useState(false)

  const { today: servicesToday, week: servicesWeek } = upcomingServices(services, today, NOTICE_AHEAD_DAYS)
  // birthdays of today and the coming week, soonest first, then by name
  const birthdays = members
    .flatMap((m) => {
      if (!m.birthMonth || !m.birthDay) return []
      const days = daysUntil(m.birthMonth, m.birthDay, today)
      return days <= NOTICE_AHEAD_DAYS ? [{ member: m, days }] : []
    })
    .toSorted((a, b) => a.days - b.days || byText(a.member.name, b.member.name))
  const birthdayDays = [...new Set(birthdays.map((b) => b.days))]
  const birthdaysToday = birthdays.some((b) => b.days === 0)

  // today's and this week's together, per kolom (today's show the time only)
  const upcoming = [...servicesToday, ...servicesWeek]
  const todayIds = new Set(servicesToday.map((s) => s.id))
  const nothing = upcoming.length + birthdays.length === 0
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

      {upcoming.length > 0 && (
        <section className="notice-section">
          <h3>⛪ Ibadah Hari Ini & Minggu Ini</h3>
          <ByKolom items={upcoming} kolomOf={(s) => s.kolom}>
            {(list) => (
              <ul className="notice-services">
                {list.map((s) => (
                  <li
                    key={s.id}
                    className={todayIds.has(s.id) ? 'today' : undefined}
                    style={{ '--type-color': s.typeColor } as CSSProperties}
                  >
                    <span className="notice-time">
                      {!todayIds.has(s.id) && (
                        <>
                          {shortDay(serviceDayKey(s))}
                          <br />
                        </>
                      )}
                      {clock(s.time)}
                    </span>
                    <span className="notice-text">
                      <strong>
                        Ibadah {s.typeName} · {serviceTitle(s)}
                      </strong>
                      {(s.leader || s.mc) && (
                        <span className="muted small">
                          {s.leader && <>Khadim: {s.leader}</>}
                          {s.leader && s.mc && ' · '}
                          {s.mc && <>Pembawa acara: {s.mc}</>}
                        </span>
                      )}
                      {s.location && <span className="muted small">Lokasi: {s.location}</span>}
                    </span>
                  </li>
                ))}
              </ul>
            )}
          </ByKolom>
        </section>
      )}

      {birthdays.length > 0 && (
        <section className="notice-section">
          <h3>🎂 Ulang Tahun Hari Ini & Minggu Ini</h3>
          {birthdayDays.map((days) => (
            <div key={days} className={days === 0 ? 'notice-day today' : 'notice-day'}>
              <h4 className="notice-day-head">{dayLabel(today, days)}</h4>
              <BirthdaysByKolom members={birthdays.filter((b) => b.days === days).map((b) => b.member)} />
            </div>
          ))}
          {birthdaysToday && <p className="muted small">Selamat ulang tahun!</p>}
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
