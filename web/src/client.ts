import { call, XrpcError, type CallOptions } from '@xgc2/xrpc-client'
import { decodeNativeSettings, type AgentProviderSettingsUpdate } from './providerSettings.js'
import { decodePromptQueue, decodeProfiles, decodeSession, decodeSessions, decodeNativeRequest, type AgentAnswer, type AgentTurnOptions, type Scope } from './state.js'

export { XrpcError }

/** Budgets of the calls, in milliseconds. */
export const AGENT_CALL_DEADLINES = {
  /** Reads, commands that only change stored state, and Stop. */
  short: 15_000,
  /** Calls that wait for a native client to be inspected or connected: create, send, settings. */
  long: 70_000,
}

export type AgentSessionListOptions = { after?: string; limit?: number }
export type AgentSessionMetadataUpdate = { expectedRevision: number; title?: string; archived?: boolean }
export type AgentQueueCommand = { operation: 'enqueue' | 'edit' | 'remove' | 'reorder' | 'pause' | 'resume'; expectedRevision?: number; id?: string; text?: string; options?: AgentTurnOptions; order?: string[] }

/**
 * Where the agent-runtime XRPC service is mounted, and how to reach it.
 *
 * `basePath` is an origin-relative path or an absolute http(s) URL of the
 * service root. Authentication belongs to the host: pass the credentials the
 * edge needs as `headers`, or as a `fetch` that adds them per request.
 */
export type AgentClientOptions = {
  basePath: string
  headers?: Record<string, string>
  fetch?: typeof globalThis.fetch
  deadlines?: Partial<typeof AGENT_CALL_DEADLINES>
}

/** A rejected answer to a request that is no longer pending: refresh instead of retrying. */
export function isStaleRequest(error: unknown): boolean {
  return error instanceof XrpcError && error.code === 'conflict' && (error.details as { reason?: unknown } | undefined)?.reason === 'stale'
}

function record(value: unknown): Record<string, unknown> {
  if (!value || typeof value !== 'object' || Array.isArray(value)) throw new Error('The operation returned no result.')
  return value as Record<string, unknown>
}

function decodeSessionPage(value: unknown) {
  const page = record(value)
  if (page.nextCursor !== undefined && (typeof page.nextCursor !== 'string' || page.nextCursor.length > 1024)) throw new Error('Invalid conversation cursor.')
  return { sessions: decodeSessions(page.sessions), nextCursor: page.nextCursor as string | undefined }
}

function decodeInputs(value: unknown) {
  if (!Array.isArray(value) || value.length > 16) throw new Error('Invalid pending inputs.')
  return value.map(entry => {
    const input = record(entry)
    if (typeof input.submitted !== 'boolean') throw new Error('Invalid pending input state.')
    return { request: decodeNativeRequest(input.request), submitted: input.submitted }
  })
}

/** The service root: an origin-relative path or an absolute http(s) URL, without query, fragment or credentials. */
export function nativeAgentBasePath(value: string): string {
  const path = value.replace(/\/$/, '')
  const relative = /^\/(?:[A-Za-z0-9_-]+\/)*[A-Za-z0-9_-]+$/.test(path)
  let absolute = false
  if (!relative) {
    try {
      const url = new URL(path)
      absolute = (url.protocol === 'https:' || url.protocol === 'http:') && !url.username && !url.password && !url.search && !url.hash && url.hostname !== ''
    } catch { /* not a URL */ }
  }
  if (!relative && !absolute) throw new Error('The client needs an origin-relative path or an http(s) URL as the API root.')
  return path
}

