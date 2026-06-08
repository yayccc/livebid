import { Radio, Store, Users } from 'lucide-react'
import { formatCompactCount } from '../../lib/format'
import type { UserLiveFeedItem, UserLiveEntry } from '../../types/domain'

type LiveRoomMetaProps = {
  item: UserLiveFeedItem
  entry?: UserLiveEntry | null
  variant?: 'preview' | 'entered'
}

export function LiveRoomMeta({ item, entry, variant = 'preview' }: LiveRoomMetaProps) {
  const room = entry?.room || item.room
  const shop = entry?.shop || item.shop
  const stats = item.stats
  const onlineCount = stats?.online_user_count ?? stats?.heat

  return (
    <section className={`live-meta${variant === 'entered' ? ' live-meta--entered' : ''}`} aria-label="直播间信息">
      <div className="live-meta__shop">
        <span className="live-meta__avatar">
          {shop?.logo ? <img src={shop.logo} alt="" /> : <Store size={16} aria-hidden="true" />}
        </span>
        <span className="live-meta__shop-text">{shop?.name || 'LiveBid 主播'}</span>
        {variant === 'entered' ? (
          <span className="live-meta__online">
            <Users size={13} aria-hidden="true" />
            {formatCompactCount(onlineCount)}
          </span>
        ) : (
          <span className="live-meta__live-tag">
            <Radio size={12} aria-hidden="true" />
            直播中
          </span>
        )}
      </div>
      <h1>{room.title || '直播竞拍间'}</h1>
      {room.description && <p>{room.description}</p>}
      {variant !== 'entered' && (
        <div className="live-meta__stats">
          <span>
            <Users size={14} aria-hidden="true" />
            {formatCompactCount(onlineCount)} 热度
          </span>
          {room.media_stream_status && <span>{room.media_stream_status}</span>}
        </div>
      )}
    </section>
  )
}
