import { requestJson } from './request'

export type Shop = {
  id: number
  username: string
  shopName: string
  logo: string
  description: string
  phone: string
  email: string
}

export type RegisterShopInput = {
  username: string
  password: string
  shopName: string
  logo?: string
  description?: string
  phone?: string
  email?: string
}

export type LoginShopInput = {
  username: string
  password: string
}

type LoginShopResponse = {
  token?: string
  access_token?: string
}

type RegisterShopResponse = {
  shopId: number
}

type UploadFileResponse = {
  url: string
  object_key: string
}

export async function registerShop(input: RegisterShopInput): Promise<RegisterShopResponse> {
  return requestJson<RegisterShopResponse>('/api/shop/register', {
    method: 'POST',
    body: input,
  })
}

export async function loginShop(input: LoginShopInput): Promise<string> {
  const data = await requestJson<LoginShopResponse>('/api/shop/login', {
    method: 'POST',
    body: input,
  })
  const token = data.token || data.access_token
  if (!token) {
    throw new Error('登录成功但未返回访问凭证')
  }
  return token
}

export async function getCurrentShop(token: string): Promise<Shop> {
  return requestJson<Shop>('/api/shop/me', {
    token,
  })
}

export async function uploadShopLogo(file: File): Promise<string> {
  const form = new FormData()
  form.append('file', file)

  const data = await requestJson<UploadFileResponse>('/api/files/upload', {
    method: 'POST',
    body: form,
  })

  return data.url
}
