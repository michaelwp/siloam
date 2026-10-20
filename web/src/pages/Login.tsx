import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { authApi } from '../api'
import { passkeyError, passkeysSupported, signInWithPasskey, useAuth } from '../auth'
import { AuthShell } from '../components/AuthShell'
import { humanDuration } from '../format'

export default function Login() {
  const { setUser } = useAuth()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [policy, setPolicy] = useState<{ maxFailedLogins: number; lockSeconds: number } | null>(null)

  useEffect(() => {
    authApi.config().then(setPolicy, () => setPolicy(null))
  }, [])

  const run = async (fn: () => Promise<void>) => {
    setBusy(true)
    setError(null)
    try {
      await fn()
    } catch (e) {
      setError(passkeyError(e))
      setBusy(false)
    }
  }

  return (
    <AuthShell title="Masuk" subtitle="Masuk untuk mengubah data jemaat dan mengelola pengguna.">
      <form
        className="auth-form"
        onSubmit={(e) => {
          e.preventDefault()
          void run(async () => setUser(await authApi.login(username, password)))
        }}
      >
        <label className="field">
          <span>Username</span>
          <input
            name="username"
            autoComplete="username"
            autoCapitalize="none"
            required
            value={username}
            onChange={(e) => setUsername(e.target.value)}
          />
        </label>
        <label className="field">
          <span>Password</span>
          <input
            name="password"
            type="password"
            autoComplete="current-password"
            required
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
        </label>
        {error && <p className="form-error inline">{error}</p>}
        {policy && (
          <p className="muted small">
            Setelah {policy.maxFailedLogins} kali salah password, akun dikunci selama{' '}
            {humanDuration(policy.lockSeconds)}.
          </p>
        )}
        <button type="submit" className="btn primary block" disabled={busy}>
          {busy ? 'Memproses…' : 'Masuk'}
        </button>
      </form>

      {passkeysSupported() && (
        <>
          <div className="divider">atau</div>
          <button
            type="button"
            className="btn block"
            disabled={busy}
            onClick={() => void run(async () => setUser(await signInWithPasskey()))}
          >
            <KeyIcon /> Masuk dengan Passkey
          </button>
        </>
      )}
      <Link to="/" className="block-link small">
        ← Kembali ke beranda
      </Link>
    </AuthShell>
  )
}

export function KeyIcon() {
  return (
    <svg viewBox="0 0 24 24" width="16" height="16" aria-hidden="true" className="btn-icon">
      <circle cx="8" cy="15" r="4" fill="none" stroke="currentColor" strokeWidth="2" />
      <path d="M11 12l9-9m-3 3l3 3m-6 0l2 2" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
    </svg>
  )
}
