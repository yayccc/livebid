export type StatusTone = 'primary' | 'success' | 'accent' | 'danger' | 'muted'

export function goodsStatusLabel(status?: number) {
  if (status === 1) {
    return '上架'
  }
  if (status === 0) {
    return '下架'
  }
  return '未返回'
}

export function goodsStatusTone(status?: number): StatusTone {
  if (status === 1) {
    return 'success'
  }
  if (status === 0) {
    return 'muted'
  }
  return 'muted'
}

export function auctionStatusLabel(status: number) {
  const labels: Record<number, string> = {
    0: '待开始',
    1: '竞拍中',
    2: '已成交',
    3: '已流拍',
    4: '已取消',
  }

  return labels[status] || '未知'
}

export function auctionStatusTone(status: number): StatusTone {
  const tones: Record<number, StatusTone> = {
    0: 'muted',
    1: 'primary',
    2: 'accent',
    3: 'muted',
    4: 'danger',
  }

  return tones[status] || 'muted'
}

export function liveRoomStatusLabel(status?: string) {
  if (status === 'living') {
    return '直播中'
  }
  if (status === 'not_live') {
    return '未开播'
  }
  return '未知'
}

export function liveRoomStatusTone(status?: string): StatusTone {
  if (status === 'living') {
    return 'primary'
  }
  return 'muted'
}

export function mediaStreamStatusLabel(status?: string) {
  if (status === 'online') {
    return '推流中'
  }
  if (status === 'offline') {
    return '未推流'
  }
  return '未知'
}

export function mediaStreamStatusTone(status?: string): StatusTone {
  if (status === 'online') {
    return 'success'
  }
  return 'muted'
}
