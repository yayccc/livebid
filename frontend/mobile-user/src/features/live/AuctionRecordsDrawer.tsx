import { Popup } from 'antd-mobile'
import { Gavel, LoaderCircle, Minus, Plus, ShoppingBag, X } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { formatCentAmount, formatTimeText } from '../../lib/format'
import type { UserLiveAuction, UserLiveAuctionRecord, UserLiveRuntime } from '../../types/domain'

type AuctionRecordsDrawerProps = {
  visible: boolean
  roomTitle: string
  currentAuction?: UserLiveAuction | null
  runtime?: UserLiveRuntime | null
  records: UserLiveAuctionRecord[]
  isBidding: boolean
  onBid: (bidPrice: number) => void
  onOpenGoods: (goodsID: string) => void
  onClose: () => void
}

type RecordFilter = 'all' | 'upcoming' | 'running' | 'deal' | 'failed' | 'cancelled'

const filters: Array<{
  key: RecordFilter
  label: string
}> = [
  { key: 'all', label: '全部' },
  { key: 'upcoming', label: '即将开始' },
  { key: 'running', label: '竞拍中' },
  { key: 'deal', label: '已成交' },
  { key: 'failed', label: '已流拍' },
  { key: 'cancelled', label: '已取消' },
]

export function AuctionRecordsDrawer({
  visible,
  roomTitle,
  currentAuction,
  runtime,
  records,
  isBidding,
  onBid,
  onOpenGoods,
  onClose,
}: AuctionRecordsDrawerProps) {
  const [filter, setFilter] = useState<RecordFilter>('all')
  const [activeBidRecord, setActiveBidRecord] = useState<UserLiveAuctionRecord | null>(null)
  const [bidPrice, setBidPrice] = useState(0)
  const [hasAdjustedBidPrice, setHasAdjustedBidPrice] = useState(false)
  const [countdownNow, setCountdownNow] = useState(Date.now())

  const mergedRecords = useMemo(() => {
    const next = [...records]
    if (currentAuction && !next.some((item) => item.id === currentAuction.id)) {
      next.unshift({
        id: currentAuction.id,
        room_id: currentAuction.room_id,
        goods_id: currentAuction.goods_id,
        shop_id: currentAuction.shop_id,
        status: currentAuction.status,
        status_text: currentAuction.status_text,
        start_price: currentAuction.start_price,
        bid_increment: currentAuction.bid_increment,
        current_price: currentAuction.current_price,
        deal_price: currentAuction.deal_price,
        bid_count: currentAuction.bid_count,
        start_time: currentAuction.start_time,
        end_time: currentAuction.end_time,
        server_time: currentAuction.server_time,
        expire_at: currentAuction.expire_at,
        version: currentAuction.version,
        countdown_received_at: currentAuction.countdown_received_at,
        winner_user_id: currentAuction.winner_user_id,
        winner_display_name: currentAuction.winner_display_name,
        created_at: currentAuction.start_time,
        updated_at: currentAuction.end_time,
        goods: null,
      })
    }

    return next
  }, [currentAuction, records])

  const filteredRecords = useMemo(
    () => mergedRecords.filter((record) => matchesFilter(record.status, filter)),
    [filter, mergedRecords],
  )
  const activeCurrentPrice =
    activeBidRecord && currentAuction?.id === activeBidRecord.id
      ? runtime?.current_price ?? activeBidRecord.current_price
      : activeBidRecord?.current_price ?? 0
  const activeBasePrice =
    activeBidRecord?.status === 1
      ? activeCurrentPrice
      : activeBidRecord?.current_price ?? 0
  const activeBidIncrement =
    activeBidRecord && currentAuction?.id === activeBidRecord.id
      ? currentAuction.bid_increment || activeBidRecord.bid_increment
      : activeBidRecord?.bid_increment || currentAuction?.bid_increment || 0
  const activeMinBidPrice =
    activeBidRecord && currentAuction?.id === activeBidRecord.id
      ? runtime?.next_bid_price || currentAuction.next_bid_price || activeBidRecord.next_bid_price || activeBasePrice + activeBidIncrement
      : activeBidRecord?.next_bid_price || activeBasePrice + activeBidIncrement
  const isCurrentBidRecord = Boolean(activeBidRecord && currentAuction?.id === activeBidRecord.id)

  useEffect(() => {
    if (!activeBidRecord || hasAdjustedBidPrice) {
      return
    }

    setBidPrice(activeMinBidPrice)
  }, [activeBidRecord, activeMinBidPrice, hasAdjustedBidPrice])

  useEffect(() => {
    if (visible) {
      return
    }

    setActiveBidRecord(null)
    setBidPrice(0)
    setHasAdjustedBidPrice(false)
  }, [visible])

  useEffect(() => {
    if (!visible || !mergedRecords.some((record) => record.status === 0 || record.status === 1)) {
      return
    }

    const timer = window.setInterval(() => setCountdownNow(Date.now()), 1000)
    return () => window.clearInterval(timer)
  }, [mergedRecords, visible])

  function openBidSheet(record: UserLiveAuctionRecord) {
    const currentPrice = currentAuction?.id === record.id ? runtime?.current_price ?? record.current_price : record.current_price
    const bidIncrement = currentAuction?.id === record.id ? currentAuction.bid_increment || record.bid_increment : record.bid_increment
    const minBidPrice =
      currentAuction?.id === record.id
        ? runtime?.next_bid_price || currentAuction.next_bid_price || record.next_bid_price || currentPrice + bidIncrement
        : record.next_bid_price || currentPrice + bidIncrement
    setActiveBidRecord(record)
    setBidPrice(minBidPrice)
    setHasAdjustedBidPrice(false)
  }

  return (
    <>
      <Popup visible={visible} onMaskClick={onClose} bodyClassName="auction-records-drawer" position="bottom">
        <header className="auction-records-drawer__header">
          <div>
            <h2>本场直播竞拍</h2>
            <p>{roomTitle}</p>
          </div>
          <button type="button" aria-label="关闭竞拍记录" onClick={onClose}>
            <X size={20} aria-hidden="true" />
          </button>
        </header>

        <div className="auction-records-drawer__filters" role="tablist" aria-label="竞拍状态筛选">
          {filters.map((item) => (
            <button
              key={item.key}
              type="button"
              role="tab"
              aria-selected={filter === item.key}
              className={filter === item.key ? 'is-active' : ''}
              onClick={() => setFilter(item.key)}
            >
              {item.label}
            </button>
          ))}
        </div>

        <div className="auction-records-drawer__list">
          {filteredRecords.length === 0 ? (
            <div className="auction-records-drawer__empty">
              <Gavel size={20} aria-hidden="true" />
              <span>当前没有可显示的竞拍记录</span>
            </div>
          ) : (
            filteredRecords.map((record, index) => {
              const statusMeta = getStatusMeta(record.status, record.status_text)
              const displayPrice = currentAuction?.id === record.id ? runtime?.current_price ?? record.current_price : record.current_price
              const isCurrentRecord = currentAuction?.id === record.id
              const displayRecord = isCurrentRecord ? mergeRecordRuntime(record, runtime, currentAuction) : record
              return (
                <article key={record.id} className={`auction-records-drawer__item auction-records-drawer__item--${statusMeta.key}`}>
                  <button
                    type="button"
                    className="auction-records-drawer__cover"
                    onClick={() => {
                      if (record.goods?.id) {
                        onOpenGoods(record.goods.id)
                      }
                    }}
                  >
                    {record.goods?.cover_url ? (
                      <img src={record.goods.cover_url} alt="" />
                    ) : (
                      <ShoppingBag size={18} aria-hidden="true" />
                    )}
                    <span className="auction-records-drawer__index">{index + 1}</span>
                  </button>
                  <div className="auction-records-drawer__body">
                    <div className="auction-records-drawer__row">
                      <button
                        type="button"
                        className="auction-records-drawer__title"
                        onClick={() => {
                          if (record.goods?.id) {
                            onOpenGoods(record.goods.id)
                          }
                        }}
                      >
                        {record.goods?.title || '竞拍商品'}
                      </button>
                      <span className={`auction-records-drawer__status auction-records-drawer__status--${statusMeta.key}`}>
                        {statusMeta.label}
                      </span>
                    </div>
                    <AuctionRecordSummary record={displayRecord} displayPrice={displayPrice} now={countdownNow} />
                    <AuctionRecordFooter record={displayRecord} />
                  </div>
                  <button
                    type="button"
                    className="auction-records-drawer__bid"
                    disabled={!isCurrentRecord}
                    onClick={() => openBidSheet(record)}
                  >
                    出价
                  </button>
                </article>
              )
            })
          )}
        </div>
      </Popup>

      <Popup
        visible={Boolean(activeBidRecord)}
        onMaskClick={() => {
          setActiveBidRecord(null)
          setHasAdjustedBidPrice(false)
        }}
        bodyClassName="auction-bid-sheet"
        position="bottom"
      >
        <header className="auction-bid-sheet__header">
          <div>
            <h2>参与出价</h2>
            <p>{activeBidRecord?.goods?.title || '竞拍商品'}</p>
          </div>
          <button
            type="button"
            aria-label="关闭出价"
            onClick={() => {
              setActiveBidRecord(null)
              setHasAdjustedBidPrice(false)
            }}
          >
            <X size={20} aria-hidden="true" />
          </button>
        </header>

        <div className="auction-bid-sheet__price">
          <span>当前价格</span>
          <strong>{formatCentAmount(activeCurrentPrice)}</strong>
        </div>

        <div className="auction-bid-sheet__stepper" aria-label="出价金额">
          <button
            type="button"
            onClick={() => {
              setHasAdjustedBidPrice(true)
              setBidPrice((value) => Math.max(activeMinBidPrice, value - activeBidIncrement))
            }}
            disabled={!isCurrentBidRecord || bidPrice <= activeMinBidPrice}
            aria-label="减少一档"
          >
            <Minus size={18} aria-hidden="true" />
          </button>
          <strong>{formatCentAmount(bidPrice || activeMinBidPrice)}</strong>
          <button
            type="button"
            onClick={() => {
              setHasAdjustedBidPrice(true)
              setBidPrice((value) => Math.max(activeMinBidPrice, value || activeMinBidPrice) + activeBidIncrement)
            }}
            disabled={!isCurrentBidRecord}
            aria-label="增加一档"
          >
            <Plus size={18} aria-hidden="true" />
          </button>
        </div>

        <button
          type="button"
          className="auction-bid-sheet__submit"
          disabled={!isCurrentBidRecord || isBidding}
          onClick={() => {
            onBid(Math.max(bidPrice || activeMinBidPrice, activeMinBidPrice))
            setHasAdjustedBidPrice(false)
          }}
        >
          {isBidding ? <LoaderCircle className="spin" size={18} aria-hidden="true" /> : null}
          {isBidding ? '出价中' : '立即出价'}
        </button>
      </Popup>
    </>
  )
}

