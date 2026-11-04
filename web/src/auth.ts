import { browserSupportsWebAuthn, startAuthentication, startRegistration, WebAuthnError } from '@simplewebauthn/browser'
import { createContext, useContext } from 'react'
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
}

/** Roles limited to some koloms and/or BIPRA categories. */
function hasScope(u: Pick<User, 'role'> | null) {
  return u?.role === 'scoped' || u?.role === 'viewer'
}

/** A viewer only looks: they change no jemaat data. */
export function isViewer(u: Pick<User, 'role'> | null) {
  return u?.role === 'viewer'
}

/** The koloms a scoped user manages, or a viewer sees (others: none). */
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

/** "Admin", "Pengguna", "Kelola Kolom 3, 5", "Kelola Kolom 3 · BIPRA Pemuda, Remaja", "Lihat Kolom 3", "Hanya lihat" */
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
  return parts.length ? `Kelola ${parts.join(' · ')}` : ROLE_LABELS.scoped
}

/**
 * The BIPRA categories of a BIPRA-only user (see bipraOnly), else null. Their
 * pages show only these categories' members and services.
 */
export function useBipraScope(): BipraCategory[] | null {
  const { user } = useAuth()
  const { categories } = useData()
  if (!bipraOnly(user)) return null
  const ids = ownBipraIds(user)
  return categories.filter((c) => ids.includes(c.id))
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
  return !hasScope(user) || isViewer(user) || limited(user)
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
  if (!user || user.mustChangePassword || !full || isViewer(user)) return false
  if (!hasScope(user)) return true
  if (!ownKoloms(user).length) return false
  return kolom !== null ? ownKoloms(user).includes(kolom) : !ownBipraIds(user).length
}

/** Whether the user may add services at all (each one is then checked with managesService). */
export function useCanAddServices() {
  const { user } = useAuth()
  const full = useFullView()
  if (!user || user.mustChangePassword || isViewer(user)) return false
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
