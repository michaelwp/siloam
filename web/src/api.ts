import type {
  AuthenticationResponseJSON,
  PublicKeyCredentialCreationOptionsJSON,
  PublicKeyCredentialRequestOptionsJSON,
  RegistrationResponseJSON,
} from '@simplewebauthn/browser'

/** A neighbourhood group; families belong to one (by number). */
export type Kolom = {
  id: string
  number: number
}

export type Family = {
  id: string
  kolom: number
  no: number // within the kolom
  name: string
  note: string
  domicile: string
  weddingDate: string | null
  memberCount: number
  weddingYears: number | null
  weddingMonth: number | null // 1–12, also for visitors
  weddingDay: number | null
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
  kolom: number // the family's kolom
  age: number | null
  ageText: string
  category: string // the main BIPRA category's key ('' = none)
  flags: string[] | null // keys of the BIPRA flags, e.g. ['Lansia']
  elderly: boolean
  birthMonth: number | null // 1–12, also for visitors
  birthDay: number | null
}

export type FamilyInput = Pick<Family, 'kolom' | 'name' | 'note' | 'domicile' | 'weddingDate'> & { no?: number }

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

/** Sends a request; throws ApiError (and signals auth changes) on failure. */
async function send(method: string, url: string, body?: unknown): Promise<Response> {
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
  return res
}

async function request<T>(method: string, url: string, body?: unknown): Promise<T> {
  const res = await send(method, url, body)
  return res.status === 204 ? (undefined as T) : res.json()
}

/** A Kategorial BIPRA category and its rule (admins manage them). */
export type BipraCategory = {
  id: string
  key: string // what members carry, e.g. 'PKB'
  name: string
  kind: 'main' | 'flag' // main: the first matching one; flag: an extra mark (e.g. Lansia)
  order: number
  color: string
  minAge: number | null
  maxAge: number | null
  gender: '' | 'Laki Laki' | 'Perempuan'
  marital: '' | 'married' | 'unmarried'
  orMarried: boolean // married members pass the age condition whatever their age
}

export type BipraInput = Omit<BipraCategory, 'id' | 'order'>

/** Which data the server sent: everything (signed in) or the redacted public view. */
export type DataView = 'full' | 'public'

async function getWithView<T>(url: string): Promise<[T, DataView]> {
  const res = await send('GET', url)
  return [await res.json(), res.headers.get('X-Data-View') === 'full' ? 'full' : 'public']
}

