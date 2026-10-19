import { useState } from 'react'
import { authApi } from '../api'
import { useAuth } from '../auth'
import { MIN_PASSWORD_LENGTH, passwordValid } from '../password'
import { PasswordStrength } from './PasswordStrength'
import { Field, FormShell } from './ui'

/** Change own username and password (forced on first sign-in, optional later). */
export function CredentialsForm({ forced }: { forced?: boolean }) {
  const { user, setUser } = useAuth()
  const [username, setUsername] = useState(user?.username ?? '')
  const [currentPassword, setCurrentPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirm, setConfirm] = useState('')

  return (
    <FormShell
      submitLabel={forced ? 'Simpan & Lanjutkan' : 'Simpan'}
      successMessage={forced ? undefined : 'Username dan password berhasil diperbarui.'}
      onSubmit={async () => {
        if (!passwordValid(newPassword, username)) throw new Error('Password baru belum memenuhi semua syarat')
        if (newPassword !== confirm) throw new Error('Konfirmasi password tidak sama')
        const u = await authApi.changeCredentials({ currentPassword, username, newPassword })
        setCurrentPassword('')
        setNewPassword('')
        setConfirm('')
        setUser(u)
      }}
    >
      <Field label="Username" wide>
        <input
          name="username"
          autoComplete="username"
          autoCapitalize="none"
          required
          minLength={3}
          maxLength={32}
          pattern="[A-Za-z0-9._\-]+"
          title="Huruf, angka, titik, garis bawah atau tanda hubung"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
        />
      </Field>
      <Field label={forced ? 'Password saat ini (sementara)' : 'Password saat ini'} wide>
        <input
          type="password"
          autoComplete="current-password"
          required
          value={currentPassword}
          onChange={(e) => setCurrentPassword(e.target.value)}
        />
      </Field>
      <Field label="Password baru">
        <input
          type="password"
          autoComplete="new-password"
          required
          minLength={MIN_PASSWORD_LENGTH}
          value={newPassword}
          onChange={(e) => setNewPassword(e.target.value)}
        />
      </Field>
      <Field label="Ulangi password baru">
        <input
          type="password"
          autoComplete="new-password"
          required
          minLength={MIN_PASSWORD_LENGTH}
          value={confirm}
          onChange={(e) => setConfirm(e.target.value)}
        />
      </Field>
      {confirm && confirm !== newPassword && <p className="wide field-hint error">Konfirmasi password tidak sama.</p>}
      <div className="wide">
        <PasswordStrength password={newPassword} username={username} />
      </div>
    </FormShell>
  )
}
