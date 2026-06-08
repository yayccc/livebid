export function formatCentAmount(value?: number | null) {
  if (value === undefined || value === null) {
    return '-'
  }

  return new Intl.NumberFormat('zh-CN', {
    style: 'currency',
    currency: 'CNY',
    minimumFractionDigits: 0,
    maximumFractionDigits: 2,
  }).format(value / 100)
}

export function formatCount(value?: number | null) {
  if (value === undefined || value === null) {
    return '-'
  }

  return new Intl.NumberFormat('zh-CN').format(value)
}

export function formatCompactCount(value?: number | null) {
  if (value === undefined || value === null) {
    return '-'
  }

  if (value >= 10000) {
    return `${(value / 10000).toFixed(value >= 100000 ? 0 : 1)}万`
  }

  return String(value)
}

export function formatTimeText(value?: string) {
  if (!value) {
    return '-'
  }

  return value.replace(/^20\d{2}-/, '').replace('T', ' ').replace(/\.\d+Z$/, '')
}

export function formatRemaining(seconds?: number | null) {
  if (seconds === undefined || seconds === null || seconds <= 0) {
    return '00:00'
  }

  const minutes = Math.floor(seconds / 60)
  const restSeconds = seconds % 60
  return `${String(minutes).padStart(2, '0')}:${String(restSeconds).padStart(2, '0')}`
}

export function getErrorText(error: unknown, fallback: string) {
  if (error instanceof Error && error.message) {
    return error.message
  }

  return fallback
}

export function cx(...values: Array<string | false | null | undefined>) {
  return values.filter(Boolean).join(' ')
}
