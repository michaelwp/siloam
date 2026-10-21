import type { ReactNode } from 'react'
import { BrowserRouter, Link, Navigate, NavLink, Route, Routes, useLocation, useNavigate } from 'react-router-dom'
import { managesAll, roleLabel, useAuth } from './auth'
import { AuthProvider } from './AuthProvider'
import { DailyNotice } from './components/DailyNotice'
import { ThemeToggle } from './components/ThemeToggle'
import { DataScope } from './DataProvider'
import { ScheduleProvider } from './ScheduleProvider'
import { kolomLabel, useData } from './data'
import Account from './pages/Account'
import AdminAndroid from './pages/AdminAndroid'
import AdminUsers from './pages/AdminUsers'
import AnggotaJemaat from './pages/AnggotaJemaat'
import AuditLog from './pages/AuditLog'
import Beranda from './pages/Beranda'
import ChangeCredentials from './pages/ChangeCredentials'
import DaftarKeluarga from './pages/DaftarKeluarga'
import DataBaseKeluarga from './pages/DataBaseKeluarga'
import HutJemaat from './pages/HutJemaat'
import HutPernikahan from './pages/HutPernikahan'
import JadwalIbadah from './pages/JadwalIbadah'
import JenisIbadah from './pages/JenisIbadah'
import KategorialBipra from './pages/KategorialBipra'
import Kolom from './pages/Kolom'
import Login from './pages/Login'

// The homepage plus one page per sheet of the original workbook. Visitors see
// them read-only; signed-in users can also edit.
const PAGES = [
  { path: '/', label: 'Beranda', element: <Beranda /> },
  { path: '/jadwal', label: 'Jadwal Ibadah', element: <JadwalIbadah /> },
  { path: '/keluarga', label: 'Daftar Keluarga', element: <DaftarKeluarga /> },
  { path: '/database', label: 'Data Base Keluarga', element: <DataBaseKeluarga /> },
  { path: '/hut-jemaat', label: 'Daftar HUT Jemaat', element: <HutJemaat /> },
  { path: '/hut-pernikahan', label: 'Daftar HUT Pernikahan', element: <HutPernikahan /> },
  { path: '/anggota', label: 'Daftar Anggota Jemaat', element: <AnggotaJemaat /> },
  { path: '/bipra', label: 'Kategorial BIPRA', element: <KategorialBipra /> },
]

export default function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <DataScope>
          <ScheduleProvider>
            <Routes>
              <Route path="/login" element={<LoginRoute />} />
              <Route path="*" element={<Gate />} />
            </Routes>
          </ScheduleProvider>
        </DataScope>
      </AuthProvider>
    </BrowserRouter>
  )
}

type FromState = { from?: string } | null

/** /login; once signed in, return to the page that asked for it (or home). */
function LoginRoute() {
  const { user, loading } = useAuth()
  const from = (useLocation().state as FromState)?.from ?? '/'
  if (loading) return <p className="status splash">Memuat…</p>
  if (user) return <Navigate to={user.mustChangePassword ? '/' : from} replace />
  return <Login />
}

/** Forced credential change for new/reset accounts; otherwise the app. */
function Gate() {
  const { user, loading } = useAuth()
  if (loading) return <p className="status splash">Memuat…</p>
  if (user?.mustChangePassword) return <ChangeCredentials />
  return <Shell />
}

/** Pages that need a signed-in user send visitors to /login. */
function RequireUser({ children, admin, all }: { children: ReactNode; admin?: boolean; all?: boolean }) {
  const { user } = useAuth()
  const location = useLocation()
  if (!user) return <Navigate to="/login" replace state={{ from: location.pathname }} />
  if (admin && user.role !== 'admin') return <Navigate to="/" replace />
  // koloms and jenis ibadah: not for a "kelola data kolom" user
  if (all && !managesAll(user)) return <Navigate to="/" replace />
  return <>{children}</>
}