function mergeRecordRuntime(
  record: UserLiveAuctionRecord,
  runtime?: UserLiveRuntime | null,
  auction?: UserLiveAuction | null,
): UserLiveAuctionRecord {
  if (!runtime && !auction) {
    return record
  }

  return {
    ...record,
    current_price: runtime?.current_price ?? auction?.current_price ?? record.current_price,
    next_bid_price: runtime?.next_bid_price ?? auction?.next_bid_price ?? record.next_bid_price,
    bid_count: runtime?.bid_count ?? auction?.bid_count ?? record.bid_count,
    status: runtime?.status ?? auction?.status ?? record.status,
    status_text: runtime?.status_text || auction?.status_text || record.status_text,
    winner_user_id: runtime?.winner_user_id ?? auction?.winner_user_id ?? record.winner_user_id,
    winner_display_name: runtime?.winner_display_name || auction?.winner_display_name || record.winner_display_name,
    server_time: runtime?.server_time ?? auction?.server_time ?? record.server_time,
    expire_at: runtime?.expire_at ?? auction?.expire_at ?? record.expire_at,
    version: runtime?.version ?? auction?.version ?? record.version,
    countdown_received_at: runtime?.countdown_received_at ?? auction?.countdown_received_at ?? record.countdown_received_at,
  }
}

