import type {
  AuthenticationResponseJSON,
  PublicKeyCredentialCreationOptionsJSON,
  PublicKeyCredentialRequestOptionsJSON,
  RegistrationResponseJSON,
} from '@simplewebauthn/browser'

export type Family = {
  id: string
  no: number
  name: string
  note: string
  domicile: string
  weddingDate: string | null
  memberCount: number
  weddingYears: number | null
}

export type Member = {
  id: string
  familyId: string
  order: number
  name: string
  birthPlace: string
  birthDate: string | null
  gender: string
  maritalStatus: string
  relationship: string
  baptized: boolean | null
  confirmed: boolean | null
  familyName: string
  age: number | null
  ageText: string
  category: string
  elderly: boolean
}

export type FamilyInput = Pick<Family, 'name' | 'note' | 'domicile' | 'weddingDate'> & { no?: number }

export type MemberInput = Pick<
  Member,
  | 'familyId'
  | 'name'
  | 'birthPlace'
  | 'birthDate'
  | 'gender'
  | 'maritalStatus'
  | 'relationship'
  | 'baptized'
  | 'confirmed'
> & { order?: number }

export class ApiError extends Error {
  status: number
  code?: string

  constructor(message: string, status: number, code?: string) {
    super(message)
    this.status = status
    this.code = code
  }
}

/** Fired when the server says the session ended or a password change is required. */
export const AUTH_CHANGED = 'auth:changed'

async function request<T>(method: string, url: string, body?: unknown): Promise<T> {
  const res = await fetch(url, {
    method,
    credentials: 'same-origin',
    headers: body === undefined ? undefined : { 'Content-Type': 'application/json' },
    body: body === undefined ? undefined : JSON.stringify(body),
  })
  if (!res.ok) {
    const data = await res.json().catch(() => null)
    const err = new ApiError(data?.error ?? `Permintaan gagal (${res.status})`, res.status, data?.code)
    if ((err.code === 'unauthenticated' || err.code === 'password_change_required') && url !== '/api/auth/me') {
      window.dispatchEvent(new Event(AUTH_CHANGED))
    }
    throw err
  }
  return res.status === 204 ? (undefined as T) : res.json()
}

export const api = {
  families: () => request<Family[]>('GET', '/api/families'),
  createFamily: (f: FamilyInput) => request<Family>('POST', '/api/families', f),
  updateFamily: (id: string, f: FamilyInput) => request<Family>('PUT', `/api/families/${id}`, f),
  deleteFamily: (id: string) => request<void>('DELETE', `/api/families/${id}`),

  members: () => request<Member[]>('GET', '/api/members'),
  createMember: (m: MemberInput) => request<Member>('POST', '/api/members', m),
  updateMember: (id: string, m: MemberInput) => request<Member>('PUT', `/api/members/${id}`, m),
  deleteMember: (id: string) => request<void>('DELETE', `/api/members/${id}`),
}

export function familyToInput(f: Family): FamilyInput {
  return { no: f.no, name: f.name, note: f.note, domicile: f.domicile, weddingDate: f.weddingDate }
}

export function memberToInput(m: Member): MemberInput {
  return {
    familyId: m.familyId,
    order: m.order,
    name: m.name,
    birthPlace: m.birthPlace,
    birthDate: m.birthDate,
    gender: m.gender,
    maritalStatus: m.maritalStatus,
    relationship: m.relationship,
    baptized: m.baptized,
    confirmed: m.confirmed,
  }
}

// ---- auth & users ----

export type Role = 'admin' | 'user'

export type Passkey = {
  id: string
  name: string
  createdAt: string
  lastUsedAt: string | null
}

export type User = {
  id: string
  username: string
  role: Role
  mustChangePassword: boolean
  failedLogins: number
  lockedAt: string | null
  lockedUntil: string | null
  passkeys: Passkey[]
  lastLoginAt: string | null
  createdAt: string
}

export const authApi = {
  config: () => request<{ maxFailedLogins: number; lockSeconds: number }>('GET', '/api/auth/config'),
  me: () => request<User>('GET', '/api/auth/me'),
  login: (username: string, password: string) => request<User>('POST', '/api/auth/login', { username, password }),
  logout: () => request<void>('POST', '/api/auth/logout'),
  changeCredentials: (body: { currentPassword: string; username: string; newPassword: string }) =>
    request<User>('POST', '/api/auth/change-credentials', body),

  passkeyLoginBegin: () =>
    request<{ publicKey: PublicKeyCredentialRequestOptionsJSON }>('POST', '/api/auth/passkey/login/begin'),
  passkeyLoginFinish: (cred: AuthenticationResponseJSON) => request<User>('POST', '/api/auth/passkey/login/finish', cred),
  passkeyRegisterBegin: () =>
    request<{ publicKey: PublicKeyCredentialCreationOptionsJSON }>('POST', '/api/auth/passkeys/register/begin'),
  passkeyRegisterFinish: (name: string, cred: RegistrationResponseJSON) =>
    request<User>('POST', `/api/auth/passkeys/register/finish?name=${encodeURIComponent(name)}`, cred),
  renamePasskey: (id: string, name: string) => request<User>('PUT', `/api/auth/passkeys/${id}`, { name }),
  deletePasskey: (id: string) => request<User>('DELETE', `/api/auth/passkeys/${id}`),
}

export const usersApi = {
  list: () => request<User[]>('GET', '/api/users'),
  create: (body: { username: string; password: string; role: Role }) => request<User>('POST', '/api/users', body),
  update: (id: string, body: { username: string; role: Role }) => request<User>('PUT', `/api/users/${id}`, body),
  resetPassword: (id: string, body: { password: string; clearPasskeys: boolean }) =>
    request<User>('POST', `/api/users/${id}/reset-password`, body),
  remove: (id: string) => request<void>('DELETE', `/api/users/${id}`),
}
