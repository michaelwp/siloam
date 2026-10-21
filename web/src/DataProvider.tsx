import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { api, type BipraCategory, type DataView, type Family, type Kolom, type Member } from './api'
import { useAuth } from './auth'
import { DataContext } from './data'

type State = {
  koloms: Kolom[]
  categories: BipraCategory[]
  families: Family[]
  members: Member[]
  view: DataView | null
  loading: boolean
  error: string | null
}

// The viewed kolom is remembered per browser (a convenience; never required).
const KOLOM_KEY = 'kolom-filter'

function savedKolom(): number | null {
  try {
    const n = Number(localStorage.getItem(KOLOM_KEY))
    return Number.isInteger(n) && n > 0 ? n : null
  } catch {
    return null
  }
}

function saveKolom(k: number | null) {
  try {
    if (k === null) localStorage.removeItem(KOLOM_KEY)
    else localStorage.setItem(KOLOM_KEY, String(k))
  } catch {
    // storage blocked: the choice lasts until the page is reloaded
  }
}

/**
 * Shares families and members across all pages. It is keyed by the signed-in
 * user (see DataScope), so signing in or out remounts it empty: data loaded
 * for one user — or the redacted public view — is never reused for another.
 */
function DataProvider({
  children,
  expectFull,
  onStale,
  startKolom,
}: {
  children: ReactNode
  expectFull: boolean
  onStale: () => void
  startKolom: number | null // a "kelola data kolom" user starts on their kolom
}) {
  const [state, setState] = useState<State>({ koloms: [], categories: [], families: [], members: [], view: null, loading: true, error: null })
  const [chosen, setChosen] = useState(() => startKolom ?? savedKolom())
  const latest = useRef(0) // ignore responses of superseded loads

  const load = useCallback(() => {
    const n = ++latest.current
    return api.load().then(
      ({ koloms, categories, families, members, view }) => {
        if (n !== latest.current) return
        setState({ koloms, categories, families, members, view, loading: false, error: null })
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

  const setKolom = useCallback((k: number | null) => {
    saveKolom(k)
    setChosen(k)
  }, [])

  // a remembered kolom that no longer exists falls back to all koloms
  const kolom = chosen !== null && state.koloms.some((k) => k.number === chosen) ? chosen : null
  const families = useMemo(
    () => (kolom === null ? state.families : state.families.filter((f) => f.kolom === kolom)),
    [state.families, kolom],
  )
  const members = useMemo(
    () => (kolom === null ? state.members : state.members.filter((m) => m.kolom === kolom)),
    [state.members, kolom],
  )

  return (
    <DataContext.Provider
      value={{
        koloms: state.koloms,
        categories: state.categories,
        kolom,
        setKolom,
        families,
        members,
        allFamilies: state.families,
        allMembers: state.members,
        view: state.view,
        loading: state.loading,
        error: state.error,
        reload: load,
      }}
    >
      {children}
    </DataContext.Provider>
  )
}

/** Provides data for the current user; waits until the sign-in state is known. */
export function DataScope({ children }: { children: ReactNode }) {
  const { user, loading, refresh } = useAuth()
  if (loading) return <p className="status splash">Memuat…</p>
  const full = !!user && !user.mustChangePassword
  const key = user ? `${user.id}:${full ? 'full' : 'limited'}` : 'visitor'
  return (
    <DataProvider
      key={key}
      expectFull={full}
      onStale={refresh}
      startKolom={user?.role === 'kolom' ? (user.kolom ?? null) : null}
    >
      {children}
    </DataProvider>
  )
}
