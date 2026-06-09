const apiBaseUrl = (import.meta.env.VITE_API_BASE_URL || '').replace(/\/$/, '')

export type ApiResponse<T> = {
  code: number
  message: string
  data?: T
}

export type RequestOptions = {
  method?: string
  body?: unknown
  token?: string
  headers?: HeadersInit
}

class RawJsonBody {
  value: string

  constructor(value: string) {
    this.value = value
  }
}

export function rawJson(value: string) {
  return new RawJsonBody(value)
}

export class HttpError extends Error {
  status: number

  code: number

  constructor(message: string, status: number, code: number) {
    super(message)
    this.name = 'HttpError'
    this.status = status
    this.code = code
  }
}

export async function requestJson<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const headers = new Headers(options.headers)
  const body = buildRequestBody(options.body, headers)

  if (options.token) {
    headers.set('Authorization', `Bearer ${options.token}`)
  }

  const response = await fetch(`${apiBaseUrl}${path}`, {
    method: options.method || 'GET',
    headers,
    body,
  })
  const payload = await parseApiResponse<T>(response)

  if (!response.ok || payload.code !== 0) {
    throw new HttpError(payload.message || '请求失败，请稍后重试', response.status, payload.code)
  }

  return payload.data as T
}

function buildRequestBody(body: unknown, headers: Headers): BodyInit | undefined {
  if (body === undefined) {
    return undefined
  }

  if (body instanceof FormData) {
    return body
  }

  if (body instanceof RawJsonBody) {
    headers.set('Content-Type', 'application/json')
    return body.value
  }

  headers.set('Content-Type', 'application/json')
  return JSON.stringify(body)
}

async function parseApiResponse<T>(response: Response): Promise<ApiResponse<T>> {
  try {
    const text = await response.text()
    const payload = JSON.parse(preserveIDIntegers(text)) as Partial<ApiResponse<T>>
    return {
      code: typeof payload.code === 'number' ? payload.code : response.status,
      message: typeof payload.message === 'string' ? payload.message : response.statusText,
      data: payload.data,
    }
  } catch {
    return {
      code: response.status,
      message: response.statusText || '请求失败，请稍后重试',
    }
  }
}

function preserveIDIntegers(json: string) {
  return json.replace(
    /"(id|shop_id|goods_id|room_id|auction_id|user_id|winner_user_id)"\s*:\s*(-?\d+)/g,
    (_match, key: string, value: string) => `"${key}":"${value}"`,
  )
}
