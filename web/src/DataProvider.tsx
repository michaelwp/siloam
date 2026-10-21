import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { api, type BipraCategory, type DataView, type Family, type Kolom, type Member } from './api'
import { useAuth } from './auth'
import { DataContext, kolomLabel } from './data'
import { inBipra } from './format'

type State = {
  koloms: Kolom[]
  categories: BipraCategory[]
  families: Family[]
  members: Member[]
  view: DataView | null
  loading: boolean
  error: string | null
}

// The viewed filter is remembered per browser (a convenience; never required):
// '' all, 'k:11' a kolom, 'b:<category id>' a BIPRA category.
const FILTER_KEY = 'kolom-filter'

function savedFilter(): string {
  try {
    const v = localStorage.getItem(FILTER_KEY) ?? ''
    return /^\d+$/.test(v) ? `k:${v}` : v // a kolom number, as saved before categories
  } catch {
    return ''
  }
}

function saveFilter(v: string) {
  try {
    if (!v) localStorage.removeItem(FILTER_KEY)
    else localStorage.setItem(FILTER_KEY, v)
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
  lockedKolom,
  bipraId,
}: {
  children: ReactNode
  expectFull: boolean
  onStale: () => void
  lockedKolom: number | null // a "kelola data kolom" user sees only their kolom
  bipraId: string | null // a "kelola data BIPRA" user sees only this category's members
}) {
  const [state, setState] = useState<State>({ koloms: [], categories: [], families: [], members: [], view: null, loading: true, error: null })
  const [chosen, setChosen] = useState(savedFilter)
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

  const setFilter = useCallback(
    (v: string) => {
      if (lockedKolom !== null) return // a kolom user cannot switch
      saveFilter(v)
      setChosen(v)
    },
    [lockedKolom],
  )
  const setKolom = useCallback((k: number | null) => setFilter(k === null ? '' : `k:${k}`), [setFilter])

  // a remembered kolom or category that no longer exists falls back to all
  const chosenKolom = chosen.startsWith('k:') ? Number(chosen.slice(2)) : null
  const kolom = lockedKolom ?? (chosenKolom !== null && state.koloms.some((k) => k.number === chosenKolom) ? chosenKolom : null)
  // a BIPRA category chosen in the filter (not for kolom users; a BIPRA user has their own)
  const bipraFilter =
    lockedKolom === null && !bipraId && chosen.startsWith('b:')
      ? (state.categories.find((c) => c.id === chosen.slice(2)) ?? null)
      : null
  // only one category's members, and the families they are in: a "kelola data
  // BIPRA" user's own, or the category chosen in the filter
  const scope = state.categories.find((c) => c.id === bipraId) ?? bipraFilter ?? undefined
  const scoped = useMemo(() => {
    if (!bipraId && !scope) return { families: state.families, members: state.members }
    const members = scope ? state.members.filter((m) => inBipra(m, scope)) : []
    const ids = new Set(members.map((m) => m.familyId))
    return { families: state.families.filter((f) => ids.has(f.id)), members }
  }, [state.families, state.members, bipraId, scope])
  const families = useMemo(
    () => (kolom === null ? scoped.families : scoped.families.filter((f) => f.kolom === kolom)),
    [scoped.families, kolom],
  )
  const members = useMemo(
    () => (kolom === null ? scoped.members : scoped.members.filter((m) => m.kolom === kolom)),
    [scoped.members, kolom],
  )

  return (
    <DataContext.Provider
      value={{
        koloms: state.koloms,
        categories: state.categories,
        kolom,
        setKolom,
        filter: kolom !== null ? `k:${kolom}` : bipraFilter ? `b:${bipraFilter.id}` : '',
        setFilter,
        bipraFilter,
        filterLabel: bipraFilter?.name ?? kolomLabel(kolom),
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
      lockedKolom={user?.role === 'kolom' ? (user.kolom ?? null) : null}
      bipraId={user?.role === 'bipra' ? (user.bipraId ?? null) : null}
    >
      {children}
    </DataProvider>
  )
}
