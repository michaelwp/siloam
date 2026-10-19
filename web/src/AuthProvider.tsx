import { useCallback, useEffect, useState, type ReactNode } from 'react'
import { ApiError, AUTH_CHANGED, authApi, type User } from './api'
import { AuthContext } from './auth'

export function AuthProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<{ user: User | null; loading: boolean }>({ user: null, loading: true })

  const load = useCallback(
    () =>
      authApi.me().then(
        (user) => setState({ user, loading: false }),
        (e) => {
          // 401 = signed out; anything else (server down) also shows the login page
          if (!(e instanceof ApiError)) console.error(e)
          setState({ user: null, loading: false })
        },
      ),
    [],
  )

  useEffect(() => {
    void load()
    const onChange = () => void load()
    window.addEventListener(AUTH_CHANGED, onChange)
    return () => window.removeEventListener(AUTH_CHANGED, onChange)
  }, [load])

  const setUser = useCallback((user: User | null) => setState({ user, loading: false }), [])

  const logout = useCallback(async () => {
    await authApi.logout()
    setUser(null)
  }, [setUser])

  return (
    <AuthContext.Provider value={{ ...state, setUser, refresh: load, logout }}>{children}</AuthContext.Provider>
  )
}
