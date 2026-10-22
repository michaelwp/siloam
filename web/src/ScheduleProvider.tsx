import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { scheduleApi, type Service, type WorshipType } from './api'
import { useAuth } from './auth'
import { useData } from './data'
import { ScheduleContext } from './schedule'

type State = { services: Service[]; types: WorshipType[]; loading: boolean; error: string | null }

// the viewed jenis ibadah is remembered per browser (a convenience; never required)
const TYPE_KEY = 'jenis-filter'

function savedType() {
  try {
    return localStorage.getItem(TYPE_KEY) ?? ''
  } catch {
    return ''
  }
}

/**
 * Loads the services and worship types once for the whole app; every page
 * and the daily notice read them from here, and a save reloads them for all.
 */
export function ScheduleProvider({ children }: { children: ReactNode }) {
  const { kolom, bipraFilter } = useData()
  const { user } = useAuth()
  const [state, setState] = useState<State>({ services: [], types: [], loading: true, error: null })
  const [chosenType, setChosenType] = useState(savedType)
  const latest = useRef(0) // ignore responses of superseded loads

  const reload = useCallback(() => {
    const n = ++latest.current
    return scheduleApi.load().then(
      ([services, types]) => {
        if (n === latest.current) setState({ services, types, loading: false, error: null })
      },
      (e: Error) => {
        if (n === latest.current) setState((s) => ({ ...s, loading: false, error: e.message }))
      },
    )
  }, [])

  useEffect(() => {
    void reload()
  }, [reload])

  const setTypeFilter = useCallback((id: string | null) => {
    try {
      if (id) localStorage.setItem(TYPE_KEY, id)
      else localStorage.removeItem(TYPE_KEY)
    } catch {
      // storage blocked: the choice lasts until the page is reloaded
    }
    setChosenType(id ?? '')
  }, [])
  // a "kelola data BIPRA" user's category (their Beranda shows only its jenis), or the chosen
  // one: the jenis linked to it; else every jenis
  const bipraId = user?.role === 'bipra' ? user.bipraId : undefined
  const ownCategory = bipraId ?? bipraFilter?.id
  const typeChoices = useMemo(
    () => (ownCategory ? state.types.filter((t) => t.bipraId === ownCategory) : state.types),
    [state.types, ownCategory],
  )
  // a remembered jenis that is gone, or not of the chosen category, falls back to all
  const typeFilter = typeChoices.find((t) => t.id === chosenType) ?? null

  const services = useMemo(() => {
    // a kolom: its services and the whole congregation's (kolom 0)
    const inKolom = kolom === null ? state.services : state.services.filter((s) => s.kolom === kolom || s.kolom === 0)
    // a jenis, or a BIPRA category (the jenis linked to it): of those, the services of that jenis
    if (typeFilter) return inKolom.filter((s) => s.typeId === typeFilter.id)
    if (!bipraFilter) return inKolom
    const linked = new Set(typeChoices.map((t) => t.id))
    return inKolom.filter((s) => linked.has(s.typeId))
  }, [state.services, kolom, bipraFilter, typeChoices, typeFilter])
  // a "kelola data BIPRA" user's home page: their category's jenis only
  const homeServices = useMemo(() => {
    if (!bipraId) return services
    const own = new Set(state.types.filter((t) => t.bipraId === bipraId).map((t) => t.id))
    return services.filter((s) => own.has(s.typeId))
  }, [services, state.types, bipraId])

  return (
    <ScheduleContext.Provider value={{ ...state, services, homeServices, allServices: state.services, typeChoices, typeFilter, setTypeFilter, reload }}>
      {children}
    </ScheduleContext.Provider>
  )
}
