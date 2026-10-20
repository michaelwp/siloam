import { useEffect, useRef, useState, type ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { useData } from '../data'
import { byText, daysUntil, parts } from '../format'

// Remembered per browser (nothing is sent to the server):
//  - "Lewati" hides it for the rest of this tab session (it comes back the next
//    time the app is opened);
//  - "Jangan tampilkan lagi hari ini" hides it until tomorrow.
const HIDE_TODAY_KEY = 'birthday-toast-hidden-on'
const SKIP_KEY = 'birthday-toast-skipped-on'

function dateKey(d: Date) {
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
 * A centered modal notification listing who has a birthday today, shown on
 * every page (visitors and signed-in users) once data has loaded. The page
 * behind is blurred and inert (native modal <dialog>) until it is closed.
 */
export function BirthdayToast() {
  const { members, loading } = useData()
  const [today] = useState(() => new Date())
  const key = dateKey(today)
  const [hidden, setHidden] = useState(
    () => read(() => localStorage, HIDE_TODAY_KEY) === key || read(() => sessionStorage, SKIP_KEY) === key,
  )
  const [dontShowToday, setDontShowToday] = useState(false)

  const todays = members
    .filter((m) => m.birthMonth && m.birthDay && daysUntil(m.birthMonth, m.birthDay, today) === 0)
    .toSorted((a, b) => byText(a.name, b.name))

  if (hidden || loading || todays.length === 0) return null

  const close = () => {
    write(() => sessionStorage, SKIP_KEY, key)
    if (dontShowToday) write(() => localStorage, HIDE_TODAY_KEY, key)
    setHidden(true)
  }

  return (
    <BirthdayDialog onClose={close}>
      <header className="birthday-toast-head">
        <span className="today-icon" aria-hidden="true">
          🎂
        </span>
        <div className="grow">
          <h2 id="birthday-toast-title">Ulang Tahun Hari Ini</h2>
          <p className="muted small">
            {todays.length} anggota jemaat berulang tahun hari ini. Selamat ulang tahun!
          </p>
        </div>
        <button type="button" className="icon-btn" aria-label="Tutup notifikasi" onClick={close}>
          ×
        </button>
      </header>

      <ul className="birthday-toast-list">
        {todays.map((m) => (
          <li key={m.id}>
            <strong>{m.name}</strong>
            <span className="muted small">
              Kel. {m.familyName}
              {m.birthDate && <> · {today.getFullYear() - parts(m.birthDate).y} tahun</>}
            </span>
          </li>
        ))}
      </ul>

      <footer className="birthday-toast-foot">
        <label className="check small">
          <input type="checkbox" checked={dontShowToday} onChange={(e) => setDontShowToday(e.target.checked)} />
          Jangan tampilkan lagi hari ini
        </label>
        <div className="birthday-toast-actions">
          <Link to="/hut-jemaat" className="btn small" onClick={close}>
            Lihat semua
          </Link>
          <button type="button" className="btn small primary" data-initial-focus onClick={close}>
            Lewati
          </button>
        </div>
      </footer>
    </BirthdayDialog>
  )
}

/**
 * Native modal dialog: centered, everything behind it inert (no clicks,
 * focus or scrolling), Escape closes it (as "Lewati"). The blur/dim of the
 * page comes from ::backdrop in CSS.
 */
function BirthdayDialog({ onClose, children }: { onClose: () => void; children: ReactNode }) {
  const ref = useRef<HTMLDialogElement>(null)
  useEffect(() => {
    const dialog = ref.current
    if (!dialog || dialog.open) return // StrictMode runs effects twice
    dialog.showModal()
    dialog.querySelector<HTMLElement>('[data-initial-focus]')?.focus()
  }, [])
  return (
    <dialog ref={ref} className="birthday-toast no-print" aria-labelledby="birthday-toast-title" onClose={onClose}>
      {children}
    </dialog>
  )
}
