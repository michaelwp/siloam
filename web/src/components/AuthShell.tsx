import type { ReactNode } from 'react'
import { ThemeToggle } from './ThemeToggle'

/** Centered card used by the sign-in and forced password-change screens. */
export function AuthShell({
  title,
  subtitle,
  children,
}: {
  title: string
  subtitle?: ReactNode
  children: ReactNode
}) {
  return (
    <main className="auth-page">
      <div className="auth-corner">
        <ThemeToggle />
      </div>
      <section className="auth-card">
        <div className="brand">
          <span className="brand-mark">GS</span>
          <div>
            <strong>GMIM Siloam</strong>
            <small>Data Base Jemaat</small>
          </div>
        </div>
        <h1>{title}</h1>
        {subtitle && <p className="muted auth-subtitle">{subtitle}</p>}
        {children}
      </section>
    </main>
  )
}
