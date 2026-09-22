import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { NativeConversation, NativeInput, mergeTimelineItems } from '../src/NativeConversation.js'
import { emptyStream } from '../src/state.js'
import type { TimelineItem } from '../src/upstream/t3/types.js'

afterEach(cleanup)
describe('shared native conversation host capabilities', () => {
  it('lets a host prepare the runtime on first send and keeps options under the editor', () => {
    const send = vi.fn(async () => undefined)
    const view = render(<NativeConversation state={emptyStream('draft','codex')} draft="Inspect current state" sendDisabled={false}
      onSend={send} onAnswer={async () => undefined} emptyState={null} composerControls={<div data-testid="choices">Model · Effort · Permissions</div>} />)
    const choices = screen.getByTestId('choices'), editor = screen.getByRole('textbox')
    const composer = view.container.querySelector('[data-xgc-role="native-agent-composer"]')
    const input = view.container.querySelector('[data-xgc-role="native-agent-composer-input"]')
    expect(composer).toBeTruthy()
    expect(composer?.contains(input)).toBe(true)
    expect(composer?.contains(choices)).toBe(true)
    expect(Boolean(editor.compareDocumentPosition(choices) & Node.DOCUMENT_POSITION_FOLLOWING)).toBe(true)
    const conversation = view.container.querySelector('[data-xgc-role="native-agent-conversation"]')
    expect(conversation?.className).toContain('relative')
    expect(composer?.parentElement === conversation).toBe(false)
    fireEvent.click(screen.getByRole('button',{name:'Send message'}))
    expect(send).toHaveBeenCalledWith('Inspect current state')
  })
  it('places a host dock above the composer without a second conversation', () => {
    const state = emptyStream('docked', 'codex')
    state.worker = 'ready'
    const view = render(<NativeConversation state={state} onAnswer={async () => undefined} onSend={async () => undefined}
      dock={<div data-testid="native-dock">pad</div>} />)
    const conversation = view.container.querySelector('[data-xgc-role="native-agent-conversation"]')
    const dock = screen.getByTestId('native-dock')
    const composer = screen.getByRole('textbox')
    expect(conversation?.contains(dock)).toBe(true)
    expect(conversation?.contains(composer)).toBe(true)
    expect(Boolean(dock.compareDocumentPosition(composer) & Node.DOCUMENT_POSITION_FOLLOWING)).toBe(true)
  })
  it('does not restore an input or stop control when the host has no handlers', () => {
    const state = { ...emptyStream('read-only', 'codex'), worker: 'running' as const }
    render(<NativeConversation state={state} onAnswer={async () => undefined} />)
    expect(screen.queryByRole('textbox')).toBeNull()
    expect(screen.queryByRole('button', { name: 'Send message' })).toBeNull()
    expect(screen.queryByRole('button', { name: 'Stop generation' })).toBeNull()
  })
  it('uses the host empty state through the shared timeline without inventing a composer', () => {
    const state = emptyStream('not-connected', 'codex')
    const view = render(<NativeConversation state={state} onAnswer={async () => undefined} />)
    expect(screen.queryByText('Send a message to start the conversation.')).toBeNull()
    view.rerender(<NativeConversation state={state} onAnswer={async () => undefined}
      emptyState={<p>Connect the selected local provider to chat.</p>} />)
    expect(screen.getByText('Connect the selected local provider to chat.')).toBeTruthy()
    expect(screen.queryByText('Send a message to start the conversation.')).toBeNull()
    expect(screen.queryByRole('textbox')).toBeNull()
    expect(screen.queryByRole('button', { name: 'Send message' })).toBeNull()
    view.rerender(<NativeConversation state={state} onAnswer={async () => undefined} emptyState={null} />)
    expect(screen.queryByText('Connect the selected local provider to chat.')).toBeNull()
    expect(screen.queryByText('Send a message to start the conversation.')).toBeNull()
  })
  it('only renders the empty state when there are no native or domain timeline items', () => {
    const state = emptyStream('has-content', 'codex')
    const props = { state, onAnswer: async () => undefined, emptyState: <p>Empty conversation</p> }
    const view = render(<NativeConversation {...props} />)
    expect(screen.getByText('Empty conversation')).toBeTruthy()
    view.rerender(<NativeConversation {...props} additionalItems={[{ kind: 'custom', id: 'decision', content: <article>Decision</article> }]} />)
    expect(screen.queryByText('Empty conversation')).toBeNull()
    view.rerender(<NativeConversation {...props} state={{ ...state, items: [{ id: 'reply', key: '["turn","reply"]', turnId: 'turn', role: 'assistant', title: '', text: 'Native reply', status: 'completed', truncated: false }] }} />)
    expect(screen.queryByText('Empty conversation')).toBeNull()
  })
  it('does not disable pending approval just because sending a prompt is unavailable', () => {
    const state = emptyStream('pending-only', 'codex')
    state.worker = 'awaiting-input'
    state.pending['request-a'] = { id: 'request-a', kind: 'permission', title: 'Review exact operation', options: [{ id: 'once', label: 'Allow once', kind: 'allow_once' }], questions: [], submitted: false }
    render(<NativeConversation state={state} onAnswer={async () => undefined} emptyState={<p>Review the pending operation.</p>} />)
    expect((screen.getByRole('button', { name: 'Allow once' }) as HTMLButtonElement).disabled).toBe(false)
    expect(screen.getByText('Review the pending operation.')).toBeTruthy()
    expect(screen.queryByRole('textbox')).toBeNull()
  })
  it('retains the selected locale when the same pending request is shown outside the conversation', () => {
    render(<NativeInput sessionId="outside" submitted={false} locale="zh" request={{id:'review',kind:'permission',title:'',options:[{id:'decline',label:'Decline',kind:'reject_once'}],questions:[]}} onAnswer={async () => undefined} />)
    expect(screen.getByRole('group', {name:'Agent 请求'})).toBeTruthy()
    expect(screen.getByRole('button', {name:'Decline'})).toBeTruthy()
  })
  it('merges dated domain cards without reordering unknown-time native history', () => {
    const native: TimelineItem[] = [
      { kind: 'message', id: 'older', role: 'assistant', text: 'Existing un-timed journal item' },
      { kind: 'message', id: 'newer', role: 'assistant', text: 'Received later', createdAt: '2026-09-06T01:02:00Z' },
    ]
    const facts: TimelineItem[] = [{ kind: 'custom', id: 'decision', content: <article>Decision</article>, createdAt: '2026-09-06T01:01:00Z' }]
    expect(mergeTimelineItems(native, facts).map(({ id }) => id)).toEqual(['older', 'decision', 'newer'])
    expect(native[0]).not.toHaveProperty('createdAt')
  })
})
