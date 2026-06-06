import { requestJson } from './request'
import type { Goods, GoodsDraft, PageResult } from '../types/domain'

export type GoodsListParams = {
  page: number
  pageSize: number
  keyword?: string
}

export async function listShopGoods(
  token: string,
  params: GoodsListParams,
): Promise<PageResult<Goods>> {
  const query = new URLSearchParams({
    page: String(params.page),
    page_size: String(params.pageSize),
  })

  if (params.keyword?.trim()) {
    query.set('keyword', params.keyword.trim())
  }

  return requestJson<PageResult<Goods>>(`/api/goods/shop/list?${query.toString()}`, {
    token,
  })
}

export async function createGoods(token: string, input: GoodsDraft): Promise<Goods> {
  return requestJson<Goods>('/api/goods', {
    method: 'POST',
    token,
    body: input,
  })
}

export async function updateGoods(token: string, goodsID: number, input: GoodsDraft): Promise<Goods> {
  return requestJson<Goods>(`/api/goods/${goodsID}`, {
    method: 'PUT',
    token,
    body: input,
  })
}

export async function deleteGoods(token: string, goodsID: number): Promise<void> {
  await requestJson<string>(`/api/goods/${goodsID}`, {
    method: 'DELETE',
    token,
  })
}

export async function putGoodsOnSale(token: string, goodsID: number): Promise<void> {
  await requestJson<string>(`/api/goods/${goodsID}/on-sale`, {
    method: 'PUT',
    token,
  })
}

export async function putGoodsOffSale(token: string, goodsID: number): Promise<void> {
  await requestJson<string>(`/api/goods/${goodsID}/off-sale`, {
    method: 'PUT',
    token,
  })
}

export async function batchGetGoods(ids: number[]): Promise<Goods[]> {
  if (ids.length === 0) {
    return []
  }

  return requestJson<Goods[]>('/api/goods/batch', {
    method: 'POST',
    body: { ids },
  })
}

export async function uploadGoodsCover(token: string, file: File): Promise<string> {
  const form = new FormData()
  form.append('file', file)

  const data = await requestJson<{ cover_url?: string; url?: string }>('/api/goods/cover/upload', {
    method: 'POST',
    token,
    body: form,
  })

  return data.cover_url || data.url || ''
}
