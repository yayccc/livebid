import { useEffect, useMemo, useRef, useState, type RefObject } from 'react'

type SRSPlayResponse = {
  code?: number
  sdp?: string
  sessionid?: string
  message?: string
}

export type WebRTCPlayState = 'idle' | 'connecting' | 'playing' | 'failed'

export function useSRSWebRTCPlayer(videoRef: RefObject<HTMLVideoElement | null>, streamURL?: string) {
  const [state, setState] = useState<WebRTCPlayState>('idle')
  const [error, setError] = useState('')
  const normalizedURL = useMemo(() => normalizeWebRTCURL(streamURL), [streamURL])
  const peerRef = useRef<RTCPeerConnection | null>(null)

  useEffect(() => {
    const video = videoRef.current
    if (!video || !normalizedURL) {
      setState('idle')
      setError('')
      return undefined
    }

    let disposed = false
    const peer = new RTCPeerConnection()
    peerRef.current = peer
    setState('connecting')
    setError('')

    peer.addTransceiver('audio', { direction: 'recvonly' })
    peer.addTransceiver('video', { direction: 'recvonly' })

    peer.ontrack = (event) => {
      if (disposed) {
        return
      }
      const [remoteStream] = event.streams
      if (remoteStream && video.srcObject !== remoteStream) {
        video.srcObject = remoteStream
        void video.play().catch(() => undefined)
        setState('playing')
      }
    }

    peer.onconnectionstatechange = () => {
      if (disposed) {
        return
      }
      if (peer.connectionState === 'failed' || peer.connectionState === 'closed') {
        setState('failed')
        setError('直播画面连接失败')
      }
    }

    playSRS(peer, normalizedURL)
      .then(() => {
        if (!disposed && peer.connectionState === 'connected') {
          setState('playing')
        }
      })
      .catch((err: unknown) => {
        if (disposed) {
          return
        }
        setState('failed')
        setError(err instanceof Error ? err.message : '直播画面加载失败')
      })

    return () => {
      disposed = true
      peerRef.current = null
      if (video.srcObject) {
        const mediaStream = video.srcObject as MediaStream
        mediaStream.getTracks().forEach((track) => track.stop())
        video.srcObject = null
      }
      peer.close()
    }
  }, [normalizedURL, videoRef])

  return {
    state,
    error,
    isWebRTC: Boolean(normalizedURL),
  }
}

async function playSRS(peer: RTCPeerConnection, streamurl: string) {
  const { endpoint, api } = srsAPIForStream(streamurl)
  const offer = await peer.createOffer()
  await peer.setLocalDescription(offer)

  const response = await fetch(endpoint, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      api,
      streamurl,
      clientip: null,
      sdp: offer.sdp,
    }),
  })
  const text = await response.text()
  let data: SRSPlayResponse
  try {
    data = JSON.parse(text) as SRSPlayResponse
  } catch {
    throw new Error(`SRS 播放信令失败: ${response.status}`)
  }

  if (!response.ok || data.code !== 0 || !data.sdp) {
    throw new Error(data.message || `SRS 播放信令失败: ${response.status}`)
  }

  await peer.setRemoteDescription({ type: 'answer', sdp: data.sdp })
}

function normalizeWebRTCURL(value?: string) {
  if (!value) {
    return ''
  }

  try {
    const url = new URL(value)
    if (url.protocol !== 'webrtc:') {
      return ''
    }
    if (url.port === '1935') {
      url.port = ''
    }
    return url.toString()
  } catch {
    return ''
  }
}

function srsAPIForStream(streamurl: string) {
  const configured = import.meta.env.VITE_SRS_WEBRTC_API_URL
  const stream = new URL(streamurl)
  const api = `http://${stream.hostname}:1985/rtc/v1/play/`
  if (configured) {
    return {
      endpoint: configured,
      api: configured,
    }
  }

  return {
    endpoint: '/rtc/v1/play/',
    api,
  }
}
