export type PageResult<T> = {
  total: number
  page: number
  page_size: number
  list: T[]
}

export type Goods = {
  id: string
  shop_id: string
  title: string
  cover_url?: string
  description?: string
  status?: number
  created_at?: string
  updated_at?: string
}

export type Auction = {
  id: string
  goods_id: string
  shop_id: string
  room_id: string
  start_price: number
  bid_increment?: number
  seal_price?: number
  current_price: number
  deal_price?: number
  bid_count: number
  status: number
  start_time?: string
  end_time?: string
  winner_user_id?: string
  created_at?: string
  updated_at?: string
}

export type BidRecord = {
  id: string
  auction_id: string
  goods_id: string
  shop_id: string
  room_id: string
  user_id: string
  bid_price: number
  bid_time?: string
  created_at?: string
  updated_at?: string
}

export type GoodsDraft = {
  title: string
  cover_url: string
  description: string
}

export type AuctionDraft = {
  goods_id: string
  room_id: string
  start_price: string
  bid_increment: string
  seal_price: string
  start_time: string
  end_time: string
}

export type LiveRoomStatus = 'not_live' | 'living' | 'unspecified'

export type MediaStreamStatus = 'offline' | 'online' | 'unspecified'

export type LiveRoom = {
  id: string
  shop_id: string
  title: string
  cover?: string
  description?: string
  status: LiveRoomStatus
  media_stream_status: MediaStreamStatus
  actual_start_time?: string
  actual_end_time?: string
  created_at?: string
  updated_at?: string
}

export type LiveRoomDraft = {
  title: string
  cover: string
  description: string
}

export type LiveStreamInfo = {
  stream_name: string
  rtmp_push_url: string
  webrtc_play_url: string
  media_stream_status: MediaStreamStatus
}
