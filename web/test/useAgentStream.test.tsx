import { describe, expect, it, vi } from 'vitest'
import { act, renderHook, waitFor } from '@testing-library/react'
import { useAgentStream, AGENT_STREAM_STATUS } from '../src/useAgentStream.js'
import { AGENT_RUNTIME_SCHEMA, type AgentSession } from '../src/state.js'
import * as streamState from '../src/state.js'
import * as fingerprints from '../src/fingerprint.js'

const session = { schemaVersion: AGENT_RUNTIME_SCHEMA, id: 's_test', provider: 'codex', state: 'ready', createdAt: '2026-09-06T00:00:00Z', lastSeq: 0, title: 'Review', archived: false, metadataRevision: 1,
  scope: { profileId: 'codex', context: { kind: 'fixture', id: 'e1' }, workspace: { id: 'debug', revision: 'v1' }, accessConfirmed: true } } as AgentSession
const delta = (seq: number, text: string) => ({ schemaVersion: AGENT_RUNTIME_SCHEMA, sessionId: 's_test', provider: 'codex', seq, kind: 'item.delta', turnId: 't_a', itemId: 'm', role: 'assistant', text })
const frame = (event: object | null, type = 'event') => `${event && 'seq' in event ? `id: ${(event as { seq: number }).seq}\n` : ''}event: ${type}\ndata: ${JSON.stringify(event ?? {})}\n\n`

/** One scripted connection: the frames it sends, then whether it stays open. */
type Connection = { status?: number; frames?: string[]; hold?: boolean }
function server(plan: Connection[], seeds: object[] = [{session,items:[],inputs:[]}]) {
  const requests: Request[] = []
  const currentRequests: Request[] = []
  const stream = (c: Connection, signal: AbortSignal | null | undefined) => {
    const encoder = new TextEncoder()
    return new ReadableStream<Uint8Array>({
      start(controller) {
        for (const text of c.frames ?? []) controller.enqueue(encoder.encode(text))
        if (!c.hold) controller.close()
        signal?.addEventListener('abort', () => { try { controller.error(new DOMException('aborted', 'AbortError')) } catch { /* closed */ } })
      },
    })
  }
  const fetch = (async (input: URL | RequestInfo, init?: RequestInit) => {
    const request = new Request(input, init)
    if (request.url.endsWith('/current')) {
      const seed = seeds[Math.min(currentRequests.length,seeds.length - 1)]!
      currentRequests.push(request)
      return new Response(JSON.stringify(seed), { status: 200, headers: { 'Content-Type': 'application/json' } })
    }
    requests.push(request)
    const connection = plan[Math.min(requests.length - 1, plan.length - 1)]!
    if (connection.status && connection.status !== 200) {
      return new Response(JSON.stringify({ error: { code: connection.status === 403 ? 'permission_denied' : 'internal', message: 'refused by the edge' } }), { status: connection.status, headers: { 'Content-Type': 'application/json' } })
    }
    return new Response(stream(connection, init?.signal), { status: 200, headers: { 'Content-Type': 'text/event-stream' } })
  }) as typeof globalThis.fetch
  return { fetch, requests, currentRequests }
}

const options = (fetch: typeof globalThis.fetch) => ({ basePath: 'https://core.test/api/agent-runtime', fetch, headers: { Authorization: 'Bearer host' } })

function controlledFrames() {
  const pending = new Map<number, FrameRequestCallback>()
  let nextId = 0, maximumPending = 0
  const request = vi.spyOn(globalThis, 'requestAnimationFrame').mockImplementation(callback => {
    pending.set(++nextId, callback)
    maximumPending = Math.max(maximumPending, pending.size)
    return nextId
  })
  const cancel = vi.spyOn(globalThis, 'cancelAnimationFrame').mockImplementation(id => { pending.delete(id) })
  return { pending, request, cancel, maximumPending: () => maximumPending, flush: () => {
    const callbacks = [...pending.values()]
    pending.clear()
    callbacks.forEach(callback => callback(0))
  } }
}

