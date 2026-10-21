import { useState } from 'react'
import { authApi, type Passkey } from '../api'
import { passkeyError, passkeysSupported, registerPasskey, roleLabel, useAuth } from '../auth'
import { CredentialsForm } from '../components/CredentialsForm'
import { ConfirmDelete, Field, FormShell, Modal, PageHeader } from '../components/ui'
import { longDate } from '../format'
import { KeyIcon } from './Login'

export default function Account() {
  const { user } = useAuth()
  if (!user) return null

  return (
    <>
      <PageHeader title="Akun Saya" subtitle={<>Masuk sebagai <b>{user.username}</b> ({roleLabel(user)})</>} />
      <section className="card panel">
        <header className="panel-head">
          <h2>Username & Password</h2>
          <p className="muted">Mengganti password akan mengeluarkan akun Anda dari perangkat lain.</p>
        </header>
        <CredentialsForm />
      </section>
      <Passkeys passkeys={user.passkeys} />
    </>
  )
}

function Passkeys({ passkeys }: { passkeys: Passkey[] }) {
  const { setUser } = useAuth()
  const [name, setName] = useState(defaultDeviceName)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [renaming, setRenaming] = useState<Passkey | null>(null)
  const [deleting, setDeleting] = useState<Passkey | null>(null)
  const supported = passkeysSupported()

  return (
    <section className="card panel">
      <header className="panel-head">
        <h2>Passkey</h2>
        <p className="muted">
          Masuk tanpa password menggunakan sidik jari, Face ID, PIN perangkat atau kunci keamanan.
        </p>
      </header>

      {passkeys.length > 0 ? (
        <ul className="passkey-list">
          {passkeys.map((p) => (
            <li key={p.id}>
              <KeyIcon />
              <div className="grow">
                <strong>{p.name}</strong>
                <span className="muted small">
                  Dibuat {longDate(p.createdAt)} · {p.lastUsedAt ? `terakhir dipakai ${longDate(p.lastUsedAt)}` : 'belum pernah dipakai'}
                </span>
              </div>
              <button className="link" onClick={() => setRenaming(p)}>
                Ganti nama
              </button>
              <button className="link danger" onClick={() => setDeleting(p)}>
                Hapus
              </button>
            </li>
          ))}
        </ul>
      ) : (
        <p className="status">Belum ada passkey.</p>
      )}

      {supported ? (
        <form
          className="passkey-add"
          onSubmit={async (e) => {
            e.preventDefault()
            setBusy(true)
            setError(null)
            try {
              setUser(await registerPasskey(name))
            } catch (err) {
              setError(passkeyError(err))
            } finally {
              setBusy(false)
            }
          }}
        >
          <label className="field grow">
            <span>Nama perangkat</span>
            <input value={name} maxLength={64} onChange={(e) => setName(e.target.value)} placeholder="mis. iPhone Michael" />
          </label>
          <button className="btn primary" disabled={busy}>
            {busy ? 'Menunggu perangkat…' : '+ Tambah Passkey'}
          </button>
        </form>
      ) : (
        <p className="status">
          Browser ini tidak mendukung passkey, atau aplikasi tidak dibuka lewat HTTPS / localhost.
        </p>
      )}
      {error && <p className="form-error">{error}</p>}

      {renaming && (
        <Modal title="Ganti Nama Passkey" onClose={() => setRenaming(null)}>
          <RenameForm passkey={renaming} onClose={() => setRenaming(null)} />
        </Modal>
      )}
      {deleting && (
        <ConfirmDelete
          title="Hapus Passkey"
          message={
            <>
              Hapus passkey <b>{deleting.name}</b>? Perangkat tersebut tidak bisa lagi dipakai untuk masuk.
            </>
          }
          onClose={() => setDeleting(null)}
          onConfirm={async () => {
            setUser(await authApi.deletePasskey(deleting.id))
            setDeleting(null)
          }}
        />
      )}
    </section>
  )
}

function RenameForm({ passkey, onClose }: { passkey: Passkey; onClose: () => void }) {
  const { setUser } = useAuth()
  const [name, setName] = useState(passkey.name)
  return (
    <FormShell
      onCancel={onClose}
      onSubmit={async () => {
        setUser(await authApi.renamePasskey(passkey.id, name))
        onClose()
      }}
    >
      <Field label="Nama" wide>
        <input required maxLength={64} value={name} onChange={(e) => setName(e.target.value)} />
      </Field>
    </FormShell>
  )
}

function defaultDeviceName() {
  const ua = navigator.userAgent
  if (/iPhone/.test(ua)) return 'iPhone'
  if (/iPad/.test(ua)) return 'iPad'
  if (/Android/.test(ua)) return 'Android'
  if (/Mac/.test(ua)) return 'Mac'
  if (/Windows/.test(ua)) return 'Windows'
  return ''
}
