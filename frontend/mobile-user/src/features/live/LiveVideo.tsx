import { useMemo, useState, useRef } from 'react'
import fallbackCover from '../../assets/hero.png'
import { cx } from '../../lib/format'
import type { UserLiveFeedItem, UserLiveStream } from '../../types/domain'
import { useSRSWebRTCPlayer } from './useSRSWebRTCPlayer'

type LiveVideoProps = {
  item: UserLiveFeedItem
  stream?: UserLiveStream | null
  enabled?: boolean
}

export function LiveVideo({ item, stream, enabled = true }: LiveVideoProps) {
  const videoRef = useRef<HTMLVideoElement | null>(null)
  const [orientation, setOrientation] = useState<'unknown' | 'landscape' | 'portrait'>('unknown')
  const activeStream = stream || item.stream
  const webRTCURL = enabled ? getWebRTCURL(activeStream) : ''
  const videoURL = enabled ? getPlayableVideoURL(activeStream) : ''
  const cover = item.room.cover || fallbackCover
  const player = useSRSWebRTCPlayer(videoRef, webRTCURL)
  const shouldRenderVideo = Boolean(webRTCURL || videoURL)
  const isConnecting = player.isWebRTC && player.state === 'connecting'
  const showCover = !shouldRenderVideo || isConnecting || player.state === 'failed'
  const statusText = useMemo(() => {
    if (isConnecting) {
      return '直播画面连接中'
    }
    if (player.state === 'failed') {
      return player.error || '直播画面加载失败'
    }
    return ''
  }, [isConnecting, player.error, player.state])

  return (
    <div className="live-video" aria-label={`${item.room.title}直播画面`}>
      <img className="live-video__backdrop" src={cover} alt="" />
      {showCover && <img className="live-video__cover" src={cover} alt="" />}
      {shouldRenderVideo && (
        <video
          ref={videoRef}
          className={cx('live-video__media', `live-video__media--${orientation}`)}
          src={videoURL || undefined}
          poster={cover}
          muted
          playsInline
          autoPlay
          loop
          onLoadedMetadata={(event) => {
            const video = event.currentTarget
            setOrientation(video.videoWidth > video.videoHeight ? 'landscape' : 'portrait')
          }}
        />
      )}
      {statusText && <div className="live-video__status">{statusText}</div>}
      <div className="live-video__shade" />
    </div>
  )
}

function getPlayableVideoURL(stream?: UserLiveStream | null) {
  const url = stream?.hls_play_url || stream?.play_url || stream?.webrtc_play_url || ''
  if (!url) {
    return ''
  }

  const lower = url.toLowerCase()
  if (lower.startsWith('webrtc://') || lower.includes('/rtc/v1/play')) {
    return ''
  }

  return url
}

function getWebRTCURL(stream?: UserLiveStream | null) {
  const url = stream?.webrtc_play_url || ''
  if (!url.toLowerCase().startsWith('webrtc://') || !isStreamPlayable(stream)) {
    return ''
  }

  return url
}

function isStreamPlayable(stream?: UserLiveStream | null) {
  const status = (stream?.media_stream_status || '').toLowerCase()
  if (!status) {
    return true
  }

  return status === 'available' || status === 'online' || status === '推流中'
}
