import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { api, type BipraCategory, type DataView, type Family, type Kolom, type Member } from './api'
import { limited, ownBipraIds, ownKoloms, useAuth } from './auth'
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
  if (!old.startsWith('b:')) return read(BIPRA_KEY)
  // moved to its own key, so a later kolom choice (which rewrites 'kolom-filter') keeps it
  save(BIPRA_KEY, old.slice(2))
  save(KOLOM_KEY, '')
  return old.slice(2)
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
  limit,
}: {
  children: ReactNode
  expectFull: boolean
  onStale: () => void
  /**
   * a scoped user or viewer limited to some koloms and/or BIPRA categories:
   * they see their koloms' data plus their categories' members in every kolom
   * (null: everything)
   */
  limit: { koloms: number[]; bipras: string[] } | null
}) {
  const [state, setState] = useState<State>({
    koloms: [],
    categories: [],
    families: [],
    members: [],
    view: null,
    loading: expectFull, // remounted (keyed) when the user changes
    error: null,
  })
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

  // visitors get no jemaat data (only the statistics summary, see Beranda)
  useEffect(() => {
    if (expectFull) void load()
  }, [load, expectFull])

  // What a limited user may pick: with categories, every kolom (their
  // members are everywhere), else their own koloms; "Semua Kolom" (all of
  // their data) unless that is a single kolom. In one of their koloms every
  // category may be picked; elsewhere (and in "Semua Data Saya") only their
  // own categories.
  const kolomChoices = useMemo(
    () => (!limit || limit.bipras.length ? state.koloms : state.koloms.filter((k) => limit.koloms.includes(k.number))),
    [limit, state.koloms],
  )
  const allKolomsAllowed = !limit || limit.bipras.length > 0 || limit.koloms.length !== 1
  const ownCats = useMemo(
    () => (limit ? state.categories.filter((c) => limit.bipras.includes(c.id)) : []),
    [limit, state.categories],
  )
  // the kolom and the category are picked separately, so both can narrow the data
  const setKolom = useCallback(
    (k: number | null) => {
      if (k === null ? !allKolomsAllowed : !kolomChoices.some((c) => c.number === k)) return
      save(KOLOM_KEY, k === null ? '' : `k:${k}`)
      setChosenKolom(k)
    },
    [allKolomsAllowed, kolomChoices],
  )

  // a remembered kolom or category that no longer exists (or is not offered)
  // falls back to all, or to a user's single kolom
  const kolom =
    chosenKolom !== null && kolomChoices.some((k) => k.number === chosenKolom)
      ? chosenKolom
      : allKolomsAllowed
        ? null
        : (kolomChoices[0]?.number ?? null)
  const bipraLimited = !!limit && limit.bipras.length > 0 && !(kolom !== null && limit.koloms.includes(kolom))
  const bipraChoices = bipraLimited ? ownCats : state.categories
  const setBipra = useCallback(
    (id: string | null) => {
      if (id && !bipraChoices.some((c) => c.id === id)) return
      save(BIPRA_KEY, id ?? '')
      setChosenBipra(id ?? '')
    },
    [bipraChoices],
  )
  // a category not offered for this kolom is set aside (it applies again in a kolom that offers it)
  const bipraFilter = chosenBipra ? (bipraChoices.find((c) => c.id === chosenBipra) ?? null) : null

  // a limited user's data: their koloms' families and members, plus their
  // categories' members (and the families they are in) in every kolom; then
  // the category chosen in the filter narrows it
  const scoped = useMemo(() => {
    let members = state.members
    let families = state.families
    if (limit) {
      members = members.filter((m) => limit.koloms.includes(m.kolom) || ownCats.some((c) => inBipra(m, c)))
      const ids = new Set(members.map((m) => m.familyId))
      families = families.filter((f) => limit.koloms.includes(f.kolom) || ids.has(f.id))
    }
    if (bipraFilter) {
      members = members.filter((m) => inBipra(m, bipraFilter))
      const ids = new Set(members.map((m) => m.familyId))
      families = families.filter((f) => ids.has(f.id))
    }
    return { families, members }
  }, [state.families, state.members, limit, ownCats, bipraFilter])
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
        kolomChoices,
        allKolomsAllowed,
        bipraFilter,
        setBipra,
        bipraChoices,
        bipraLimited,
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
  const ks = ownKoloms(user).join(',')
  const bs = ownBipraIds(user).join(',')
  const isLimited = limited(user)
  // stable while the user's koloms and categories stay the same
  const limit = useMemo(
    () => (isLimited ? { koloms: ks ? ks.split(',').map(Number) : [], bipras: bs ? bs.split(',') : [] } : null),
    [isLimited, ks, bs],
  )
  if (loading) return <p className="status splash">Memuat…</p>
  const full = !!user && !user.mustChangePassword
  const key = user ? `${user.id}:${full ? 'full' : 'limited'}` : 'visitor'
  return (
    <DataProvider key={key} expectFull={full} onStale={refresh} limit={limit}>
      {children}
    </DataProvider>
  )
}
