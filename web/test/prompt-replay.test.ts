// @vitest-environment node
import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'
import { applyEvent, decodeEvent, emptyStream, type NativeEvent } from '../src/state.js'
import { nativeConversationModel } from '../src/nativePresentation.js'

// The real nine-event failure, with message bodies, identities and times replaced.
// Go tests consume these same bytes and assert the producer's typed receipt.
const fixture = JSON.parse(readFileSync(new URL('../../testdata/prompt-options-replay.json', import.meta.url), 'utf8')) as NativeEvent[]
const receipt = fixture[2]!
const nativeOptions = { model: 'gpt-5.3-codex-spark', effort: 'high' }
const empty = () => emptyStream('s_prompt_replay', 'codex')
const beforeReceipt = () => fixture.slice(0, 2).reduce(applyEvent, empty())
const typedReceipt = (extra: Record<string, unknown> = {}) => ({ ...receipt, details: { type: 'userMessage', nativeOptions }, ...extra })

describe('persisted native prompt options replay', () => {
  it('retains permission across partial selections and resets effort on model changes', () => {
    let state = applyEvent(beforeReceipt(), typedReceipt({ details: { type: 'userMessage', nativeOptions: { ...nativeOptions, permission: 'full-access' } } }))
    state = applyEvent(state, { ...typedReceipt(), seq: 4, turnId: 't_next', details: { type: 'userMessage', nativeOptions: { model: 'another-model' } } })
    expect(state.options).toEqual({ model: 'another-model', permission: 'full-access' })
  })
  it('replays the existing nine events through the incremental reply and authoritative completion', () => {
    let state = beforeReceipt()
    state = applyEvent(state, receipt)
    expect(state.items[0]).toMatchObject({ role: 'user', text: 'Fixture prompt', status: 'submitted', details: { type: 'userMessage', nativeOptions } })
    for (const event of fixture.slice(3, 6)) state = applyEvent(state, event)
    expect(state.cursor).toBe(6)
    expect(nativeConversationModel(state).items).toMatchObject([
      { kind: 'message', role: 'user', text: 'Fixture prompt', streaming: false },
      { kind: 'message', role: 'assistant', text: 'Fixture reply', streaming: true },
    ])
    for (const event of fixture.slice(6)) state = applyEvent(state, event)
    expect(state).toMatchObject({ cursor: 9, worker: 'ready', activeTurnId: '', lastTurnStatus: 'completed', pending: {} })
    expect(state.items).toHaveLength(2)
    expect(state.items[1]).toMatchObject({ text: 'Fixture reply', status: 'completed', turnStatus: 'completed' })
    expect(nativeConversationModel(state).items[1]).toMatchObject({ kind: 'message', streaming: false })
    expect(nativeConversationModel(state).items).toHaveLength(2)
    expect(new Set(nativeConversationModel(state).items.map(item => item.id)).size).toBe(2)
    for (const event of fixture) expect(applyEvent(state, event)).toBe(state)
  })

  it('uses one decoded contract for old and typed receipts, preserving every selected option', () => {
    const legacy = applyEvent(beforeReceipt(), receipt)
    const typed = applyEvent(beforeReceipt(), typedReceipt())
    expect(typed).toEqual(legacy)
    expect(applyEvent(legacy, typedReceipt())).toBe(legacy)
    expect(applyEvent(typed, receipt)).toBe(typed)
    const selected = { ...nativeOptions, permission: 'approval-required' }
    expect(decodeEvent(typedReceipt({ details: { type: 'userMessage', nativeOptions: selected } }), 's_prompt_replay', 'codex').details)
      .toEqual({ type: 'userMessage', nativeOptions: selected })
  })

  it('detects changed selections on a repeated event sequence', () => {
    const state = applyEvent(beforeReceipt(), receipt)
    expect(() => applyEvent(state, typedReceipt({ details: { type: 'userMessage', nativeOptions: { ...nativeOptions, effort: 'low' } } })))
      .toThrow('同一事件序号出现不同内容')
  })

  it('keeps prompts without explicit selections valid without inventing metadata', () => {
    const state = applyEvent(beforeReceipt(), { ...receipt, details: undefined })
    expect(state.items[0]?.details).toBeUndefined()
    expect(state.items[0]?.text).toBe('Fixture prompt')
  })

  it.each([
    { name: 'unknown type', details: { type: 'futureType', nativeOptions } },
    { name: 'private reasoning', details: { type: 'reasoning', content: 'unlisted' } },
    { name: 'legacy extra fields', details: { nativeOptions, extra: true } },
    { name: 'typed extra fields', details: { type: 'userMessage', nativeOptions, extra: true } },
    { name: 'typed truncation', details: { type: 'userMessage', nativeOptions, truncated: true } },
    { name: 'missing options', details: { type: 'userMessage' } },
    { name: 'empty options', details: { nativeOptions: {} } },
    { name: 'unknown option', details: { nativeOptions: { ...nativeOptions, env: 'unlisted' } } },
    { name: 'malformed option', details: { nativeOptions: { model: 7 } } },
    { name: 'oversized option', details: { nativeOptions: { model: 'm'.repeat(257) } } },
  ])('rejects $name rather than swallowing unknown details', ({ details }) => {
    expect(() => applyEvent(beforeReceipt(), { ...receipt, details })).toThrow()
  })

  it.each([
    { name: 'assistant', role: 'assistant' },
    { name: 'tool', role: 'tool' },
    { name: 'delta', kind: 'item.delta' },
    { name: 'patch', kind: 'item.patch' },
    { name: 'other item', itemId: 'another-user' },
    { name: 'unsubmitted', status: 'completed' },
  ])('rejects prompt metadata on $name for both old and typed details', ({ name: _name, ...extra }) => {
    expect(() => applyEvent(beforeReceipt(), { ...receipt, ...extra })).toThrow()
    expect(() => applyEvent(beforeReceipt(), typedReceipt(extra))).toThrow()
  })
})
