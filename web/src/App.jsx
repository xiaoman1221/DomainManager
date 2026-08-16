import { Navigate, Route, Routes } from 'react-router-dom'
import { lazy, Suspense } from 'react'
import MainLayout from './layouts/MainLayout'
import { useAuth } from './context/AuthContext'

const Login = lazy(() => import('./pages/Login'))

const Dashboard = lazy(() => import('./pages/Dashboard'))
const Domains = lazy(() => import('./pages/Domains'))
const Price = lazy(() => import('./pages/Price'))
const Registrars = lazy(() => import('./pages/Registrars'))
const Certificates = lazy(() => import('./pages/Certificates'))
const Notifications = lazy(() => import('./pages/Notifications'))
const Profile = lazy(() => import('./pages/Profile'))

function Page({ children }) {
  return <Suspense fallback={null}>{children}</Suspense>
}

function RequireAuth({ children }) {
  const { token } = useAuth()
  if (!token) return <Navigate to="/login" replace />
  return children
}

function GuestOnly({ children }) {
  const { token } = useAuth()
  if (token) return <Navigate to="/" replace />
  return children
}

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<GuestOnly><Page><Login /></Page></GuestOnly>} />
      <Route path="/register" element={<GuestOnly><Page><Login /></Page></GuestOnly>} />
      <Route path="/" element={<RequireAuth><MainLayout /></RequireAuth>}>
        <Route index element={<Page><Dashboard /></Page>} />
        <Route path="domains" element={<Page><Domains /></Page>} />
        <Route path="price" element={<Page><Price /></Page>} />
        <Route path="registrars" element={<Page><Registrars /></Page>} />
        <Route path="certificates" element={<Page><Certificates /></Page>} />
        <Route path="notifications" element={<Page><Notifications /></Page>} />
        <Route path="profile" element={<Page><Profile /></Page>} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}
