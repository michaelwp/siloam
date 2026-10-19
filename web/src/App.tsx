import { BrowserRouter, Navigate, NavLink, Route, Routes } from 'react-router-dom'
import { useAuth } from './auth'
import { AuthProvider } from './AuthProvider'
import { ThemeToggle } from './components/ThemeToggle'
import { DataProvider } from './DataProvider'
import Account from './pages/Account'
import AdminUsers from './pages/AdminUsers'
import AnggotaJemaat from './pages/AnggotaJemaat'
import ChangeCredentials from './pages/ChangeCredentials'
import DaftarKeluarga from './pages/DaftarKeluarga'
import DataBaseKeluarga from './pages/DataBaseKeluarga'
import HutJemaat from './pages/HutJemaat'
import HutPernikahan from './pages/HutPernikahan'
import KategorialBipra from './pages/KategorialBipra'
import Login from './pages/Login'

// One page per sheet of the original workbook.
const PAGES = [
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
        <Gate />
      </AuthProvider>
    </BrowserRouter>
  )
}

/** Sign-in → forced credential change → app. */
function Gate() {
  const { user, loading } = useAuth()
  if (loading) return <p className="status splash">Memuat…</p>
  if (!user) return <Login />
  if (user.mustChangePassword) return <ChangeCredentials />
  return (
    <DataProvider>
      <Shell />
    </DataProvider>
  )
}

function Shell() {
  const { user, logout } = useAuth()
  const isAdmin = user?.role === 'admin'

  return (
    <div className="layout">
      <aside className="sidebar no-print">
        <div className="brand">
          <span className="brand-mark">K9</span>
          <div>
            <strong>GMIM Siloam</strong>
            <small>Jemaat Kolom 9</small>
          </div>
        </div>
        <nav>
          {PAGES.map((p) => (
            <NavLink key={p.path} to={p.path}>
              {p.label}
            </NavLink>
          ))}
        </nav>
        <div className="sidebar-footer">
          <div className="user-chip">
            <span className="avatar" aria-hidden="true">
              {user?.username.slice(0, 1).toUpperCase()}
            </span>
            <div className="grow">
              <strong>{user?.username}</strong>
              <small>{isAdmin ? 'Admin' : 'Pengguna'}</small>
            </div>
            <ThemeToggle />
          </div>
          <nav>
            <NavLink to="/akun">Akun Saya</NavLink>
            {isAdmin && <NavLink to="/admin/pengguna">Pengguna</NavLink>}
            <button type="button" className="nav-button" onClick={() => void logout()}>
              Keluar
            </button>
          </nav>
        </div>
      </aside>
      <main className="content">
        <Routes>
          {PAGES.map((p) => (
            <Route key={p.path} path={p.path} element={p.element} />
          ))}
          <Route path="/akun" element={<Account />} />
          {isAdmin && <Route path="/admin/pengguna" element={<AdminUsers />} />}
          <Route path="*" element={<Navigate to="/keluarga" replace />} />
        </Routes>
      </main>
    </div>
  )
}
