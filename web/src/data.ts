import { createContext, useContext } from 'react'
import type { DataView, Family, Member } from './api'

export type Data = {
  families: Family[]
  members: Member[]
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
