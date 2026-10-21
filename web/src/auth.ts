import { browserSupportsWebAuthn, startAuthentication, startRegistration, WebAuthnError } from '@simplewebauthn/browser'
import { createContext, useContext } from 'react'
import { authApi, type Role, type User } from './api'
import { useData } from './data'

export type Auth = {
  user: User | null
  loading: boolean
  /** replace the signed-in user (after login, change, passkey edits) */
  setUser: (u: User | null) => void
  refresh: () => Promise<void>
  logout: () => Promise<void>
}

export const AuthContext = createContext<Auth | null>(null)

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used inside AuthProvider')
  return ctx
}

export const ROLE_LABELS: Record<Role, string> = {
  admin: 'Admin',
  user: 'Pengguna',
  kolom: 'Pengguna – kelola data kolom',
}

/** "Admin", "Pengguna", "Kelola Kolom 3" */
export function roleLabel(u: Pick<User, 'role' | 'kolom'>) {
  return u.role === 'kolom' ? `Kelola Kolom ${u.kolom}` : ROLE_LABELS[u.role]
}

/** Admins and regular users change every kolom's data, the koloms and the jenis ibadah. */
export function managesAll(u: User | null) {
  return !!u && (u.role === 'admin' || u.role === 'user')
}

/** Whether u may change (and see the private data of) kolom k; a kolom user only their own. */
export function managesKolom(u: User | null, k: number) {
  return managesAll(u) || (!!u && u.role === 'kolom' && u.kolom === k)
}

/**
 * Whether the loaded data is the full (signed-in) view, as reported by the
 * server — never guessed from the sign-in state, which can be stale. A
 * "kelola data kolom" user gets it only for their own kolom (the server sends
 * the other koloms redacted), so other koloms show as for visitors.
 */
export function useFullView() {
  const { view, kolom } = useData()
  const { user } = useAuth()
  return view === 'full' && (user?.role !== 'kolom' || kolom === user.kolom)
}

/**
 * Data can be changed only when signed in AND the loaded data is the full
 * view, so an edit form is never filled from redacted (blank) fields.
 */
export function useCanEdit() {
  const { user } = useAuth()
  const full = useFullView()
  return !!user && !user.mustChangePassword && full
}

/** Passkeys need WebAuthn and a secure context (HTTPS or localhost). */
export function passkeysSupported() {
  return window.isSecureContext && browserSupportsWebAuthn()
}

export async function signInWithPasskey(): Promise<User> {
  const { publicKey } = await authApi.passkeyLoginBegin()
  const cred = await startAuthentication({ optionsJSON: publicKey })
  return authApi.passkeyLoginFinish(cred)
}

export async function registerPasskey(name: string): Promise<User> {
  const { publicKey } = await authApi.passkeyRegisterBegin()
  const cred = await startRegistration({ optionsJSON: publicKey })
  return authApi.passkeyRegisterFinish(name, cred)
}

/** Human message for WebAuthn errors; null when the user simply cancelled. */
export function passkeyError(e: unknown): string | null {
  if (e instanceof WebAuthnError) {
    if (e.code === 'ERROR_AUTHENTICATOR_PREVIOUSLY_REGISTERED') return 'Passkey untuk perangkat ini sudah terdaftar.'
    if (e.code === 'ERROR_CEREMONY_ABORTED') return null
  }
  if (e instanceof Error && (e.name === 'NotAllowedError' || e.name === 'AbortError')) return null
  return e instanceof Error ? e.message : 'Passkey gagal'
}
