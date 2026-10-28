import { createHash } from 'node:crypto'
import { describe, expect, it, vi } from 'vitest'
import * as fingerprint from '../src/fingerprint.js'
import { AGENT_RUNTIME_SCHEMA, applyEvent, decodeCurrentSeed, decodeEvent, emptyStream, type AgentSession } from '../src/state.js'

const session = { schemaVersion: AGENT_RUNTIME_SCHEMA, id: 's_test', provider: 'codex', state: 'ready', createdAt: '2026-09-06T00:00:00Z', lastSeq: 0, title: 'Review', archived: false, metadataRevision: 1,
  scope: { profileId: 'codex', context: { kind: 'fixture', id: 'e1' }, workspace: { id: 'debug', revision: 'v1' }, accessConfirmed: true } } as AgentSession
const delta = (seq: number, text: string) => ({ schemaVersion: AGENT_RUNTIME_SCHEMA, sessionId: session.id, provider: 'codex', seq, kind: 'item.delta', turnId: 't_a', itemId: 'message', role: 'assistant', text })

function eventSerializations(spy: ReturnType<typeof vi.spyOn>) {
  return spy.mock.calls.filter(([value]) => value && typeof value === 'object' && 'schemaVersion' in value && 'kind' in value)
}

describe('ordered replay fingerprints', () => {
  it('uses SHA-256 of the decoded event once per accepted or verifiable duplicate event', () => {
    const hash = vi.spyOn(fingerprint, 'fingerprintBytes')
    const serialize = vi.spyOn(JSON, 'stringify')
    const event = delta(1,'é🚁')
    const state = applyEvent(emptyStream(session.id, session.provider),event)
    expect(hash).toHaveBeenCalledOnce()
    expect(eventSerializations(serialize)).toHaveLength(1)
    const encodedBytes = hash.mock.calls[0]![0]
    const expectedJSON = JSON.stringify(decodeEvent(event,session.id,session.provider))
    expect(encodedBytes.byteLength).toBe(Buffer.byteLength(expectedJSON))
    expect(Buffer.from(state.recent[1]!)).toEqual(createHash('sha256').update(expectedJSON).digest())
    hash.mockClear(); serialize.mockClear()
    expect(applyEvent(state,event)).toBe(state)
    expect(hash).toHaveBeenCalledOnce()
    expect(eventSerializations(serialize)).toHaveLength(1)
    hash.mockClear(); serialize.mockClear()
    expect(() => applyEvent(state,delta(1,'different'))).toThrow(/different content/)
    expect(hash).toHaveBeenCalledOnce()
    expect(eventSerializations(serialize)).toHaveLength(1)
    expect(state.cursor).toBe(1)
  })

  it('retains 2,048 fingerprint bytes after 64 near-3 MiB events and counts the processed bytes', () => {
    const hash = vi.spyOn(fingerprint, 'fingerprintBytes')
    const serialize = vi.spyOn(JSON, 'stringify')
    const payload = 'x'.repeat(3 * 1024 * 1024 - 1024)
    let state = emptyStream(session.id,session.provider), serialized = 0, serializedBytes = 0, hashedBytes = 0, hashes = 0, displayCopies = 0, displayCopyBytes = 0
    for (let seq = 1; seq <= 64; seq++) {
      state = applyEvent(state,delta(seq,payload))
      expect(hash).toHaveBeenCalledOnce()
      hashes++
      hashedBytes += hash.mock.calls[0]![0].byteLength
      serialized += eventSerializations(serialize).length
      const eventIndex = serialize.mock.calls.findIndex(([value]) => value && typeof value === 'object' && 'schemaVersion' in value && 'kind' in value)
      serializedBytes += Buffer.byteLength(serialize.mock.results[eventIndex]!.value as string)
      const displayIndex = serialize.mock.calls.findIndex(([value]) => typeof value === 'string')
      if (displayIndex >= 0) { displayCopies++; displayCopyBytes += Buffer.byteLength(serialize.mock.results[displayIndex]!.value as string) }
      // A test spy must not itself retain 64 large temporary input buffers.
      hash.mockClear(); serialize.mockClear()
    }
    const retained = Object.values(state.recent)
    expect(hashes).toBe(64)
    expect(serialized).toBe(64)
    expect(serializedBytes).toBe(hashedBytes)
    expect(hashedBytes).toBe(201272055)
    expect(displayCopies).toBe(1)
    expect(displayCopyBytes).toBe(262146)
    expect(hashedBytes).toBeGreaterThan(64 * payload.length)
    expect(hashedBytes).toBeLessThan(64 * 3 * 1024 * 1024)
    expect(retained).toHaveLength(64)
    expect(new Set(retained.map(digest => digest.buffer)).size).toBe(64)
    expect(retained.reduce((bytes,digest) => bytes + digest.buffer.byteLength,0)).toBe(2048)
    expect(state.cursor).toBe(64)
    expect(state.items[0]?.text).toHaveLength(262144)
    expect(state.items[0]?.truncated).toBe(true)
    expect(applyEvent(state,delta(64,payload))).toBe(state)
    expect(hash).toHaveBeenCalledOnce()
    expect(hash.mock.calls[0]![0].byteLength).toBeGreaterThan(payload.length)
    expect(eventSerializations(serialize)).toHaveLength(1)
    hash.mockClear(); serialize.mockClear()
    expect(() => applyEvent(state,delta(64,`${payload.slice(0,-1)}y`))).toThrow(/different content/)
    expect(hash).toHaveBeenCalledOnce()
    expect(eventSerializations(serialize)).toHaveLength(1)
    hash.mockClear(); serialize.mockClear()
    state = applyEvent(state,delta(65,'small'))
    hash.mockClear(); serialize.mockClear()
    expect(() => applyEvent(state,delta(1,payload))).toThrow(/outside the verification window/)
    expect(hash).not.toHaveBeenCalled()
    expect(eventSerializations(serialize)).toHaveLength(0)
    expect(state.cursor).toBe(65)
  }, 30000)

  it('counts structured-detail validation, item-key and fingerprint serialization separately', () => {
    const hash = vi.spyOn(fingerprint, 'fingerprintBytes')
    const serialize = vi.spyOn(JSON, 'stringify')
    const details = {type:'mcpToolCall',server:'local',tool:'image',result:{content:[{type:'image',mimeType:'image/png',data:'A'.repeat(1024 * 1024),width:1,height:1}]}}
    const event = {...delta(1,'tool output'),kind:'item.snapshot',role:'tool',details}
    const state = applyEvent(emptyStream(session.id,session.provider),event)
    expect(serialize).toHaveBeenCalledTimes(3)
    const serialized = serialize.mock.results.map(result => Buffer.byteLength(result.value as string))
    expect(serialized[0]).toBeGreaterThan(1024 * 1024) // Structured-detail size validation.
    expect(serialized[1]).toBe(Buffer.byteLength('["t_a","message"]')) // Item identity.
    expect(serialized[2]).toBe(hash.mock.calls[0]![0].byteLength) // Fingerprint input.
    expect(state.recent[1]!.buffer.byteLength).toBe(32)
    hash.mockClear(); serialize.mockClear()
    expect(applyEvent(state,event)).toBe(state)
    expect(serialize).toHaveBeenCalledTimes(2) // Detail validation and duplicate fingerprint; no item update.
    expect(hash).toHaveBeenCalledOnce()
    hash.mockClear(); serialize.mockClear()
    const seededSession = {...session,lastSeq:1}
    const seeded = decodeCurrentSeed({session:seededSession,items:[event],inputs:[]},seededSession)
    expect(seeded.items[0]?.details?.type).toBe('mcpToolCall')
    expect(serialize).toHaveBeenCalledTimes(3) // Detail validation and two seed/item identity keys.
    expect(hash).not.toHaveBeenCalled()
    expect(Object.keys(seeded.recent)).toHaveLength(0)
  })

  it('copies a clipped display prefix without changing UTF-16 or recopying later deltas', () => {
    const serialize = vi.spyOn(JSON, 'stringify')
    const text = `${'a'.repeat(262143)}🚁 trailing`
    const state = applyEvent(emptyStream(session.id,session.provider),delta(1,text))
    expect(state.items[0]?.text).toBe(text.slice(0,262144))
    expect(state.items[0]?.text.charCodeAt(262143)).toBe(0xd83d)
    expect(state.items[0]?.truncated).toBe(true)
    expect(serialize.mock.calls.filter(([value]) => typeof value === 'string')).toHaveLength(1)
    serialize.mockClear()
    const next = applyEvent(state,delta(2,' more'))
    expect(next.items[0]?.text).toBe(state.items[0]?.text)
    expect(next.items[0]?.truncated).toBe(true)
    expect(serialize.mock.calls.filter(([value]) => typeof value === 'string')).toHaveLength(0)
    const snapshot = applyEvent(next,{...delta(3,'short'),kind:'item.snapshot'})
    expect(snapshot.items[0]?.text).toBe('short')
    expect(snapshot.items[0]?.truncated).toBe(false)
  })

  it('does not serialize or hash gaps, foreign events, invalid transitions or discarded seed projections', () => {
    const hash = vi.spyOn(fingerprint, 'fingerprintBytes')
    const serialize = vi.spyOn(JSON, 'stringify')
    const seededSession = {...session,state:'awaiting-input',lastSeq:100} as AgentSession
    const item = {...delta(95,'complete'),kind:'item.snapshot'}
    const request = {id:'q_1',kind:'permission',title:'Approve',options:[],questions:[]}
    const state = decodeCurrentSeed({session:seededSession,activeTurnId:'t_a',items:[item],inputs:[{request,submitted:true,decision:{outcome:'allow'}}]},seededSession)
    expect(state.cursor).toBe(100)
    expect(state.pending.q_1?.submitted).toBe(true)
    expect(Object.keys(state.recent)).toHaveLength(0)
    expect(hash).not.toHaveBeenCalled()
    expect(eventSerializations(serialize)).toHaveLength(0)
    expect(() => applyEvent(state,delta(95,'complete'))).toThrow(/outside the verification window/)
    expect(() => applyEvent(state,delta(102,'gap'))).toThrow(/gap/)
    expect(() => applyEvent(state,{...delta(101,'wrong'),sessionId:'s_other'})).toThrow(/identity chain/)
    expect(() => applyEvent(state,{...delta(101,'wrong'),role:'tool'})).toThrow(/role of an item changed/)
    expect(hash).not.toHaveBeenCalled()
    expect(eventSerializations(serialize)).toHaveLength(0)
    const next = applyEvent(state,delta(101,' more'))
    expect(next.items[0]?.text).toBe('complete more')
    expect(next.cursor).toBe(101)
    expect(hash).toHaveBeenCalledOnce()
    expect(eventSerializations(serialize)).toHaveLength(1)
  })
})
