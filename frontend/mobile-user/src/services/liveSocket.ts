import type { IncomingLiveEvent } from '../types/domain'
import type { EntityID } from '../types/domain'

export type LiveSocketHandlers = {
  onOpen?: () => void
  onEvent?: (event: IncomingLiveEvent) => void
  onError?: () => void
  onClose?: () => void
}

export type LiveSocketClient = {
  sendBid: (auctionID: EntityID, bidPrice: number) => boolean
  close: () => void
}

export function openLiveSocket(
  rawURL: string,
  roomID: EntityID | undefined,
  token: string | null,
  handlers: LiveSocketHandlers,
): LiveSocketClient | null {
  const url = buildSocketURL(rawURL, roomID, token)
  if (!url) {
    return null
  }

  const socket = new WebSocket(url)
  socket.addEventListener('open', () => handlers.onOpen?.())
  socket.addEventListener('error', () => handlers.onError?.())
  socket.addEventListener('close', () => handlers.onClose?.())
  socket.addEventListener('message', (message) => {
    const event = parseSocketEvent(message.data)
    if (event) {
      handlers.onEvent?.(event)
    }
  })

  return {
    sendBid(auctionID: EntityID, bidPrice: number) {
      if (socket.readyState !== WebSocket.OPEN) {
        return false
      }

      socket.send(
        JSON.stringify({
          type: 'place_bid',
          request_id: `bid_${auctionID}_${Date.now()}`,
          timestamp: Date.now(),
          data: {
            auction_id: auctionID,
            bid_price: bidPrice,
          },
        }),
      )
      return true
    },
    close() {
      socket.close()
    },
  }
}

function buildSocketURL(rawURL: string, roomID: EntityID | undefined, token: string | null) {
  if (!rawURL) {
    return null
  }

  const base = window.location.origin
  const url = new URL(rawURL, base)
  if (url.protocol === 'http:') {
    url.protocol = 'ws:'
  }
  if (url.protocol === 'https:') {
    url.protocol = 'wss:'
  }
  if (token && !url.searchParams.has('token')) {
    url.searchParams.set('token', token)
  }
  if (roomID && !url.searchParams.has('room_id')) {
    url.searchParams.set('room_id', String(roomID))
  }

  return url.toString()
}

function parseSocketEvent(data: unknown): IncomingLiveEvent | null {
  if (typeof data !== 'string') {
    return null
  }

  try {
    return JSON.parse(data) as IncomingLiveEvent
  } catch {
    return {
      type: 'message',
      message: data,
    }
  }
}
