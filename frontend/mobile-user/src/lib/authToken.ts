type JwtPayload = {
  sub?: string
  exp?: number
}

export function getJwtSubject(token: string | null) {
  if (!token) {
    return null
  }

  const payload = decodeJwtPayload(token)
  if (!payload?.sub) {
    return null
  }

  return payload.sub.trim() || null
}

export function isJwtExpired(token: string | null) {
  if (!token) {
    return true
  }

  const payload = decodeJwtPayload(token)
  if (!payload?.exp) {
    return false
  }

  return payload.exp * 1000 <= Date.now()
}

function decodeJwtPayload(token: string): JwtPayload | null {
  const parts = token.split('.')
  if (parts.length < 2) {
    return null
  }

  try {
    const base64 = parts[1].replace(/-/g, '+').replace(/_/g, '/')
    const json = decodeURIComponent(
      atob(base64)
        .split('')
        .map((char) => `%${char.charCodeAt(0).toString(16).padStart(2, '0')}`)
        .join(''),
    )
    return JSON.parse(json) as JwtPayload
  } catch {
    return null
  }
}
