import type { BidEventMessage } from '../../types/domain'

type LiveMessageListProps = {
  messages: BidEventMessage[]
  variant?: 'overlay' | 'dock'
}

export function LiveMessageList({ messages, variant = 'overlay' }: LiveMessageListProps) {
  if (messages.length === 0) {
    return null
  }

  return (
    <div className={`live-messages${variant === 'dock' ? ' live-messages--dock' : ''}`} aria-live="polite" aria-label="直播间消息">
      {messages.slice(-7).map((message) => (
        <p key={message.id} className={`live-messages__item live-messages__item--${message.type}`}>
          {message.text}
        </p>
      ))}
    </div>
  )
}
