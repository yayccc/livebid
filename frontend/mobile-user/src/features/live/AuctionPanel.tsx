import { Clock3, Gavel, LoaderCircle, Radio, Timer } from 'lucide-react'
import { formatCentAmount, formatCount, formatRemaining } from '../../lib/format'
import type { UserLiveAuction, UserLiveRuntime } from '../../types/domain'

type AuctionPanelProps = {
  auction: UserLiveAuction | null
  runtime?: UserLiveRuntime | null
  canBid: boolean
  isBidding: boolean
  onBid: () => void
}

export function AuctionPanel({
  auction,
  runtime,
  canBid,
  isBidding,
  onBid,
}: AuctionPanelProps) {
  if (!auction) {
    return (
      <aside className="auction-panel auction-panel--empty" aria-label="当前竞拍">
        <Radio size={18} aria-hidden="true" />
        <span>等待主播发起竞拍</span>
      </aside>
    )
  }

  const currentPrice = runtime?.current_price ?? auction.current_price
  const bidCount = runtime?.bid_count ?? auction.bid_count
  const nextBidPrice = runtime?.next_bid_price || auction.next_bid_price || currentPrice + auction.bid_increment
  const status = runtime?.status ?? auction.status
  const isBidEnabled = status === 1 && canBid && !isBidding
  const bidLabel = canBid ? `出价 ${formatCentAmount(nextBidPrice)}` : '登录出价'

  return (
    <aside className="auction-panel" aria-label="当前竞拍">
      <div className="auction-panel__summary">
        <span className="auction-panel__label">当前价</span>
        <strong>{formatCentAmount(currentPrice)}</strong>
        <span className="auction-panel__leader">
          {auction.winner_display_name || runtime?.winner_display_name || '暂无领先用户'}
        </span>
      </div>

      <div className="auction-panel__facts">
        <span>
          <Timer size={13} aria-hidden="true" />
          {formatRemaining(runtime?.remaining_seconds)}
        </span>
        <span>
          <Gavel size={13} aria-hidden="true" />
          {formatCount(bidCount)} 次
        </span>
        <span>
          <Clock3 size={13} aria-hidden="true" />
          加价 {formatCentAmount(auction.bid_increment)}
        </span>
      </div>

      <button
        type="button"
        className="auction-panel__bid"
        disabled={!isBidEnabled}
        onClick={onBid}
      >
        {isBidding ? <LoaderCircle className="spin" size={18} aria-hidden="true" /> : null}
        {isBidding ? '出价中' : bidLabel}
      </button>
    </aside>
  )
}
