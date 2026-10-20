import { useCallback, useEffect, useRef, useState } from 'react'
import { scheduleApi, type Service, type WorshipType } from './api'
import { useData } from './data'
import { parts } from './format'

/** Services (of the viewed kolom) and worship types, with a reload. */
export function useSchedule() {
  const { kolom } = useData()
  const [state, setState] = useState<{ services: Service[]; types: WorshipType[]; loading: boolean; error: string | null }>({
    services: [],
    types: [],
    loading: true,
    error: null,
  })
  const latest = useRef(0)
  const reload = useCallback(() => {
    const n = ++latest.current
    return scheduleApi.load().then(
      ([services, types]) => {
        if (n === latest.current) setState({ services, types, loading: false, error: null })
      },
      (e: Error) => {
        if (n === latest.current) setState((s) => ({ ...s, loading: false, error: e.message }))
      },
    )
  }, [])
  useEffect(() => {
    void reload()
  }, [reload])
  const services = kolom === null ? state.services : state.services.filter((s) => s.kolom === kolom)
  return { ...state, allServices: state.services, services, reload }
}

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
    `Kolom ${s.kolom}`,
    s.leader && `Pemimpin Ibadah: ${s.leader}`,
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

/** The subscribable feed (all services, or one kolom). */
export function feedUrl(kolom: number | null) {
  return `${window.location.origin}/api/calendar.ics${kolom === null ? '' : `?kolom=${kolom}`}`
}
