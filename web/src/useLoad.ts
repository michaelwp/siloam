import { useCallback, useEffect, useState } from 'react'

import { ApiError } from './api'

/**
 * Loads (again whenever key or load changes: load must be stable) and keeps
 * the answer with its key, so a slower answer to an earlier request is never
 * shown. reload() fetches the same key again while the current answer stays
 * on screen (no "loading" flash after a save).
 */
export function useLoad<T>(key: string, load: () => Promise<T>) {
  const [result, setResult] = useState<{
    key: string
    data?: T
    error?: string
    status?: number
  } | null>(null)
  useEffect(() => {
    let live = true
    load().then(
      (data) => {
        if (live) setResult({ key, data })
      },
      (e: Error) => {
        if (live) setResult({ key, error: e.message, status: e instanceof ApiError ? e.status : undefined })
      },
    )
    return () => {
      live = false
    }
  }, [key, load])
  // the same key again: the answer replaces the current one only if that key is still shown
  const reload = useCallback(() => {
    const keep = (next: { key: string; data?: T; error?: string; status?: number }) =>
      setResult((r) => (r?.key === key ? next : r))
    load().then(
      (data) => keep({ key, data }),
      (e: Error) => keep({ key, error: e.message, status: e instanceof ApiError ? e.status : undefined }),
    )
  }, [key, load])
  const current = result?.key === key ? result : null
  return {
    data: current?.data ?? null,
    error: current?.error ?? null,
    /** the HTTP status of a failed load (e.g. 403), if any */
    status: current?.status ?? null,
    loading: !current,
    reload,
  }
}
