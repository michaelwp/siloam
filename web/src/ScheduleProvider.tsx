import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { scheduleApi, type Service, type WorshipType } from './api'
import { limited, ownBipraIds, ownKoloms, useAuth } from './auth'
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

  const signedIn = !!user && !user.mustChangePassword
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

  // visitors get no schedule (it is jemaat data)
  useEffect(() => {
    if (signedIn) void reload()
  }, [reload, signedIn])

  // a limited user: their koloms' services (and the whole congregation's),
  // plus their categories' services in every kolom
  const ownKs = limited(user) ? ownKoloms(user).join(',') : ''
  const ownBs = limited(user) ? ownBipraIds(user).join(',') : ''
  const isLimited = limited(user)
  const services = useMemo(() => {
    const linkedTo = (ids: string[]) =>
      new Set(state.types.filter((t) => t.bipraId && ids.includes(t.bipraId)).map((t) => t.id))
    let list = state.services
    if (isLimited) {
      const ks = ownKs ? ownKs.split(',').map(Number) : []
      const own = linkedTo(ownBs ? ownBs.split(',') : [])
      list = list.filter((s) => ks.includes(s.kolom) || (ks.length > 0 && s.kolom === 0) || own.has(s.typeId))
    }
    // a kolom: its services and the whole congregation's (kolom 0)
    if (kolom !== null) list = list.filter((s) => s.kolom === kolom || s.kolom === 0)
    // a BIPRA category: the services of the jenis linked to it
    if (bipraFilter) {
      const linked = linkedTo([bipraFilter.id])
      list = list.filter((s) => linked.has(s.typeId))
    }
    return list
  }, [state.services, state.types, kolom, bipraFilter, isLimited, ownKs, ownBs])
  // Beranda and the daily notice show the same
  const homeServices = services

  return (
    <ScheduleContext.Provider
      value={
        signedIn
          ? { ...state, services, homeServices, allServices: state.services, reload }
          : { services: [], types: [], loading: false, error: null, homeServices: [], allServices: [], reload }
      }
    >
      {children}
    </ScheduleContext.Provider>
  )
}
