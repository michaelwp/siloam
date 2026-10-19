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
