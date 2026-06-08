import { requestJson } from './request'
import type { Auction, AuctionDraft, BidRecord, PageResult } from '../types/domain'
import { parseCentInput } from '../lib/format'

export type AuctionListParams = {
  page: number
  pageSize: number
  status?: string
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

export async function getAuction(auctionID: number): Promise<Auction> {
  return requestJson<Auction>(`/api/auctions/${auctionID}`)
}

export async function createAuction(token: string, input: AuctionDraft): Promise<Auction> {
  return requestJson<Auction>('/api/auctions', {
    method: 'POST',
    token,
    body: toAuctionPayload(input),
  })
}

export async function updateAuction(
  token: string,
  auctionID: number,
  input: AuctionDraft,
): Promise<Record<string, never>> {
  return requestJson<Record<string, never>>(`/api/auctions/${auctionID}`, {
    method: 'PUT',
    token,
    body: toAuctionPayload(input),
  })
}

export async function startAuction(token: string, auctionID: number): Promise<Auction> {
  return requestJson<Auction>(`/api/auctions/${auctionID}/start`, {
    method: 'POST',
    token,
  })
}

export async function finishAuction(token: string, auctionID: number): Promise<Auction> {
  return requestJson<Auction>(`/api/auctions/${auctionID}/finish`, {
    method: 'POST',
    token,
  })
}

export async function cancelAuction(token: string, auctionID: number): Promise<Auction> {
  return requestJson<Auction>(`/api/auctions/${auctionID}/cancel`, {
    method: 'POST',
    token,
  })
}

export async function deleteAuction(token: string, auctionID: number): Promise<boolean> {
  return requestJson<boolean>(`/api/auctions/${auctionID}`, {
    method: 'DELETE',
    token,
  })
}

export async function listBidRecords(
  auctionID: number,
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

function toAuctionPayload(input: AuctionDraft) {
  return {
    goods_id: Number(input.goods_id),
    room_id: Number(input.room_id),
    start_price: parseCentInput(input.start_price) || 0,
    bid_increment: parseCentInput(input.bid_increment) || 0,
    seal_price: parseCentInput(input.seal_price),
    start_time: input.start_time.trim(),
    end_time: input.end_time.trim(),
  }
}
