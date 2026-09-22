export const AGENT_RUNTIME_SCHEMA = 'xgc.agent-runtime/v1' as const
export const PROVIDERS = ['codex', 'claude', 'cursor', 'opencode', 'grok'] as const
export type AgentProvider = typeof PROVIDERS[number]
export type WorkerState = 'starting' | 'ready' | 'running' | 'awaiting-input' | 'cancelling' | 'disconnected' | 'closed'
export type AgentTurnStatus = 'completed' | 'failed' | 'cancelled' | 'unknown' | 'incomplete' | 'refused' | 'blocked'
export type Role = 'user' | 'assistant' | 'tool' | 'plan' | 'activity'
export type AgentJson = null | boolean | number | string | AgentJson[] | { [key: string]: AgentJson }
export type AgentCommandAction = { type: 'read' | 'listFiles' | 'search' | 'unknown'; command: string; name?: string; path?: string; query?: string }
export type AgentFileChange = { path: string; kind: { type: 'add' | 'delete' | 'update'; movePath?: string }; diff: string }
export type AgentToolContent = { type: 'text'; text: string } | { type: 'resource_link'; uri: string; name?: string; mimeType?: string; description?: string } | { type: 'image'; mimeType: 'image/jpeg' | 'image/png'; data: string; width: number; height: number }
export type AgentItemDetails = (
  | { type: 'userMessage'; providerOptions: AgentTurnOptions }
  | { type: 'agentMessage'; phase?: 'commentary' | 'final_answer' }
  | { type: 'commandExecution'; command: string; cwd?: string; exitCode?: number; durationMs?: number; commandActions?: AgentCommandAction[] }
  | { type: 'fileChange'; changes: AgentFileChange[] }
  | { type: 'mcpToolCall'; server: string; tool: string; arguments?: AgentJson; result?: { content: AgentToolContent[]; structuredContent?: AgentJson }; error?: { message: string }; durationMs?: number }
  | { type: 'webSearch'; query: string; action?: { type: string; query?: string; queries?: string[]; url?: string; pattern?: string } }
  | { type: 'reasoningSummary'; summaryIndex?: number }
) & { truncated?: boolean }
export type AgentRequestDetails = { command?: string; cwd?: string; reason?: string; grantRoot?: string; network?: { protocol: string; host: string }; toolName?: string; target?: string; preview?: string; truncated?: boolean }
export type DecisionOption = { id: string; label: string; kind: string; description?: string }
export type AgentQuestion = { id: string; text: string; options: DecisionOption[]; multiple: boolean; freeText: boolean; header?: string }
export type AgentRequest = { id: string; kind: 'permission' | 'question' | 'plan'; title: string; text?: string; options: DecisionOption[]; questions: AgentQuestion[]; sourceMethod?: string; providerThreadId?: string; providerTurnId?: string; providerItemId?: string; createdAt?: string; details?: AgentRequestDetails }
export type AgentAnswer = { optionId?: string; answers?: Record<string, string[]>; cancel?: boolean }
export type ContextRef = { kind: string; id: string }
export type WorkspaceRef = { id: string; revision: string }
export type AgentTurnOptions = { model?: string; effort?: string; permission?: string }
export function mergeNativeOptions(base: AgentTurnOptions = {}, override: AgentTurnOptions = {}): AgentTurnOptions {
  const merged = { ...base }
  if (override.model && override.model !== base.model) delete merged.effort
  for (const key of ['model', 'effort', 'permission'] as const) {
    if (override[key]) merged[key] = override[key]
  }
  return merged
}
export type Scope = { profileId: string; context: ContextRef; workspace: WorkspaceRef; accessConfirmed: boolean; options?: AgentTurnOptions }
export type AgentSession = { schemaVersion: typeof AGENT_RUNTIME_SCHEMA; id: string; scope: Scope; options?: AgentTurnOptions; provider: AgentProvider; state: WorkerState; providerSessionId?: string; createdAt: string; lastSeq: number; title: string; archived: boolean; metadataRevision: number; runtimeId?: string }
export type AgentConversationMetadata = Pick<AgentSession, 'title' | 'archived' | 'metadataRevision'>
export type AgentDecision = { outcome: 'allow' | 'deny' | 'cancel' | 'answer'; optionId?: string; optionKind?: string; actor?: { id: string; label?: string }; policyId?: string; policyRevision?: string }
export type AgentDecisionRecord = { request: AgentRequest; turnId: string; status: 'pending' | 'submitted' | 'answered' | 'expired'; decision?: AgentDecision; updatedAt?: string }
export type AgentProfile = { id: string; provider: AgentProvider; protocol: string; available: boolean; detail: string; reviewedVersion: string; interactiveRequests: boolean; toolMode: string }
export type QueuedPrompt = { id: string; text: string; options: AgentTurnOptions; requestOptions: AgentTurnOptions; createdAt: string }
export type PromptQueue = { revision: number; paused: boolean; items: QueuedPrompt[] }
export type AgentEvent = {
  queue?: PromptQueue
  schemaVersion: typeof AGENT_RUNTIME_SCHEMA; sessionId: string; seq: number; provider: AgentProvider
  kind: 'prompt.queue' | 'session.state' | 'session.identity' | 'session.metadata' | 'session.runtime' | 'item.delta' | 'item.snapshot' | 'item.patch' | 'input.request' | 'input.submitted' | 'input.resolved' | 'turn.end' | 'notice'
  turnId?: string; itemId?: string; role?: Role; text?: string; title?: string; status?: string
  sourceMethod?: string; providerSessionId?: string; request?: AgentRequest
  createdAt?: string; providerThreadId?: string; providerTurnId?: string; details?: AgentItemDetails
  decision?: AgentDecision
  runtimeId?: string; metadata?: AgentConversationMetadata
}
export type AgentItem = { key: string; turnId: string; id: string; role: Role; text: string; title: string; status: string; truncated: boolean; turnStatus?: AgentTurnStatus; sourceMethod?: string; createdAt?: string; updatedAt?: string; providerThreadId?: string; providerTurnId?: string; details?: AgentItemDetails }
export type StreamState = {
  queue?: PromptQueue
  options?: AgentTurnOptions
  sessionId: string; provider: AgentProvider; cursor: number; worker: WorkerState; activeTurnId: string
  providerSessionId: string; items: AgentItem[]; pending: Record<string, AgentRequest & { submitted: boolean }>
  decisions: Record<string, AgentDecisionRecord>
  runtimeId: string; metadata?: AgentConversationMetadata
  notices: string[]; lastTurnStatus: string; recent: Record<number, string>
  turnFailures?: Record<string, { message: string; createdAt?: string }>
}
const workerStates: WorkerState[] = ['starting', 'ready', 'running', 'awaiting-input', 'cancelling', 'disconnected', 'closed']
const kinds: AgentEvent['kind'][] = ['prompt.queue', 'session.state', 'session.identity', 'session.metadata', 'session.runtime', 'item.delta', 'item.snapshot', 'item.patch', 'input.request', 'input.submitted', 'input.resolved', 'turn.end', 'notice']
const roles: Role[] = ['user', 'assistant', 'tool', 'plan', 'activity']
const ID = /^[A-Za-z0-9][A-Za-z0-9._-]{0,95}$/
function record(value: unknown): Record<string, unknown> {
  if (!value || typeof value !== 'object' || Array.isArray(value)) throw new Error('协议对象无效。')
  return value as Record<string, unknown>
}
function string(value: unknown, limit = 262144): string {
  if (typeof value !== 'string' || value.length > limit) throw new Error('协议文本无效或过长。')
  return value
}
function safeId(value: unknown): string { const id = string(value, 96); if (!ID.test(id)) throw new Error('会话标识无效。'); return id }
function boolean(value: unknown): boolean { if (typeof value !== 'boolean') throw new Error('协议布尔值无效。'); return value }
function integer(value: unknown): number { if (!Number.isSafeInteger(value) || (value as number) < 0) throw new Error('事件序号无效。'); return value as number }
function list<T>(value: unknown, decode: (v: unknown) => T, max: number): T[] {
  if (!Array.isArray(value) || value.length > max) throw new Error('协议集合无效。')
  return value.map(decode)
}
function provider(value: unknown): AgentProvider { if (!PROVIDERS.includes(value as AgentProvider)) throw new Error('未知供应者。'); return value as AgentProvider }
function worker(value: unknown): WorkerState { if (!workerStates.includes(value as WorkerState)) throw new Error('未知工作者状态。'); return value as WorkerState }
function optionalText(source: Record<string, unknown>, key: string, limit = 8192): Record<string, string> {
  return source[key] === undefined ? {} : { [key]: string(source[key], limit) }
}
function nativeTimestamp(value: unknown): string {
  const result = string(value, 64)
  if (!Number.isFinite(Date.parse(result))) throw new Error('事件时间无效。')
  return result
}
function nativeNumber(value: unknown, nonnegative = true): number {
  if (!Number.isSafeInteger(value) || (nonnegative && (value as number) < 0)) throw new Error('结构化数值无效。')
  return value as number
}
function hiddenNativeField(key: string) {
  return ['auth','authorization','authentication','accesstoken','refreshtoken','idtoken','apikey','password','secret','clientsecret','cookie','setcookie','env','environment','headers','meta','proto','constructor','prototype'].includes(key.toLowerCase().replace(/[-_.]/g, ''))
}
function nativeJson(value: unknown, budget = { nodes: 0 }, depth = 0): AgentJson {
  if (++budget.nodes > 1024 || depth > 6) throw new Error('工具数据超过结构限制。')
  if (value === null || typeof value === 'boolean') return value
  if (typeof value === 'string') return string(value, 16 * 1024)
  if (typeof value === 'number') { if (!Number.isFinite(value)) throw new Error('工具数值无效。'); return value }
  if (Array.isArray(value)) return list(value, (entry) => nativeJson(entry, budget, depth + 1), 64)
  const source = record(value), entries = Object.entries(source)
  if (entries.length > 64) throw new Error('工具数据超过结构限制。')
  return Object.fromEntries(entries.map(([key, entry]) => {
    if (key.length > 256 || hiddenNativeField(key)) throw new Error('工具数据含禁止的配置字段。')
    return [key, nativeJson(entry, budget, depth + 1)]
  }))
}
function nativeDetails(value: unknown): AgentItemDetails {
  const d = record(value)
  if (JSON.stringify(d).length > 2 * 1024 * 1024) throw new Error('工具详情过长。')
  const truncated = d.truncated === undefined ? {} : { truncated: boolean(d.truncated) }
  switch (d.type) {
    case 'userMessage': {
      if (Object.keys(d).some(key => !['type', 'providerOptions'].includes(key))) throw new Error('用户消息详情字段无效。')
      const providerOptions = decodeNativeTurnOptions(d.providerOptions)
      if (!Object.keys(providerOptions).length) throw new Error('用户消息选项缺失。')
      return { type: d.type, providerOptions }
    }
    case 'agentMessage': {
      if (d.phase !== undefined && d.phase !== 'commentary' && d.phase !== 'final_answer') throw new Error('消息阶段无效。')
      return { type: d.type, ...truncated, ...(d.phase ? { phase: d.phase as 'commentary' | 'final_answer' } : {}) }
    }
    case 'commandExecution':
      return { type: d.type, ...truncated, command: string(d.command, 16 * 1024), ...optionalText(d, 'cwd', 4096),
        ...(d.exitCode === undefined ? {} : { exitCode: nativeNumber(d.exitCode, false) }),
        ...(d.durationMs === undefined ? {} : { durationMs: nativeNumber(d.durationMs) }),
        ...(d.commandActions === undefined ? {} : { commandActions: list(d.commandActions, (entry) => {
          const action = record(entry)
          if (!['read','listFiles','search','unknown'].includes(String(action.type))) throw new Error('命令操作类型无效。')
          return { type: action.type as AgentCommandAction['type'], command: string(action.command, 16 * 1024), ...optionalText(action, 'name', 4096), ...optionalText(action, 'path', 4096), ...optionalText(action, 'query', 4096) }
        }, 64) }) }
    case 'fileChange':
      return { type: d.type, ...truncated, changes: list(d.changes, (entry) => {
        const change = record(entry), kind = record(change.kind)
        if (!['add','delete','update'].includes(String(kind.type))) throw new Error('文件变更类型无效。')
        return { path: string(change.path, 4096), diff: string(change.diff, 64 * 1024), kind: { type: kind.type as AgentFileChange['kind']['type'], ...optionalText(kind, 'movePath', 4096) } }
      }, 64) }
    case 'mcpToolCall': {
      const result = d.result === undefined ? undefined : record(d.result)
      const error = d.error === undefined ? undefined : record(d.error)
      let imageBytesLeft = 2 * 1024 * 1024, imageCount = 0
      return { type: d.type, ...truncated, server: string(d.server, 4096), tool: string(d.tool, 4096),
        ...(d.arguments === undefined ? {} : { arguments: nativeJson(d.arguments) }),
        ...(d.durationMs === undefined ? {} : { durationMs: nativeNumber(d.durationMs) }),
        ...(error === undefined ? {} : { error: { message: string(error.message, 4096) } }),
        ...(result === undefined ? {} : { result: { content: list(result.content, (entry): AgentToolContent => {
          const block = record(entry)
          if (block.type === 'text') return { type: block.type, text: string(block.text, 16 * 1024) }
          if (block.type === 'image') {
            const data = string(block.data, 4 * Math.ceil(imageBytesLeft / 3))
            const width = nativeNumber(block.width), height = nativeNumber(block.height)
            if (!['image/jpeg','image/png'].includes(String(block.mimeType)) || (!/^[A-Za-z0-9+/]*={0,2}$/.test(data) || data.length % 4 !== 0)
              || !data || width < 1 || height < 1 || width > 8192 || height > 8192 || width * height > 32 * 1024 * 1024 || ++imageCount > 8) throw new Error('Invalid tool image.')
            imageBytesLeft -= data.length / 4 * 3 - (data.endsWith('==') ? 2 : data.endsWith('=') ? 1 : 0)
            if (imageBytesLeft < 0) throw new Error('Tool images exceed the display limit.')
            return { type: 'image', mimeType: block.mimeType as 'image/jpeg' | 'image/png', data, width, height }
          }
          if (block.type !== 'resource_link') throw new Error('工具内容类型不支持。')
          return { type: block.type, uri: string(block.uri, 4096), ...optionalText(block, 'name', 4096), ...optionalText(block, 'mimeType', 4096), ...optionalText(block, 'description', 4096) }
        }, 64), ...(result.structuredContent === undefined ? {} : { structuredContent: nativeJson(result.structuredContent) }) } }) }
    }
    case 'webSearch': {
      const action = d.action === undefined ? undefined : record(d.action)
      return { type: d.type, ...truncated, query: string(d.query, 8192), ...(action === undefined ? {} : { action: {
        type: string(action.type, 64), ...optionalText(action, 'query'), ...optionalText(action, 'url'), ...optionalText(action, 'pattern'),
        ...(action.queries === undefined ? {} : { queries: list(action.queries, (entry) => string(entry, 8192), 32) }),
      } }) }
    }
    case 'reasoningSummary':
      return { type: d.type, ...truncated, ...(d.summaryIndex === undefined ? {} : { summaryIndex: nativeNumber(d.summaryIndex) }) }
    default: throw new Error('未知结构化条目类型。')
  }
}
function nativeRequestDetails(value: unknown): AgentRequestDetails {
  const details = record(value), network = details.network === undefined ? undefined : record(details.network)
  return { ...optionalText(details, 'command'), ...optionalText(details, 'cwd'), ...optionalText(details, 'reason'), ...optionalText(details, 'grantRoot'),
    ...optionalText(details, 'toolName', 256), ...optionalText(details, 'target'), ...optionalText(details, 'preview', 16 * 1024),
    ...(details.truncated === undefined ? {} : { truncated: boolean(details.truncated) }),
    ...(network === undefined ? {} : { network: { protocol: string(network.protocol, 64), host: string(network.host, 4096) } }) }
}

