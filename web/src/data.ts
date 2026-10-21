import { createContext, useContext } from 'react'
import type { BipraCategory, DataView, Family, Kolom, Member } from './api'

export type Data = {
  koloms: Kolom[]
  /** Kategorial BIPRA categories, in order (main categories and flags) */
  categories: BipraCategory[]
  /** the kolom being viewed; null = all koloms (or a BIPRA category is viewed) */
  kolom: number | null
  setKolom: (k: number | null) => void
  /** the filter: '' all, 'k:11' a kolom, 'b:<id>' a BIPRA category */
  filter: string
  setFilter: (v: string) => void
  /** the BIPRA category being viewed (its members, their families and its services), or null */
  bipraFilter: BipraCategory | null
  /** "Semua Kolom", "Kolom 11" or the category's name */
  filterLabel: string
  /** families and members of the viewed kolom (all when kolom is null) */
  families: Family[]
  members: Member[]
  /** every family, whatever kolom is viewed (forms pick from these) */
  allFamilies: Family[]
  allMembers: Member[]
  /** what the server sent: 'full' (signed in) or the redacted 'public' view */
  view: DataView | null
  loading: boolean
  error: string | null
  reload: () => Promise<void>
}

export const DataContext = createContext<Data | null>(null)

export function useData() {
  const ctx = useContext(DataContext)
  if (!ctx) throw new Error('useData must be used inside DataProvider')
  return ctx
}

export function kolomLabel(k: number | null) {
  return k === null ? 'Semua Kolom' : `Kolom ${k}`
}

/** "3. Kel. Najoan" — prefixed with the kolom when several koloms are listed. */
export function familyOption(f: Family, withKolom: boolean) {
  return `${withKolom ? `${kolomLabel(f.kolom)} · ` : ''}${f.no}. Kel. ${f.name}`
}
