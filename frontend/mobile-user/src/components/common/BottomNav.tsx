import { Heart, Home, MessageCircle, UserRound } from 'lucide-react'
import { NavLink } from 'react-router-dom'

const navItems = [
  { to: '/', label: '首页', icon: Home },
  { to: '/follow', label: '关注', icon: Heart },
  { to: '/messages', label: '消息', icon: MessageCircle },
  { to: '/profile', label: '我的', icon: UserRound },
]

export function BottomNav() {
  return (
    <nav className="bottom-nav" aria-label="底部导航">
      {navItems.map((item) => {
        const Icon = item.icon
        return (
          <NavLink
            key={item.to}
            to={item.to}
            className={({ isActive }) => `bottom-nav__item${isActive ? ' is-active' : ''}`}
            end={item.to === '/'}
          >
            <Icon size={22} aria-hidden="true" />
            <span>{item.label}</span>
          </NavLink>
        )
      })}
    </nav>
  )
}
