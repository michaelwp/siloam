import { useAuth } from '../auth'
import { AuthShell } from '../components/AuthShell'
import { CredentialsForm } from '../components/CredentialsForm'

/** Shown instead of the app until a new/reset account sets its own password. */
export default function ChangeCredentials() {
  const { user, logout } = useAuth()
  return (
    <AuthShell
      title="Perbarui Akun"
      subtitle={
        <>
          Halo <b>{user?.username}</b>, demi keamanan silakan ganti username dan password sementara sebelum
          melanjutkan.
        </>
      }
    >
      <div className="auth-form flush">
        <CredentialsForm forced />
      </div>
      <button type="button" className="link block-link" onClick={() => void logout()}>
        Keluar
      </button>
    </AuthShell>
  )
}
