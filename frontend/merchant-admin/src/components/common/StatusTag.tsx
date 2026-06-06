import type { StatusTone } from '../../lib/status'

type StatusTagProps = {
  label: string
  tone?: StatusTone
}

export function StatusTag({ label, tone = 'muted' }: StatusTagProps) {
  return <span className={`status-tag ${tone}`}>{label}</span>
}
