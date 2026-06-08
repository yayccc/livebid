export type PageResult<T> = {
  total: number
  page: number
  page_size: number
  list: T[]
}

export type EntityID = string

export type UserProfile = {
  id: EntityID
  username: string
  nickname?: string
  avatar?: string
  gender?: number
  birthday?: string
  phone?: string
  email?: string
  status?: number
  created_at?: string
  updated_at?: string
}

export type UserLiveShop = {
  id: EntityID
  name: string
  logo?: string
  description?: string
}

export type UserLiveStream = {
  webrtc_play_url?: string
  play_url?: string
  hls_play_url?: string
  media_stream_status?: string
}

export type UserLiveStats = {
  online_user_count?: number
  heat?: number
}

export type UserLiveRoom = {
  id: EntityID
  shop_id?: EntityID
  title: string
  cover?: string
  description?: string
  status?: string
  status_text?: string
  media_stream_status?: string
}

export type UserLiveAuctionHint = {
  auction_id: EntityID
  goods_id?: EntityID
  current_price?: number
  next_bid_price?: number
  bid_count?: number
  status?: number
  status_text?: string
  end_time?: string
  winner_user_id?: EntityID
  winner_display_name?: string
}

export type UserLiveFeedItem = {
  room: UserLiveRoom
  shop?: UserLiveShop | null
  stream?: UserLiveStream | null
  stats?: UserLiveStats | null
  auction_hint?: UserLiveAuctionHint | null
}

export type UserLiveViewer = {
  entered: boolean
  user_id?: EntityID
  nickname?: string
}

export type UserLiveAuction = {
  id: EntityID
  goods_id: EntityID
  shop_id?: EntityID
  room_id?: EntityID
  start_price: number
  bid_increment: number
  seal_price?: number
  current_price: number
  next_bid_price?: number
  deal_price?: number
  bid_count: number
  status: number
  status_text?: string
  start_time?: string
  end_time?: string
  winner_user_id?: EntityID
  winner_display_name?: string
}

export type UserLiveGoods = {
  id: EntityID
  shop_id?: EntityID
  title: string
  cover_url?: string
  description?: string
  status?: number
}

export type UserLiveRuntime = {
  current_price?: number
  bid_count?: number
  status?: number
  status_text?: string
  winner_user_id?: EntityID
  winner_display_name?: string
  remaining_seconds?: number
  expire_at?: string
}

export type UserLiveWSConfig = {
  url: string
  room_id?: EntityID
  heartbeat_interval_seconds?: number
}

export type UserLiveEntry = {
  room: UserLiveRoom
  shop?: UserLiveShop | null
  stream?: UserLiveStream | null
  viewer?: UserLiveViewer | null
  current_auction?: UserLiveAuction | null
  auction?: UserLiveAuction | null
  goods?: UserLiveGoods | null
  runtime?: UserLiveRuntime | null
  ws?: UserLiveWSConfig | null
}

export type UserLiveAuctionSnapshot = {
  viewer?: UserLiveViewer | null
  current_auction?: UserLiveAuction | null
  auction?: UserLiveAuction | null
  goods?: UserLiveGoods | null
  runtime?: UserLiveRuntime | null
}

export type BidEventMessage = {
  id: string
  type: 'system' | 'bid' | 'deal' | 'error'
  text: string
  createdAt: number
}

export type IncomingLiveEvent = {
  type?: string
  request_id?: string
  timestamp?: number
  data?: unknown
  message?: string
}
