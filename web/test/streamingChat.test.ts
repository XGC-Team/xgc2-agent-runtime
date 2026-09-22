import { afterEach, describe, expect, it, vi } from 'vitest'
import { createHash, webcrypto } from 'node:crypto'
import { isApprovalStale, APPROVAL_STALE_AFTER_MS } from '../src/approvalStaleness.js'
import { optimisticTimelineItems, type NativeOptimisticMessage } from '../src/optimisticMessages.js'
import { nativePromptTurnId } from '../src/promptIdentity.js'
import { isTimelinePinnedToBottom } from '../src/timelineState.js'
import type { NativeItem } from '../src/state.js'

afterEach(() => vi.unstubAllGlobals())
const outgoing: NativeOptimisticMessage = { id: 'request-a', sessionId: 's', turnId: 'turn-a', text: 'same text', status: 'pending' }
function canonical(turnId: string): NativeItem {
  return { key: JSON.stringify([turnId, 'user']), turnId, id: 'user', role: 'user', text: 'same text', title: '', status: 'submitted', truncated: false }
}

describe('optimistic journal reconciliation', () => {
  it('uses the broker SHA-256 turn identity, including the NUL delimiter', async () => {
    vi.stubGlobal('crypto', webcrypto)
    const expected = `t_${createHash('sha256').update('session-a\0request-a').digest('hex').slice(0, 32)}`
    expect(await nativePromptTurnId('session-a', 'request-a')).toBe(expected)
  })
  it('shows pending immediately without a canonical message', () => {
    const row = optimisticTimelineItems('s', [], [outgoing], 'zh')[0]!
    expect(row.optimistic).toBe(true)
    expect(row.delivery?.label).toBe('发送中')
    expect(row.id).toBe(JSON.stringify(['s', 'turn-a', 'user']))
  })
  it('replaces only the confirmed identity; identical texts remain separate', () => {
    const rows = optimisticTimelineItems('s', [canonical('turn-a')], [outgoing, { ...outgoing, id: 'request-b', turnId: 'turn-b' }], 'en')
    expect(rows.map(row => row.id)).toEqual([JSON.stringify(['s', 'turn-b', 'user'])])
  })
  it('deduplicates the same queue/outbox identity and excludes other sessions', () => {
    const rows = optimisticTimelineItems('s', [], [outgoing, outgoing, { ...outgoing, sessionId: 'another' }], 'en')
    expect(rows).toHaveLength(1)
  })
  it('retains failed text and retries its original outbox id', async () => {
    const retry = vi.fn(async () => undefined)
    const row = optimisticTimelineItems('s', [], [{ ...outgoing, status: 'failed', error: 'offline' }], 'zh', retry)[0]!
    expect(row.delivery).toEqual({ state: 'failed', label: '发送失败', detail: 'offline' })
    await row.retry!.run()
    expect(retry).toHaveBeenCalledWith('request-a')
  })
})

describe('stale is an advisory, not an expiry terminal', () => {
  const createdAt = '2026-09-14T08:00:00Z'
  const created = Date.parse(createdAt)
  it('uses a strict greater-than five-minute boundary', () => {
    expect(isApprovalStale({ open: true, createdAt, now: created + APPROVAL_STALE_AFTER_MS })).toBe(false)
    expect(isApprovalStale({ open: true, createdAt, now: created + APPROVAL_STALE_AFTER_MS + 1 })).toBe(true)
  })
  it('does not age closed or undated cards into a new terminal', () => {
    expect(isApprovalStale({ open: false, createdAt, now: created + 600_000 })).toBe(false)
    expect(isApprovalStale({ open: true, createdAt: 'invalid', now: created + 600_000 })).toBe(false)
    expect(isApprovalStale({ open: true, now: created + 600_000 })).toBe(false)
  })
  it('identifies a provider-lost request independently of its age', () => {
    expect(isApprovalStale({ open: false, createdAt, now: created + 1, missing: true })).toBe(true)
  })
})

describe('timeline pin threshold', () => {
  it('pins at the actual end, allowing only small fractional-pixel drift', () => {
    expect(isTimelinePinnedToBottom({ scrollHeight: 5000, clientHeight: 500, scrollTop: 4499 })).toBe(true)
  })
  it('does not treat a full viewport of scroll-up as being at the end', () => {
    expect(isTimelinePinnedToBottom({ scrollHeight: 5000, clientHeight: 500, scrollTop: 4100 })).toBe(false)
    expect(isTimelinePinnedToBottom({ scrollHeight: 5000, clientHeight: 500, scrollTop: 4470 })).toBe(false)
  })
})
