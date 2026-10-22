import type { CSSProperties, ReactNode } from 'react'
import { MONTHS } from '../format'
import { longDay, monthDays, moveCursor, WEEKDAYS, type Cursor } from '../schedule'

// The Google-Calendar-like month grid shared by Jadwal Ibadah and Daftar HUT Jemaat.

const MAX_PER_CELL = 3

/**
 * A month grid: each day shows up to three of its items (`renderItem`), and
 * "n lainnya" (`onMore`) for the rest. With `onAdd`, the empty part of a day
 * (of the `addable` ones) is a button too (labelled `addLabel`).
 */
export function MonthGrid<T>({
  cursor,
  todayKey,
  byDay,
  itemKey,
  renderItem,
  onMore,
  onAdd,
  addable,
  addLabel = 'Tambah',
}: {
  cursor: Cursor
  todayKey: string
  byDay: Map<string, T[]>
  itemKey: (item: T) => string
  renderItem: (item: T) => ReactNode
  onMore: (key: string) => void
  onAdd?: (key: string) => void
  /** the days `onAdd` applies to (default: all) */
  addable?: (key: string) => boolean
  addLabel?: string
}) {
  const days = monthDays(cursor.y, cursor.m)
  // only the weeks that contain days of this month (4–6)
  const weeks = Math.ceil((new Date(cursor.y, cursor.m, 1).getDay() + new Date(cursor.y, cursor.m + 1, 0).getDate()) / 7)
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
            {onAdd && (addable?.(day.key) ?? true) && (
              <button className="cal-cell-add" aria-label={`${addLabel} ${longDay(day.key)}`} onClick={() => onAdd(day.key)} />
            )}
            <span className={`cal-date${day.key === todayKey ? ' today' : ''}`}>
              {day.d === 1 ? `${day.d} ${MONTHS[day.m].slice(0, 3)}` : day.d}
            </span>
            {shown.map((item) => (
              <div key={itemKey(item)} className="cal-item">
                {renderItem(item)}
              </div>
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

/** "Hari ini", ‹ › and "Oktober 2026" (with a subtitle) above a month grid. */
export function MonthNav({
  cursor,
  today,
  subtitle,
  onChange,
  children,
}: {
  cursor: Cursor
  today: Date
  subtitle: ReactNode
  onChange: (c: Cursor) => void
  children?: ReactNode
}) {
  return (
    <header className="cal-toolbar">
      <button className="btn" onClick={() => onChange({ y: today.getFullYear(), m: today.getMonth() })}>
        Hari ini
      </button>
      <div className="cal-nav">
        <button className="icon-btn" aria-label="Bulan sebelumnya" onClick={() => onChange(moveCursor(cursor, -1))}>
          ‹
        </button>
        <button className="icon-btn" aria-label="Bulan berikutnya" onClick={() => onChange(moveCursor(cursor, 1))}>
          ›
        </button>
      </div>
      <h1 className="cal-title">
        {MONTHS[cursor.m]} {cursor.y}
        <small>{subtitle}</small>
      </h1>
      {children}
    </header>
  )
}
