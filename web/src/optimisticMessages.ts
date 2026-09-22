import type { AgentItem } from './state.js'
import type { TimelineMessage } from './upstream/t3/types.js'

/** A projection of the host's existing outbox/queue, not another message journal. */
export type AgentOptimisticMessage = {
  id: string
  sessionId?: string
  turnId?: string
  text: string
  createdAt?: string
  status: 'pending' | 'queued' | 'failed'
  error?: string
}

/** Identity, never text equality, reconciles the pending row with the broker receipt. */
export function optimisticTimelineItems(
  sessionId: string,
  canonical: readonly AgentItem[],
  outgoing: readonly AgentOptimisticMessage[],
  locale: 'en' | 'zh',
  retry?: (id: string) => Promise<unknown>,
): TimelineMessage[] {
  const received = new Set(canonical.filter(item => item.role === 'user').map(item => item.turnId))
  const seen = new Set<string>()
  const labels = locale === 'zh'
    ? { pending: '发送中', queued: '已排队', failed: '发送失败', retry: '重试' }
    : { pending: 'Sending', queued: 'Queued', failed: 'Send failed', retry: 'Retry' }
  return outgoing.flatMap(message => {
    if (message.sessionId && message.sessionId !== sessionId) return []
    if (message.turnId && received.has(message.turnId)) return []
    const id = message.turnId
      ? JSON.stringify([sessionId, message.turnId, 'user'])
      : `optimistic:${message.id}`
    if (seen.has(id)) return []
    seen.add(id)
    return [{
      kind: 'message' as const, id, role: 'user' as const, text: message.text,
      createdAt: message.createdAt, optimistic: true,
      delivery: {
        state: message.status === 'failed' ? 'failed' as const : 'sending' as const,
        label: labels[message.status], detail: message.error,
      },
      ...(message.status === 'failed' && retry ? {
        retry: { label: labels.retry, run: () => retry(message.id) },
      } : {}),
    }]
  })
}
