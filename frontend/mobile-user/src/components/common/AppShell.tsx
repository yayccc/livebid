import { Outlet, useLocation } from 'react-router-dom'
import { BottomNav } from './BottomNav'

export function AppShell() {
  const location = useLocation()
  const isHome = location.pathname === '/'
  const isLightPage = location.pathname === '/messages' || location.pathname === '/profile'
  const shellClassName = [
    'app-shell',
    isHome ? 'app-shell--immersive' : '',
    isLightPage ? 'app-shell--light' : '',
  ]
    .filter(Boolean)
    .join(' ')

  return (
    <main className={shellClassName}>
      <Outlet />
      <BottomNav />
    </main>
  )
}
