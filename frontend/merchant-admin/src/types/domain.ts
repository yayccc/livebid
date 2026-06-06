export type PageResult<T> = {
  total: number
  page: number
  page_size: number
  list: T[]
}

export type Goods = {
  id: number
  shop_id: number
  title: string
  cover_url?: string
  description?: string
  status?: number
  created_at?: string
  updated_at?: string
}

export type Auction = {
  id: number
  goods_id: number
  shop_id: number
  start_price: number
  bid_increment?: number
  seal_price?: number
  current_price: number
  deal_price?: number
  bid_count: number
  status: number
  start_time?: string
  end_time?: string
  winner_user_id?: number
  created_at?: string
  updated_at?: string
}

export type BidRecord = {
  id: number
  auction_id: number
  goods_id: number
  shop_id: number
  user_id: number
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
  start_price: string
  bid_increment: string
  seal_price: string
  start_time: string
  end_time: string
}
