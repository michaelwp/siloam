import { useEffect, useRef, useState, type CSSProperties, type ReactNode } from 'react'
import { useCanEdit } from '../auth'
import { useData } from '../data'

export function Modal({
  title,
  onClose,
  children,
}: {
  title: string
  onClose: () => void
  children: ReactNode
}) {
  const ref = useRef<HTMLDialogElement>(null)

  // native modal dialog: traps focus, closes on Escape (fires "close").
  // No close() in cleanup: unmounting removes it, and a queued "close" event
  // would dismiss the remounted dialog under StrictMode.
  useEffect(() => {
    const dialog = ref.current
    if (!dialog || dialog.open) return
    dialog.showModal()
    dialog.querySelector<HTMLElement>('form input, form select')?.focus()
  }, [])

  return (
    <dialog ref={ref} className="modal" aria-label={title} onClose={onClose}>
      <header className="modal-head">
        <h2>{title}</h2>
        <button className="icon-btn" onClick={onClose} aria-label="Tutup">
          ×
        </button>
      </header>
      {children}
    </dialog>
  )
}

/**
 * Generic form shell that shows the server's error message. In a modal pass
 * onCancel (shows "Batal"); on a page pass successMessage to confirm a save.
 */
export function FormShell({
  onSubmit,
  onCancel,
  submitLabel = 'Simpan',
  successMessage,
  children,
}: {
  onSubmit: () => Promise<void>
  onCancel?: () => void
  submitLabel?: string
  successMessage?: string
  children: ReactNode
}) {
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [saved, setSaved] = useState(false)

  return (
    <form
      // stale errors/confirmations disappear as soon as the user edits a field
      onInput={() => {
        setError(null)
        setSaved(false)
      }}
      onSubmit={async (e) => {
        e.preventDefault()
        setBusy(true)
        setError(null)
        setSaved(false)
        try {
          await onSubmit()
          setSaved(true)
        } catch (err) {
          setError((err as Error).message)
        } finally {
          setBusy(false)
        }
      }}
    >
      <div className="form-grid">{children}</div>
      {error && <p className="form-error">{error}</p>}
      {saved && successMessage && <p className="form-success">{successMessage}</p>}
      <footer className="modal-foot">
        {onCancel && (
          <button type="button" className="btn" onClick={onCancel}>
            Batal
          </button>
        )}
        <button type="submit" className="btn primary" disabled={busy}>
          {busy ? 'Menyimpan…' : submitLabel}
        </button>
      </footer>
    </form>
  )
}

export function Field({ label, wide, children }: { label: string; wide?: boolean; children: ReactNode }) {
  return (
    <label className={wide ? 'field wide' : 'field'}>
      <span>{label}</span>
      {children}
    </label>
  )
}

export function ConfirmDelete({
  title,
  message,
  onConfirm,
  onClose,
}: {
  title: string
  message: ReactNode
  onConfirm: () => Promise<void>
  onClose: () => void
}) {
  return (
    <Modal title={title} onClose={onClose}>
      <FormShell onSubmit={onConfirm} onCancel={onClose} submitLabel="Hapus">
        <p className="wide confirm-text">{message}</p>
      </FormShell>
    </Modal>
  )
}

export function PageHeader({
  title,
  subtitle,
  children,
}: {
  title: string
  subtitle?: ReactNode
  children?: ReactNode
}) {
  return (
    <header className="page-head">
      <div>
        <h1>{title}</h1>
        {subtitle && <p className="muted">{subtitle}</p>}
      </div>
      {children && <div className="page-actions">{children}</div>}
    </header>
  )
}

export function Search({ value, onChange, placeholder = 'Cari nama…' }: {
  value: string
  onChange: (v: string) => void
  placeholder?: string
}) {
  return (
    <input
      className="search"
      type="search"
      value={value}
      placeholder={placeholder}
      onChange={(e) => onChange(e.target.value)}
    />
  )
}

/** Renders its children only for signed-in users (public visitors see read-only pages). */
export function EditOnly({ children }: { children: ReactNode }) {
  return useCanEdit() ? <>{children}</> : null
}

/** Header cell for the row-actions column; absent in the read-only view. */
export function ActionsHeader() {
  return useCanEdit() ? (
    <th className="no-print">
      <span className="sr-only">Aksi</span>
    </th>
  ) : null
}

export function RowActions({ onEdit, onDelete }: { onEdit: () => void; onDelete: () => void }) {
  if (!useCanEdit()) return null
  return (
    <td className="actions no-print">
      <button className="link" onClick={onEdit}>
        Ubah
      </button>
      <button className="link danger" onClick={onDelete}>
        Hapus
      </button>
    </td>
  )
}

export function Status({ loading, error, empty }: { loading: boolean; error: string | null; empty?: boolean }) {
  if (loading) return <p className="status">Memuat data…</p>
  if (error) return <p className="status error">Gagal memuat data: {error}</p>
  if (empty) return <p className="status">Belum ada data.</p>
  return null
}

export function Badge({ children, tone }: { children: ReactNode; tone?: string }) {
  return <span className={`badge ${tone ?? ''}`}>{children}</span>
}

// the default categories keep their hand-tuned light/dark badge colors
const BIPRA_TONES = new Set(['anak', 'remaja', 'pemuda', 'pkb', 'wki', 'lansia'])

/** A member's BIPRA category or flag; others than the defaults show their color as a dot. */
export function BipraBadge({ code }: { code: string }) {
  const c = useData().categories.find((x) => x.key === code)
  const tone = code.toLowerCase()
  if (BIPRA_TONES.has(tone)) return <Badge tone={tone}>{code}</Badge>
  return (
    <Badge>
      {c && <span className="badge-dot" style={{ '--dot': c.color } as CSSProperties} aria-hidden="true" />}
      {code}
    </Badge>
  )
}

/**
 * Lists items per kolom: "Semua kolom" (services for the whole congregation,
 * kolom 0) first, then Kolom 1, 2, … under a small heading each. With items of
 * one kolom only (e.g. one kolom is viewed) there are no headings.
 */
export function ByKolom<T>({
  items,
  kolomOf,
  children,
}: {
  items: T[]
  kolomOf: (item: T) => number
  children: (items: T[]) => ReactNode
}) {
  const koloms = [...new Set(items.map(kolomOf))].toSorted((a, b) => a - b)
  if (koloms.length <= 1) return <>{children(items)}</>
  return (
    <>
      {koloms.map((k) => (
        <div key={k} className="kolom-group">
          <h4 className="kolom-group-head">{k === 0 ? 'Semua kolom' : `Kolom ${k}`}</h4>
          {children(items.filter((i) => kolomOf(i) === k))}
        </div>
      ))}
    </>
  )
}
