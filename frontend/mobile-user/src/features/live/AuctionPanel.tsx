import { Radio, ShoppingBag, X } from 'lucide-react'
import { formatCentAmount } from '../../lib/format'
import type { UserLiveAuction, UserLiveGoods } from '../../types/domain'

type AuctionPanelProps = {
  auction: UserLiveAuction | null
  goods?: UserLiveGoods | null
  onOpenGoods?: () => void
  onOpenRecords?: () => void
  onClose?: () => void
  variant?: 'overlay' | 'dock'
}

export function AuctionPanel({
  auction,
  goods,
  onOpenGoods,
  onOpenRecords,
  onClose,
  variant = 'overlay',
}: AuctionPanelProps) {
  if (!auction) {
    return (
      <aside className="auction-panel auction-panel--empty" aria-label="当前竞拍">
        <Radio size={18} aria-hidden="true" />
        <span>等待主播发起竞拍</span>
      </aside>
    )
  }

  const statusMeta = getAuctionPanelStatus(auction)

  return (
    <aside className={`auction-panel${variant === 'dock' ? ' auction-panel--dock' : ''}`} aria-label="当前竞拍">
      <div className={`auction-panel__card auction-panel__card--${statusMeta.key}`}>
        <span className={`auction-panel__badge auction-panel__badge--${statusMeta.key}`}>{statusMeta.badge}</span>
        {onClose ? (
          <button type="button" className="auction-panel__dismiss" aria-label="关闭竞拍卡片" onClick={onClose}>
            <X size={12} aria-hidden="true" />
          </button>
        ) : null}
        <button type="button" className="auction-panel__goods" onClick={onOpenGoods}>
          <span className="auction-panel__cover">
            {goods?.cover_url ? (
              <img src={goods.cover_url} alt="" />
            ) : (
              <ShoppingBag size={24} aria-hidden="true" />
            )}
          </span>
          <span className="auction-panel__info">
            <span className="auction-panel__price">
              <strong>{formatCentAmount(statusMeta.price)}</strong>
              <small>{statusMeta.priceLabel}</small>
            </span>
          </span>
        </button>

        <button
          type="button"
          className="auction-panel__open"
          onClick={onOpenRecords}
        >
          <span>去看看</span>
        </button>
      </div>
    </aside>
  )
}

function getAuctionPanelStatus(auction: UserLiveAuction) {
  switch (auction.status) {
    case 0:
      return {
        key: 'upcoming',
        badge: auction.status_text || '即将开拍',
        priceLabel: '起拍价',
        price: auction.start_price,
      }
    case 1:
      return {
        key: 'running',
        badge: auction.status_text || '正在竞拍',
        priceLabel: '当前价',
        price: auction.current_price,
      }
    case 2:
      return {
        key: 'deal',
        badge: auction.status_text || '已成交',
        priceLabel: '成交价',
        price: auction.deal_price || auction.current_price,
      }
    case 3:
      return {
        key: 'failed',
        badge: auction.status_text || '已流拍',
        priceLabel: '最高价',
        price: auction.current_price || auction.start_price,
      }
    case 4:
      return {
        key: 'cancelled',
        badge: auction.status_text || '已取消',
        priceLabel: '起拍价',
        price: auction.start_price,
      }
    default:
      return {
        key: 'unknown',
        badge: auction.status_text || '竞拍商品',
        priceLabel: '当前价',
        price: auction.current_price || auction.start_price,
      }
  }
}
