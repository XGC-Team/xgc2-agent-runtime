import { decodeNativeSettings, type NativeProviderSettingsUpdate } from './providerSettings.js'
import { decodeDecisionFacts } from './decisionPolicy.js'
export { decodeDecisionFacts, type DecisionFacts } from './decisionPolicy.js'
import { decodePromptQueue, decodeProfiles, decodeSession, decodeSessions, decodeNativeRequest, type NativeAnswer, type NativeTurnOptions, type Scope } from './state.js'

export type NativeSessionListOptions = { after?: string; limit?: number }
export type NativeSessionMetadataUpdate = { expectedRevision: number; title?: string; archived?: boolean }

function decodeSessionPage(value: unknown) {
  const page = record(value)
  if (page.nextCursor !== undefined && (typeof page.nextCursor !== 'string' || page.nextCursor.length > 1024)) throw new Error('Invalid native conversation cursor.')
  return { sessions: decodeSessions(page.sessions), nextCursor: page.nextCursor as string | undefined }
}

function decodeInputs(value: unknown) {
  if (!Array.isArray(value) || value.length > 16) throw new Error('Invalid native pending inputs.')
  return value.map(entry => {
    const input = record(entry)
    if (typeof input.submitted !== 'boolean') throw new Error('Invalid native pending input state.')
    return { request: decodeNativeRequest(input.request), submitted: input.submitted, ...(input.facts === undefined ? {} : {facts:decodeDecisionFacts(input.facts)}) }
  })
}

export const NATIVE_CLIENT_HEADER = 'X-XGC-Native-Client'
export type NativeClientOptions = { basePath: string; fetch?: typeof globalThis.fetch }

export class NativeAgentClientError extends Error {
  readonly status: number
  readonly code: string
  constructor(status: number, code: string, message: string) {
    super(message)
    this.name = 'NativeAgentClientError'
    this.status = status
    this.code = code
  }
}

function record(value: unknown): Record<string, unknown> {
  if (!value || typeof value !== 'object' || Array.isArray(value)) throw new Error('原生操作缺少响应。')
  return value as Record<string, unknown>
}

export function nativeAgentBasePath(value: string): string {
  const path = value.replace(/\/$/, '')
  if (!/^\/(?:[A-Za-z0-9_-]+\/)*[A-Za-z0-9_-]+$/.test(path)) throw new Error('原生客户端需要同源 API 根路径。')
  return path
}

