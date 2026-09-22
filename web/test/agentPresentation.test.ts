import { describe, expect, it } from 'vitest'
import { emptyStream, type AgentItem, type AgentRequest, type StreamState } from '../src/state.js'
import { nativeApprovalAnswer, nativeCancelAnswer, nativeConversationModel, nativeQuestionAnswer } from '../src/agentPresentation.js'

function state(): StreamState { return { ...emptyStream('session-a', 'codex'), worker: 'running', activeTurnId: 'turn-a' } }
function item(patch: Partial<AgentItem> = {}): AgentItem {
  return { id: 'item-a', key: '["turn-a","item-a"]', turnId: 'turn-a', role: 'assistant', title: '', text: 'Hello', status: 'running', truncated: false, ...patch }
}
function permission(): AgentRequest & { submitted: boolean } {
  return { id: 'approval-a', kind: 'permission', title: 'Read the log', options: [{ id: 'accept', label: 'Allow once', kind: 'allow_once' }, { id: 'decline', label: 'Decline', kind: 'reject_once' }], questions: [], submitted: false }
}
function question(): AgentRequest & { submitted: boolean } {
  return { id: 'question-a', kind: 'question', title: 'Choose evidence', options: [], submitted: false, questions: [
    { id: 'logs', header: 'Evidence', text: 'Which logs?', options: [{ id: 'flight', label: 'Flight log', kind: 'answer', description: 'Flight controller log' }, { id: 'ros', label: 'ROS log', kind: 'answer' }], multiple: true, freeText: false },
  ] }
}

