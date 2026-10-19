import { useCallback, useEffect, useState } from 'react'
import { usersApi, type Role, type User } from '../api'
import { useAuth } from '../auth'
import { PasswordStrength } from '../components/PasswordStrength'
import { Badge, ConfirmDelete, Field, FormShell, Modal, PageHeader, Status } from '../components/ui'
import { clockTime, longDate } from '../format'
import { generatePassword, MIN_PASSWORD_LENGTH, passwordValid } from '../password'

type Dialog =
  | { kind: 'add' }
  | { kind: 'edit' | 'reset' | 'delete'; user: User }
  | { kind: 'created'; user: User; password: string }
  | null

export default function AdminUsers() {
  const { user: me } = useAuth()
  const [users, setUsers] = useState<User[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [dialog, setDialog] = useState<Dialog>(null)
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
                      <Badge tone={u.role === 'admin' ? 'pkb' : undefined}>{u.role === 'admin' ? 'Admin' : 'Pengguna'}</Badge>
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
                    <td>{u.lastLoginAt ? longDate(u.lastLoginAt) : <span className="muted">Belum pernah</span>}</td>
                    <td className="actions">
                      <button className="link" onClick={() => setDialog({ kind: 'edit', user: u })}>
                        Ubah
                      </button>
                      <button className="link" onClick={() => setDialog({ kind: 'reset', user: u })}>
                        {locked ? 'Buka Kunci' : 'Reset Password'}
                      </button>
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
      {dialog?.kind === 'edit' && <EditUser user={dialog.user} self={dialog.user.id === me?.id} onClose={close} onSaved={load} />}
      {dialog?.kind === 'reset' && (
        <ResetPassword user={dialog.user} locked={isLocked(dialog.user, loadedAt)} onClose={close} onSaved={load} />
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

function PasswordInput({ value, username, onChange }: { value: string; username: string; onChange: (v: string) => void }) {
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
  const [password, setPassword] = useState(generatePassword)
  return (
    <Modal title="Tambah Pengguna" onClose={onClose}>
      <FormShell
        onCancel={onClose}
        submitLabel="Tambah"
        onSubmit={async () => {
          requireValid(password, username)
          onCreated(await usersApi.create({ username, password, role }), password)
        }}
      >
        <Field label="Username">
          <input autoCapitalize="none" required minLength={3} maxLength={32} value={username} onChange={(e) => setUsername(e.target.value)} />
        </Field>
        <Field label="Role">
          <select value={role} onChange={(e) => setRole(e.target.value as Role)}>
            <option value="user">Pengguna — kelola data jemaat</option>
            <option value="admin">Admin — juga kelola pengguna</option>
          </select>
        </Field>
        <Field label="Password sementara" wide>
          <PasswordInput value={password} username={username} onChange={setPassword} />
        </Field>
        <p className="wide muted small">Pengguna wajib mengganti username/password saat pertama kali masuk.</p>
      </FormShell>
    </Modal>
  )
}

function EditUser({ user, self, onClose, onSaved }: { user: User; self: boolean; onClose: () => void; onSaved: () => Promise<void> }) {
  const [username, setUsername] = useState(user.username)
  const [role, setRole] = useState<Role>(user.role)
  return (
    <Modal title="Ubah Pengguna" onClose={onClose}>
      <FormShell
        onCancel={onClose}
        onSubmit={async () => {
          await usersApi.update(user.id, { username, role })
          await onSaved()
          onClose()
        }}
      >
        <Field label="Username">
          <input autoCapitalize="none" required minLength={3} maxLength={32} value={username} onChange={(e) => setUsername(e.target.value)} />
        </Field>
        <Field label="Role">
          <select value={role} disabled={self} onChange={(e) => setRole(e.target.value as Role)}>
            <option value="user">Pengguna</option>
            <option value="admin">Admin</option>
          </select>
        </Field>
        {self && <p className="wide muted small">Role akun sendiri tidak dapat diubah.</p>}
      </FormShell>
    </Modal>
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
