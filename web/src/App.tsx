import { useEffect, useState, type ReactNode } from 'react'
import { BrowserRouter, Link, Navigate, NavLink, Route, Routes, useLocation, useNavigate } from 'react-router-dom'
import { versionApi } from './api'
import { bipraOnly, limited, managesAll, roleLabel, useAuth } from './auth'
import { AuthProvider } from './AuthProvider'
import { DailyNotice } from './components/DailyNotice'
import { ThemeToggle } from './components/ThemeToggle'
import { DataScope } from './DataProvider'
import { ScheduleProvider } from './ScheduleProvider'
import { kolomLabel, useData } from './data'
import Account from './pages/Account'
import AdminAndroid from './pages/AdminAndroid'
import AdminBipra from './pages/AdminBipra'
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
import Kas from './pages/Kas'
import KategorialBipra from './pages/KategorialBipra'
import Kolom from './pages/Kolom'
import Login from './pages/Login'

// The homepage plus one page per sheet of the original workbook. Visitors see
// only the homepage's statistics summary; every other page needs a sign-in.
const PAGES = [
  { path: '/', label: 'Beranda', element: <Beranda /> },
  { path: '/jadwal', label: 'Jadwal Ibadah', element: <JadwalIbadah /> },
  { path: '/keluarga', label: 'Daftar Keluarga', element: <DaftarKeluarga />, noBipra: true },
  { path: '/database', label: 'Data Base Keluarga', element: <DataBaseKeluarga />, noBipra: true },
  { path: '/hut-jemaat', label: 'Daftar HUT Jemaat', element: <HutJemaat /> },
  { path: '/hut-pernikahan', label: 'Daftar HUT Pernikahan', element: <HutPernikahan />, noBipra: true },
  { path: '/anggota', label: 'Daftar Anggota Jemaat', element: <AnggotaJemaat /> },
  { path: '/bipra', label: 'Kategorial BIPRA', element: <KategorialBipra />, noBipra: true },
  { path: '/kas', label: 'Kas Kolom', element: <Kas />, noBipra: true },
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
  // koloms and jenis ibadah: not for a scoped user
  if (all && !managesAll(user)) return <Navigate to="/" replace />
  return <>{children}</>
}

function Shell() {
  const { user, logout } = useAuth()
  const { categories } = useData()
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
        {/* visitors cannot filter: they get the whole congregation's summary only */}
        {user && <KolomPicker />}
        <nav>
          {/* a BIPRA-only user sees only their categories' members: no family pages */}
          {PAGES.filter((p) => (user ? !(p.noBipra && bipraOnly(user)) : p.path === '/')).map((p) => (
            <NavLink key={p.path} to={p.path} end={p.path === '/'}>
              {p.label}
            </NavLink>
          ))}
        </nav>
        <div className="sidebar-footer">
          <AppVersion />
          {user ? (
            <>
              <div className="user-chip">
                <span className="avatar" aria-hidden="true">
                  {user.username.slice(0, 1).toUpperCase()}
                </span>
                <div className="grow">
                  <strong>{user.username}</strong>
                  <small>{roleLabel(user, categories)}</small>
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
                    <NavLink to="/admin/bipra">Kategori BIPRA</NavLink>
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
      {user && <DailyNotice />}
      <main className="content">
        <Routes>
          {PAGES.map((p) => (
            <Route
              key={p.path}
              path={p.path}
              element={
                p.path === '/' ? (
                  p.element
                ) : p.noBipra && bipraOnly(user) ? (
                  <Navigate to="/" replace />
                ) : (
                  <RequireUser>{p.element}</RequireUser>
                )
              }
            />
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
            path="/admin/bipra"
            element={
              <RequireUser admin>
                <AdminBipra />
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

/**
 * Which data every page shows: a kolom ("Semua Kolom": the whole congregation)
 * and a BIPRA category ("Semua BIPRA"), picked separately so both can apply.
 */
function KolomPicker() {
  const { kolomChoices, allKolomsAllowed, bipraChoices, bipraLimited, kolom, setKolom, bipraFilter, setBipra } = useData()
  const { user } = useAuth()
  // a limited user's "all": their koloms plus their categories' members everywhere
  const allLabel = limited(user) ? 'Semua Data Saya' : kolomLabel(null)
  return (
    <div className="data-pickers">
      {kolomChoices.length === 1 && !allKolomsAllowed ? (
        <p className="kolom-picker kolom-fixed" title="Kolom Anda">
          {kolomLabel(kolomChoices[0].number)}
        </p>
      ) : (
        <label className="kolom-picker">
          <span className="sr-only">Kolom</span>
          <select value={kolom ?? ''} onChange={(e) => setKolom(e.target.value ? Number(e.target.value) : null)}>
            {allKolomsAllowed && <option value="">{allLabel}</option>}
            {kolomChoices.map((k) => (
              <option key={k.id} value={k.number}>
                {kolomLabel(k.number)}
              </option>
            ))}
          </select>
        </label>
      )}
      {bipraLimited && bipraChoices.length === 1 ? (
        <p className="kolom-picker kolom-fixed" title="Kategori BIPRA Anda">
          {bipraChoices[0].name}
        </p>
      ) : (
        bipraChoices.length > 0 && (
          <label className="kolom-picker">
            <span className="sr-only">Kategori BIPRA</span>
            <select value={bipraFilter?.id ?? ''} onChange={(e) => setBipra(e.target.value || null)}>
              <option value="">{bipraLimited ? 'Semua BIPRA Saya' : 'Semua BIPRA'}</option>
              {bipraChoices.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                </option>
              ))}
            </select>
          </label>
        )
      )}
    </div>
  )
}

/** "Versi v3.1.0": the version of the server and this web app (they are built together). */
function AppVersion() {
  const [version, setVersion] = useState<string | null>(null)
  useEffect(() => {
    versionApi.get().then(setVersion, () => {})
  }, [])
  if (!version) return null
  return <p className="app-version">Versi {version}</p>
}