describe('useAgentStream over the XRPC event stream', () => {
  it('applies the journal, sends the host headers and reports the connection', async () => {
    const { fetch, requests } = server([{ frames: [frame(delta(1, 'hel')), frame(delta(2, 'lo'))], hold: true }])
    const { result, unmount } = renderHook(() => useAgentStream(session, 0, options(fetch)))
    await waitFor(() => expect(result.current.state.cursor).toBe(2))
    expect(result.current.state.items[0]?.text).toBe('hello')
    expect(result.current.connection).toBe(AGENT_STREAM_STATUS.connected)
    const first = requests[0]!
    expect(first.url).toBe('https://core.test/api/agent-runtime/sessions/s_test/events')
    expect(first.headers.get('Accept')).toBe('text/event-stream')
    expect(first.headers.get('Authorization')).toBe('Bearer host')
    expect(first.headers.get('Last-Event-ID')).toBe('0')
    unmount()
  })

  it('resumes from the last applied event after the connection drops and never needs the prompt again', async () => {
    const hash = vi.spyOn(fingerprints,'fingerprintBytes')
    const { fetch, requests } = server([{ frames: [frame(delta(1, 'a')), frame(delta(2, 'b'))] }, { frames: [frame(delta(3, 'c'))], hold: true }])
    const { result, unmount } = renderHook(() => useAgentStream(session, 0, options(fetch)))
    await waitFor(() => expect(result.current.state.cursor).toBe(3), { timeout: 4000 })
    expect(result.current.state.items[0]?.text).toBe('abc')
    expect(requests[1]!.headers.get('Last-Event-ID')).toBe('2')
    expect(requests.every(request => request.method === 'GET')).toBe(true)
    expect(hash).toHaveBeenCalledTimes(3)
    unmount()
  })

  it('replaces stale state with a bounded current seed on reset', async () => {
    const fresh = {...delta(1,'fresh'),kind:'item.snapshot'}
    const { fetch, currentRequests, requests } = server([
      { frames: [frame(delta(1, 'old'), 'event'), frame(delta(2, ' state'))] },
      { frames: ['event: reset\ndata: {}\n\n', frame(delta(1, 'stale replay'))], hold: true },
      { hold: true },
    ], [{session,items:[],inputs:[]}, {session:{...session,lastSeq:1},items:[fresh],inputs:[],historyBefore:1}])
    const { result, unmount } = renderHook(() => useAgentStream(session, 0, options(fetch)))
    await waitFor(() => expect(result.current.state.items[0]?.text).toBe('fresh'), { timeout: 4000 })
    expect(result.current.state.cursor).toBe(1)
    expect(result.current.state.items).toHaveLength(1)
    expect(currentRequests).toHaveLength(2)
    expect(requests[2]!.headers.get('Last-Event-ID')).toBe('1')
    unmount()
  })

  it('starts at the current cursor with complete items and pending controls without reading history', async () => {
    const hash = vi.spyOn(fingerprints,'fingerprintBytes')
    const currentSession = {...session,state:'awaiting-input',lastSeq:100,runtimeId:'r_live'}
    const item = {...delta(95,'complete answer'),kind:'item.snapshot'}
    const tool = {...delta(96,'tool output'),kind:'item.snapshot',itemId:'tool_1',role:'tool',details:{type:'commandExecution',command:'inspect'}}
    const request = {id:'q_1',kind:'permission',title:'Approve',options:[],questions:[]}
    const {fetch,requests,currentRequests} = server([{frames:[frame(delta(101,' more'))],hold:true}], [{session:currentSession,activeTurnId:'t_a',items:[item,tool],inputs:[{request,submitted:true,decision:{outcome:'allow'}}],historyBefore:90}])
    const {result,unmount} = renderHook(() => useAgentStream(currentSession as AgentSession,0,options(fetch)))
    await waitFor(() => expect(result.current.state.cursor).toBe(101))
    expect(result.current.state.items[0]?.text).toBe('complete answer more')
    expect(result.current.state.items[1]?.text).toBe('tool output')
    expect(result.current.state.pending.q_1?.submitted).toBe(true)
    expect(result.current.state.activeTurnId).toBe('t_a')
    expect(result.current.state.historyBefore).toBe(90)
    expect(currentRequests).toHaveLength(1)
    expect(requests).toHaveLength(1)
    expect(requests[0]!.headers.get('Last-Event-ID')).toBe('100')
    expect([...requests,...currentRequests].every(request => !request.url.includes('/history'))).toBe(true)
    expect(hash).toHaveBeenCalledOnce() // The seed's items and requests are not replay events.
    unmount()
  })

  it('drops pending hash work on reset, rehydrates without hashing, then resumes at the new cursor', async () => {
    const frames = controlledFrames()
    const hash = vi.spyOn(fingerprints,'fingerprintBytes')
    const serialize = vi.spyOn(JSON,'stringify')
    const fresh = {...delta(9,'fresh'),kind:'item.snapshot'}
    const {fetch,currentRequests,requests} = server([
      {frames:[frame(delta(1,'discarded')), 'event: reset\ndata: {}\n\n'],hold:true},
      {frames:[frame(delta(11,' new'))],hold:true},
    ], [{session,items:[],inputs:[]},{session:{...session,lastSeq:10},items:[fresh],inputs:[]}])
    serialize.mockClear()
    const eventSerializations = () => serialize.mock.calls.filter(([value]) => value && typeof value === 'object' && 'schemaVersion' in value && 'kind' in value)
    const {result,unmount} = renderHook(() => useAgentStream(session,0,options(fetch)))
    await waitFor(() => expect(currentRequests).toHaveLength(2))
    await waitFor(() => expect(frames.pending.size).toBe(1))
    expect(requests[1]!.headers.get('Last-Event-ID')).toBe('10')
    expect(result.current.state.items[0]?.text).toBe('fresh')
    expect(result.current.state.cursor).toBe(10)
    expect(frames.cancel).toHaveBeenCalledOnce()
    expect(hash).not.toHaveBeenCalled()
    expect(eventSerializations()).toHaveLength(0)
    act(frames.flush)
    expect(result.current.state.cursor).toBe(11)
    expect(result.current.state.items[0]?.text).toBe('fresh new')
    expect(hash).toHaveBeenCalledOnce()
    expect(eventSerializations()).toHaveLength(1)
    expect(hash.mock.calls[0]![0].byteLength).toBe(new TextEncoder().encode(JSON.stringify(streamState.decodeEvent(delta(11,' new'),session.id,session.provider))).byteLength)
    unmount()
    expect(requests[1]!.signal.aborted).toBe(true)
  })

  it('keeps a completed accepted result in the seed and rejects a stale pre-seed event', async () => {
    const completed = {...session,lastSeq:10}
    const item = {...delta(8,'accepted result'),kind:'item.snapshot'}
    const end = {schemaVersion:AGENT_RUNTIME_SCHEMA,sessionId:'s_test',provider:'codex',seq:9,kind:'turn.end',turnId:'t_a',status:'completed'}
    const {fetch} = server([{frames:[frame(delta(10,'wrong'))],hold:true}], [{session:completed,items:[item],inputs:[],result:end}])
    const {result,unmount} = renderHook(() => useAgentStream(completed,0,options(fetch)))
    await waitFor(() => expect(result.current.connection).toBe(AGENT_STREAM_STATUS.invalid))
    expect(result.current.state.items[0]?.text).toBe('accepted result')
    expect(result.current.state.items[0]?.turnStatus).toBe('completed')
    expect(result.current.error).toMatch(/cursor/)
    unmount()
  })

  it('deduplicates identical delivered events after the seed', async () => {
    const currentSession = {...session,lastSeq:10}
    const {fetch} = server([{frames:[frame(delta(11,'a')),frame(delta(11,'a')),frame(delta(12,'b'))],hold:true}], [{session:currentSession,items:[],inputs:[]}])
    const {result,unmount} = renderHook(() => useAgentStream(currentSession,0,options(fetch)))
    await waitFor(() => expect(result.current.state.cursor).toBe(12))
    expect(result.current.state.items[0]?.text).toBe('ab')
    unmount()
  })

  it('accepts a bounded event above the transport default and marks display truncation', async () => {
    const {fetch} = server([{frames:[frame(delta(1,'x'.repeat(1_200_000)))],hold:true}])
    const {result,unmount} = renderHook(() => useAgentStream(session,0,options(fetch)))
    await waitFor(() => expect(result.current.state.cursor).toBe(1))
    expect(result.current.state.items[0]?.truncated).toBe(true)
    expect(result.current.state.items[0]?.text).toHaveLength(262144)
    unmount()
  })

  it('rejects a foreign current seed before opening a stream', async () => {
    const foreign = {...session,scope:{...session.scope,context:{kind:'fixture',id:'foreign'}}}
    const {fetch,requests} = server([{hold:true}], [{session:foreign,items:[],inputs:[]}])
    const {result,unmount} = renderHook(() => useAgentStream(session,0,options(fetch)))
    await waitFor(() => expect(result.current.connection).toBe(AGENT_STREAM_STATUS.refused))
    expect(result.current.error).toMatch(/identity/)
    expect(requests).toHaveLength(0)
    unmount()
  })

  it('offers the newest explicit history page when a current seed exceeds capacity', async () => {
    const fetch = vi.fn(async () => new Response(JSON.stringify({error:{code:'resource_exhausted',message:'current view too large'}}),{status:429,headers:{'Content-Type':'application/json'}})) as typeof globalThis.fetch
    const {result,unmount} = renderHook(() => useAgentStream(session,0,options(fetch)))
    await waitFor(() => expect(result.current.connection).toBe(AGENT_STREAM_STATUS.refused))
    expect(result.current.state.historyBefore).toBe(0)
    expect(result.current.error).toContain('resource_exhausted')
    expect(fetch).toHaveBeenCalledTimes(1)
    unmount()
  })

  it('stops with an English diagnosis when an event fails validation', async () => {
    const { fetch } = server([{ frames: [frame(delta(1, 'a')), frame(delta(3, 'gap'))], hold: true }])
    const { result, unmount } = renderHook(() => useAgentStream(session, 0, options(fetch)))
    await waitFor(() => expect(result.current.connection).toBe(AGENT_STREAM_STATUS.invalid))
    expect(result.current.error).toMatch(/event stream has a gap/)
    expect(result.current.state.cursor).toBeLessThan(3) // nothing after the gap is shown
    unmount()
  })

  it('reports a refusal the service will keep giving instead of retrying forever', async () => {
    const { fetch, requests } = server([{ status: 403 }])
    const { result, unmount } = renderHook(() => useAgentStream(session, 0, options(fetch)))
    await waitFor(() => expect(result.current.connection).toBe(AGENT_STREAM_STATUS.refused))
    expect(result.current.error).toContain('permission_denied')
    expect(requests).toHaveLength(1)
    unmount()
  })

  it('bounds a 769-event journal burst to three full 256-item commits and one pending presentation frame', async () => {
    const frames = controlledFrames()
    const apply = vi.spyOn(streamState, 'applyEvent')
    const { fetch, requests } = server([{ frames: Array.from({ length: 769 }, (_, index) => frame(delta(index + 1, 'a'))), hold: true }])
    const { result, unmount } = renderHook(() => useAgentStream(session, 0, options(fetch)))
    await waitFor(() => expect(frames.request).toHaveBeenCalledTimes(4))
    expect(apply).toHaveBeenCalledTimes(768)
    expect(result.current.state.cursor).toBe(768)
    expect(frames.maximumPending()).toBe(1)
    expect(frames.pending.size).toBe(1)
    expect(frames.cancel).toHaveBeenCalledTimes(3)
    expect(requests).toHaveLength(1)
    act(frames.flush)
    expect(apply).toHaveBeenCalledTimes(769)
    expect(result.current.state.cursor).toBe(769)
    expect(result.current.state.items[0]?.text).toBe('a'.repeat(769))
    expect(frames.pending.size).toBe(0)
    unmount()
    expect(requests[0]!.signal.aborted).toBe(true)
  })

  it('cancels a pending presentation and rejects its late callback after the stream owner unmounts', async () => {
    const frames = controlledFrames()
    const apply = vi.spyOn(streamState, 'applyEvent')
    const hash = vi.spyOn(fingerprints,'fingerprintBytes')
    const { fetch, requests } = server([{ frames: [frame(delta(1, 'pending'))], hold: true }])
    const { result, unmount } = renderHook(() => useAgentStream(session, 0, options(fetch)))
    await waitFor(() => expect(frames.pending.size).toBe(1))
    const late = [...frames.pending.values()][0]!
    expect(result.current.state.cursor).toBe(0)
    unmount()
    expect(requests[0]!.signal.aborted).toBe(true)
    expect(frames.cancel).toHaveBeenCalledOnce()
    expect(frames.pending.size).toBe(0)
    act(() => late(0))
    expect(apply).not.toHaveBeenCalled()
    expect(hash).not.toHaveBeenCalled()
  })

  it('releases hidden work and does not hash the authoritative seed on reveal', async () => {
    const frames = controlledFrames()
    const hash = vi.spyOn(fingerprints,'fingerprintBytes')
    const serialize = vi.spyOn(JSON,'stringify')
    const fresh = {...delta(1,'accepted while hidden'),kind:'item.snapshot'}
    const {fetch,requests,currentRequests} = server([
      {frames:[frame(delta(1,'old pending'))],hold:true},
      {frames:[frame(delta(2,' then live'))],hold:true},
    ], [{session,items:[],inputs:[]},{session:{...session,lastSeq:1},items:[fresh],inputs:[]}])
    serialize.mockClear()
    const eventSerializations = () => serialize.mock.calls.filter(([value]) => value && typeof value === 'object' && 'schemaVersion' in value && 'kind' in value)
    const {result,rerender,unmount} = renderHook(({visible}) => useAgentStream(visible ? session : undefined,0,options(fetch)),{initialProps:{visible:true}})
    await waitFor(() => expect(frames.pending.size).toBe(1))
    const late = [...frames.pending.values()][0]!
    rerender({visible:false})
    expect(requests[0]!.signal.aborted).toBe(true)
    expect(frames.pending.size).toBe(0)
    act(() => late(0))
    expect(hash).not.toHaveBeenCalled()
    expect(eventSerializations()).toHaveLength(0)
    rerender({visible:true})
    await waitFor(() => expect(currentRequests).toHaveLength(2))
    await waitFor(() => expect(frames.pending.size).toBe(1))
    expect(result.current.state.items[0]?.text).toBe('accepted while hidden')
    expect(requests[1]!.headers.get('Last-Event-ID')).toBe('1')
    expect(hash).not.toHaveBeenCalled()
    expect(eventSerializations()).toHaveLength(0)
    act(frames.flush)
    expect(result.current.state.items[0]?.text).toBe('accepted while hidden then live')
    expect(result.current.state.cursor).toBe(2)
    expect(hash).toHaveBeenCalledOnce()
    expect(eventSerializations()).toHaveLength(1)
    unmount()
  })
})
