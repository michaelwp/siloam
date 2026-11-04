import { browserSupportsWebAuthn, startAuthentication, startRegistration, WebAuthnError } from '@simplewebauthn/browser'
import { createContext, useContext, useMemo } from 'react'
import { authApi, type BipraCategory, type Role, type Service, type User, type WorshipType } from './api'
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
  scoped: 'Pengguna – kelola kolom/BIPRA',
  viewer: 'Pengguna – hanya lihat',
  treasurer: 'Bendahara Kolom',
}

/** Roles limited to some koloms and/or BIPRA categories (a treasurer: koloms). */
function hasScope(u: Pick<User, 'role'> | null) {
  return u?.role === 'scoped' || u?.role === 'viewer' || u?.role === 'treasurer'
}

/** A viewer or treasurer changes no jemaat data (a treasurer only their koloms' kas). */
export function readsOnly(u: Pick<User, 'role'> | null) {
  return u?.role === 'viewer' || u?.role === 'treasurer'
}

/** Whether u may add, change and remove the kas of kolom k: admins, and that kolom's treasurers. */
export function managesKas(u: Pick<User, 'role' | 'koloms'> | null, k: number) {
  return u?.role === 'admin' || (u?.role === 'treasurer' && ownKoloms(u).includes(k))
}

/** The koloms a scoped user manages, a viewer sees, or a treasurer keeps the kas of (others: none). */
export function ownKoloms(u: Pick<User, 'role' | 'koloms'> | null): number[] {
  return hasScope(u) ? (u?.koloms ?? []) : []
}

/** The ids of the BIPRA categories a scoped user manages, or a viewer sees. */
export function ownBipraIds(u: Pick<User, 'role' | 'bipraIds'> | null): string[] {
  return hasScope(u) ? (u?.bipraIds ?? []) : []
}

/**
 * A scoped user or viewer limited to some koloms and/or BIPRA categories
 * (not a viewer of everything). Their data is their koloms' plus their
 * categories' members (in every kolom); see DataProvider.
 */
export function limited(u: Pick<User, 'role' | 'koloms' | 'bipraIds'> | null) {
  return hasScope(u) && (ownKoloms(u).length > 0 || ownBipraIds(u).length > 0)
}

/**
 * A scoped user or viewer with BIPRA categories and no koloms: they see only
 * their categories' members and services, no family pages, and change only
 * those services.
 */
export function bipraOnly(u: Pick<User, 'role' | 'koloms' | 'bipraIds'> | null) {
  return hasScope(u) && !ownKoloms(u).length && ownBipraIds(u).length > 0
}

/** "Admin", "Pengguna", "Kelola Kolom 3, 5", "Kelola Kolom 3 · BIPRA Pemuda, Remaja", "Lihat Kolom 3", "Hanya lihat", "Bendahara Kolom 3" */
export function roleLabel(u: Pick<User, 'role' | 'koloms' | 'bipraIds'>, categories: BipraCategory[] = []) {
  if (!hasScope(u)) return ROLE_LABELS[u.role]
  const parts: string[] = []
  const ks = ownKoloms(u)
  if (ks.length) parts.push(`Kolom ${ks.join(', ')}`)
  const ids = ownBipraIds(u)
  if (ids.length) {
    const names = categories.filter((c) => ids.includes(c.id)).map((c) => c.name)
    parts.push(`BIPRA ${names.join(', ')}`.trim())
  }
  if (u.role === 'viewer') return parts.length ? `Lihat ${parts.join(' · ')}` : 'Hanya lihat'
  if (u.role === 'treasurer') return `Bendahara ${parts.join(' · ')}`.trim()
  return parts.length ? `Kelola ${parts.join(' · ')}` : ROLE_LABELS.scoped
}

/**
 * The BIPRA categories of a BIPRA-only user (see bipraOnly), else null. Their
 * pages show only these categories' members and services.
 */
export function useBipraScope(): BipraCategory[] | null {
  const { user } = useAuth()
  const { categories } = useData()
  const ids = bipraOnly(user) ? ownBipraIds(user).join(',') : ''
  // the same array while nothing changes (callers memoize on it)
  return useMemo(() => (ids ? categories.filter((c) => ids.split(',').includes(c.id)) : null), [ids, categories])
}

/**
 * Whether u may change service s (the server decides the same way): every
 * service of their koloms, and their categories' services in every kolom.
 */
export function managesService(u: User | null, s: Pick<Service, 'kolom' | 'typeId'>, types: WorshipType[]) {
  if (!u || u.mustChangePassword) return false
  if (s.kolom > 0 && managesKolom(u, s.kolom)) return true
  if (managesAll(u)) return true
  return managesType(u, types.find((x) => x.id === s.typeId))
}

/** Whether a scoped user may change services of jenis t in every kolom (linked to one of their categories). */
export function managesType(u: User | null, t: Pick<WorshipType, 'bipraId'> | undefined) {
  return u?.role === 'scoped' && !!t?.bipraId && ownBipraIds(u).includes(t.bipraId)
}

/** Admins and regular users change every kolom's data, the koloms and the jenis ibadah. */
export function managesAll(u: User | null) {
  return !!u && (u.role === 'admin' || u.role === 'user')
}

/** Whether u may change the data of kolom k (families, members, every service there); a scoped user only their own. */
export function managesKolom(u: User | null, k: number) {
  return managesAll(u) || (u?.role === 'scoped' && ownKoloms(u).includes(k))
}

/**
 * Whether the loaded data is the full (signed-in) view, as reported by the
 * server — never guessed from the sign-in state, which can be stale. A
 * limited user's data is narrowed to what the server sends them in full
 * (their koloms and their categories' members; see DataProvider).
 */
export function useFullView() {
  const { view } = useData()
  const { user } = useAuth()
  if (view !== 'full') return false
  return !hasScope(user) || readsOnly(user) || limited(user)
}

/**
 * Data can be changed only when signed in AND the loaded data is the full
 * view, so an edit form is never filled from redacted (blank) fields. A
 * scoped user changes families and members while viewing one of their koloms
 * (or all of them, if they have no categories: then nothing else shows).
 */
export function useCanEdit() {
  const { user } = useAuth()
  const { kolom } = useData()
  const full = useFullView()
  if (!user || user.mustChangePassword || !full || readsOnly(user)) return false
  if (!hasScope(user)) return true
  if (!ownKoloms(user).length) return false
  return kolom !== null ? ownKoloms(user).includes(kolom) : !ownBipraIds(user).length
}

/** Whether the user may add services at all (each one is then checked with managesService). */
export function useCanAddServices() {
  const { user } = useAuth()
  const full = useFullView()
  if (!user || user.mustChangePassword || readsOnly(user)) return false
  return ownBipraIds(user).length > 0 || full
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