export function createNativeAgentClient(options: NativeClientOptions) {
  const root = nativeAgentBasePath(options.basePath)
  const fetcher = options.fetch ?? ((...args) => globalThis.fetch(...args))
  async function request<T>(path: string, decode: (value: unknown) => T, init?: RequestInit): Promise<T> {
    const response = await fetcher(`${root}${path}`, { ...init, headers: { Accept: 'application/json', ...init?.headers } })
    const payload: unknown = await response.json().catch(() => null)
    const envelope = payload && typeof payload === 'object' && !Array.isArray(payload) ? payload as Record<string, unknown> : undefined
    if (!response.ok) {
      const error = envelope?.error && typeof envelope.error === 'object' ? envelope.error as Record<string, unknown> : undefined
      throw new NativeAgentClientError(response.status, typeof error?.code === 'string' ? error.code : `http_${response.status}`,
        typeof error?.message === 'string' ? error.message : `原生请求失败（HTTP ${response.status}）。`)
    }
    if (!envelope || !Object.prototype.hasOwnProperty.call(envelope, 'data')) throw new NativeAgentClientError(502, 'invalid_response', '原生响应缺少 data envelope。')
    return decode(envelope.data)
  }
  function post(body: unknown, key?: string): RequestInit {
    return { method: 'POST', headers: { 'Content-Type': 'application/json', [NATIVE_CLIENT_HEADER]: '1', ...(key ? { 'Idempotency-Key': key } : {}) }, body: JSON.stringify(body) }
  }
  const sessionPath = (id: string) => `/sessions/${encodeURIComponent(id)}`
  const getNativeSessionPage = (signal?: AbortSignal, options: NativeSessionListOptions = {}) => {
    const query = new URLSearchParams()
    if (options.after) query.set('after', options.after)
    if (options.limit !== undefined) query.set('limit', String(options.limit))
    return request(`/sessions${query.size ? `?${query}` : ''}`, decodeSessionPage, { signal })
  }
  return {
    basePath: root,
    getNativeSettings: (signal?: AbortSignal) => request('/settings', decodeNativeSettings, { signal }),
    updateNativeSettings: (update: NativeProviderSettingsUpdate) => request('/settings', decodeNativeSettings, post(update)),
    refreshNativeSettings: (id: string) => request('/settings/refresh', decodeNativeSettings, post({ id })),
    getNativeProfiles: (signal?: AbortSignal) => request('/providers', decodeProfiles, { signal }),
    getNativeSessionPage,
    // Existing embedded clients request a complete inventory. Follow the same
    // paginated authority instead of silently returning only the first page.
    getNativeSessions: async (signal?: AbortSignal) => {
      const sessions = [] as ReturnType<typeof decodeSessions>
      const seen = new Set<string>()
      let after: string | undefined
      do {
        const page = await getNativeSessionPage(signal,{after})
        sessions.push(...page.sessions)
        after = page.nextCursor
        if (after && seen.has(after)) throw new Error('Native conversation pagination repeated a cursor.')
        if (after) seen.add(after)
      } while (after)
      return sessions
    },
    getNativeSession: (id: string, signal?: AbortSignal) => request(sessionPath(id), decodeSession, { signal }),
    getNativeInputs: (id: string, signal?: AbortSignal) => request(`${sessionPath(id)}/inputs`, decodeInputs, { signal }),
    evaluateNativeInputs: (id:string) => request(`${sessionPath(id)}/evaluate-inputs`,record,post({})),
    updateNativeSession: (id: string, update: NativeSessionMetadataUpdate) => request(`${sessionPath(id)}/metadata`, decodeSession, post(update)),
    createNativeSession: (scope: Scope, key: string) => request('/sessions', (value) => {
      const session = decodeSession(value)
      if (session.scope.profileId !== scope.profileId || session.scope.nativeAccessConfirmed !== scope.nativeAccessConfirmed
        || session.scope.context.kind !== scope.context.kind || session.scope.context.id !== scope.context.id
        || session.scope.workspace.id !== scope.workspace.id || session.scope.workspace.revision !== scope.workspace.revision) {
        throw new Error('创建响应的会话范围不匹配。')
      }
      const expectedOptions = scope.options
      if (['model', 'effort', 'permission'].some(key => session.scope.options?.[key as keyof NativeTurnOptions] !== expectedOptions?.[key as keyof NativeTurnOptions])) {
        throw new Error('Native session options do not match the request.')
      }
      return session
    }, post(scope, key)),
    updateNativePromptQueue: (id: string, command: {operation: 'enqueue' | 'edit' | 'remove' | 'reorder' | 'pause' | 'resume'; expectedRevision?: number; id?: string; text?: string; options?: NativeTurnOptions; order?: string[]}, key: string) => request(`${sessionPath(id)}/queue`, decodePromptQueue, post(command, key)),
    sendNativePrompt: (id: string, text: string, key: string, options?: NativeTurnOptions) => request(`${sessionPath(id)}/prompts`, (value) => {
      const result = record(value)
      if (typeof result.turnId !== 'string' || !/^t_[a-f0-9]{32}$/.test(result.turnId)) throw new Error('原生任务缺少稳定轮次标识。')
      return result.turnId
    }, post({ text, ...(options ? { options } : {}) }, key)),
    answerNativeRequest: (id: string, requestId: string, answer: NativeAnswer) => request(`${sessionPath(id)}/inputs/${encodeURIComponent(requestId)}`, record, post(answer)),
    cancelNativeTurn: (id: string) => request(`${sessionPath(id)}/cancel`, record, post({})),
    reconnectNativeSession: (id: string) => request(`${sessionPath(id)}/reconnect`, record, post({})),
    closeNativeSession: (id: string) => request(`${sessionPath(id)}/close`, record, post({})),
  }
}

export type NativeAgentClient = ReturnType<typeof createNativeAgentClient>
