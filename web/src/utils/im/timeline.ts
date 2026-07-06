import type { ChatMessage } from '@/types/im'

const TIME_DIVIDER_INTERVAL = 15 * 60 * 1000

export type MessageTimelineItem =
  | {
      type: 'time'
      id: string
      text: string
    }
  | {
      type: 'message'
      id: string
      message: ChatMessage
    }

const pad2 = (value: number) => value.toString().padStart(2, '0')

const isSameDay = (a: Date, b: Date) =>
  a.getFullYear() === b.getFullYear() &&
  a.getMonth() === b.getMonth() &&
  a.getDate() === b.getDate()

export function formatMessageDividerTime(value: string | undefined, now = new Date()) {
  const date = value ? new Date(value) : new Date()
  if (Number.isNaN(date.getTime())) {
    return `${now.getHours()}:${pad2(now.getMinutes())}`
  }

  const time = `${date.getHours()}:${pad2(date.getMinutes())}`
  if (isSameDay(date, now)) {
    return time
  }

  const monthDay = `${date.getMonth() + 1}-${date.getDate()} ${time}`
  if (date.getFullYear() === now.getFullYear()) {
    return monthDay
  }

  return `${date.getFullYear()}-${date.getMonth() + 1}-${date.getDate()} ${time}`
}

export function buildMessageTimeline(messages: ChatMessage[]): MessageTimelineItem[] {
  const result: MessageTimelineItem[] = []
  let previousMessageTime = 0

  for (const message of messages) {
    const messageTime = new Date(message.created_at ?? Date.now()).getTime()
    const normalizedTime = Number.isNaN(messageTime) ? Date.now() : messageTime
    const shouldShowTime =
      previousMessageTime === 0 || normalizedTime - previousMessageTime >= TIME_DIVIDER_INTERVAL

    if (shouldShowTime) {
      result.push({
        type: 'time',
        id: `time-${message.id ?? message.client_message_id ?? normalizedTime}`,
        text: formatMessageDividerTime(message.created_at),
      })
    }

    result.push({
      type: 'message',
      id: message.id ?? message.client_message_id ?? `message-${normalizedTime}-${result.length}`,
      message,
    })

    previousMessageTime = normalizedTime
  }

  return result
}
