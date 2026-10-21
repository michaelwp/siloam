import { createContext, useContext } from 'react'
import type { Service, WorshipType } from './api'
import { MONTHS, parts } from './format'

export type Schedule = {
  /** services of the viewed kolom, and those of the whole congregation (kolom 0); all when no kolom is chosen */
  services: Service[]
  /** services for Beranda and the daily notice: for a "kelola data BIPRA" user only their category's */
  homeServices: Service[]
  allServices: Service[]
  types: WorshipType[]
  loading: boolean
  error: string | null
  reload: () => Promise<void>
}

export const ScheduleContext = createContext<Schedule | null>(null)

/** The schedule shared by every page (see ScheduleProvider), so a change shows everywhere. */
export function useSchedule() {
  const ctx = useContext(ScheduleContext)
  if (!ctx) throw new Error('useSchedule must be used inside ScheduleProvider')
  return ctx
}

/** Services shown as "this week": the days after today, up to this many. */
export const WEEK_AHEAD_DAYS = 6

/** Google Calendar's event colors, offered for worship types. */
export const TYPE_COLORS = [
  { hex: '#039be5', name: 'Biru Laut' },
  { hex: '#3f51b5', name: 'Biru Tua' },
  { hex: '#7986cb', name: 'Lavender' },
  { hex: '#8e24aa', name: 'Ungu' },
  { hex: '#e67c73', name: 'Merah Muda' },
  { hex: '#d50000', name: 'Merah' },
  { hex: '#f4511e', name: 'Jingga' },
  { hex: '#f6bf26', name: 'Kuning' },
  { hex: '#33b679', name: 'Hijau Muda' },
  { hex: '#0b8043', name: 'Hijau' },
  { hex: '#616161', name: 'Abu-abu' },
]

export const WEEKDAYS = ['Minggu', 'Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat', 'Sabtu']

/** "2026-01-21" for a local date. */
export function dayKey(y: number, m: number, d: number) {
  return `${y}-${String(m + 1).padStart(2, '0')}-${String(d).padStart(2, '0')}`
}

export function serviceDayKey(s: Service) {
  const p = parts(s.date)
  return dayKey(p.y, p.m, p.d)
}

/** "19:00" → "19.00" (Indonesian style) */
export function clock(t: string) {
  return t.replace(':', '.')
}

/** "19.00 – 20.30" from the service's start/end */
export function timeRange(s: Service) {
  const end = new Date(s.end)
  const start = new Date(s.start)
  const mins = Math.round((end.getTime() - start.getTime()) / 60000)
  const [h, m] = s.time.split(':').map(Number)
  const total = h * 60 + m + mins
  const e = `${String(Math.floor(total / 60) % 24).padStart(2, '0')}.${String(total % 60).padStart(2, '0')}`
  return `${clock(s.time)} – ${e}`
}

export function serviceTitle(s: Service) {
  return s.host ? `Kel. ${s.host}` : `Ibadah ${s.typeName}`
}

function gcalStamp(iso: string) {
  return new Date(iso).toISOString().replace(/[-:]/g, '').replace(/\.\d{3}/, '')
}

function details(s: Service) {
  return [
    s.kolom ? `Kolom ${s.kolom}` : 'Semua kolom',
    s.leader && `Khadim: ${s.leader}`,
    s.mc && `Pembawa Acara: ${s.mc}`,
    s.note,
  ]
    .filter(Boolean)
    .join('\n')
}

export function serviceLocation(s: Service) {
  return s.location || (s.host ? `Rumah Kel. ${s.host}` : '')
}

/** Opens Google Calendar's "new event" page filled with this service. */
export function googleCalendarUrl(s: Service) {
  const q = new URLSearchParams({
    action: 'TEMPLATE',
    text: `Ibadah ${s.typeName}${s.host ? ` · Kel. ${s.host}` : ''}`,
    dates: `${gcalStamp(s.start)}/${gcalStamp(s.end)}`,
    details: details(s),
    location: serviceLocation(s),
  })
  return `https://calendar.google.com/calendar/render?${q}`
}

/** The .ics download: Apple Calendar (and Outlook) add it on open. */
export function icsUrl(s: Service) {
  return `/api/services/${s.id}/ics`
}

/** The subscribable feed: all services, or one kolom; typeIds (when given) limits the jenis. */
export function feedUrl(kolom: number | null, typeIds?: string[]) {
  const q = new URLSearchParams()
  if (kolom !== null) q.set('kolom', String(kolom))
  for (const id of typeIds ?? []) q.append('type', id)
  const qs = q.toString()
  return `${window.location.origin}/api/calendar.ics${qs ? `?${qs}` : ''}`
}

/** Chronological order: date, then time. */
export function byStart(a: Service, b: Service) {
  return serviceDayKey(a).localeCompare(serviceDayKey(b)) || a.time.localeCompare(b.time)
}

/** "Rabu, 21 Okt" from a "2026-10-21" key */
export function shortDay(key: string) {
  const [y, m, d] = key.split('-').map(Number)
  return `${WEEKDAYS[new Date(y, m - 1, d).getDay()]}, ${d} ${MONTHS[m - 1].slice(0, 3)}`
}

/**
 * Services of today, of the next `days` days after today, and the first one
 * after that (for "berikutnya" when the week is empty), in time order.
 */
export function upcomingServices(services: Service[], today: Date, days: number) {
  const todayKey = dayKey(today.getFullYear(), today.getMonth(), today.getDate())
  const end = new Date(today.getFullYear(), today.getMonth(), today.getDate() + days)
  const endKey = dayKey(end.getFullYear(), end.getMonth(), end.getDate())
  const sorted = services.toSorted(byStart)
  return {
    today: sorted.filter((s) => serviceDayKey(s) === todayKey),
    week: sorted.filter((s) => serviceDayKey(s) > todayKey && serviceDayKey(s) <= endKey),
    next: sorted.find((s) => serviceDayKey(s) > endKey) ?? null,
  }
}
