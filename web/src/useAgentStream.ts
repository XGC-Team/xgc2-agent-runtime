import { useEffect, useRef, useState } from 'react'
import { events, XrpcError } from '@xgc2/xrpc-client'
import { applyEvent, emptyStream, type AgentSession } from './state.js'
import { createAgentClient, nativeAgentBasePath } from './client.js'

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
 * event it applied. The first connection and a cursor reset read the bounded
 * current projection before following its exact cursor. Reconnect never resends
 * a prompt or automatically reads older history.
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
    let queue: unknown[] = [], queueBytes = 0
    let invalid = false, streamAbort: AbortController | undefined
    const fail = (cause: unknown, status: string) => {
      if (!active || invalid) return
      invalid = true; abort.abort(); streamAbort?.abort(); queue = []; queueBytes = 0; setConnection(status)
      setError(cause instanceof Error ? cause.message : String(cause))
    }
    const flush = () => {
      frame = undefined
      if (!active) return
      try {
        let next = current.current
        for (const event of queue) next = applyEvent(next, event)
        queue = []; queueBytes = 0; current.current = next; setState(next)
      } catch (cause) { fail(cause, AGENT_STREAM_STATUS.invalid) }
    }
    const { headers: extra, fetch: transportFetch } = transport.current
    const client = createAgentClient({basePath:root,...(extra ? {headers:extra} : {}),...(transportFetch ? {fetch:transportFetch} : {})})
    const follow = async () => {
      let resets = 0
      while (active && !invalid) {
        let seed
        try {
          seed = await client.getNativeCurrent(session,abort.signal)
        } catch (cause) {
          if (active && cause instanceof XrpcError && cause.code === 'resource_exhausted') {
            // Zero is a local request for the newest explicit history page;
            // it does not guess a cursor from the possibly stale catalog row.
            current.current = {...emptyStream(session.id,session.provider),worker:session.state,historyBefore:0}
            setState(current.current)
          }
          throw cause
        }
        if (!active || invalid) return
        current.current = seed; setState(seed)
        streamAbort = new AbortController()
        const child = streamAbort
        const stopChild = () => child.abort()
        abort.signal.addEventListener('abort',stopChild,{once:true})
        let reset = false
        try {
          await events(`${root}/sessions/${encodeURIComponent(session.id)}/events`, {
            after: String(seed.cursor),
            signal: child.signal,
            maxEventChars: 3 * 1024 * 1024,
            ...(extra ? { headers: extra } : {}),
            ...(transportFetch ? { fetch: transportFetch } : {}),
            onOpen: () => { if (active && !invalid) setConnection(AGENT_STREAM_STATUS.connected) },
            onError: () => { if (active && !invalid) setConnection(AGENT_STREAM_STATUS.interrupted) },
            onClosing: () => { if (active && !invalid) setConnection(AGENT_STREAM_STATUS.draining) },
            onReset: () => {
              if (!active || invalid) return
              reset = true; child.abort()
              if (frame !== undefined) cancelAnimationFrame(frame)
              frame = undefined; queue = []; queueBytes = 0
              setConnection(AGENT_STREAM_STATUS.connecting)
            },
            onEvent: ({ event, data, id }) => {
              if (!active || invalid || child.signal.aborted || event !== 'event') return
              let parsed: unknown
              try {
                parsed = JSON.parse(data)
                const seq = (parsed as {seq?:unknown} | null)?.seq
                if (!Number.isSafeInteger(seq) || (seq as number) <= seed.cursor || id !== String(seq)) throw new Error('The stream cursor does not match its event.')
              } catch (cause) { fail(cause, AGENT_STREAM_STATUS.invalid); return }
              if (queue.length && queueBytes + data.length > 4 * 1024 * 1024) { if (frame !== undefined) cancelAnimationFrame(frame); flush() }
              if (!active || invalid) return
              queue.push(parsed); queueBytes += data.length
              if (queue.length >= 256) { if (frame !== undefined) cancelAnimationFrame(frame); flush() }
              else if (frame === undefined) frame = requestAnimationFrame(flush)
            },
          })
        } finally {
          abort.signal.removeEventListener('abort',stopChild)
        }
        if (!reset || !active || invalid) return
        if (++resets > 2) throw new Error('The conversation cursor repeatedly reset; reopen the conversation.')
      }
    }
    void follow().catch((cause: unknown) => {
      if (abort.signal.aborted || !active) return
      fail(cause instanceof XrpcError ? new Error(`${cause.message} (${cause.code})`) : cause, AGENT_STREAM_STATUS.refused)
    })
    return () => {
      active = false; abort.abort(); streamAbort?.abort()
      if (frame !== undefined) cancelAnimationFrame(frame)
      queue = []; queueBytes = 0
    }
  }, [session?.id, session?.provider, reload, root])
  return { state: state.sessionId === session?.id ? state : initial(), connection, error }
}
