import { useEffect, useRef, useState } from 'react'
import { events, XrpcError } from '@xgc2/xrpc-client'
import { applyEvent, emptyStream, type AgentSession } from './state.js'
import { nativeAgentBasePath } from './client.js'

/**
 * `headers` and `fetch` reach the event stream as they do the client's calls;
 * a `fetch` that adds credentials per request keeps them fresh across the
 * stream's reconnects. They are read when the stream opens: change `reload` to
 * reopen it with new ones.
 */
export type AgentStreamOptions = { basePath: string; headers?: Record<string, string>; fetch?: typeof globalThis.fetch }

export const AGENT_STREAM_STATUS = {
  idle: 'Not connected',
  connecting: 'Connecting',
  connected: 'Connected',
  interrupted: 'Connection interrupted; resuming from the event cursor. The task is not resent.',
  draining: 'The service is restarting; resuming from the event cursor. The task is not resent.',
  invalid: 'Event validation failed',
  refused: 'The event stream was refused',
}

/**
 * Follows a conversation's event stream (the XRPC http.v1 event stream of the
 * service) and reduces it into display state. The stream resumes from the last
 * event it applied; when the service no longer knows that cursor it starts over
 * from the first event. A reconnect only replays the journal and never resends
 * a prompt.
 */
export function useAgentStream(session: AgentSession | undefined, reload: number, { basePath, headers, fetch }: AgentStreamOptions) {
  const root = nativeAgentBasePath(basePath)
  const transport = useRef({ headers, fetch })
  transport.current = { headers, fetch }
  const initial = () => emptyStream(session?.id ?? '', session?.provider ?? 'codex')
  const current = useRef(initial())
  const lastReload = useRef(reload)
  const [state, setState] = useState(initial)
  const [connection, setConnection] = useState(AGENT_STREAM_STATUS.idle)
  const [error, setError] = useState('')
  useEffect(() => {
    if (!session) return
    if (current.current.sessionId !== session.id || lastReload.current !== reload) {
      current.current = emptyStream(session.id, session.provider)
      setState(current.current)
      lastReload.current = reload
    }
    setError(''); setConnection(AGENT_STREAM_STATUS.connecting)
    const abort = new AbortController()
    let active = true, frame: number | undefined
    let queue: unknown[] = []
    let invalid = false
    const fail = (cause: unknown, status: string) => {
      if (!active || invalid) return
      invalid = true; abort.abort(); queue = []; setConnection(status)
      setError(cause instanceof Error ? cause.message : String(cause))
    }
    const flush = () => {
      frame = undefined
      if (!active) return
      try {
        let next = current.current
        for (const event of queue) next = applyEvent(next, event)
        queue = []; current.current = next; setState(next)
      } catch (cause) { fail(cause, AGENT_STREAM_STATUS.invalid) }
    }
    const { headers: extra, fetch: transportFetch } = transport.current
    const cursor = current.current.cursor
    events(`${root}/sessions/${encodeURIComponent(session.id)}/events`, {
      ...(cursor > 0 ? { after: String(cursor) } : {}),
      signal: abort.signal,
      ...(extra ? { headers: extra } : {}),
      ...(transportFetch ? { fetch: transportFetch } : {}),
      onOpen: () => { if (active && !invalid) setConnection(AGENT_STREAM_STATUS.connected) },
      onError: () => { if (active && !invalid) setConnection(AGENT_STREAM_STATUS.interrupted) },
      onClosing: () => { if (active && !invalid) setConnection(AGENT_STREAM_STATUS.draining) },
      // The service does not know the cursor (its journal restarted or was
      // restored): what was shown may not be what is stored. Start over.
      onReset: () => {
        if (!active || invalid) return
        if (frame !== undefined) cancelAnimationFrame(frame)
        frame = undefined; queue = []
        current.current = emptyStream(session.id, session.provider)
        setState(current.current)
      },
      onEvent: ({ event, data }) => {
        if (!active || invalid || event !== 'event') return
        let parsed: unknown
        try { parsed = JSON.parse(data) } catch (cause) { fail(cause, AGENT_STREAM_STATUS.invalid); return }
        queue.push(parsed)
        if (queue.length >= 256) { if (frame !== undefined) cancelAnimationFrame(frame); flush() }
        else if (frame === undefined) frame = requestAnimationFrame(flush)
      },
    }).catch((cause: unknown) => {
      // Only a failure retrying cannot fix ends the stream: the service refused it.
      fail(cause instanceof XrpcError ? new Error(`${cause.message} (${cause.code})`) : cause, AGENT_STREAM_STATUS.refused)
    })
    return () => {
      active = false; abort.abort()
      if (frame !== undefined) cancelAnimationFrame(frame)
      // Unrendered frames are replayed on the next mount from our committed
      // cursor. Hiding the UI never sends cancel/close to the native process.
      queue = []
    }
  }, [session?.id, session?.provider, reload, root])
  return { state: state.sessionId === session?.id ? state : initial(), connection, error }
}
