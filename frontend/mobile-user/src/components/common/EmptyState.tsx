import type { ReactNode } from 'react'

type EmptyStateProps = {
  title: string
  description: string
  action?: ReactNode
}

export function EmptyState({ title, description, action }: EmptyStateProps) {
  return (
    <section className="empty-state">
      <div className="empty-state__mark" aria-hidden="true" />
      <h1>{title}</h1>
      <p>{description}</p>
      {action && <div className="empty-state__action">{action}</div>}
    </section>
  )
}
