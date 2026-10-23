import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { scheduleApi, type Service, type WorshipType } from './api'
import { useAuth } from './auth'
import { useData } from './data'
import { ScheduleContext } from './schedule'

type State = { services: Service[]; types: WorshipType[]; loading: boolean; error: string | null }

/**
 * Loads the services and worship types once for the whole app; every page
 * and the daily notice read them from here, and a save reloads them for all.
 */
export function ScheduleProvider({ children }: { children: ReactNode }) {
  const { kolom, bipraFilter } = useData()
  const { user } = useAuth()
  const [state, setState] = useState<State>({ services: [], types: [], loading: true, error: null })
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

  const services = useMemo(() => {
    // a kolom: its services and the whole congregation's (kolom 0)
    const inKolom = kolom === null ? state.services : state.services.filter((s) => s.kolom === kolom || s.kolom === 0)
    if (!bipraFilter) return inKolom
    // a BIPRA category: of those, the services of the jenis linked to it
    const linked = new Set(state.types.filter((t) => t.bipraId === bipraFilter.id).map((t) => t.id))
    return inKolom.filter((s) => linked.has(s.typeId))
  }, [state.services, state.types, kolom, bipraFilter])
  // a "kelola data BIPRA" user's home page: their category's jenis only
  const bipraId = user?.role === 'bipra' ? user.bipraId : undefined
  const homeServices = useMemo(() => {
    if (!bipraId) return services
    const own = new Set(state.types.filter((t) => t.bipraId === bipraId).map((t) => t.id))
    return services.filter((s) => own.has(s.typeId))
  }, [services, state.types, bipraId])

  return (
    <ScheduleContext.Provider value={{ ...state, services, homeServices, allServices: state.services, reload }}>
      {children}
    </ScheduleContext.Provider>
  )
}
