import { requestJson } from './request'
import type {
  EntityID,
  PageResult,
  UserLiveAuction,
  UserLiveAuctionRecord,
  UserLiveAuctionRecordPage,
  UserLiveAuctionSnapshot,
  UserLiveEntry,
  UserLiveFeedItem,
  UserLiveGoods,
  UserLiveRoom,
  UserLiveRuntime,
  UserLiveShop,
  UserLiveStats,
  UserLiveStream,
  UserLiveViewer,
} from '../types/domain'

type RawFeedResponse =
  | PageResult<RawFeedItem>
  | {
      total?: number
      page?: number
      page_size?: number
      has_more?: boolean
      list?: RawFeedItem[]
      rooms?: RawFeedItem[]
      live_rooms?: RawFeedItem[]
    }

type RawFeedItem = UserLiveFeedItem & {
  room_id?: EntityID
  title?: string
  cover?: string
  description?: string
  status?: string | number
  status_text?: string
  media_stream_status?: string | number
  media_stream_status_text?: string
  shop?: RawShop | null
  preview_stream?: RawStream | null
  current_auction_hint?: RawAuctionHint | null
}

type RawShop = Partial<UserLiveShop> & {
  shop_id?: EntityID
  shop_name?: string
}

type RawStream = UserLiveStream & {
  preview_stream?: UserLiveStream
  play_status?: string
}

type RawAuctionHint = {
  auction_id?: EntityID
  id?: EntityID
  goods_id?: EntityID
  current_price?: number
  next_bid_price?: number
  bid_count?: number
  status?: number
  status_text?: string
  end_time?: string | number
  winner_user_id?: EntityID
  winner_display_name?: string
}

type RawEntry = Omit<UserLiveEntry, 'room' | 'shop' | 'stream' | 'current_auction' | 'auction' | 'goods' | 'viewer'> & {
  room?: RawRoom
  shop?: RawShop | null
  stream?: RawStream | null
  current_auction?: RawAuction | null
  auction?: RawAuction | null
  goods?: RawGoods | null
  viewer?: RawViewer | null
  runtime?: RawRuntime | null
}

type RawAuctionSnapshot = Omit<UserLiveAuctionSnapshot, 'current_auction' | 'auction' | 'goods' | 'viewer' | 'runtime'> & {
  current_auction?: RawAuction | null
  auction?: RawAuction | null
  goods?: RawGoods | null
  viewer?: RawViewer | null
  runtime?: RawRuntime | null
}

type RawRoom = Omit<Partial<UserLiveRoom>, 'status' | 'media_stream_status'> & {
  room_id?: EntityID
  status?: string | number
  media_stream_status?: string | number
  media_stream_status_text?: string
}

type RawAuction = UserLiveAuction & {
  auction_id?: EntityID
}

type RawAuctionRecord = Omit<UserLiveAuctionRecord, 'goods'> & {
  auction_id?: EntityID
  goods?: RawGoods | null
}

type RawAuctionRecordPage = Omit<UserLiveAuctionRecordPage, 'room_id' | 'list'> & {
  room_id?: EntityID
  list?: RawAuctionRecord[]
}

type RawGoods = Partial<UserLiveGoods> & {
  goods_id?: EntityID
  cover?: string
}

type RawViewer = Partial<UserLiveViewer> & {
  is_logged_in?: boolean
  can_bid?: boolean
  enter_state?: string
}

type RawRuntime = Omit<Partial<UserLiveRuntime>, 'expire_at'> & {
  auction_id?: EntityID
  next_bid_price?: number
  server_time?: number
  expire_at?: number | string
  version?: number
}

export async function getUserLiveFeed(page: number, pageSize: number, signal?: AbortSignal) {
  const params = new URLSearchParams({
    page: String(page),
    page_size: String(pageSize),
  })
  const data = await requestJson<RawFeedResponse>(`/api/user/live/feed?${params}`, { signal })
  const list = 'list' in data && data.list ? data.list : 'rooms' in data && data.rooms ? data.rooms : 'live_rooms' in data && data.live_rooms ? data.live_rooms : []

  return {
    total: data.total || list.length,
    page: data.page || page,
    page_size: data.page_size || pageSize,
    list: list.map(normalizeFeedItem),
  }
}

export async function getUserLivePreview(roomID: EntityID, signal?: AbortSignal) {
  const data = await requestJson<RawEntry>(`/api/user/live/rooms/${roomID}/preview`, { signal })
  return normalizeEntry(data)
}

export async function getUserLiveEntry(roomID: EntityID, token?: string | null, signal?: AbortSignal) {
  const data = await requestJson<RawEntry>(`/api/user/live/rooms/${roomID}/entry`, { token, signal })
  return normalizeEntry(data)
}

export async function getUserLiveAuctionSnapshot(
  roomID: EntityID,
  token?: string | null,
  signal?: AbortSignal,
) {
  const data = await requestJson<RawAuctionSnapshot>(`/api/user/live/rooms/${roomID}/auction-snapshot`, {
    token,
    signal,
  })
  return normalizeAuctionSnapshot(data)
}