function Shell() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const isAdmin = user?.role === 'admin'

  return (
    <div className="layout">
      <aside className="sidebar no-print">
        <Link to="/" className="brand">
          <span className="brand-mark">GS</span>
          <div>
            <strong>GMIM Siloam</strong>
            <small>Data Base Jemaat</small>
          </div>
        </Link>
        <KolomPicker />
        <nav>
          {PAGES.map((p) => (
            <NavLink key={p.path} to={p.path} end={p.path === '/'}>
              {p.label}
            </NavLink>
          ))}
        </nav>
        <div className="sidebar-footer">
          {user ? (
            <>
              <div className="user-chip">
                <span className="avatar" aria-hidden="true">
                  {user.username.slice(0, 1).toUpperCase()}
                </span>
                <div className="grow">
                  <strong>{user.username}</strong>
                  <small>{roleLabel(user)}</small>
                </div>
                <ThemeToggle />
              </div>
              <nav>
                {managesAll(user) && (
                  <>
                    <NavLink to="/kolom">Kelola Kolom</NavLink>
                    <NavLink to="/jenis-ibadah">Jenis Ibadah</NavLink>
                  </>
                )}
                <NavLink to="/akun">Akun Saya</NavLink>
                {isAdmin && (
                  <>
                    <NavLink to="/admin/pengguna">Pengguna</NavLink>
                    <NavLink to="/admin/android">Aplikasi Android</NavLink>
                    <NavLink to="/admin/log">Log Aktivitas</NavLink>
                  </>
                )}
                <button
                  type="button"
                  className="nav-button"
                  onClick={async () => {
                    // leave signed-in-only pages first, or their guard redirects to /login
                    navigate('/')
                    await logout()
                  }}
                >
                  Keluar
                </button>
              </nav>
            </>
          ) : (
            <div className="user-chip">
              <Link to="/login" state={{ from: location.pathname }} className="btn primary grow">
                Masuk
              </Link>
              <ThemeToggle />
            </div>
          )}
        </div>
      </aside>
      <DailyNotice />
      <main className="content">
        <Routes>
          {PAGES.map((p) => (
            <Route key={p.path} path={p.path} element={p.element} />
          ))}
          <Route
            path="/jenis-ibadah"
            element={
              <RequireUser all>
                <JenisIbadah />
              </RequireUser>
            }
          />
          <Route
            path="/kolom"
            element={
              <RequireUser all>
                <Kolom />
              </RequireUser>
            }
          />
          <Route
            path="/akun"
            element={
              <RequireUser>
                <Account />
              </RequireUser>
            }
          />
          <Route
            path="/admin/pengguna"
            element={
              <RequireUser admin>
                <AdminUsers />
              </RequireUser>
            }
          />
          <Route
            path="/admin/android"
            element={
              <RequireUser admin>
                <AdminAndroid />
              </RequireUser>
            }
          />
          <Route
            path="/admin/log"
            element={
              <RequireUser admin>
                <AuditLog />
              </RequireUser>
            }
          />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </main>
    </div>
  )
}

/** Which kolom every page shows; "Semua Kolom" shows the whole congregation. */
function KolomPicker() {
  const { koloms, kolom, setKolom } = useData()
  const { user } = useAuth()
  // a "kelola data kolom" user only reads the other koloms (as visitors see them)
  const own = user?.role === 'kolom' ? user.kolom : undefined
  return (
    <>
      <label className="kolom-picker">
        <span className="sr-only">Kolom yang ditampilkan</span>
        <select value={kolom ?? ''} onChange={(e) => setKolom(e.target.value ? Number(e.target.value) : null)}>
          <option value="">{kolomLabel(null)}</option>
          {koloms.map((k) => (
            <option key={k.id} value={k.number}>
              {kolomLabel(k.number)}
            </option>
          ))}
        </select>
      </label>
      {own !== undefined && kolom !== own && (
        <p className="kolom-note small">
          Hanya baca. Anda mengelola data Kolom {own}.{' '}
          <button type="button" className="link" onClick={() => setKolom(own)}>
            Buka Kolom {own}
          </button>
        </p>
      )}
    </>
  )
}