function nativeDecision(value: unknown): AgentDecision {
  const d = record(value)
  if (!['allow', 'deny', 'cancel', 'answer'].includes(String(d.outcome))) throw new Error('Unknown native decision outcome.')
  const actor = d.actor === undefined ? undefined : record(d.actor)
  return { outcome: d.outcome as AgentDecision['outcome'], ...optionalText(d, 'optionId', 256), ...optionalText(d, 'optionKind', 32),
    ...optionalText(d, 'policyId', 256), ...optionalText(d, 'policyRevision', 256),
    ...(actor === undefined ? {} : { actor: { id: string(actor.id, 256), ...optionalText(actor, 'label', 256) } }) }
}

function options(value: unknown): DecisionOption[] {
  const result = list(value, (entry) => { const o = record(entry); return { id: string(o.id, 8192), label: string(o.label, 8192), kind: string(o.kind, 32), ...optionalText(o, 'description') } }, 32)
  if (result.some(({ id }) => !id) || new Set(result.map(({ id }) => id)).size !== result.length) throw new Error('请求选项标识重复或缺失。')
  return result
}
function request(value: unknown): AgentRequest {
  const r = record(value)
  if (!['permission', 'question', 'plan'].includes(String(r.kind))) throw new Error('未知请求类型。')
  const questions = list(r.questions, (entry) => {
    const q = record(entry)
    return { id: string(q.id, 256), text: string(q.text, 8192), options: options(q.options), multiple: boolean(q.multiple), freeText: boolean(q.freeText), ...optionalText(q, 'header', 256) }
  }, 16)
  if (questions.some(({ id }) => !id) || new Set(questions.map(({ id }) => id)).size !== questions.length) throw new Error('问题标识重复或缺失。')
  return { id: safeId(r.id), kind: r.kind as AgentRequest['kind'], title: string(r.title, 4096), ...(r.text === undefined ? {} : { text: string(r.text) }), options: options(r.options), questions,
    ...optionalText(r, 'sourceMethod', 512), ...optionalText(r, 'providerThreadId', 512), ...optionalText(r, 'providerTurnId', 512), ...optionalText(r, 'providerItemId', 512),
    ...(r.createdAt === undefined ? {} : { createdAt: nativeTimestamp(r.createdAt) }), ...(r.details === undefined ? {} : { details: nativeRequestDetails(r.details) }) }
}
export const decodeNativeRequest = request
export function decodeNativeTurnOptions(value: unknown): AgentTurnOptions {
  const source = record(value)
  if (Object.keys(source).some(key => !['model', 'effort', 'permission'].includes(key))) throw new Error('Unknown native option.')
  const result: AgentTurnOptions = {}
  for (const key of ['model', 'effort', 'permission'] as const) {
    if (source[key] === undefined) continue
    const value = string(source[key], 256)
    if (!value || /[\0\r\n]/.test(value)) throw new Error('Invalid native option.')
    result[key] = value
  }
  return result
}
function nativeMetadata(value: unknown): AgentConversationMetadata {
  const s = record(value), metadataRevision = integer(s.metadataRevision), title = string(s.title, 640)
  if (metadataRevision < 1 || [...title].length > 160 || title.trim() !== title || /\p{Cc}/u.test(title)) throw new Error('Conversation metadata is invalid.')
  return { title, archived: boolean(s.archived), metadataRevision }
}
export function decodeSession(value: unknown): AgentSession {
  const s = record(value), scope = record(s.scope)
  if (s.schemaVersion !== AGENT_RUNTIME_SCHEMA) throw new Error('会话版本不兼容。')
  const context = record(scope.context), workspace = record(scope.workspace)
  const revision = string(workspace.revision, 256)
  if (!revision || /[\0\r\n]/.test(revision)) throw new Error('缺少已确认的工作区版本。')
  if (boolean(scope.accessConfirmed) !== true) throw new Error('会话缺少明确本机授权。')
  const createdAt = string(s.createdAt, 64); if (!Number.isFinite(Date.parse(createdAt))) throw new Error('会话时间无效。')
  return { schemaVersion: AGENT_RUNTIME_SCHEMA, id: safeId(s.id), provider: provider(s.provider), state: worker(s.state), createdAt,
    ...nativeMetadata(s),
    ...(s.runtimeId === undefined ? {} : { runtimeId: safeId(s.runtimeId) }),
    ...(s.options === undefined ? {} : { options: decodeNativeTurnOptions(s.options) }),
    lastSeq: integer(s.lastSeq), ...(s.providerSessionId === undefined ? {} : { providerSessionId: string(s.providerSessionId, 512) }),
    scope: { ...(scope.options === undefined ? {} : { options: decodeNativeTurnOptions(scope.options) }), profileId: safeId(scope.profileId), context: { kind: safeId(context.kind), id: safeId(context.id) }, workspace: { id: safeId(workspace.id), revision }, accessConfirmed: boolean(scope.accessConfirmed) } }
}
export function decodeSessions(value: unknown): AgentSession[] { return list(value, decodeSession, 128) }
export function decodeProfiles(value: unknown): AgentProfile[] {
  return list(value, (entry) => { const p = record(entry); return { id: safeId(p.id), provider: provider(p.provider), protocol: string(p.protocol, 64), available: boolean(p.available), detail: string(p.detail, 4096), reviewedVersion: string(p.reviewedVersion, 256), interactiveRequests: boolean(p.interactiveRequests), toolMode: string(p.toolMode, 128) } }, 16)
}
export function decodeEvent(value: unknown, sessionId: string, expectedProvider: AgentProvider): AgentEvent {
  const e = record(value)
  if (e.schemaVersion !== AGENT_RUNTIME_SCHEMA || e.sessionId !== sessionId || e.provider !== expectedProvider) throw new Error('事件身份链不匹配，已停止显示。')
  if (!kinds.includes(e.kind as AgentEvent['kind'])) throw new Error('未知事件版本，已停止显示。')
  const result: AgentEvent = { schemaVersion: AGENT_RUNTIME_SCHEMA, sessionId, provider: expectedProvider, kind: e.kind as AgentEvent['kind'], seq: integer(e.seq) }
  if (result.seq === 0) throw new Error('事件序号必须从 1 开始。')
  for (const key of ['turnId', 'itemId', 'text', 'title', 'status', 'sourceMethod', 'providerSessionId', 'providerThreadId', 'providerTurnId'] as const) {
    if (e[key] !== undefined) result[key] = string(e[key], key === 'text' ? 4 * 1024 * 1024 : key === 'title' ? 4096 : 512)
  }
  if (e.createdAt !== undefined) result.createdAt = nativeTimestamp(e.createdAt)
  if (e.runtimeId !== undefined) result.runtimeId = safeId(e.runtimeId)
  if (result.kind === 'session.runtime' && !result.runtimeId) throw new Error('Runtime identity is required.')
  if (result.kind === 'prompt.queue') result.queue = decodePromptQueue(e.queue)
  if (result.kind === 'session.metadata') result.metadata = nativeMetadata(e.metadata)
  if (e.decision !== undefined) {
    if (e.kind !== 'input.submitted' && e.kind !== 'input.resolved') throw new Error('A decision requires a response event.')
    result.decision = nativeDecision(e.decision)
  }
  if (e.details !== undefined) {
    const details = record(e.details)
    // The initial options producer persisted this exact untagged user receipt.
    // Normalize only that journal shape; unknown native structures stay invalid.
    const legacyPrompt = e.kind === 'item.snapshot' && e.role === 'user' && e.itemId === 'user' && e.status === 'submitted'
      && Object.keys(details).length === 1 && Object.keys(details)[0] === 'providerOptions'
    result.details = nativeDetails(legacyPrompt ? { type: 'userMessage', ...details } : details)
  }
  if (e.role !== undefined) { if (!roles.includes(e.role as Role)) throw new Error('未知消息角色。'); result.role = e.role as Role }
  if (result.kind.startsWith('item.') && (!result.turnId || !result.itemId || !result.role)) throw new Error('消息缺少轮次、条目或角色。')
  if (result.details) {
    const expectedRole = result.details.type === 'userMessage' ? 'user' : result.details.type === 'agentMessage' ? 'assistant' : result.details.type === 'reasoningSummary' ? 'activity' : 'tool'
    if (!result.kind.startsWith('item.') || result.role !== expectedRole) throw new Error('结构化条目与消息角色不匹配。')
    if (result.details.type === 'userMessage' && (result.kind !== 'item.snapshot' || result.itemId !== 'user' || result.status !== 'submitted')) throw new Error('用户消息回执无效。')
  }
  if (result.kind === 'session.state') worker(result.status)
  if (result.kind === 'session.identity' && (!result.providerSessionId || result.providerSessionId.length > 512)) throw new Error('会话身份缺失。')
  if (result.kind.startsWith('input.') && (!result.itemId || !result.turnId)) throw new Error('请求身份缺失。')
  if (result.kind === 'input.request') { result.request = request(e.request); if (result.request.id !== result.itemId) throw new Error('审批标识不匹配。') }
  if (result.kind === 'turn.end' && (!result.turnId || !['completed', 'failed', 'cancelled', 'unknown', 'incomplete', 'refused', 'blocked'].includes(result.status ?? ''))) throw new Error('轮次终态无效。')
  return result
}
export function emptyStream(sessionId: string, source: AgentProvider): StreamState {
  return { sessionId, provider: source, cursor: 0, worker: 'starting', activeTurnId: '', providerSessionId: '', runtimeId: '', items: [], pending: {}, decisions: {}, notices: [], lastTurnStatus: '', recent: {} }
}
const ITEM_LIMIT = 262144
export function applyEvent(state: StreamState, raw: unknown): StreamState {
  const e = decodeEvent(raw, state.sessionId, state.provider)
  const fingerprint = JSON.stringify(e)
  if (e.seq <= state.cursor) {
    if (state.recent[e.seq] && state.recent[e.seq] !== fingerprint) throw new Error('同一事件序号出现不同内容。')
    return state
  }
  if (e.seq !== state.cursor + 1) throw new Error('事件流存在缺口；请重新读取会话，不要重新提交任务。')
  const recent = { ...state.recent, [e.seq]: fingerprint }; delete recent[e.seq - 64]
  const next = { ...state, cursor: e.seq, recent }
  if (e.runtimeId) {
    if (state.runtimeId && e.kind !== 'session.runtime' && e.runtimeId !== state.runtimeId) throw new Error('The runtime identity changed without a new runtime event.')
    next.runtimeId = e.runtimeId
  }
  switch (e.kind) {
    case 'session.metadata':
      if (state.metadata && e.metadata!.metadataRevision <= state.metadata.metadataRevision) throw new Error('Conversation metadata revision did not advance.')
      next.metadata = e.metadata
      break
    case 'session.runtime':
      next.pending = {}
      next.decisions = expireDecisions(state.decisions, e.createdAt)
      next.activeTurnId = ''
      break
    case 'session.identity':
      if (state.providerSessionId && state.providerSessionId !== e.providerSessionId) throw new Error('会话身份发生变化。')
      next.providerSessionId = e.providerSessionId!; break
    case 'session.state':
      next.worker = worker(e.status)
      // State events establish the active public turn. Cancellation may omit it;
      // retain only a previously observed turn, never infer one from an item.
      next.activeTurnId = ['running', 'awaiting-input', 'cancelling'].includes(next.worker) ? e.turnId || state.activeTurnId : ''
      if (['closed', 'disconnected'].includes(next.worker)) {
        next.pending = {}
        next.decisions = expireDecisions(state.decisions, e.createdAt)
      }
      break
    case 'item.delta': case 'item.snapshot': case 'item.patch': {
      if (e.details?.type === 'userMessage') next.options = mergeNativeOptions(state.options, e.details.providerOptions)
      const key = JSON.stringify([e.turnId, e.itemId])
      const index = state.items.findIndex((item) => item.key === key)
      if (index < 0 && state.items.length >= 2048) throw new Error('会话显示条目已达上限；原始事件仍保留在本地日志。')
      const existing = state.items[index]
      if (existing && existing.role !== e.role) throw new Error('同一条目的角色发生变化。')
      for (const key of ['providerThreadId', 'providerTurnId'] as const) { if (existing?.[key] && e[key] && existing[key] !== e[key]) throw new Error('同一条目的身份发生变化。') }
      if (existing?.details && e.details && existing.details.type !== e.details.type) throw new Error('同一条目的结构类型发生变化。')
      const content = e.kind === 'item.delta' ? (existing?.text ?? '') + (e.text ?? '') : e.kind === 'item.snapshot' ? e.text ?? '' : existing?.text ?? ''
      const item: AgentItem = { key, turnId: e.turnId!, id: e.itemId!, role: e.role!, text: content.slice(0, ITEM_LIMIT), title: e.title || existing?.title || '', status: e.status || existing?.status || 'running', truncated: content.length > ITEM_LIMIT || (e.kind !== 'item.snapshot' && Boolean(existing?.truncated)),
        ...(existing?.turnStatus ? { turnStatus: existing.turnStatus } : {}),
        ...((e.sourceMethod ?? existing?.sourceMethod) ? { sourceMethod: e.sourceMethod ?? existing?.sourceMethod } : {}),
        ...((existing?.createdAt ?? e.createdAt) ? { createdAt: existing?.createdAt ?? e.createdAt } : {}),
        ...((e.createdAt ?? existing?.updatedAt) ? { updatedAt: e.createdAt ?? existing?.updatedAt } : {}),
        ...((e.providerThreadId ?? existing?.providerThreadId) ? { providerThreadId: e.providerThreadId ?? existing?.providerThreadId } : {}),
        ...((e.providerTurnId ?? existing?.providerTurnId) ? { providerTurnId: e.providerTurnId ?? existing?.providerTurnId } : {}),
        ...((e.details ?? existing?.details) ? { details: e.details ?? existing?.details } : {}),
      }
      next.items = [...state.items]
      if (index < 0) next.items.push(item); else next.items[index] = item
      break
    }
    case 'input.request':
      next.pending = { ...state.pending, [e.itemId!]: { ...e.request!, submitted: false } }
      if (e.request!.kind !== 'question') next.decisions = { ...state.decisions, [e.itemId!]: { request: e.request!, turnId: e.turnId!, status: 'pending', updatedAt: e.createdAt } }
      break
    case 'input.submitted':
      if (state.pending[e.itemId!]) next.pending = { ...state.pending, [e.itemId!]: { ...state.pending[e.itemId!]!, submitted: true } }
      if (state.decisions[e.itemId!]) next.decisions = { ...state.decisions, [e.itemId!]: { ...state.decisions[e.itemId!]!, status: 'submitted', decision: e.decision, updatedAt: e.createdAt } }
      break
    case 'input.resolved':
      next.pending = { ...state.pending }; delete next.pending[e.itemId!]
      if (state.decisions[e.itemId!]) next.decisions = { ...state.decisions, [e.itemId!]: { ...state.decisions[e.itemId!]!, status: e.status === 'expired' ? 'expired' : 'answered', decision: e.decision ?? state.decisions[e.itemId!]!.decision, updatedAt: e.createdAt } }
      break
    case 'prompt.queue':
      if (state.queue && e.queue!.revision <= state.queue.revision) throw new Error('Queue revision did not advance.')
      next.queue = e.queue
      break
    case 'turn.end':
      next.pending = {}
      next.decisions = expireDecisions(state.decisions, e.createdAt)
      next.lastTurnStatus = e.status!
      if (e.status === 'failed') next.turnFailures = { ...state.turnFailures, [e.turnId!]: { message: e.text ?? '', createdAt: e.createdAt } }
      if (state.activeTurnId === e.turnId) next.activeTurnId = ''
      // A turn result does not certify the terminal result of any tool item.
      next.items = state.items.map((item) => item.turnId === e.turnId ? { ...item, turnStatus: e.status as AgentTurnStatus } : item)
      if (e.text) next.notices = [...state.notices, e.text].slice(-32)
      break
    case 'notice':
      if (e.text) next.notices = [...state.notices, e.text].slice(-32)
      break
  }
  return next
}

function expireDecisions(decisions: StreamState['decisions'], at?: string): StreamState['decisions'] {
  return Object.fromEntries(Object.entries(decisions).map(([id, value]) => [id,
    value.status === 'pending' || value.status === 'submitted' ? { ...value, status: 'expired', updatedAt: at } : value]))
}

export function decodePromptQueue(value: unknown): PromptQueue {
  const q = record(value)
  const items = list(q.items, value => { const p = record(value); return {id: safeId(p.id),text: string(p.text, 131072),options: decodeNativeTurnOptions(p.options),requestOptions: decodeNativeTurnOptions(p.requestOptions),createdAt: nativeTimestamp(p.createdAt)} }, 20)
  if (new Set(items.map(p => p.id)).size !== items.length) throw new Error('Duplicate queued message identity.')
  return {revision: integer(q.revision),paused: boolean(q.paused),items}
}
