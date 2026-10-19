import { useCallback, useEffect, useState, type ReactNode } from 'react'
import { api, type Family, type Member } from './api'
import { DataContext } from './data'

type State = { families: Family[]; members: Member[]; loading: boolean; error: string | null }

const fetchAll = () => Promise.all([api.families(), api.members()])

/** Loads families and members once and shares them across all pages. */
export function DataProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<State>({ families: [], members: [], loading: true, error: null })

  const apply = useCallback((p: ReturnType<typeof fetchAll>) => {
    return p.then(
      ([families, members]) => setState({ families, members, loading: false, error: null }),
      (e: Error) => setState((s) => ({ ...s, loading: false, error: e.message })),
    )
  }, [])

  const reload = useCallback(() => apply(fetchAll()), [apply])

  useEffect(() => {
    void apply(fetchAll())
  }, [apply])

  return <DataContext.Provider value={{ ...state, reload }}>{children}</DataContext.Provider>
}
