import type { BipraCategory, BipraInput } from './api'

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

/** Short age hint of a BIPRA rule: "0 – 11", "31+ / menikah", "60+". */
export function bipraRange(c: BipraInput) {
  const age =
    c.minAge !== null && c.maxAge !== null
      ? `${c.minAge} – ${c.maxAge}`
      : c.minAge !== null
        ? `${c.minAge}+`
        : c.maxAge !== null
          ? `≤ ${c.maxAge}`
          : ''
  const married = c.orMarried || c.marital === 'married'
  return married ? (age ? `${age} / menikah` : 'menikah') : age
}

/** The whole rule in words, for the admin page. */
export function bipraRule(c: BipraInput) {
  const bits: string[] = []
  if (c.gender) bits.push(c.gender === 'Laki Laki' ? 'laki-laki' : 'perempuan')
  const age = bipraRange({ ...c, orMarried: false, marital: '' })
  if (age) bits.push(`umur ${age}${c.orMarried ? ' atau sudah menikah' : ''}`)
  else if (c.orMarried) bits.push('sudah menikah')
  if (c.marital === 'married') bits.push('sudah menikah')
  if (c.marital === 'unmarried') bits.push('belum menikah')
  return bits.length ? bits.join(' · ') : 'semua anggota yang memiliki tanggal lahir'
}

/**
 * Whether a member falls under rule c (the server's model.BipraCategory.Matches),
 * for the admin page's preview. Members without an age match nothing.
 */
export function matchesBipra(m: { age: number | null; gender: string; maritalStatus: string }, c: BipraInput) {
  if (m.age === null) return false
  const married = m.maritalStatus !== '' && m.maritalStatus !== 'Belum Menikah'
  const ageOK =
    (c.orMarried && married) || ((c.minAge === null || m.age >= c.minAge) && (c.maxAge === null || m.age <= c.maxAge))
  if (!ageOK) return false
  if (c.gender && c.gender !== m.gender) return false
  if (c.marital === 'married') return married
  if (c.marital === 'unmarried') return !married
  return true
}

/** Whether member m is in category c (its main category, or one of its flags). */
export function inBipra(m: { category: string; flags: string[] | null }, c: BipraCategory) {
  return c.kind === 'flag' ? (m.flags ?? []).includes(c.key) : m.category === c.key
}

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
