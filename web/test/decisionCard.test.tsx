import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { ReactNode } from 'react'
import type { TimelineItem } from '../src/upstream/t3/types.js'
import { DecisionCard } from '../src/DecisionCard.js'
import { AgentConversation, AgentInput } from '../src/AgentConversation.js'
import { applyEvent, emptyStream, AGENT_RUNTIME_SCHEMA, type AgentEvent, type AgentRequest } from '../src/state.js'

// jsdom has no viewport. Preserve the real timeline row renderer while
// replacing only the list's layout/virtualization boundary.
vi.mock('@legendapp/list/react', () => ({ LegendList: ({ data, renderItem }: { data: TimelineItem[]; renderItem: (value: { item: TimelineItem; index: number }) => ReactNode }) => <>{data.map((item, index) => <div key={item.id}>{renderItem({ item, index })}</div>)}</> }))

afterEach(cleanup)

const request: AgentRequest = { id: 'review', kind: 'permission', title: 'Edit', sourceMethod: 'claude:can_use_tool',
  options: [{ id: 'allow', label: 'Allow once', kind: 'allow_once' }, { id: 'deny', label: 'Decline', kind: 'reject_once' }], questions: [],
  createdAt: '2026-09-08T03:00:00Z', details: { toolName: 'Edit', target: '/reviewed/flight.yaml', preview: '--- Before\ngain: 1\n+++ After\ngain: 2' } }
function event(seq: number, patch: Partial<AgentEvent>): AgentEvent {
  return { schemaVersion: AGENT_RUNTIME_SCHEMA, sessionId: 'conversation', provider: 'claude', seq, turnId: 'turn', itemId: request.id,
    kind: 'input.request', createdAt: '2026-09-08T03:00:00Z', ...patch }
}

describe('shared decision surface', () => {
  it('shares one shell with domain decisions without owning their executor', () => {
    const respond = vi.fn()
    const view = render(<DecisionCard identity="experiment:arm" title="Arm selected robots" state="pending" actions={<button onClick={respond}>Confirm arm</button>}>
      <p>Scout 1</p>
    </DecisionCard>)
    expect(view.container.querySelector('[data-xgc-role="decision-card"][data-xgc-id="experiment:arm"]')?.classList.contains('xgc-decision-card')).toBe(true)
    fireEvent.click(screen.getByRole('button', { name: 'Confirm arm' }))
    expect(respond).toHaveBeenCalledTimes(1)
    expect(screen.queryByRole('button', { name: 'Allow once' })).toBeNull()
  })

  it('reviews exact Claude changes in the notification without a tool timeline', async () => {
    const onAnswer = vi.fn(async () => undefined)
    const view = render(<AgentInput sessionId="conversation" request={request} submitted={false} onAnswer={onAnswer} />)
    const card = screen.getByRole('article', { name: 'Edit' })
    const details = card.querySelector('details')!
    expect(details.open).toBe(false)
    fireEvent.click(details.querySelector('summary')!)
    expect(details.open).toBe(true)
    expect(screen.getByRole('group', { name: 'Edit' })).toBeTruthy()
    expect(card.textContent).toContain('/reviewed/flight.yaml')
    expect(card.textContent).toContain('--- Before\ngain: 1\n+++ After\ngain: 2')
    fireEvent.click(screen.getByRole('button', { name: 'Allow once' }))
    await waitFor(() => expect(onAnswer).toHaveBeenCalledWith({ optionId: 'allow' }))
    expect((screen.getByRole('button', { name: 'Allow once' }) as HTMLButtonElement).disabled).toBe(true)
    expect(screen.getByText('Response submitted')).toBeTruthy()
    expect(view.container.querySelector('[data-xgc-decision-state="submitted"]')).toBeTruthy()
    expect(screen.queryByText('Allowed')).toBeNull()
  })

  it.each(['allow', 'deny'] as const)('restores the %s decision from journal without asserting execution success', (outcome) => {
    const frames = [event(1, { request }), event(2, { kind: 'input.submitted', status: 'submitted', decision: { outcome, optionId: outcome, optionKind: outcome === 'allow' ? 'allow_once' : 'reject_once', actor: { id: 'operator', label: 'Alice' } } }), event(3, { kind: 'input.resolved', status: 'answered' })]
    const state = frames.reduce(applyEvent, emptyStream('conversation', 'claude'))
    render(<AgentConversation state={state} onAnswer={async () => undefined} />)
    expect(screen.getByRole('article', { name: 'Edit' }).textContent).toContain(outcome === 'allow' ? 'Allowed · Alice' : 'Declined · Alice')
    expect(screen.queryByRole('button', { name: 'Allow once' })).toBeNull()
    expect(state.pending).toEqual({})
    expect(screen.queryByText('Execution succeeded')).toBeNull()
  })

  it('retains expiry and never labels a legacy answered event as approval', () => {
    let state = applyEvent(emptyStream('conversation', 'claude'), event(1, { request }))
    state = applyEvent(state, event(2, { kind: 'input.resolved', status: 'answered' }))
    const view = render(<AgentConversation state={state} onAnswer={async () => undefined} />)
    expect(screen.getByText('Response completed')).toBeTruthy()
    expect(screen.queryByText('Allowed')).toBeNull()
    state = applyEvent(emptyStream('conversation', 'claude'), event(1, { request }))
    state = applyEvent(state, event(2, { kind: 'session.state', status: 'disconnected' }))
    view.rerender(<AgentConversation state={state} onAnswer={async () => undefined} />)
    expect(screen.getByText('Request expired')).toBeTruthy()
    expect(screen.queryByRole('button', { name: 'Allow once' })).toBeNull()
  })

  it('marks incomplete previews and keeps questions out of permission history', () => {
    const view = render(<AgentInput sessionId="conversation" request={{ ...request, details: { ...request.details, truncated: true } }} submitted={false} onAnswer={async () => undefined} />)
    expect(view.container.querySelector('[data-approval-detail="partial"]')?.textContent).toContain('Preview truncated')
    const question: AgentRequest = { id: 'question', kind: 'question', title: 'Choose a run', options: [], questions: [{ id: 'run', text: 'Which run?', multiple: false, freeText: true, options: [] }] }
    const state = applyEvent(emptyStream('conversation', 'claude'), event(1, { itemId: question.id, request: question }))
    expect(state.decisions).toEqual({})
  })

  it('keeps decision history across runtime replacement and rejects late runtime events', () => {
    let state = applyEvent(emptyStream('conversation', 'claude'), event(1, { request, runtimeId: 'runtime-a' }))
    state = applyEvent(state, event(2, { kind: 'session.runtime', runtimeId: 'runtime-b' }))
    expect(state.pending).toEqual({})
    expect(state.decisions.review?.status).toBe('expired')
    state = applyEvent(state, event(3, { kind: 'session.metadata', metadata: { title: 'Flight review', archived: false, metadataRevision: 2 } }))
    expect(state.metadata?.title).toBe('Flight review')
    expect(() => applyEvent(state, event(4, { kind: 'input.resolved', runtimeId: 'runtime-a', status: 'answered' }))).toThrow('runtime identity')
  })
})
