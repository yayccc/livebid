import type { BidEventMessage } from '../../types/domain'

type LiveMessageListProps = {
  messages: BidEventMessage[]
}

export function LiveMessageList({ messages }: LiveMessageListProps) {
  if (messages.length === 0) {
    return null
  }

  return (
    <div className="live-messages" aria-live="polite" aria-label="直播间消息">
      {messages.slice(-5).map((message) => (
        <p key={message.id} className={`live-messages__item live-messages__item--${message.type}`}>
          {message.text}
        </p>
      ))}
    </div>
  )
}
