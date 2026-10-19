import { useState } from 'react'

type Theme = 'light' | 'dark'

// index.html sets data-theme before first paint (saved choice, else system)
function readTheme(): Theme {
  return document.documentElement.dataset.theme === 'dark' ? 'dark' : 'light'
}

export function ThemeToggle({ withLabel }: { withLabel?: boolean }) {
  const [theme, setTheme] = useState<Theme>(readTheme)
  const next: Theme = theme === 'dark' ? 'light' : 'dark'
  const label = next === 'dark' ? 'Mode gelap' : 'Mode terang'

  return (
    <button
      type="button"
      className={withLabel ? 'btn theme-toggle' : 'icon-btn theme-toggle'}
      aria-label={label}
      title={label}
      onClick={() => {
        document.documentElement.dataset.theme = next
        try {
          localStorage.setItem('theme', next)
        } catch {
          // storage blocked: the choice lasts for this page only
        }
        setTheme(next)
      }}
    >
      {next === 'dark' ? (
        <svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true">
          <path fill="currentColor" d="M21 14.5A9 9 0 0 1 9.5 3a7 7 0 1 0 11.5 11.5Z" />
        </svg>
      ) : (
        <svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true">
          <circle cx="12" cy="12" r="4.5" fill="currentColor" />
          <path
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4"
          />
        </svg>
      )}
      {withLabel && <span>{label}</span>}
    </button>
  )
}
