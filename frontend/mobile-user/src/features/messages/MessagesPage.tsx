import { Bell, MessageCircle, UserRound } from 'lucide-react'
import { EmptyState } from '../../components/common/EmptyState'

const messageTypes = [
  { label: '系统通知', icon: Bell },
  { label: '主播消息', icon: MessageCircle },
  { label: '好友消息', icon: UserRound },
]

export function MessagesPage() {
  return (
    <div className="plain-page">
      <EmptyState title="消息功能开发中" description="后续会展示主播、好友和系统通知" />
      <div className="message-placeholders" aria-label="预留消息分类">
        {messageTypes.map((item) => {
          const Icon = item.icon
          return (
            <div key={item.label} className="message-placeholders__item">
              <Icon size={18} aria-hidden="true" />
              <span>{item.label}</span>
            </div>
          )
        })}
      </div>
    </div>
  )
}
