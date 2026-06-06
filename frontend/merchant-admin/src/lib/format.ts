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

export function compactTime(value?: string) {
  if (!value) {
    return '-'
  }

  return value.replace(/^20\d{2}-/, '')
}

export function getErrorText(error: unknown, fallback: string) {
  if (error instanceof Error && error.message) {
    return error.message
  }

  return fallback
}

export function parseCentInput(value: string) {
  const normalized = value.trim()
  if (!normalized) {
    return undefined
  }

  const amount = Number(normalized)
  if (!Number.isFinite(amount)) {
    return undefined
  }

  return Math.round(amount * 100)
}