function AuctionRecordFooter({ record }: { record: UserLiveAuctionRecord }) {
  if (record.status === 2) {
    return (
      <div className="auction-records-drawer__footer">
        <span>{record.winner_display_name ? `成交者 ${record.winner_display_name}` : '成交者待同步'}</span>
        <span>{formatTimeText(record.end_time || record.updated_at)}</span>
      </div>
    )
  }

  return (
    <div className="auction-records-drawer__footer">
      <span>{record.winner_display_name || '暂无出价者'}</span>
      <span>{formatTimeText(record.created_at || record.start_time)}</span>
    </div>
  )
}

function AuctionRecordSummary({
  record,
  displayPrice,
  now,
}: {
  record: UserLiveAuctionRecord
  displayPrice: number
  now: number
}) {
  if (record.status === 2) {
    return (
      <div className="auction-records-drawer__summary auction-records-drawer__summary--deal">
        <span className="auction-records-drawer__summary-main">
          <span>落槌价</span>
          <strong>{formatCentAmount(record.deal_price || displayPrice)}</strong>
        </span>
      </div>
    )
  }

  if (record.status === 1) {
    const hasBid = record.bid_count > 0
    return (
      <div className="auction-records-drawer__summary auction-records-drawer__summary--running">
        <span className="auction-records-drawer__summary-main">
          <span>{hasBid ? '当前最高出价' : '起拍价'}</span>
          <strong>{formatCentAmount(hasBid ? displayPrice : record.start_price)}</strong>
        </span>
        <small
          className={`auction-records-drawer__countdown auction-records-drawer__countdown--running${
            hasBid ? ' auction-records-drawer__countdown--hammer' : ''
          }`}
        >
          {hasBid ? '距落锤' : '距截拍'} {formatRunningCountdown(record, now)}
        </small>
      </div>
    )
  }

  if (record.status === 0) {
    return (
      <div className="auction-records-drawer__summary auction-records-drawer__summary--upcoming">
        <span className="auction-records-drawer__summary-main">
          <span>起拍价</span>
          <strong>{formatCentAmount(record.start_price)}</strong>
        </span>
        <small className="auction-records-drawer__countdown">距开始 {formatAuctionCountdown(record.start_time, now)}</small>
      </div>
    )
  }

  if (record.status === 3) {
    return (
      <div className="auction-records-drawer__meta">
        <span>最高 {formatCentAmount(displayPrice || record.start_price)}</span>
        <span>出价 {record.bid_count}</span>
      </div>
    )
  }

  return (
    <div className="auction-records-drawer__summary auction-records-drawer__summary--cancelled">
      <span className="auction-records-drawer__summary-main">
        <span>起拍价</span>
        <strong>{formatCentAmount(record.start_price)}</strong>
      </span>
      <small>竞拍已取消</small>
    </div>
  )
}

