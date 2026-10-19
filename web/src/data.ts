import { createContext, useContext } from 'react'
import type { Family, Member } from './api'

export type Data = {
  families: Family[]
  members: Member[]
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
