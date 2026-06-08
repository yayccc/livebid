import { Outlet, useLocation } from 'react-router-dom'
import { BottomNav } from './BottomNav'

export function AppShell() {
  const location = useLocation()
  const isHome = location.pathname === '/'

  return (
    <main className={isHome ? 'app-shell app-shell--immersive' : 'app-shell'}>
      <Outlet />
      <BottomNav />
    </main>
  )
}
