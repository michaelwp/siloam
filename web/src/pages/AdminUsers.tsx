import { useCallback, useEffect, useState } from 'react'
import { loginsApi, usersApi, type LoginSource, type LoginStats, type Role, type User } from '../api'
import { roleLabel, useAuth } from '../auth'
import { PasswordStrength } from '../components/PasswordStrength'
import { Badge, ConfirmDelete, Field, FormShell, Modal, PageHeader, Status } from '../components/ui'
import { useData } from '../data'
import { clockTime, localDateTime, localToday, longDate } from '../format'
import { generatePassword, MIN_PASSWORD_LENGTH, passwordValid } from '../password'

type Dialog =
  | { kind: 'add' }
  | { kind: 'edit' | 'reset' | 'delete'; user: User }
  | { kind: 'logout'; user: { id: string; username: string } }
  | { kind: 'created'; user: User; password: string }
  | null

export default function AdminUsers() {
  const { user: me } = useAuth()
  const { categories } = useData()
  const [users, setUsers] = useState<User[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [dialog, setDialog] = useState<Dialog>(null)
  const [statsKey, setStatsKey] = useState(0)
  // locks expire on their own; judge them against the time the list was loaded
  const [loadedAt, setLoadedAt] = useState(0)

  const load = useCallback(
    () =>
      usersApi.list().then(
        (u) => {
          setUsers(u)
          setLoadedAt(Date.now())
          setError(null)
          setLoading(false)
        },
        (e: Error) => {
          setError(e.message)
          setLoading(false)
        },
      ),
    [],
  )

  useEffect(() => {
    void load()
  }, [load])

  const close = () => setDialog(null)

  return (
    <>
      <PageHeader title="Pengguna" subtitle="Kelola akun yang dapat masuk ke aplikasi">
        <button className="btn primary" onClick={() => setDialog({ kind: 'add' })}>
          + Tambah Pengguna
        </button>
      </PageHeader>

      <Status loading={loading} error={error} empty={!users.length} />
      {users.length > 0 && (
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Username</th>
                <th>Role</th>
                <th>Status</th>
                <th className="num">Passkey</th>
                <th>Terakhir Masuk</th>
                <th>
                  <span className="sr-only">Aksi</span>
                </th>
              </tr>
            </thead>
            <tbody>
              {users.map((u) => {
                const self = u.id === me?.id
                const locked = isLocked(u, loadedAt)
                return (
                  <tr key={u.id}>
                    <td className="strong">
                      {u.username} {self && <span className="muted">(Anda)</span>}
                    </td>
                    <td>
                      <Badge tone={u.role === 'admin' ? 'pkb' : u.role === 'scoped' ? 'pemuda' : u.role === 'viewer' ? 'amber' : undefined}>
                        {roleLabel(u, categories)}
                      </Badge>
                    </td>
                    <td>
                      {locked ? (
                        <Badge tone="red">Terkunci s.d. {clockTime(u.lockedUntil!)}</Badge>
                      ) : u.mustChangePassword ? (
                        <Badge tone="amber">Wajib ganti password</Badge>
                      ) : (
                        <Badge tone="green">Aktif</Badge>
                      )}
                    </td>
                    <td className="num">{u.passkeys.length}</td>
                    <td>
                      {u.lastLoginAt ? (
                        <>
                          {localDateTime(u.lastLoginAt)}
                          {u.lastLoginSource && <span className="muted"> · {SOURCE_LABELS[u.lastLoginSource]}</span>}
                        </>
                      ) : (
                        <span className="muted">Belum pernah</span>
                      )}
                    </td>
                    <td className="actions">
                      <button className="link" onClick={() => setDialog({ kind: 'edit', user: u })}>
                        Ubah
                      </button>
                      <button className="link" onClick={() => setDialog({ kind: 'reset', user: u })}>
                        {locked ? 'Buka Kunci' : 'Reset Password'}
                      </button>
                      {!self && (
                        <button className="link" onClick={() => setDialog({ kind: 'logout', user: u })}>
                          Keluarkan
                        </button>
                      )}
                      {!self && (
                        <button className="link danger" onClick={() => setDialog({ kind: 'delete', user: u })}>
                          Hapus
                        </button>
                      )}
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}

      <UserStats reload={statsKey} onLogout={(user) => setDialog({ kind: 'logout', user })} />

      {dialog?.kind === 'add' && (
        <AddUser
          onClose={close}
          onCreated={(user, password) => {
            void load()
            setDialog({ kind: 'created', user, password })
          }}
        />
      )}
      {dialog?.kind === 'created' && (
        <Modal title="Pengguna Dibuat" onClose={close}>
          <TempPassword username={dialog.user.username} password={dialog.password} />
          <footer className="modal-foot">
            <button className="btn primary" onClick={close}>
              Selesai
            </button>
          </footer>
        </Modal>
      )}
      {dialog?.kind === 'edit' && (
        <EditUser user={dialog.user} self={dialog.user.id === me?.id} onClose={close} onSaved={load} />
      )}
      {dialog?.kind === 'reset' && (
        <ResetPassword user={dialog.user} locked={isLocked(dialog.user, loadedAt)} onClose={close} onSaved={load} />
      )}
      {dialog?.kind === 'logout' && (
        <LogoutUser
          user={dialog.user}
          onClose={close}
          onDone={() => {
            setStatsKey((k) => k + 1) // the "signed in now" list changed (the chosen date stays)
            close()
          }}
        />
      )}
      {dialog?.kind === 'delete' && (
        <ConfirmDelete
          title="Hapus Pengguna"
          message={
            <>
              Hapus akun <b>{dialog.user.username}</b>? Pengguna ini tidak bisa masuk lagi.
            </>
          }
          onClose={close}
          onConfirm={async () => {
            await usersApi.remove(dialog.user.id)
            await load()
            close()
          }}
        />
      )}
    </>
  )
}

function PasswordInput({
  value,
  username,
  onChange,
}: {
  value: string
  username: string
  onChange: (v: string) => void
}) {
  return (
    <>
      <div className="input-row">
        <input
          type="text"
          autoComplete="off"
          spellCheck={false}
          required
          minLength={MIN_PASSWORD_LENGTH}
          value={value}
          onChange={(e) => onChange(e.target.value)}
        />
        <button type="button" className="btn" onClick={() => onChange(generatePassword())}>
          Buat acak
        </button>
      </div>
      <PasswordStrength password={value} username={username} />
    </>
  )
}

function requireValid(password: string, username: string) {
  if (!passwordValid(password, username)) throw new Error('Password sementara belum memenuhi semua syarat')
}

function TempPassword({ username, password }: { username: string; password: string }) {
  const [copied, setCopied] = useState(false)
  return (
    <div className="form-grid">
      <p className="wide">
        Berikan data berikut kepada pengguna. Password ini hanya ditampilkan sekali, dan pengguna wajib menggantinya saat
        pertama kali masuk.
      </p>
      <Field label="Username">
        <input readOnly value={username} />
      </Field>
      <Field label="Password sementara">
        <div className="input-row">
          <input readOnly value={password} className="mono" />
          <button
            type="button"
            className="btn"
            onClick={() => {
              void navigator.clipboard?.writeText(password).then(() => setCopied(true))
            }}
          >
            {copied ? 'Tersalin' : 'Salin'}
          </button>
        </div>
      </Field>
    </div>
  )
}

function AddUser({ onClose, onCreated }: { onClose: () => void; onCreated: (u: User, password: string) => void }) {
  const [username, setUsername] = useState('')
  const [role, setRole] = useState<Role>('user')
  const [koloms, setKoloms] = useState<number[]>([])
  const [bipraIds, setBipraIds] = useState<string[]>([])
  const [password, setPassword] = useState(generatePassword)
  return (
    <Modal title="Tambah Pengguna" onClose={onClose}>
      <FormShell
        onCancel={onClose}
        submitLabel="Tambah"
        onSubmit={async () => {
          requireValid(password, username)
          requireScope(role, koloms, bipraIds)
          onCreated(
            await usersApi.create({
              username,
              password,
              role,
              ...(role === 'scoped' || role === 'viewer' ? { koloms, bipraIds } : {}),
            }),
            password,
          )
        }}
      >
        <Field label="Username">
          <input autoCapitalize="none" required minLength={3} maxLength={32} value={username} onChange={(e) => setUsername(e.target.value)} />
        </Field>
        <RoleFields
          role={role}
          koloms={koloms}
          bipraIds={bipraIds}
          onRole={setRole}
          onKoloms={setKoloms}
          onBipras={setBipraIds}
        />
        <Field label="Password sementara" wide>
          <PasswordInput value={password} username={username} onChange={setPassword} />
        </Field>
        <p className="wide muted small">Pengguna wajib mengganti username/password saat pertama kali masuk.</p>
      </FormShell>
    </Modal>
  )
}

function EditUser({
  user,
  self,
  onClose,
  onSaved,
}: {
  user: User
  self: boolean
  onClose: () => void
  onSaved: () => Promise<void>
}) {
  const [username, setUsername] = useState(user.username)
  const [role, setRole] = useState<Role>(user.role)
  const [koloms, setKoloms] = useState<number[]>(user.koloms ?? [])
  const [bipraIds, setBipraIds] = useState<string[]>(user.bipraIds ?? [])
  return (
    <Modal title="Ubah Pengguna" onClose={onClose}>
      <FormShell
        onCancel={onClose}
        onSubmit={async () => {
          requireScope(role, koloms, bipraIds)
          await usersApi.update(user.id, {
            username,
            role,
            ...(role === 'scoped' || role === 'viewer' ? { koloms, bipraIds } : {}),
          })
          await onSaved()
          onClose()
        }}
      >
        <Field label="Username">
          <input autoCapitalize="none" required minLength={3} maxLength={32} value={username} onChange={(e) => setUsername(e.target.value)} />
        </Field>
        <RoleFields
          role={role}
          koloms={koloms}
          bipraIds={bipraIds}
          disabled={self}
          onRole={setRole}
          onKoloms={setKoloms}
          onBipras={setBipraIds}
        />
        {self && <p className="wide muted small">Role akun sendiri tidak dapat diubah.</p>}
      </FormShell>
    </Modal>
  )
}

function requireScope(role: Role, koloms: number[], bipraIds: string[]) {
  if (role === 'scoped' && !koloms.length && !bipraIds.length)
    throw new Error('Pilih minimal satu kolom atau kategori BIPRA yang dikelola pengguna ini')
}

/** Adds or removes v from list. */
function toggle<T>(list: T[], v: T, on: boolean) {
  return on ? (list.includes(v) ? list : [...list, v]) : list.filter((x) => x !== v)
}

/**
 * Role, and the koloms and/or BIPRA categories a scoped user manages
 * (at least one) or a viewer sees (none: everything).
 */
function RoleFields({
  role,
  koloms: chosenKoloms,
  bipraIds,
  disabled,
  onRole,
  onKoloms,
  onBipras,
}: {
  role: Role
  koloms: number[]
  bipraIds: string[]
  disabled?: boolean
  onRole: (r: Role) => void
  onKoloms: (k: number[]) => void
  onBipras: (ids: string[]) => void
}) {
  const { koloms, categories } = useData()
  const verb = role === 'viewer' ? 'dilihat' : 'dikelola'
  return (
    <>
      <Field label="Role" wide>
        <select value={role} disabled={disabled} onChange={(e) => onRole(e.target.value as Role)}>
          <option value="user">Pengguna — kelola data semua kolom</option>
          <option value="scoped">Pengguna – kelola kolom/BIPRA — kolom dan/atau kategori BIPRA yang dipilih</option>
          <option value="viewer">Pengguna – hanya lihat — tidak dapat mengubah data</option>
          <option value="admin">Admin — juga kelola pengguna</option>
        </select>
      </Field>
      {(role === 'scoped' || role === 'viewer') && (
        <>
          <fieldset className="wide check-group" disabled={disabled}>
            <legend>Kolom yang {verb} (boleh lebih dari satu)</legend>
            {koloms.map((k) => (
              <label key={k.id} className="check">
                <input
                  type="checkbox"
                  checked={chosenKoloms.includes(k.number)}
                  onChange={(e) => onKoloms(toggle(chosenKoloms, k.number, e.target.checked).toSorted((a, b) => a - b))}
                />
                Kolom {k.number}
              </label>
            ))}
          </fieldset>
          <fieldset className="wide check-group" disabled={disabled}>
            <legend>Kategori BIPRA yang {verb} (boleh lebih dari satu)</legend>
            {categories.map((c) => (
              <label key={c.id} className="check">
                <input
                  type="checkbox"
                  checked={bipraIds.includes(c.id)}
                  onChange={(e) => onBipras(toggle(bipraIds, c.id, e.target.checked))}
                />
                {c.name}
              </label>
            ))}
          </fieldset>
        </>
      )}
      {role === 'viewer' && (
        <p className="wide muted small">
          Hanya dapat melihat data, tidak dapat menambah, mengubah atau menghapus apa pun (kecuali username, password dan
          passkey akunnya sendiri). <b>Tanpa pilihan:</b> melihat data seluruh jemaat. <b>Kolom:</b> semua data kolom
          tersebut, termasuk BIPRA-nya. <b>Kategori BIPRA:</b> anggota dan jadwal ibadah kategori tersebut di semua
          kolom. Keduanya boleh dipilih bersama; pemberitahuan mengikuti pilihan ini.
        </p>
      )}
      {role === 'scoped' && (
        <p className="wide muted small">
          Pilih minimal satu kolom atau kategori BIPRA; keduanya boleh dipilih bersama. <b>Kolom:</b> dapat menambah,
          mengubah dan menghapus keluarga, anggota dan semua jadwal ibadah kolom tersebut (termasuk jadwal BIPRA), dan
          melihat data pribadinya. <b>Kategori BIPRA:</b> dapat mengelola jadwal ibadah jenis yang terhubung ke kategori
          tersebut di semua kolom (atau untuk semua kolom), dan melihat data pribadi anggotanya di semua kolom.
          Pemberitahuan (jadwal dan ulang tahun) mengikuti pilihan ini. Tanpa kolom, halaman keluarga, HUT pernikahan
          dan Kategorial BIPRA disembunyikan.
        </p>
      )}
    </>
  )
}

function isLocked(u: User, at: number) {
  return !!u.lockedUntil && new Date(u.lockedUntil).getTime() > at
}

function ResetPassword({
  user,
  locked,
  onClose,
  onSaved,
}: {
  user: User
  locked: boolean
  onClose: () => void
  onSaved: () => Promise<void>
}) {
  const [password, setPassword] = useState(generatePassword)
  const [clearPasskeys, setClearPasskeys] = useState(false)
  const [done, setDone] = useState(false)

  if (done) {
    return (
      <Modal title={locked ? 'Akun Dibuka & Password Direset' : 'Password Direset'} onClose={onClose}>
        <TempPassword username={user.username} password={password} />
        <footer className="modal-foot">
          <button className="btn primary" onClick={onClose}>
            Selesai
          </button>
        </footer>
      </Modal>
    )
  }

  return (
    <Modal title={`${locked ? 'Buka Kunci & Reset Password' : 'Reset Password'} — ${user.username}`} onClose={onClose}>
      <FormShell
        onCancel={onClose}
        submitLabel={locked ? 'Buka Kunci' : 'Reset'}
        onSubmit={async () => {
          requireValid(password, user.username)
          await usersApi.resetPassword(user.id, { password, clearPasskeys })
          await onSaved()
          setDone(true)
        }}
      >
        {locked && (
          <p className="wide form-error flush">
            Akun ini dikunci sampai pukul {clockTime(user.lockedUntil!)} karena terlalu banyak percobaan masuk yang gagal,
            lalu terbuka otomatis. Reset password untuk membukanya sekarang.
          </p>
        )}
        <Field label="Password sementara baru" wide>
          <PasswordInput value={password} username={user.username} onChange={setPassword} />
        </Field>
        {user.passkeys.length > 0 && (
          <label className="check wide">
            <input type="checkbox" checked={clearPasskeys} onChange={(e) => setClearPasskeys(e.target.checked)} />
            Hapus juga {user.passkeys.length} passkey milik pengguna ini (mis. perangkat hilang)
          </label>
        )}
        <p className="wide muted small">
          Pengguna akan dikeluarkan dari semua perangkat dan wajib mengganti password saat masuk berikutnya.
        </p>
      </FormShell>
    </Modal>
  )
}

const SOURCE_LABELS: Record<LoginSource, string> = { web: 'Web', android: 'Android' }

/** Ends all of a user's sessions; the reason is required and kept in the activity log. */
function LogoutUser({ user, onClose, onDone }: { user: { id: string; username: string }; onClose: () => void; onDone: () => void }) {
  const [reason, setReason] = useState('')
  return (
    <Modal title={`Keluarkan ${user.username}`} onClose={onClose}>
      <FormShell
        onCancel={onClose}
        submitLabel="Keluarkan"
        onSubmit={async () => {
          if (reason.trim().length < 3) throw new Error('Alasan wajib diisi')
          await usersApi.logout(user.id, reason)
          onDone()
        }}
      >
        <p className="wide">
          <b>{user.username}</b> akan dikeluarkan dari semua perangkat (web dan Android). Pengguna tetap dapat masuk lagi;
          untuk mencegahnya, reset password atau hapus akunnya.
        </p>
        <Field label="Alasan (wajib, dicatat di Log Aktivitas)" wide>
          <textarea required minLength={3} maxLength={200} rows={2} value={reason} onChange={(e) => setReason(e.target.value)} />
        </Field>
      </FormShell>
    </Modal>
  )
}

/** Who signed in on a day (with source and role), and who is signed in now. */
function UserStats({ reload, onLogout }: { reload: number; onLogout: (u: { id: string; username: string }) => void }) {
  const { user: me } = useAuth()
  const { categories } = useData()
  // a day in the viewer's time zone (the server keeps every sign-in in UTC)
  const [date, setDate] = useState(localToday)
  // a result with the request it answers, so a slower answer for another date is never shown under this one
  const [result, setResult] = useState<{ key: string; stats?: LoginStats; error?: string } | null>(null)
  const key = `${date}|${reload}`
  useEffect(() => {
    let live = true
    loginsApi.get(date).then(
      (s) => {
        if (live) setResult({ key, stats: s })
      },
      (e: Error) => {
        if (live) setResult({ key, error: e.message })
      },
    )
    return () => {
      live = false
    }
  }, [date, key])
  const current = result?.key === key ? result : null
  const stats = current?.stats ?? null
  const error = current?.error ?? null
  const today = localToday()
  const isToday = date === today
  return (
    <section className="user-stats">
      <header className="family-head">
        <h2 className="section-title">Statistik Pengguna</h2>
        <label className="inline-field">
          <span className="muted small">Tanggal</span>
          <input type="date" value={date} max={today} onChange={(e) => setDate(e.target.value || today)} />
        </label>
      </header>
      <Status loading={!stats && !error} error={error} />
      {stats && (
        <>
          <div className="stats">
            <div className="stat">
              <span className="stat-value">{stats.users}</span>
              <span className="stat-label">Pengguna masuk {isToday ? 'hari ini' : `(${longDate(date)})`}</span>
            </div>
            <div className="stat">
              <span className="stat-value">{stats.web}</span>
              <span className="stat-label">Masuk lewat web</span>
            </div>
            <div className="stat">
              <span className="stat-value">{stats.android}</span>
              <span className="stat-label">Masuk lewat Android</span>
            </div>
            <div className="stat">
              <span className="stat-value">{stats.active.length}</span>
              <span className="stat-label">Sedang masuk sekarang</span>
            </div>
          </div>

          <h3 className="subsection-title">Daftar masuk {isToday ? 'hari ini' : longDate(date)}</h3>
          {stats.logins.length ? (
            <div className="table-wrap">
              <table>
                <thead>
                  <tr>
                    <th>Jam</th>
                    <th>Username</th>
                    <th>Role</th>
                    <th>Sumber</th>
                  </tr>
                </thead>
                <tbody>
                  {stats.logins.map((l) => (
                    <tr key={l.id}>
                      <td>{clockTime(l.at)}</td>
                      <td className="strong">{l.username}</td>
                      <td>{roleLabel(l, categories)}</td>
                      <td>{SOURCE_LABELS[l.source] ?? l.source}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <p className="status">Tidak ada yang masuk pada tanggal ini.</p>
          )}

          <h3 className="subsection-title">Sedang masuk sekarang</h3>
          {stats.active.length ? (
            <div className="table-wrap">
              <table>
                <thead>
                  <tr>
                    <th>Username</th>
                    <th>Role</th>
                    <th>Perangkat</th>
                    <th>Masuk terakhir</th>
                    <th>
                      <span className="sr-only">Aksi</span>
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {stats.active.map((a) => (
                    <tr key={a.userId}>
                      <td className="strong">{a.username}</td>
                      <td>{roleLabel(a, categories)}</td>
                      <td>
                        {[a.web && `${a.web} web`, a.android && `${a.android} Android`].filter(Boolean).join(' · ')}
                      </td>
                      <td>{localDateTime(a.since)}</td>
                      <td className="actions">
                        {a.userId !== me?.id && (
                          <button className="link" onClick={() => onLogout({ id: a.userId, username: a.username })}>
                            Keluarkan
                          </button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <p className="status">Tidak ada.</p>
          )}
        </>
      )}
    </section>
  )
}
