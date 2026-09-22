import { act, cleanup, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { StaleApprovalNotice } from '../src/StaleApprovalNotice.js'
import { T3PendingRequests } from '../src/T3Conversation.js'

afterEach(() => { cleanup(); vi.useRealTimers() })
describe('stale approval card content', () => {
  it('becomes stale without another incoming event and refreshes through the provided authority', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-09-14T08:00:00Z'))
    const refresh = vi.fn(async () => undefined)
    const view = render(<StaleApprovalNotice identity="approval" open createdAt="2026-09-14T08:00:00Z" locale="zh" onRefresh={refresh} />)
    act(() => vi.advanceTimersByTime(300_000))
    expect(view.container.querySelector('[data-xgc-stale]')).toBeNull()
    act(() => vi.advanceTimersByTime(1))
    expect(screen.getByText('此授权请求可能已过期')).toBeTruthy()
    await act(async () => fireEvent.click(screen.getByRole('button', { name: '刷新' })))
    expect(refresh).toHaveBeenCalledTimes(1)
    view.rerender(<StaleApprovalNotice identity="approval" open={false} createdAt="2026-09-14T08:00:00Z" locale="zh" onRefresh={refresh} />)
    expect(view.container.querySelector('[data-xgc-stale]')).toBeNull()
  })
  it('keeps a provider-lost approval in its one existing card with no actionable grant', async () => {
    const onApproval = vi.fn(async () => undefined)
    const refresh = vi.fn(async () => undefined)
    const model = { sessionKey: 'session', approvals: [{ requestId: 'r', requestKind: 'permission', title: 'Run command', createdAt: new Date().toISOString(), options: [{ decision: 'allow', label: 'Allow' }] }], userInputs: [] }
    const view = render(<T3PendingRequests model={model} active locale="zh" staleRequestIds={['r']}
      onRefreshRequests={refresh} onApproval={onApproval} onUserInput={vi.fn()} onCancelRequest={vi.fn()} />)
    expect(view.container.querySelectorAll('[data-xgc-stale]')).toHaveLength(1)
    expect(screen.getByRole('button', { name: 'Allow' }).hasAttribute('disabled')).toBe(true)
    await act(async () => fireEvent.click(screen.getByRole('button', { name: '刷新' })))
    expect(refresh).toHaveBeenCalledTimes(1)
    expect(onApproval).not.toHaveBeenCalled()
  })
  it('surfaces a real refresh failure without claiming that the request was resolved', async () => {
    const view = render(<StaleApprovalNotice identity="lost" open={false} missing locale="zh"
      onRefresh={async () => { throw new Error('Inventory offline') }} />)
    await act(async () => fireEvent.click(screen.getByRole('button', { name: '刷新' })))
    expect(screen.getByRole('alert').textContent).toBe('Inventory offline')
    expect(view.container.querySelector('[data-xgc-stale="true"]')).toBeTruthy()
  })
})
