import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { AgentConversation } from '../src/AgentConversation.js'
import { applyEvent, emptyStream, AGENT_RUNTIME_SCHEMA } from '../src/state.js'
import type { TimelineItem } from '../src/upstream/t3/types.js'

// The adapter test does not emulate virtual-list viewport measurement.
vi.mock('../src/upstream/t3/MessagesTimeline.js', () => ({MessagesTimeline: ({items}:{items:TimelineItem[]}) => <>{items.map(item => item.kind === 'custom' ? <div key={item.id}>{item.content}</div> : null)}</>}))
afterEach(cleanup)
describe('native failure projection', () => {
  it('replays a failed response visibly without inventing a tool or assistant reply', () => {
    const message = "You've hit your usage limit. Try again at 9:10 PM."
    let state = applyEvent(emptyStream('failed','codex'), {schemaVersion:AGENT_RUNTIME_SCHEMA,sessionId:'failed',provider:'codex',seq:1,kind:'turn.end',turnId:'turn',status:'failed',text:message,createdAt:'2026-09-08T09:55:00Z'})
    state = applyEvent(state, {schemaVersion:AGENT_RUNTIME_SCHEMA,sessionId:'failed',provider:'codex',seq:2,kind:'session.state',status:'ready'})
    const props = {state,onAnswer:async()=>undefined}
    const view=render(<AgentConversation {...props} />)
    expect(screen.getByRole('alert').textContent).toBe(message)
    expect(view.container.querySelector('[data-xgc-role="agent-turn-error"]')?.getAttribute('data-xgc-id')).toBe('failed:turn')
    expect(view.container.querySelector('[data-xgc-role="agent-tool-disclosure"]')).toBeNull()
    view.unmount()
    render(<AgentConversation {...props} />)
    expect(screen.getByRole('alert').textContent).toBe(message)
  })
})
