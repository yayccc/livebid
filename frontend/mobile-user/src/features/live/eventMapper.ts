import { formatCentAmount } from '../../lib/format'
import type {
  BidEventMessage,
  IncomingLiveEvent,
  UserLiveAuction,
  UserLiveRuntime,
  UserLiveStats,
} from '../../types/domain'

export type LiveEventPatch = {
  runtime?: UserLiveRuntime
  stats?: UserLiveStats
  message?: BidEventMessage
  bidResolved?: boolean
  bidError?: string
}

export function mapIncomingLiveEvent(
  event: IncomingLiveEvent,
  auction: UserLiveAuction | null,
): LiveEventPatch {
  const type = event.type || ''
  const data = toRecord(event.data)

  if (type === 'room_online_changed') {
    return {
      stats: {
        online_user_count: toNumber(data.online_user_count),
      },
    }
  }

  if (type === 'response') {
    const code = toNumber(data.code ?? toRecord(event).code) ?? 0
    const message = toString(toRecord(event).message) || 'success'
    if (code !== 0) {
      return {
        bidResolved: true,
        bidError: message,
        message: buildMessage('error', message),
      }
    }

    if (data.accepted === true) {
      const bidPrice = toNumber(data.current_price) ?? toNumber(data.bid_price)
      return {
        bidResolved: true,
        runtime: buildRuntimePatch(data, auction, 1),
        message: buildMessage('bid', `出价成功 ${formatCentAmount(bidPrice || auction?.current_price || 0)}`),
      }
    }

    return {
      bidResolved: true,
      runtime: buildRuntimePatch(data, auction),
      message: buildMessage('system', '出价已提交，等待直播间同步'),
    }
  }

  if (type === 'auction_started') {
    return {
      runtime: buildRuntimePatch(data, auction, 1),
      message: buildMessage('system', '新一轮竞拍开始'),
    }
  }

  if (type === 'bid_accepted') {
    const bidPrice = toNumber(data.bid_price) ?? toNumber(data.current_price)
    return {
      runtime: buildRuntimePatch(data, auction, 1),
      message: buildMessage(
        'bid',
        `${winnerText(data)} 出价 ${formatCentAmount(bidPrice || 0)}`,
      ),
    }
  }

  if (type === 'auction_finished' || type === 'auction_deal') {
    return {
      runtime: buildRuntimePatch(data, auction, 2),
      message: buildMessage('deal', `竞拍成交，成交价 ${formatCentAmount(toNumber(data.deal_price) || toNumber(data.current_price) || auction?.current_price)}`),
    }
  }

  if (type === 'auction_failed') {
    return {
      runtime: buildRuntimePatch(data, auction, 3),
      message: buildMessage('system', '本轮竞拍已流拍'),
    }
  }

  if (type === 'auction_cancelled' || type === 'auction_canceled') {
    return {
      runtime: buildRuntimePatch(data, auction, 4),
      message: buildMessage('system', '本轮竞拍已取消'),
    }
  }

  return {}
}

function buildRuntimePatch(
  data: Record<string, unknown>,
  auction: UserLiveAuction | null,
  fallbackStatus?: number,
): UserLiveRuntime {
  const currentPrice = toNumber(data.current_price) ?? toNumber(data.bid_price)
  const bidCount = toNumber(data.bid_count)
  const winnerUserID = toID(data.winner_user_id) ?? toID(data.user_id)
  const expireAt = toNumber(data.expire_at)

  return {
    next_bid_price:
      currentPrice !== undefined && auction?.bid_increment
        ? currentPrice + auction.bid_increment
        : undefined,
    current_price: currentPrice ?? auction?.current_price,
    bid_count: bidCount ?? auction?.bid_count,
    status: toNumber(data.status) ?? fallbackStatus ?? auction?.status,
    winner_user_id: winnerUserID,
    winner_display_name: winnerUserID ? `用户${winnerUserID}` : auction?.winner_display_name,
    remaining_seconds: expireAt ? Math.max(0, Math.ceil((expireAt - Date.now()) / 1000)) : undefined,
  }
}

function winnerText(data: Record<string, unknown>) {
  const userID = toID(data.winner_user_id) ?? toID(data.user_id)
  return userID ? `用户${userID}` : '用户'
}

function buildMessage(type: BidEventMessage['type'], text: string): BidEventMessage {
  const createdAt = Date.now()
  return {
    id: `${createdAt}_${Math.random().toString(16).slice(2)}`,
    type,
    text,
    createdAt,
  }
}

function toRecord(value: unknown): Record<string, unknown> {
  if (value && typeof value === 'object' && !Array.isArray(value)) {
    return value as Record<string, unknown>
  }

  return {}
}

function toNumber(value: unknown) {
  if (typeof value === 'number' && Number.isFinite(value)) {
    return value
  }
  if (typeof value === 'string') {
    const parsed = Number(value)
    return Number.isFinite(parsed) ? parsed : undefined
  }

  return undefined
}

function toID(value: unknown) {
  if (typeof value === 'string') {
    return value.trim() || undefined
  }
  if (typeof value === 'number' && Number.isFinite(value)) {
    return String(value)
  }

  return undefined
}

function toString(value: unknown) {
  return typeof value === 'string' ? value : ''
}
