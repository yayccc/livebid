type StateBlockProps = {
  title: string
  description: string
  actionLabel?: string
  onAction?: () => void
}

export function StateBlock({ title, description, actionLabel, onAction }: StateBlockProps) {
  return (
    <div className="state-block">
      <strong>{title}</strong>
      <span>{description}</span>
      {actionLabel && onAction && (
        <button type="button" className="ghost-button" onClick={onAction}>
          {actionLabel}
        </button>
      )}
    </div>
  )
}
