import { request, type BipraCategory, type Family, type Member } from './api'
import { inBipra, MARITAL_STATUSES } from './format'

export type YesNo = { yes: number; no: number; unknown: number }

/** The statistics summary: what visitors get from /api/stats, or computed from the loaded data. */
export type Stats = {
  families: number
  members: number
  male: number
  female: number
  baptized: YesNo
  confirmed: YesNo
  maritalStatuses: { label: string; value: number }[]
  categories: (BipraCategory & { count: number; male: number; female: number })[]
  /** members born in each month, January first */
  birthMonths: number[]
  /** [i]: families with i+1 members */
  familySizes: number[]
}

export const statsApi = {
  /** the whole congregation's summary (no sign-in needed) */
  get: () => request<Stats>('GET', '/api/stats'),
}

function yesNo(values: (boolean | null)[]): YesNo {
  return {
    yes: values.filter((v) => v === true).length,
    no: values.filter((v) => v === false).length,
    unknown: values.filter((v) => v === null).length,
  }
}

/** The same summary as /api/stats, for the data a signed-in user has loaded (and filtered). */
export function computeStats(families: Family[], members: Member[], categories: BipraCategory[]): Stats {
  const count = (pred: (m: Member) => boolean) => members.filter(pred).length
  // the known ones in order, then any other the data has sorted, as /api/stats lists them
  const statuses = [
    ...MARITAL_STATUSES,
    ...[...new Set(members.map((m) => m.maritalStatus).filter((s) => s && !MARITAL_STATUSES.includes(s)))].toSorted((a, b) =>
      a < b ? -1 : a > b ? 1 : 0,
    ),
  ]
  const maxSize = Math.max(0, ...families.map((f) => f.memberCount))
  return {
    families: families.length,
    members: members.length,
    male: count((m) => m.gender === 'Laki Laki'),
    female: count((m) => m.gender === 'Perempuan'),
    baptized: yesNo(members.map((m) => m.baptized)),
    confirmed: yesNo(members.map((m) => m.confirmed)),
    maritalStatuses: statuses.map((s) => ({ label: s, value: count((m) => m.maritalStatus === s) })),
    categories: categories.map((c) => {
      const inCat = members.filter((m) => inBipra(m, c))
      return {
        ...c,
        count: inCat.length,
        male: inCat.filter((m) => m.gender === 'Laki Laki').length,
        female: inCat.filter((m) => m.gender === 'Perempuan').length,
      }
    }),
    birthMonths: Array.from({ length: 12 }, (_, i) => count((m) => m.birthMonth === i + 1)),
    familySizes: Array.from({ length: maxSize }, (_, i) => families.filter((f) => f.memberCount === i + 1).length),
  }
}
