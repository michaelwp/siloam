import { checkRules, passwordStrength } from '../password'

/** Strength meter plus a live checklist of the password rules. */
export function PasswordStrength({ password, username }: { password: string; username?: string }) {
  // the length cap is only listed once it is exceeded, to keep the list short
  const rules = checkRules(password, username).filter((r) => r.id !== 'max' || !r.ok)
  const { score, label } = passwordStrength(password, username)

  return (
    <div className="pw-strength" aria-live="polite">
      <div className="pw-meter-row">
        <div className={`pw-meter s${password ? score : 'none'}`} aria-hidden="true">
          {[0, 1, 2, 3].map((i) => (
            <span key={i} className={password && i < Math.max(score, 1) ? 'on' : ''} />
          ))}
        </div>
        <span className="pw-label">{password ? `Kekuatan: ${label}` : 'Kekuatan password'}</span>
      </div>
      <ul className="pw-rules">
        {rules.map((r) => (
          <li key={r.id} className={r.ok && password ? 'ok' : ''}>
            <span aria-hidden="true">{r.ok && password ? '✓' : '•'}</span>
            {r.label}
            <span className="sr-only">{r.ok && password ? ' (terpenuhi)' : ' (belum)'}</span>
          </li>
        ))}
      </ul>
    </div>
  )
}
