import { browserSupportsWebAuthn, startAuthentication, startRegistration, WebAuthnError } from '@simplewebauthn/browser'
import { createContext, useContext } from 'react'
import { authApi, type User } from './api'

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
