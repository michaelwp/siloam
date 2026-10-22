import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { api, type BipraCategory, type DataView, type Family, type Kolom, type Member } from './api'
import { useAuth } from './auth'
import { DataContext, filterLabel } from './data'
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

// The viewed kolom and BIPRA category are remembered per browser (a
// convenience; never required): 'kolom-filter' is 'k:11' (or '11', as saved
// before categories), 'bipra-filter' a category id. Before the two were
// separate, 'kolom-filter' held either: 'b:<category id>' moves over.
const KOLOM_KEY = 'kolom-filter'
const BIPRA_KEY = 'bipra-filter'

function read(key: string) {
  try {
    return localStorage.getItem(key) ?? ''
  } catch {
    return ''
  }
}

function save(key: string, v: string) {
  try {
    if (!v) localStorage.removeItem(key)
    else localStorage.setItem(key, v)
  } catch {
    // storage blocked: the choice lasts until the page is reloaded
  }
}

function savedKolom(): number | null {
  const v = read(KOLOM_KEY).replace(/^k:/, '')
  return /^\d+$/.test(v) ? Number(v) : null
}

function savedBipra(): string {
  const old = read(KOLOM_KEY)
  return old.startsWith('b:') ? old.slice(2) : read(BIPRA_KEY)
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
  const [chosenKolom, setChosenKolom] = useState(savedKolom)
  const [chosenBipra, setChosenBipra] = useState(savedBipra)
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

  // the kolom and the category are picked separately, so both can narrow the data
  const setKolom = useCallback(
    (k: number | null) => {
      if (lockedKolom !== null) return // a kolom user cannot switch
      save(KOLOM_KEY, k === null ? '' : `k:${k}`)
      setChosenKolom(k)
    },
    [lockedKolom],
  )
  const setBipra = useCallback(
    (id: string | null) => {
      if (bipraId) return // a BIPRA user has their own
      save(BIPRA_KEY, id ?? '')
      setChosenBipra(id ?? '')
    },
    [bipraId],
  )

  // a remembered kolom or category that no longer exists falls back to all
  const kolom = lockedKolom ?? (chosenKolom !== null && state.koloms.some((k) => k.number === chosenKolom) ? chosenKolom : null)
  // a BIPRA category chosen in the filter (a BIPRA user has their own: see scope)
  const bipraFilter = !bipraId && chosenBipra ? (state.categories.find((c) => c.id === chosenBipra) ?? null) : null
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
        bipraFilter,
        setBipra,
        filterLabel: filterLabel(kolom, bipraFilter),
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