export const api = {
  /** Families and members, plus which view the server actually returned. */
  load: async () => {
    const [koloms, categories, [families, fv], [members, mv]] = await Promise.all([
      request<Kolom[]>('GET', '/api/koloms'),
      request<BipraCategory[]>('GET', '/api/bipra-categories'),
      getWithView<Family[]>('/api/families'),
      getWithView<Member[]>('/api/members'),
    ])
    return {
      koloms,
      categories,
      families,
      members,
      view: (fv === 'full' && mv === 'full' ? 'full' : 'public') as DataView,
    }
  },
  createKolom: (number: number) => request<Kolom>('POST', '/api/koloms', { number }),
  deleteKolom: (id: string) => request<void>('DELETE', `/api/koloms/${id}`),
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
  return { kolom: f.kolom, no: f.no, name: f.name, note: f.note, domicile: f.domicile, weddingDate: f.weddingDate }
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

export type Role = 'admin' | 'user' | 'kolom'

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
  kolom?: number // role 'kolom': the kolom they manage
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

// ---- admin: Android version & audit log ----

export type AndroidRelease = {
  latestVersion: string
  downloadUrl: string
  updatedAt?: string
  updatedBy?: string
}

/** What the app's update check reports, and where it comes from. */
export type AndroidReleaseView = AndroidRelease & {
  source: 'database' | 'env' | 'none'
  server: AndroidRelease // the server's ANDROID_* settings
}

export type AuditEntry = {
  id: string
  at: string
  userId?: string
  username: string
  category: string
  action: string
  target?: string
  targetId?: string
  method: string
  path: string
  status: number
  ip: string
}

export type AuditFilter = {
  username?: string
  category?: string
  result?: '' | 'ok' | 'failed'
  from?: string // YYYY-MM-DD
  to?: string
  before?: string
}

export const adminApi = {
  android: () => request<AndroidReleaseView>('GET', '/api/admin/android'),
  setAndroid: (body: { latestVersion: string; downloadUrl: string }) =>
    request<AndroidReleaseView>('PUT', '/api/admin/android', body),
  clearAndroid: () => request<AndroidReleaseView>('DELETE', '/api/admin/android'),
  audit: (f: AuditFilter) => {
    const q = new URLSearchParams(Object.entries(f).filter((e): e is [string, string] => !!e[1]))
    return request<AuditEntry[]>('GET', `/api/admin/audit?${q}`)
  },
  createBipra: (body: BipraInput) => request<BipraCategory>('POST', '/api/admin/bipra', body),
  updateBipra: (id: string, body: BipraInput) => request<BipraCategory>('PUT', `/api/admin/bipra/${id}`, body),
  deleteBipra: (id: string) => request<void>('DELETE', `/api/admin/bipra/${id}`),
  reorderBipra: (ids: string[]) => request<BipraCategory[]>('PUT', '/api/admin/bipra/order', { ids }),
  auditMeta: () =>
    request<{ categories: { key: string; label: string }[]; usernames: string[] }>('GET', '/api/admin/audit/meta'),
}

export const usersApi = {
  list: () => request<User[]>('GET', '/api/users'),
  create: (body: { username: string; password: string; role: Role; kolom?: number }) =>
    request<User>('POST', '/api/users', body),
  update: (id: string, body: { username: string; role: Role; kolom?: number }) =>
    request<User>('PUT', `/api/users/${id}`, body),
  resetPassword: (id: string, body: { password: string; clearPasskeys: boolean }) =>
    request<User>('POST', `/api/users/${id}/reset-password`, body),
  remove: (id: string) => request<void>('DELETE', `/api/users/${id}`),
}

// ---- jadwal ibadah ----

/** A kind of service ("jenis ibadah") with its calendar color. */
export type WorshipType = {
  id: string
  name: string
  color: string // #rrggbb
  order: number
}

export type Service = {
  id: string
  kolom: number
  typeId: string
  date: string // midnight UTC of the local date
  time: string // "19:00"
  host: string // host family, without "Kel."
  location: string
  leader: string // khadim, who leads the service (the member's current name when linked)
  leaderId: string | null // the member, or null for someone from outside
  mc: string // pembawa acara
  mcId: string | null
  note: string
  typeName: string
  typeColor: string
  start: string // absolute start/end (RFC 3339), for calendar links
  end: string
}

export type ServiceInput = Pick<
  Service,
  'kolom' | 'typeId' | 'time' | 'host' | 'location' | 'leader' | 'leaderId' | 'mc' | 'mcId' | 'note'
> & {
  date: string | null
}

export function serviceToInput(s: Service): ServiceInput {
  const { kolom, typeId, date, time, host, location, leader, leaderId, mc, mcId, note } = s
  return { kolom, typeId, date, time, host, location, leader, leaderId, mc, mcId, note }
}

export const scheduleApi = {
  load: () =>
    Promise.all([request<Service[]>('GET', '/api/services'), request<WorshipType[]>('GET', '/api/worship-types')]),
  createService: (s: ServiceInput) => request<Service>('POST', '/api/services', s),
  updateService: (id: string, s: ServiceInput) => request<Service>('PUT', `/api/services/${id}`, s),
  deleteService: (id: string) => request<void>('DELETE', `/api/services/${id}`),
  createType: (t: { name: string; color: string }) => request<WorshipType>('POST', '/api/worship-types', t),
  updateType: (id: string, t: { name: string; color: string }) => request<WorshipType>('PUT', `/api/worship-types/${id}`, t),
  deleteType: (id: string) => request<void>('DELETE', `/api/worship-types/${id}`),
}
