import { formatCentAmount } from '../../lib/format'
import type {
  BidEventMessage,
  IncomingLiveEvent,
  UserLiveAuctionRecord,
  UserLiveAuction,
  UserLiveRuntime,
  UserLiveStats,
} from '../../types/domain'

export type LiveEventPatch = {
  runtime?: UserLiveRuntime
  stats?: UserLiveStats
  message?: BidEventMessage
  messages?: BidEventMessage[]
  bidResolved?: boolean
  bidError?: string
  danmakuResolved?: boolean
  danmakuError?: string
  auctionRecord?: UserLiveAuctionRecord
}

export function mapIncomingLiveEvent(
  event: IncomingLiveEvent,
  auction: UserLiveAuction | null,
): LiveEventPatch {
  const type = event.type || event.event_type || ''
  const envelope = toRecord(event)
  const eventData = toRecord(event.data)
  const data: Record<string, unknown> = {
    ...eventData,
    auction_id: eventData.auction_id ?? envelope.auction_id,
    room_id: eventData.room_id ?? envelope.room_id,
    server_time: eventData.server_time ?? envelope.server_time,
    version: eventData.version ?? envelope.version,
    code: eventData.code ?? envelope.code,
    message: eventData.message ?? envelope.message,
  }
  const requestType = toString(event.request_type)
  const requestID = toString(envelope.request_id)
  const isBidResponse = requestType === 'place_bid' || requestID.startsWith('bid_')

  if (type === 'room_online_changed') {
    return {
      stats: {
        online_user_count: toNumber(data.online_user_count),
      },
    }
  }

  if (type === 'response') {
    if (requestType === 'connect') {
      const messages = buildRecentDanmakuMessages(data.recent_danmaku)
      return messages.length > 0 ? { messages } : {}
    }

    if (requestType === 'ping') {
      return {}
    }

    if (isBidResponse) {
      const code = toNumber(data.code) ?? 0
      const message = toString(data.message) || 'success'
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

    if (requestType === 'send_danmaku' || requestID.startsWith('dm_')) {
      const code = toNumber(data.code) ?? 0
      if (code !== 0) {
        return {
          danmakuResolved: true,
          danmakuError: danmakuErrorText(code, toString(data.message)),
        }
      }

      return {
        danmakuResolved: true,
      }
    }

    return {}
  }

  if (type === 'danmaku_created') {
    const message = buildDanmakuMessage(data)
    return message ? { message } : {}
  }

  if (type === 'ai_interaction_created') {
    const message = buildAIInteractionMessage(data)
    return message ? { message } : {}
  }

  if (type === 'auction_started') {
    return {
      runtime: buildRuntimePatch(data, auction, 1),
      auctionRecord: buildAuctionRecord(data, auction, 1),
      message: buildMessage('system', '新一轮竞拍开始'),
    }
  }

  if (type === 'bid_accepted') {
    const bidPrice = toNumber(data.bid_price) ?? toNumber(data.current_price)
    const userID = toID(data.winner_user_id) ?? toID(data.user_id)
    const displayName = displayNameFromData(data)
    return {
      runtime: buildRuntimePatch(data, auction, 1),
      auctionRecord: buildAuctionRecord(data, auction, 1),
      message: buildMessage(
        'bid',
        `${displayName || '用户'} 出价 ${formatCentAmount(bidPrice || 0)}`,
        {
          userID,
          bidPrice,
          displayName,
        },
      ),
    }
  }

  if (type === 'auction_finished' || type === 'auction_deal') {
    return {
      runtime: buildRuntimePatch(data, auction, 2),
      auctionRecord: buildAuctionRecord(data, auction, 2),
      message: buildMessage('deal', `竞拍成交，成交价 ${formatCentAmount(toNumber(data.deal_price) || toNumber(data.current_price) || auction?.current_price)}`),
    }
  }

  if (type === 'auction_failed') {
    return {
      runtime: buildRuntimePatch(data, auction, 3),
      auctionRecord: buildAuctionRecord(data, auction, 3),
      message: buildMessage('system', '本轮竞拍已流拍'),
    }
  }

  if (type === 'auction_cancelled' || type === 'auction_canceled') {
    return {
      runtime: buildRuntimePatch(data, auction, 4),
      auctionRecord: buildAuctionRecord(data, auction, 4),
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
  const winnerDisplayName = displayNameFromData(data) ?? auction?.winner_display_name
  const expireAt = toNumber(data.expire_at)
  const serverTime = toNumber(data.server_time)
  const version = toNumber(data.version)

  return {
    next_bid_price:
      currentPrice !== undefined && auction?.bid_increment
        ? currentPrice + auction.bid_increment
        : undefined,
    current_price: currentPrice ?? auction?.current_price,
    bid_count: bidCount ?? auction?.bid_count,
    status: toNumber(data.status) ?? fallbackStatus ?? auction?.status,
    winner_user_id: winnerUserID,
    winner_display_name: winnerDisplayName,
    remaining_seconds: expireAt ? Math.max(0, Math.ceil((expireAt - (serverTime || Date.now())) / 1000)) : undefined,
    server_time: serverTime,
    expire_at: expireAt,
    version,
    countdown_received_at: expireAt ? Date.now() : undefined,
  }
}

function buildAuctionRecord(
  data: Record<string, unknown>,
  auction: UserLiveAuction | null,
  statusOverride?: number,
): UserLiveAuctionRecord | undefined {
  const auctionID = toID(data.auction_id) ?? auction?.id
  if (!auctionID) {
    return undefined
  }

  const currentPrice = toNumber(data.current_price) ?? toNumber(data.bid_price) ?? auction?.current_price ?? 0
  const bidCount = toNumber(data.bid_count) ?? auction?.bid_count ?? 0
  const status = toNumber(data.status) ?? statusOverride ?? auction?.status ?? 0
  const dealPrice = toNumber(data.deal_price)
  const winnerUserID = toID(data.winner_user_id) ?? toID(data.user_id) ?? auction?.winner_user_id
  const winnerDisplayName = displayNameFromData(data) ?? auction?.winner_display_name
  const serverTime = toNumber(data.server_time)
  const expireAt = toNumber(data.expire_at)

  return {
    id: auctionID,
    room_id: toID(data.room_id) ?? auction?.room_id,
    goods_id: toID(data.goods_id) ?? auction?.goods_id ?? '',
    shop_id: toID(data.shop_id) ?? auction?.shop_id,
    status,
    status_text: statusText(status, toString(data.status_text) || auction?.status_text),
    start_price: toNumber(data.start_price) ?? auction?.start_price ?? currentPrice,
    bid_increment: toNumber(data.bid_increment) ?? auction?.bid_increment ?? 0,
    current_price: currentPrice,
    deal_price: dealPrice,
    bid_count: bidCount,
    start_time: toTimeValue(data.start_time) ?? auction?.start_time,
    end_time: toTimeValue(data.end_time) ?? auction?.end_time,
    server_time: serverTime,
    expire_at: expireAt,
    version: toNumber(data.version),
    countdown_received_at: expireAt ? Date.now() : undefined,
    winner_user_id: winnerUserID,
    winner_display_name: winnerDisplayName,
    created_at: toTimeValue(data.created_at),
    updated_at: toTimeValue(data.updated_at),
  }
}

function displayNameFromData(data: Record<string, unknown>) {
  return (
    toString(data.winner_display_name) ||
    toString(data.bidder_display_name) ||
    toString(data.display_name) ||
    toString(data.nickname) ||
    toString(data.username) ||
    toString(data.user_name) ||
    undefined
  )
}

function buildRecentDanmakuMessages(value: unknown) {
  if (!Array.isArray(value)) {
    return []
  }

  return value
    .map((item) => buildDanmakuMessage(toRecord(item)))
    .filter((message): message is BidEventMessage => Boolean(message))
}

function buildDanmakuMessage(data: Record<string, unknown>): BidEventMessage | undefined {
  const status = toString(data.status)
  if (status && status !== 'visible') {
    return undefined
  }

  const content = toString(data.content).trim()
  if (!content) {
    return undefined
  }

  const messageID = toString(data.message_id)
  const displayName = displayNameFromData(data) || '用户'
  const userID = toID(data.user_id)
  const createdAt = toNumber(data.server_time) || Date.now()

  return {
    id: messageID ? `danmaku_${messageID}` : `danmaku_${createdAt}_${Math.random().toString(16).slice(2)}`,
    type: 'chat',
    text: `${displayName}：${content}`,
    createdAt,
    userID,
    displayName,
  }
}

function buildAIInteractionMessage(data: Record<string, unknown>): BidEventMessage | undefined {
  const status = toString(data.status)
  if (status && status !== 'visible') {
    return undefined
  }

  const content = toString(data.content).trim()
  if (!content) {
    return undefined
  }

  const messageID = toString(data.message_id)
  const displayName = toString(data.display_name) || 'AI互动助手'
  const createdAt = toNumber(data.server_time) || Date.now()

  return {
    id: messageID ? `ai_${messageID}` : `ai_${createdAt}_${Math.random().toString(16).slice(2)}`,
    type: 'ai',
    text: `${displayName}：${content}`,
    createdAt,
    displayName,
  }
}

function danmakuErrorText(code: number, fallback: string) {
  switch (code) {
    case 400:
      return '弹幕内容需要在 1-30 个字符内'
    case 401:
      return '请先登录后发送弹幕'
    case 403:
      return '当前直播间不允许发送弹幕'
    case 429:
      return '发送太频繁，请稍后再试'
    default:
      return fallback || '弹幕发送失败，请稍后重试'
  }
}

function statusText(status: number, fallback?: string) {
  if (fallback) {
    return fallback
  }

  switch (status) {
    case 0:
      return '即将开始'
    case 1:
      return '竞拍中'
    case 2:
      return '已成交'
    case 3:
      return '已流拍'
    case 4:
      return '已取消'
    default:
      return ''
  }
}

function buildMessage(
  type: BidEventMessage['type'],
  text: string,
  metadata?: Pick<BidEventMessage, 'userID' | 'bidPrice' | 'displayName'>,
): BidEventMessage {
  const createdAt = Date.now()
  return {
    id: `${createdAt}_${Math.random().toString(16).slice(2)}`,
    type,
    text,
    createdAt,
    ...metadata,
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

function toTimeValue(value: unknown): string | number | undefined {
  if (typeof value === 'number' && Number.isFinite(value)) {
    return value
  }
  if (typeof value === 'string' && value.trim()) {
    return value
  }

  return undefined
}