function matchesFilter(status: number, filter: RecordFilter) {
  switch (filter) {
    case 'all':
      return true
    case 'upcoming':
      return status === 0
    case 'running':
      return status === 1
    case 'deal':
      return status === 2
    case 'failed':
      return status === 3
    case 'cancelled':
      return status === 4
    default:
      return true
  }
}

function getStatusMeta(status: number, fallback?: string) {
  switch (status) {
    case 0:
      return { key: 'upcoming', label: fallback || '即将开始' }
    case 1:
      return { key: 'running', label: fallback || '竞拍中' }
    case 2:
      return { key: 'deal', label: fallback || '已成交' }
    case 3:
      return { key: 'failed', label: fallback || '已流拍' }
    case 4:
      return { key: 'cancelled', label: fallback || '已取消' }
    default:
      return { key: 'running', label: fallback || '未知状态' }
  }
}

function formatAuctionCountdown(value?: string | number | null, now = Date.now()) {
  const timestamp = timeToTimestamp(value)
  if (!timestamp) {
    return '-'
  }

  const diffSeconds = Math.max(0, Math.floor((timestamp - now) / 1000))
  if (diffSeconds <= 0) {
    return '即将开始'
  }

  const hours = Math.floor(diffSeconds / 3600)
  const minutes = Math.floor((diffSeconds % 3600) / 60)
  const seconds = diffSeconds % 60

  if (hours > 0) {
    return `${hours}小时${String(minutes).padStart(2, '0')}分`
  }

  return `${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`
}

function formatRunningCountdown(record: UserLiveAuctionRecord, now = Date.now()) {
  const timestamp = runningExpireTimestamp(record, now)
  if (!timestamp) {
    return '--:--'
  }

  const diffSeconds = Math.max(0, Math.ceil((timestamp - now) / 1000))
  if (diffSeconds <= 0) {
    return '即将落槌'
  }

  const minutes = Math.floor(diffSeconds / 60)
  const seconds = diffSeconds % 60
  return `${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`
}

function runningExpireTimestamp(record: UserLiveAuctionRecord, now: number) {
  if (record.expire_at) {
    const expireAt = timeToTimestamp(record.expire_at)
    const serverTime = timeToTimestamp(record.server_time)
    if (expireAt && serverTime) {
      return (record.countdown_received_at || now) + Math.max(0, expireAt - serverTime)
    }
    return expireAt
  }

  return timeToTimestamp(record.end_time)
}

function timeToTimestamp(value?: string | number | null) {
  if (!value) {
    return 0
  }
  if (typeof value === 'number') {
    return value > 10_000_000_000 ? value : value * 1000
  }

  const timestamp = new Date(value).getTime()
  return Number.isFinite(timestamp) ? timestamp : 0
}
