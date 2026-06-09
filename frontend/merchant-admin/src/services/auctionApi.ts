import { rawJson, requestJson } from './request'
import type { Auction, AuctionDraft, BidRecord, Goods, LiveRoom, PageResult } from '../types/domain'
import { parseCentInput } from '../lib/format'

export type AuctionListParams = {
  page: number
  pageSize: number
  status?: string
}

export type AuctionCreateOptions = {
  goods: Goods[]
  live_rooms: LiveRoom[]
}

export async function listShopAuctions(
  token: string,
  params: AuctionListParams,
): Promise<PageResult<Auction>> {
  const query = new URLSearchParams({
    page: String(params.page),
    page_size: String(params.pageSize),
  })

  if (params.status !== undefined && params.status !== '') {
    query.set('status', params.status)
  }

  return requestJson<PageResult<Auction>>(`/api/auctions/shop?${query.toString()}`, {
    token,
  })
}

export async function getAuction(auctionID: string): Promise<Auction> {
  return requestJson<Auction>(`/api/auctions/${auctionID}`)
}

export async function createAuction(token: string, input: AuctionDraft): Promise<Auction> {
  return requestJson<Auction>('/api/auctions', {
    method: 'POST',
    token,
    body: rawJson(toAuctionPayloadJSON(input)),
  })
}

export async function getAuctionCreateOptions(token: string): Promise<AuctionCreateOptions> {
  const data = await requestJson<
    Partial<AuctionCreateOptions> & {
      rooms?: LiveRoom[]
      liveRooms?: LiveRoom[]
      available_goods?: Goods[]
      available_live_rooms?: LiveRoom[]
    }
  >('/api/merchant/auctions/create-options', { token })

  return {
    goods: data.goods || data.available_goods || [],
    live_rooms: data.live_rooms || data.rooms || data.available_live_rooms || data.liveRooms || [],
  }
}

export async function updateAuction(
  token: string,
  auctionID: string,
  input: AuctionDraft,
): Promise<Record<string, never>> {
  return requestJson<Record<string, never>>(`/api/auctions/${auctionID}`, {
    method: 'PUT',
    token,
    body: rawJson(toAuctionPayloadJSON(input)),
  })
}

export async function startAuction(token: string, auctionID: string): Promise<Auction> {
  return requestJson<Auction>(`/api/auctions/${auctionID}/start`, {
    method: 'POST',
    token,
  })
}

export async function finishAuction(token: string, auctionID: string): Promise<Auction> {
  return requestJson<Auction>(`/api/auctions/${auctionID}/finish`, {
    method: 'POST',
    token,
  })
}

export async function cancelAuction(token: string, auctionID: string): Promise<Auction> {
  return requestJson<Auction>(`/api/auctions/${auctionID}/cancel`, {
    method: 'POST',
    token,
  })
}

export async function deleteAuction(token: string, auctionID: string): Promise<boolean> {
  return requestJson<boolean>(`/api/auctions/${auctionID}`, {
    method: 'DELETE',
    token,
  })
}

export async function listBidRecords(
  auctionID: string,
  page = 1,
  pageSize = 20,
): Promise<PageResult<BidRecord>> {
  const query = new URLSearchParams({
    page: String(page),
    page_size: String(pageSize),
  })

  return requestJson<PageResult<BidRecord>>(
    `/api/auctions/${auctionID}/bids?${query.toString()}`,
  )
}

function toAuctionPayloadJSON(input: AuctionDraft) {
  const fields = [
    `"goods_id":${assertIntegerID(input.goods_id)}`,
    `"room_id":${assertIntegerID(input.room_id)}`,
    `"start_price":${parseCentInput(input.start_price) || 0}`,
    `"bid_increment":${parseCentInput(input.bid_increment) || 0}`,
    `"start_time":${JSON.stringify(toServerDateTime(input.start_time))}`,
    `"end_time":${JSON.stringify(toServerDateTime(input.end_time))}`,
  ]
  const sealPrice = parseCentInput(input.seal_price)
  if (sealPrice !== undefined) {
    fields.push(`"seal_price":${sealPrice}`)
  }
  return `{${fields.join(',')}}`
}

function assertIntegerID(id: string) {
  if (!/^\d+$/.test(id)) {
    throw new Error('ID 格式无效')
  }
  return id
}

function toServerDateTime(value: string) {
  const normalized = value.trim()
  if (!normalized) {
    return ''
  }

  const parsed = new Date(normalized)
  if (Number.isFinite(parsed.getTime())) {
    return parsed.toISOString()
  }

  return normalized
}