export async function getUserLiveAuctionRecords(
  roomID: EntityID,
  token?: string | null,
  signal?: AbortSignal,
) {
  const params = new URLSearchParams({
    page: '1',
    page_size: '50',
  })
  const data = await requestJson<RawAuctionRecordPage>(`/api/user/live/rooms/${roomID}/auction-records?${params}`, {
    token,
    signal,
  })
  const list = data.list || []

  return {
    room_id: idValue(data.room_id, roomID),
    total: numberValue(data.total, list.length),
    page: numberValue(data.page, 1),
    page_size: numberValue(data.page_size, 50),
    list: list.map(normalizeAuctionRecord),
  }
}

export async function getGoods(goodsID: EntityID, signal?: AbortSignal) {
  const data = await requestJson<RawGoods>(`/api/goods/${goodsID}`, { signal })
  return normalizeGoods(data)
}

export async function batchGetGoods(goodsIDs: EntityID[], signal?: AbortSignal) {
  const ids = uniqueIntegerIDs(goodsIDs)
  if (ids.length === 0) {
    return []
  }

  const data = await requestJson<RawGoods[]>('/api/goods/batch', {
    method: 'POST',
    body: {
      ids,
    },
    signal,
  })

  return data.map(normalizeGoods).filter((goods): goods is NonNullable<ReturnType<typeof normalizeGoods>> => Boolean(goods))
}

function normalizeFeedItem(item: RawFeedItem): UserLiveFeedItem {
  if (item.room) {
    return {
    ...item,
      room: normalizeRoom(item.room),
      shop: normalizeShop(item.shop),
      stream: normalizeStream(item.stream || item.preview_stream),
      auction_hint: normalizeAuctionHint(item.auction_hint || item.current_auction_hint),
    }
  }

  return {
    room: normalizeRoom(item),
    shop: normalizeShop(item.shop),
    stream: normalizeStream(item.preview_stream || item.stream),
    stats: item.stats ? normalizeStats(item.stats) : null,
    auction_hint: normalizeAuctionHint(item.current_auction_hint || item.auction_hint),
  }
}

function normalizeEntry(entry: RawEntry): UserLiveEntry {
  return {
    ...entry,
    room: normalizeRoom(entry.room || {}),
    shop: normalizeShop(entry.shop),
    stream: normalizeStream(entry.stream),
    viewer: normalizeViewer(entry.viewer),
    current_auction: normalizeAuction(entry.current_auction),
    auction: normalizeAuction(entry.auction),
    goods: normalizeGoods(entry.goods),
    runtime: normalizeRuntime(entry.runtime),
  }
}

function normalizeAuctionSnapshot(snapshot: RawAuctionSnapshot): UserLiveAuctionSnapshot {
  return {
    ...snapshot,
    viewer: normalizeViewer(snapshot.viewer),
    current_auction: normalizeAuction(snapshot.current_auction),
    auction: normalizeAuction(snapshot.auction),
    goods: normalizeGoods(snapshot.goods),
    runtime: normalizeRuntime(snapshot.runtime),
  }
}

function normalizeRoom(room: RawRoom): UserLiveRoom {
  return {
    id: idValue(room.id ?? room.room_id),
    shop_id: optionalID(room.shop_id),
    title: stringValue(room.title, '直播竞拍间'),
    cover: stringValue(room.cover),
    description: stringValue(room.description),
    status: stringValue(room.status),
    status_text: stringValue(room.status_text),
    media_stream_status: stringValue(room.media_stream_status_text ?? room.media_stream_status),
  }
}

function normalizeShop(shop?: RawShop | null): UserLiveShop | null {
  if (!shop) {
    return null
  }

  return {
    id: idValue(shop.id ?? shop.shop_id),
    name: stringValue(shop.name || shop.shop_name, 'LiveBid 主播'),
    logo: stringValue(shop.logo),
    description: stringValue(shop.description),
  }
}

function normalizeStream(stream?: RawStream | null): UserLiveStream | null {
  if (!stream) {
    return null
  }

  return {
    webrtc_play_url: stream.webrtc_play_url,
    play_url: stream.play_url,
    hls_play_url: stream.hls_play_url,
    media_stream_status: stringValue(stream.play_status ?? stream.media_stream_status),
  }
}

function normalizeStats(stats: UserLiveStats): UserLiveStats {
  return {
    online_user_count: stats.online_user_count,
    heat: stats.heat,
  }
}

function normalizeAuctionHint(hint?: RawAuctionHint | null) {
  if (!hint) {
    return null
  }

  return {
    auction_id: idValue(hint.auction_id ?? hint.id),
    goods_id: optionalID(hint.goods_id),
    current_price: optionalNumber(hint.current_price),
    next_bid_price: optionalNumber(hint.next_bid_price),
    bid_count: optionalNumber(hint.bid_count),
    status: optionalNumber(hint.status),
    status_text: stringValue(hint.status_text),
    end_time: timeValue(hint.end_time),
    winner_user_id: optionalID(hint.winner_user_id),
    winner_display_name: stringValue(hint.winner_display_name),
  }
}

