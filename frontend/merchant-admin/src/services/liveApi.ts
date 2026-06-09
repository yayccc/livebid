import { requestJson } from './request'
import type { LiveRoom, LiveRoomDraft, LiveStreamInfo, PageResult } from '../types/domain'

export type LiveRoomListParams = {
  page: number
  pageSize: number
  status?: string
}

type CreateLiveRoomResponse = {
  live_room: LiveRoom
  initial_stream_code: string
  rtmp_push_url: string
  webrtc_play_url: string
}

export async function listMerchantLiveRooms(
  token: string,
  params: LiveRoomListParams,
): Promise<PageResult<LiveRoom>> {
  const query = new URLSearchParams({
    page: String(params.page),
    page_size: String(params.pageSize),
  })

  if (params.status) {
    query.set('status', params.status)
  }

  return requestJson<PageResult<LiveRoom>>(`/api/merchant/live/rooms?${query.toString()}`, {
    token,
  })
}

export async function createLiveRoom(
  token: string,
  input: LiveRoomDraft,
): Promise<CreateLiveRoomResponse> {
  return requestJson<CreateLiveRoomResponse>('/api/live/rooms', {
    method: 'POST',
    token,
    body: {
      title: input.title.trim(),
      cover: input.cover.trim(),
      description: input.description.trim(),
    },
  })
}

export async function startLive(token: string, roomID: string): Promise<LiveRoom> {
  const data = await requestJson<{ live_room: LiveRoom }>(`/api/live/rooms/${roomID}/start`, {
    method: 'POST',
    token,
  })
  return data.live_room
}

export async function endLive(token: string, roomID: string): Promise<LiveRoom> {
  const data = await requestJson<{ live_room: LiveRoom }>(`/api/live/rooms/${roomID}/end`, {
    method: 'POST',
    token,
  })
  return data.live_room
}

export async function getLiveStreamInfo(
  token: string,
  roomID: string,
): Promise<LiveStreamInfo> {
  const data = await requestJson<{ stream_info: LiveStreamInfo }>(
    `/api/live/rooms/${roomID}/stream`,
    { token },
  )
  return data.stream_info
}

export async function uploadLiveCover(file: File): Promise<string> {
  const form = new FormData()
  form.append('file', file)

  const data = await requestJson<{ url: string }>('/api/files/upload', {
    method: 'POST',
    body: form,
  })

  return data.url
}