describe('native protocol to migrated T3 presentation', () => {
  it('keeps turn-scoped identity and does not invent times for old journals', () => {
    const input = state()
    input.items = [item(), item({ key: '["turn-b","item-a"]', turnId: 'turn-b' })]
    const model = nativeConversationModel(input)
    expect(model.items.map((entry) => entry.id)).toEqual(input.items.map((entry) => JSON.stringify([input.sessionId, entry.turnId, entry.id])))
    expect(model.items[0]).toMatchObject({ kind: 'message', createdAt: undefined, updatedAt: undefined })
  })
  it('retains partial text while stopping the streaming indicator after disconnect', () => {
    const input = state(); input.items = [item({ text: 'partial' })]
    expect(nativeConversationModel(input).items[0]).toMatchObject({ text: 'partial', streaming: true })
    input.worker = 'disconnected'
    expect(nativeConversationModel(input).items[0]).toMatchObject({ text: 'partial', streaming: false })
  })
  it('uses structured tool facts and preserves zero exit status/duration', () => {
    const input = state()
    input.items = [item({ role: 'tool', title: 'Inspect', text: 'output', status: 'completed', details: { type: 'commandExecution', command: 'printf ok', cwd: '/reviewed', exitCode: 0, durationMs: 0 } })]
    expect(nativeConversationModel(input).items[0]).toMatchObject({ kind: 'work', detail: 'output', tone: 'tool', toolData: { command: 'printf ok', cwd: '/reviewed', exitCode: 0, durationMs: 0 } })
  })
  it('does not parse a title or command-looking text into an executable tool fact', () => {
    const input = state(); input.items = [item({ role: 'tool', title: 'rm /fake', text: '$ rm /fake' })]
    expect(nativeConversationModel(input).items[0]).toMatchObject({ toolData: undefined, command: undefined })
  })
  it('keeps rename destinations and partial display status', () => {
    const input = state(); input.items = [item({ role: 'tool', truncated: true, details: { type: 'fileChange', changes: [{ path: 'before.ts', kind: { type: 'update', movePath: 'after.ts' }, diff: '@@\n-old\n+new' }] } })]
    expect(nativeConversationModel(input).items[0]).toMatchObject({ displayTruncated: true, toolData: { changes: [{ path: 'before.ts', movePath: 'after.ts' }] } })
  })
  it('preserves multi-select, custom-answer policy and provider option identities', () => {
    const input = state(); input.pending = { 'question-a': question() }
    expect(nativeConversationModel(input).userInputs[0]?.questions[0]).toMatchObject({ multiSelect: true, allowCustomAnswer: false, options: [{ id: 'flight', value: 'flight', label: 'Flight log', description: 'Flight controller log' }, { id: 'ros', value: 'ros' }] })
  })
  it('shows exactly the offered approval choices and keeps submitted requests disabled', () => {
    const input = state(); input.pending = { 'approval-a': { ...permission(), submitted: true } }
    expect(nativeConversationModel(input).approvals[0]).toMatchObject({ submitted: true, options: [{ decision: 'accept' }, { decision: 'decline' }] })
    expect(() => nativeApprovalAnswer(input, 'approval-a', 'accept')).toThrow('no longer')
  })
  it('rejects stale, foreign, and unoffered approval callbacks', () => {
    const input = state(); input.pending = { 'approval-a': permission() }
    expect(() => nativeApprovalAnswer(input, 'other', 'accept')).toThrow()
    expect(() => nativeApprovalAnswer(input, 'approval-a', 'acceptForSession')).toThrow('not offered')
    expect(nativeApprovalAnswer(input, 'approval-a', 'decline')).toEqual({ optionId: 'decline' })
    expect(nativeCancelAnswer(input, 'approval-a')).toEqual({ cancel: true })
  })
  it('returns multiple answer IDs intact and rejects extra or forged answers', () => {
    const input = state(); input.pending = { 'question-a': question() }
    expect(nativeQuestionAnswer(input, 'question-a', { logs: ['flight', 'ros'] })).toEqual({ answers: { logs: ['flight', 'ros'] } })
    expect(() => nativeQuestionAnswer(input, 'question-a', { logs: ['invented'] })).toThrow()
    expect(() => nativeQuestionAnswer(input, 'question-a', { logs: ['flight'], extra: ['ros'] })).toThrow()
  })
  it('does not animate a cancelled partial answer when a later turn starts', () => {
    const input = state(); input.activeTurnId = 'turn-b'
    input.items = [item({ turnStatus: 'cancelled' })]
    expect(nativeConversationModel(input).items[0]).toMatchObject({ text: 'Hello', streaming: false })
  })
  it('does not manufacture a chat activity from normal completion or hide actual completed tools and decisions', () => {
    const input = state(); input.worker = 'ready'; input.activeTurnId = ''; input.lastTurnStatus = 'completed'
    input.items = [item({ role: 'user', text: 'Fixture prompt', status: 'submitted' }), item({ id: 'reply', role: 'assistant', text: 'Fixture reply', status: 'completed' })]
    expect(nativeConversationModel(input).items).toMatchObject([
      { kind: 'message', role: 'user', text: 'Fixture prompt' },
      { kind: 'message', role: 'assistant', text: 'Fixture reply' },
    ])
    expect(nativeConversationModel(input).items).toHaveLength(2)
    input.items.push(item({ id: 'tool', role: 'tool', status: 'completed', title: 'Actual tool', text: 'Native output' }))
    input.pending = { 'approval-a': permission(), 'question-a': question() }
    const model = nativeConversationModel(input)
    expect(model.items).toHaveLength(3)
    expect(model.items[2]).toMatchObject({ kind: 'work', title: 'Actual tool', detail: 'Native output', status: 'completed' })
    expect(model.approvals[0]?.requestId).toBe('approval-a')
    expect(model.userInputs[0]?.requestId).toBe('question-a')
    expect(input.lastTurnStatus).toBe('completed')
  })
  it.each(['failed', 'unknown', 'incomplete', 'refused', 'blocked', 'cancelled'])('retains the operator-relevant terminal result %s', (status) => {
    const input = state(); input.lastTurnStatus = status
    expect(nativeConversationModel(input).items).toMatchObject([{ kind: 'work', id: 'session-a:last-turn', status }])
  })
  it('retains unknown completion without presenting an unfinished tool as running', () => {
    const input = state(); input.worker = 'disconnected'; input.activeTurnId = ''; input.lastTurnStatus = 'unknown'
    input.items = [item({ role: 'tool', status: 'inProgress' })]; input.notices = ['Transport disconnected']
    expect(nativeConversationModel(input).items).toMatchObject([
      { kind: 'work', status: 'Completion unconfirmed', tone: 'warning' },
      { kind: 'work', status: 'unknown' },
    ])
  })
  it('does not invent Session notice work rows from journal notices or session resume copy', () => {
    const input = state(); input.worker = 'ready'; input.activeTurnId = ''; input.lastTurnStatus = 'completed'
    input.notices = ['Explicit native resume; no prior prompt is replayed.', 'Host restarted. Prior turn outcome may be unknown; prompts will not be resent.']
    expect(nativeConversationModel(input).items).toEqual([])
    expect(nativeConversationModel(input).items.some((entry) => entry.kind === 'work' && 'title' in entry && entry.title === 'Session notice')).toBe(false)
  })
  it('displays the exact grant root and cwd even when legacy text contains only whitespace', () => {
    const input = state(); input.pending = { 'approval-a': { ...permission(), sourceMethod: 'item/fileChange/requestApproval', text: '\n', details: { grantRoot: '/reviewed/assets', cwd: '/reviewed/experiment' } } }
    const approval = nativeConversationModel(input).approvals[0]!
    expect(approval.detail).toContain('Requested access: /reviewed/assets')
    expect(approval.detail).toContain('Working directory: /reviewed/experiment')
  })
  it('does not revive an unfinished tool after reconnecting and starting a new turn', () => {
    const input = state(); input.activeTurnId = 'turn-b'
    input.items = [item({ role: 'tool', status: 'inProgress' })]
    expect(nativeConversationModel(input).items[0]).toMatchObject({ status: 'Completion unconfirmed', tone: 'warning' })
  })
  it('does not ask the operator to confirm a thought, including a finished greeting', () => {
    const live = state()
    live.items = [item({ id: 'segment-1', role: 'activity', title: '', text: 'The user just said "hi".', status: 'running' })]
    const thinking = nativeConversationModel(live, 'zh').items[0]
    expect(thinking).toMatchObject({ kind: 'work', title: '活动', tone: 'info', detail: 'The user just said "hi".' })
    expect(thinking).not.toHaveProperty('status')
    const done = state(); done.worker = 'ready'; done.activeTurnId = ''; done.lastTurnStatus = 'completed'
    done.items = [
      item({ role: 'user', text: 'hi', status: 'submitted' }),
      item({ id: 'segment-1', role: 'activity', title: '', text: 'The user just said "hi".', status: 'running', turnStatus: 'completed' }),
      item({ id: 'segment-2', role: 'assistant', text: 'Hi. What are you working on?', status: 'completed', turnStatus: 'completed' }),
    ]
    const model = nativeConversationModel(done, 'zh')
    expect(model.items[1]).toMatchObject({ kind: 'work', title: '活动', tone: 'info' })
    expect(model.items[1]).not.toHaveProperty('status')
    expect(JSON.stringify(model.items)).not.toContain('完成结果未确认')
    expect(JSON.stringify(model.items)).not.toContain('Completion unconfirmed')
  })
})