function normalizeAuction(auction?: RawAuction | null): UserLiveAuction | null {
  if (!auction) {
    return null
  }
  const expireAt = optionalNumber(auction.expire_at)

  return {
    ...auction,
    id: idValue(auction.id ?? auction.auction_id),
    goods_id: idValue(auction.goods_id),
    start_price: numberValue(auction.start_price),
    bid_increment: numberValue(auction.bid_increment),
    current_price: numberValue(auction.current_price),
    bid_count: numberValue(auction.bid_count),
    status: numberValue(auction.status),
    start_time: timeValue(auction.start_time),
    end_time: timeValue(auction.end_time),
    server_time: optionalNumber(auction.server_time),
    expire_at: expireAt,
    version: optionalNumber(auction.version),
    countdown_received_at: expireAt ? Date.now() : undefined,
  }
}

function normalizeAuctionRecord(record: RawAuctionRecord): UserLiveAuctionRecord {
  const expireAt = optionalNumber(record.expire_at)
  return {
    ...record,
    id: idValue(record.id ?? record.auction_id),
    room_id: optionalID(record.room_id),
    goods_id: idValue(record.goods_id),
    shop_id: optionalID(record.shop_id),
    goods: record.goods ? normalizeGoods(record.goods) : null,
    status: numberValue(record.status),
    status_text: stringValue(record.status_text),
    start_price: numberValue(record.start_price),
    bid_increment: numberValue(record.bid_increment),
    seal_price: optionalNumber(record.seal_price),
    current_price: numberValue(record.current_price),
    next_bid_price: optionalNumber(record.next_bid_price),
    deal_price: optionalNumber(record.deal_price),
    bid_count: numberValue(record.bid_count),
    start_time: timeValue(record.start_time),
    end_time: timeValue(record.end_time),
    server_time: optionalNumber(record.server_time),
    expire_at: expireAt,
    version: optionalNumber(record.version),
    countdown_received_at: expireAt ? Date.now() : undefined,
    winner_user_id: optionalID(record.winner_user_id),
    winner_display_name: stringValue(record.winner_display_name),
    created_at: timeValue(record.created_at),
    updated_at: timeValue(record.updated_at),
  }
}

function normalizeGoods(goods?: RawGoods | null): UserLiveGoods | null {
  if (!goods) {
    return null
  }

  return {
    id: idValue(goods.id ?? goods.goods_id),
    shop_id: optionalID(goods.shop_id),
    title: stringValue(goods.title, '竞拍商品'),
    cover_url: stringValue(goods.cover_url || goods.cover),
    description: stringValue(goods.description),
    status: optionalNumber(goods.status),
    created_at: timeValue(goods.created_at),
    updated_at: timeValue(goods.updated_at),
  }
}

function normalizeViewer(viewer?: RawViewer | null): UserLiveViewer | null {
  if (!viewer) {
    return null
  }

  return {
    entered: viewer.entered ?? viewer.enter_state === 'entered',
    can_bid: viewer.can_bid,
    user_id: optionalID(viewer.user_id),
    nickname: stringValue(viewer.nickname),
  }
}

function normalizeRuntime(runtime?: RawRuntime | null): UserLiveRuntime | null {
  if (!runtime) {
    return null
  }
  const expireAt = optionalNumber(runtime.expire_at)

  return {
    auction_id: optionalID(runtime.auction_id),
    current_price: optionalNumber(runtime.current_price),
    next_bid_price: optionalNumber(runtime.next_bid_price),
    bid_count: optionalNumber(runtime.bid_count),
    status: optionalNumber(runtime.status),
    status_text: stringValue(runtime.status_text),
    winner_user_id: optionalID(runtime.winner_user_id),
    winner_display_name: stringValue(runtime.winner_display_name),
    remaining_seconds: optionalNumber(runtime.remaining_seconds),
    server_time: optionalNumber(runtime.server_time),
    expire_at: expireAt,
    version: optionalNumber(runtime.version),
    countdown_received_at: expireAt ? Date.now() : undefined,
  }
}

function numberValue(value: unknown, fallback = 0) {
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : fallback
}

function idValue(value: unknown, fallback = '') {
  const text = stringValue(value)
  return text || fallback
}

function optionalID(value: unknown) {
  const text = stringValue(value)
  return text || undefined
}

function optionalNumber(value: unknown) {
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : undefined
}

function timeValue(value: unknown) {
  if (typeof value === 'number' && Number.isFinite(value)) {
    return value
  }

  return stringValue(value)
}

function stringValue(value: unknown, fallback = '') {
  if (value === undefined || value === null) {
    return fallback
  }

  return String(value)
}

function uniqueIntegerIDs(ids: EntityID[]) {
  const seen = new Set<number>()
  for (const id of ids) {
    if (!/^\d+$/.test(id)) {
      continue
    }
    seen.add(Number(id))
  }

  return [...seen]
}
