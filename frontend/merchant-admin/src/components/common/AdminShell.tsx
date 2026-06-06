import type React from 'react'
import type { Shop } from '../../services/shopApi'

export type AdminPageKey = 'dashboard' | 'goods' | 'auctions' | 'live' | 'shop'

type AdminShellProps = {
  shop: Shop | null
  activePage: AdminPageKey
  onNavigate: (page: AdminPageKey) => void
  onRefreshShop: () => void
  onLogout: () => void
  children: React.ReactNode
}

const menuItems: Array<{ key: AdminPageKey; label: string; icon: string }> = [
  { key: 'dashboard', label: '工作台', icon: '▦' },
  { key: 'goods', label: '商品管理', icon: '□' },
  { key: 'auctions', label: '竞拍管理', icon: '◎' },
  { key: 'live', label: '直播管理', icon: '◉' },
  { key: 'shop', label: '商铺设置', icon: '◇' },
]

export function AdminShell({
  shop,
  activePage,
  onNavigate,
  onRefreshShop,
  onLogout,
  children,
}: AdminShellProps) {
  return (
    <main className="admin-layout">
      <header className="admin-header">
        <p className="brand">
          <span>LiveBid</span>
          商家后台
        </p>
        <div className="header-meta">
          <span className="shop-chip">{shop?.shopName || '商铺管理台'}</span>
          <span className="status-tag success">已登录</span>
          <button type="button" className="icon-button" title="刷新商铺信息" onClick={onRefreshShop}>
            ↻
          </button>
          <button type="button" className="ghost-button danger" onClick={onLogout}>
            退出
          </button>
        </div>
      </header>

      <div className="admin-body">
        <aside className="sidebar" aria-label="后台导航">
          {menuItems.map((item) => (
            <button
              key={item.key}
              type="button"
              className={`menu-item ${activePage === item.key ? 'active' : ''}`}
              onClick={() => onNavigate(item.key)}
            >
              <span aria-hidden="true">{item.icon}</span>
              {item.label}
            </button>
          ))}
        </aside>
        <section className="admin-content">{children}</section>
      </div>
    </main>
  )
}
