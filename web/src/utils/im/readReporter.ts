import type { ConversationReadState } from '@/sdk/im'

type ReadSDK = {
  markConversationRead: (
    conversationId: string,
    lastReadSequence?: number,
  ) => Promise<ConversationReadState>
}

type AckHandler = (state: ConversationReadState) => void

const DEFAULT_DELAY = 800

export class ReadReporter {
  private readonly pending = new Map<string, number>()
  private readonly acked = new Map<string, number>()
  private readonly timers = new Map<string, number>()
  private readonly inflight = new Map<string, Promise<void>>()

  constructor(
    private readonly sdkProvider: () => ReadSDK | null,
    private readonly onAck?: AckHandler,
    private readonly delay = DEFAULT_DELAY,
  ) {}

  schedule(conversationId: string, sequence: number) {
    if (!conversationId || !Number.isFinite(sequence) || sequence <= 0) return

    const pendingSeq = Math.max(this.pending.get(conversationId) ?? 0, sequence)
    const ackedSeq = this.acked.get(conversationId) ?? 0
    if (pendingSeq <= ackedSeq) return

    this.pending.set(conversationId, pendingSeq)
    if (this.timers.has(conversationId)) return

    const timer = window.setTimeout(() => {
      this.timers.delete(conversationId)
      void this.flush(conversationId)
    }, this.delay)
    this.timers.set(conversationId, timer)
  }

  ack(conversationId: string, sequence: number) {
    if (!conversationId || !Number.isFinite(sequence) || sequence <= 0) return
    const nextAcked = Math.max(this.acked.get(conversationId) ?? 0, sequence)
    this.acked.set(conversationId, nextAcked)
    if ((this.pending.get(conversationId) ?? 0) <= nextAcked) {
      this.pending.delete(conversationId)
    }
  }

  async flush(conversationId: string) {
    if (!conversationId) return

    const timer = this.timers.get(conversationId)
    if (timer != null) {
      window.clearTimeout(timer)
      this.timers.delete(conversationId)
    }

    if (this.inflight.has(conversationId)) return

    const sequence = this.pending.get(conversationId) ?? 0
    const ackedSeq = this.acked.get(conversationId) ?? 0
    if (sequence <= ackedSeq) {
      this.pending.delete(conversationId)
      return
    }

    const sdk = this.sdkProvider()
    if (!sdk) return

    this.pending.delete(conversationId)
    const request = sdk.markConversationRead(conversationId, sequence)
      .then((state) => {
        const nextAcked = Math.max(this.acked.get(conversationId) ?? 0, state.last_read_sequence ?? 0)
        this.acked.set(conversationId, nextAcked)
        this.onAck?.(state)
      })
      .catch((error) => {
        this.pending.set(conversationId, Math.max(this.pending.get(conversationId) ?? 0, sequence))
        console.error('标记会话已读失败:', error)
      })
      .finally(() => {
        this.inflight.delete(conversationId)
      })

    this.inflight.set(conversationId, request)
    return request
  }

  async flushAll() {
    await Promise.allSettled([...this.pending.keys()].map((conversationId) => this.flush(conversationId)))
  }

  reset(conversationId?: string) {
    if (conversationId) {
      this.clearTimer(conversationId)
      this.pending.delete(conversationId)
      this.acked.delete(conversationId)
      return
    }

    for (const id of this.timers.keys()) {
      this.clearTimer(id)
    }
    this.pending.clear()
    this.acked.clear()
  }

  private clearTimer(conversationId: string) {
    const timer = this.timers.get(conversationId)
    if (timer != null) {
      window.clearTimeout(timer)
      this.timers.delete(conversationId)
    }
  }
}
