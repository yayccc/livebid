import { Button } from 'antd-mobile'
import {
  ChevronRight,
  LogOut,
  MapPin,
  Package,
  Settings,
  ShoppingBag,
  UserRound,
} from 'lucide-react'
import { AuthPanel } from './AuthPanel'
import { useAuthProfile } from '../../hooks/useAuthProfile'
import { useAuthStore } from '../../stores/authStore'

const featureItems = [
  { label: '我的订单', description: '订单功能暂未实现', icon: Package },
  { label: '我的竞拍', description: '竞拍记录后续接入', icon: ShoppingBag },
  { label: '收藏竞拍', description: '收藏功能开发中', icon: UserRound },
  { label: '收货地址', description: '地址能力后续开放', icon: MapPin },
  { label: '设置', description: '偏好设置开发中', icon: Settings },
]

export function ProfilePage() {
  const token = useAuthStore((state) => state.token)
  const userID = useAuthStore((state) => state.userID)
  const profile = useAuthStore((state) => state.profile)
  const clearSession = useAuthStore((state) => state.clearSession)
  const profileQuery = useAuthProfile()

  if (!token) {
    return (
      <div className="profile-page profile-page--auth">
        <AuthPanel />
      </div>
    )
  }

  const displayName = profile?.nickname || profile?.username || `用户${userID || ''}`

  return (
    <div className="profile-page">
      <section className="profile-header">
        <div className="profile-header__avatar">
          {profile?.avatar ? <img src={profile.avatar} alt="" /> : <UserRound size={34} aria-hidden="true" />}
        </div>
        <div className="profile-header__body">
          <h1>{displayName}</h1>
          <p>{profileQuery.isLoading ? '正在同步用户信息' : profile?.username || '用户资料待完善'}</p>
          <span>{profile?.phone || profile?.email || '登录后可参与直播竞拍'}</span>
        </div>
      </section>

      <section className="profile-actions" aria-label="用户功能入口">
        {featureItems.map((item) => {
          const Icon = item.icon
          return (
            <button key={item.label} type="button" className="profile-action">
              <span className="profile-action__icon">
                <Icon size={19} aria-hidden="true" />
              </span>
              <span className="profile-action__body">
                <span>{item.label}</span>
                <small>{item.description}</small>
              </span>
              <ChevronRight size={18} aria-hidden="true" />
            </button>
          )
        })}
      </section>

      <Button className="logout-button" block fill="outline" onClick={clearSession}>
        <LogOut size={16} aria-hidden="true" />
        退出登录
      </Button>
    </div>
  )
}
