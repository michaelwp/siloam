import { useEffect, useState } from 'react'

/**
 * Loads (again whenever key or load changes: load must be stable) and keeps
 * the answer with its key, so a slower answer to an earlier request is never shown.
 */
export function useLoad<T>(key: string, load: () => Promise<T>) {
  const [result, setResult] = useState<{
    key: string
    data?: T
    error?: string
  } | null>(null)
  useEffect(() => {
    let live = true
    load().then(
      (data) => {
        if (live) setResult({ key, data })
      },
      (e: Error) => {
        if (live) setResult({ key, error: e.message })
      },
    )
    return () => {
      live = false
    }
  }, [key, load])
  const current = result?.key === key ? result : null
  return {
    data: current?.data ?? null,
    error: current?.error ?? null,
    loading: !current,
  }
}
