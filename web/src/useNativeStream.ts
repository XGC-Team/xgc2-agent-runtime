import { useEffect, useRef, useState } from 'react'
import { applyEvent, emptyStream, type NativeSession } from './state.js'
import { nativeAgentBasePath } from './client.js'

export type NativeStreamTransport = (options: {
  url: string
  lastEventId: () => string
  onEvent: (event: unknown) => void
  onOpen: () => void
  onError: (cause?: unknown) => void
  onInvalid: (cause: unknown) => void
}) => { close: () => void }

export type NativeStreamOptions = { basePath: string; openStream?: NativeStreamTransport }

const openEventSource: NativeStreamTransport = (options) => {
  const stream = new EventSource(options.url)
  stream.addEventListener('native-agent', (message) => {
    try { options.onEvent(JSON.parse((message as MessageEvent<string>).data) as unknown) }
    catch (cause) { options.onInvalid(cause) }
  })
  stream.onopen = options.onOpen
  stream.onerror = options.onError
  return stream
}

export function useNativeStream(session: NativeSession | undefined, reload: number, { basePath, openStream = openEventSource }: NativeStreamOptions) {
  const root = nativeAgentBasePath(basePath)
  const initial = () => emptyStream(session?.id ?? '', session?.provider ?? 'codex')
  const current = useRef(initial())
  const lastReload = useRef(reload)
  const [state, setState] = useState(initial)
  const [connection, setConnection] = useState('未连接')
  const [error, setError] = useState('')
  useEffect(() => {
    if (!session) return
    if (current.current.sessionId !== session.id || lastReload.current !== reload) {
      current.current = emptyStream(session.id, session.provider)
      setState(current.current)
      lastReload.current = reload
    }
    setError(''); setConnection('正在连接')
    let active = true, frame: number | undefined
    let queue: unknown[] = []
    // Transport callbacks may run before openStream returns its close handle.
    // eslint-disable-next-line prefer-const
    let stream: ReturnType<NativeStreamTransport> | undefined
    let invalid = false
    const fail = (cause: unknown) => {
      if (!active) return
      invalid = true; stream?.close(); queue = []; setConnection('事件校验失败')
      setError(cause instanceof Error ? cause.message : String(cause))
    }
    const flush = () => {
      frame = undefined
      if (!active) return
      try {
        let next = current.current
        for (const event of queue) next = applyEvent(next, event)
        queue = []; current.current = next; setState(next)
      } catch (cause) { fail(cause) }
    }
    stream = openStream({
      url: `${root}/sessions/${encodeURIComponent(session.id)}/events?after=${current.current.cursor}`,
      lastEventId: () => String(current.current.cursor),
      onEvent: (event) => {
        if (!active || invalid) return
        queue.push(event)
        if (queue.length >= 256) { if (frame !== undefined) cancelAnimationFrame(frame); flush() }
        else if (frame === undefined) frame = requestAnimationFrame(flush)
      },
      onOpen: () => { if (active && !invalid) setConnection('已连接') },
      onError: () => { if (active && !invalid) setConnection('连接中断，正在按事件游标恢复；任务未重发') },
      onInvalid: fail,
    })
    if (invalid) stream.close()
    return () => {
      active = false; stream?.close()
      if (frame !== undefined) cancelAnimationFrame(frame)
      // Unrendered frames are replayed on the next mount from our committed
      // cursor. Hiding the UI never sends cancel/close to the native process.
      queue = []
    }
  }, [session?.id, session?.provider, reload, root, openStream])
  return { state: state.sessionId === session?.id ? state : initial(), connection, error }
}