export function createAgentClient(options: AgentClientOptions) {
  const root = nativeAgentBasePath(options.basePath)
  const deadlines = { ...AGENT_CALL_DEADLINES, ...options.deadlines }
  /** One XRPC call: the answer is decoded, a failure is thrown as the XrpcError the service sent. */
  async function request<T>(method: 'GET' | 'POST', path: string, decode: (value: unknown) => T, init: { body?: unknown; deadlineMs?: number; requestId?: string; signal?: AbortSignal } = {}): Promise<T> {
    const callOptions: CallOptions = { deadlineMs: init.deadlineMs ?? deadlines.short }
    if (init.requestId !== undefined) callOptions.requestId = init.requestId
    if (init.signal) callOptions.signal = init.signal
    if (options.headers) callOptions.headers = options.headers
    if (options.fetch) callOptions.fetch = options.fetch
    const result = await call(`${root}${path}`, method, init.body, callOptions)
    if (!result.ok) throw result.error
    return decode(result.body)
  }
  const sessionPath = (id: string) => `/sessions/${encodeURIComponent(id)}`
  const getNativeSessionPage = (signal?: AbortSignal, listOptions: AgentSessionListOptions = {}) => {
    const query = new URLSearchParams()
    if (listOptions.after) query.set('after', listOptions.after)
    if (listOptions.limit !== undefined) query.set('limit', String(listOptions.limit))
    return request('GET', `/sessions${query.size ? `?${query}` : ''}`, decodeSessionPage, { signal })
  }
  return {
    basePath: root,
    getNativeSettings: (signal?: AbortSignal) => request('GET', '/settings', decodeNativeSettings, { signal }),
    updateNativeSettings: (update: AgentProviderSettingsUpdate) => request('POST', '/settings', decodeNativeSettings, { body: update, deadlineMs: deadlines.long }),
    refreshNativeSettings: (id: string) => request('POST', '/settings/refresh', decodeNativeSettings, { body: { id }, deadlineMs: deadlines.long }),
    getNativeProfiles: (signal?: AbortSignal) => request('GET', '/providers', decodeProfiles, { signal }),
    getNativeSessionPage,
    // Existing embedded clients request a complete inventory. Follow the same
    // paginated authority instead of silently returning only the first page.
    getNativeSessions: async (signal?: AbortSignal) => {
      const sessions = [] as ReturnType<typeof decodeSessions>
      const seen = new Set<string>()
      let after: string | undefined
      do {
        const page = await getNativeSessionPage(signal, { after })
        sessions.push(...page.sessions)
        after = page.nextCursor
        if (after && seen.has(after)) throw new Error('Conversation pagination repeated a cursor.')
        if (after) seen.add(after)
      } while (after)
      return sessions
    },
    getNativeSession: (id: string, signal?: AbortSignal) => request('GET', sessionPath(id), decodeSession, { signal }),
    getNativeInputs: (id: string, signal?: AbortSignal) => request('GET', `${sessionPath(id)}/inputs`, decodeInputs, { signal }),
    /** Asks the host's decision evaluator to look at the pending requests again; it answers nothing itself. */
    evaluateNativeInputs: (id: string) => request('POST', `${sessionPath(id)}/evaluate-inputs`, record),
    updateNativeSession: (id: string, update: AgentSessionMetadataUpdate) => request('POST', `${sessionPath(id)}/metadata`, decodeSession, { body: update }),
    /** `key` is the idempotency identity of the operation: it travels as the XRPC request id. */
    createNativeSession: (scope: Scope, key: string) => request('POST', '/sessions', (value) => {
      const session = decodeSession(value)
      if (session.scope.profileId !== scope.profileId || session.scope.accessConfirmed !== scope.accessConfirmed
        || session.scope.context.kind !== scope.context.kind || session.scope.context.id !== scope.context.id
        || session.scope.workspace.id !== scope.workspace.id || session.scope.workspace.revision !== scope.workspace.revision) {
        throw new Error('The created conversation does not match the requested scope.')
      }
      const expectedOptions = scope.options
      if (['model', 'effort', 'permission'].some(option => session.scope.options?.[option as keyof AgentTurnOptions] !== expectedOptions?.[option as keyof AgentTurnOptions])) {
        throw new Error('Session options do not match the request.')
      }
      return session
    }, { body: scope, requestId: key, deadlineMs: deadlines.long }),
    updateNativePromptQueue: (id: string, command: AgentQueueCommand, key: string) => request('POST', `${sessionPath(id)}/queue`, decodePromptQueue, { body: command, requestId: key }),
    sendNativePrompt: (id: string, text: string, key: string, promptOptions?: AgentTurnOptions) => request('POST', `${sessionPath(id)}/prompts`, (value) => {
      const result = record(value)
      if (typeof result.turnId !== 'string' || !/^t_[a-f0-9]{32}$/.test(result.turnId)) throw new Error('The prompt was accepted without a stable turn identity.')
      return result.turnId
    }, { body: { text, ...(promptOptions ? { options: promptOptions } : {}) }, requestId: key, deadlineMs: deadlines.long }),
    answerNativeRequest: (id: string, requestId: string, answer: AgentAnswer) => request('POST', `${sessionPath(id)}/inputs/${encodeURIComponent(requestId)}`, record, { body: answer }),
    cancelNativeTurn: (id: string) => request('POST', `${sessionPath(id)}/cancel`, record),
    reconnectNativeSession: (id: string) => request('POST', `${sessionPath(id)}/reconnect`, record),
    closeNativeSession: (id: string) => request('POST', `${sessionPath(id)}/close`, record),
  }
}

export type AgentClient = ReturnType<typeof createAgentClient>
