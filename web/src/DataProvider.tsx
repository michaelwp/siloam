import { useCallback, useEffect, useRef, useState, type ReactNode } from 'react'
import { api, type DataView, type Family, type Member } from './api'
import { useAuth } from './auth'
import { DataContext } from './data'

type State = { families: Family[]; members: Member[]; view: DataView | null; loading: boolean; error: string | null }

/**
 * Shares families and members across all pages. It is keyed by the signed-in
 * user (see DataScope), so signing in or out remounts it empty: data loaded
 * for one user — or the redacted public view — is never reused for another.
 */
function DataProvider({ children, expectFull, onStale }: { children: ReactNode; expectFull: boolean; onStale: () => void }) {
  const [state, setState] = useState<State>({ families: [], members: [], view: null, loading: true, error: null })
  const latest = useRef(0) // ignore responses of superseded loads

  const load = useCallback(() => {
    const n = ++latest.current
    return api.load().then(
      ({ families, members, view }) => {
        if (n !== latest.current) return
        setState({ families, members, view, loading: false, error: null })
        // signed in, but the server sent the public view: the session is gone
        if (expectFull && view !== 'full') onStale()
      },
      (e: Error) => {
        if (n === latest.current) setState((s) => ({ ...s, loading: false, error: e.message }))
      },
    )
  }, [expectFull, onStale])

  useEffect(() => {
    void load()
  }, [load])

  return <DataContext.Provider value={{ ...state, reload: load }}>{children}</DataContext.Provider>
}

/** Provides data for the current user; waits until the sign-in state is known. */
export function DataScope({ children }: { children: ReactNode }) {
  const { user, loading, refresh } = useAuth()
  if (loading) return <p className="status splash">Memuat…</p>
  const full = !!user && !user.mustChangePassword
  const key = user ? `${user.id}:${full ? 'full' : 'limited'}` : 'visitor'
  return (
    <DataProvider key={key} expectFull={full} onStale={refresh}>
      {children}
    </DataProvider>
  )
}
