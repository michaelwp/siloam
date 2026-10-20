export const MONTHS = [
  'Januari',
  'Februari',
  'Maret',
  'April',
  'Mei',
  'Juni',
  'Juli',
  'Agustus',
  'September',
  'Oktober',
  'November',
  'Desember',
]

export const GENDERS = ['Laki Laki', 'Perempuan']

export const MARITAL_STATUSES = [
  'Belum Menikah',
  'Menikah',
  'Cerai Hidup / Duda',
  'Cerai Hidup / Janda',
  'Cerai Mati / Duda',
  'Cerai Mati / Janda',
]

export const RELATIONSHIPS = [
  'Kepala Keluarga',
  'Istri',
  'Suami',
  'Anak',
  'Orang Tua',
  'Menantu',
  'Cucu',
  'Family Lain',
]

export const CATEGORIES = [
  { key: 'Anak', label: 'Anak (ASM)', range: '0 – 11' },
  { key: 'Remaja', label: 'Remaja', range: '12 – 16' },
  { key: 'Pemuda', label: 'Pemuda', range: '17 – 30' },
  { key: 'PKB', label: 'PKB', range: '31+ / menikah' },
  { key: 'WKI', label: 'WKI', range: '31+ / menikah' },
]

/** "2021-09-10T00:00:00Z" → { y, m (0-11), d } read in UTC */
export function parts(iso: string) {
  const d = new Date(iso)
  return { y: d.getUTCFullYear(), m: d.getUTCMonth(), d: d.getUTCDate() }
}

/** "10 September 2021" */
export function longDate(iso: string | null) {
  if (!iso) return '–'
  const p = parts(iso)
  return `${p.d} ${MONTHS[p.m]} ${p.y}`
}

/** API timestamp → value for <input type="date"> */
export function toInputDate(iso: string | null) {
  return iso ? iso.slice(0, 10) : ''
}

/** <input type="date"> value → API timestamp */
export function fromInputDate(v: string) {
  return v ? `${v}T00:00:00Z` : null
}

export function triState(v: boolean | null, yes: string) {
  return v === true ? yes : v === false ? 'Belum' : '–'
}

export function byText(a: string, b: string) {
  return a.localeCompare(b, 'id', { sensitivity: 'base' })
}

export function matches(q: string, ...fields: (string | null | undefined)[]) {
  const needle = q.trim().toLowerCase()
  return !needle || fields.some((f) => f?.toLowerCase().includes(needle))
}

/** Lock duration in Indonesian, e.g. 180 → "3 menit" (matches the server's wording). */
export function humanDuration(seconds: number) {
  if (seconds >= 3600 && seconds % 3600 === 0) return `${seconds / 3600} jam`
  if (seconds >= 60) return `${Math.ceil(seconds / 60)} menit`
  return `${seconds} detik`
}

/** "14.05" — local time of an API timestamp */
export function clockTime(iso: string) {
  return new Date(iso).toLocaleTimeString('id-ID', { hour: '2-digit', minute: '2-digit' })
}

/** "13 November" from 1-based month and day (public birthday/anniversary) */
export function dayMonth(day: number | null, month: number | null) {
  return day && month ? `${day} ${MONTHS[month - 1]}` : '–'
}

/** "Manado, 13 November 1992" — full birth line for signed-in users */
export function birthLine(m: { birthPlace: string; birthDate: string | null }) {
  return [m.birthPlace, m.birthDate ? longDate(m.birthDate) : ''].filter(Boolean).join(', ') || '–'
}

const DAY_MS = 24 * 60 * 60 * 1000

function isLeapYear(y: number) {
  return (y % 4 === 0 && y % 100 !== 0) || y % 400 === 0
}

/** The date of a month/day anniversary in a given year; 29 February falls on the 28th in non-leap years. */
export function anniversaryIn(year: number, month: number, day: number) {
  if (month === 2 && day === 29 && !isLeapYear(year)) return new Date(year, 1, 28)
  return new Date(year, month - 1, day)
}

/** Whole days from `today` to the next occurrence of a 1-based month/day (0 = today). */
export function daysUntil(month: number, day: number, today: Date) {
  const start = new Date(today.getFullYear(), today.getMonth(), today.getDate())
  let next = anniversaryIn(today.getFullYear(), month, day)
  if (next < start) next = anniversaryIn(today.getFullYear() + 1, month, day)
  return Math.round((next.getTime() - start.getTime()) / DAY_MS)
}
