import { describe, expect, it } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { useAgentStream, AGENT_STREAM_STATUS } from '../src/useAgentStream.js'
import { AGENT_RUNTIME_SCHEMA, type AgentSession } from '../src/state.js'

const session = { schemaVersion: AGENT_RUNTIME_SCHEMA, id: 's_test', provider: 'codex', state: 'ready', createdAt: '2026-09-06T00:00:00Z', lastSeq: 0, title: 'Review', archived: false, metadataRevision: 1,
  scope: { profileId: 'codex', context: { kind: 'fixture', id: 'e1' }, workspace: { id: 'debug', revision: 'v1' }, accessConfirmed: true } } as AgentSession
const delta = (seq: number, text: string) => ({ schemaVersion: AGENT_RUNTIME_SCHEMA, sessionId: 's_test', provider: 'codex', seq, kind: 'item.delta', turnId: 't_a', itemId: 'm', role: 'assistant', text })
const frame = (event: object | null, type = 'event') => `${event && 'seq' in event ? `id: ${(event as { seq: number }).seq}\n` : ''}event: ${type}\ndata: ${JSON.stringify(event ?? {})}\n\n`

/** One scripted connection: the frames it sends, then whether it stays open. */
type Connection = { status?: number; frames?: string[]; hold?: boolean }
function server(plan: Connection[]) {
  const requests: Request[] = []
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
    requests.push(request)
    const connection = plan[Math.min(requests.length - 1, plan.length - 1)]!
    if (connection.status && connection.status !== 200) {
      return new Response(JSON.stringify({ error: { code: connection.status === 403 ? 'permission_denied' : 'internal', message: 'refused by the edge' } }), { status: connection.status, headers: { 'Content-Type': 'application/json' } })
    }
    return new Response(stream(connection, init?.signal), { status: 200, headers: { 'Content-Type': 'text/event-stream' } })
  }) as typeof globalThis.fetch
  return { fetch, requests }
}

const options = (fetch: typeof globalThis.fetch) => ({ basePath: 'https://core.test/api/agent-runtime', fetch, headers: { Authorization: 'Bearer host' } })

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
    expect(first.headers.get('Last-Event-ID')).toBeNull()
    unmount()
  })

  it('resumes from the last applied event after the connection drops and never needs the prompt again', async () => {
    const { fetch, requests } = server([{ frames: [frame(delta(1, 'a')), frame(delta(2, 'b'))] }, { frames: [frame(delta(3, 'c'))], hold: true }])
    const { result, unmount } = renderHook(() => useAgentStream(session, 0, options(fetch)))
    await waitFor(() => expect(result.current.state.cursor).toBe(3), { timeout: 4000 })
    expect(result.current.state.items[0]?.text).toBe('abc')
    expect(requests[1]!.headers.get('Last-Event-ID')).toBe('2')
    expect(requests.every(request => request.method === 'GET')).toBe(true)
    unmount()
  })

  it('starts over when the service no longer knows the cursor', async () => {
    const { fetch } = server([
      { frames: [frame(delta(1, 'old'), 'event'), frame(delta(2, ' state'))] },
      { frames: ['event: reset\ndata: {}\n\n', frame(delta(1, 'fresh'))], hold: true },
    ])
    const { result, unmount } = renderHook(() => useAgentStream(session, 0, options(fetch)))
    await waitFor(() => expect(result.current.state.items[0]?.text).toBe('fresh'), { timeout: 4000 })
    expect(result.current.state.cursor).toBe(1)
    expect(result.current.state.items).toHaveLength(1)
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
})
