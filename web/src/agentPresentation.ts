import type { AgentAnswer, AgentItem, AgentRequest, StreamState } from './state.js'
import type { PendingApproval, PendingUserInput, T3ConversationModel, TimelineItem, ToolData } from './upstream/t3/types.js'

export type AgentLocale = 'en' | 'zh'
const copy = {
  en: { tool: 'Tool activity', activity: 'Activity', plan: 'Plan', request: 'Agent request', unconfirmed: 'Completion unconfirmed', cancelled: 'Cancelled', lastTurn: 'Last turn' },
  zh: { tool: '工具活动', activity: '活动', plan: '计划', request: 'Agent 请求', unconfirmed: '完成结果未确认', cancelled: '已取消', lastTurn: '上一轮结果' },
}

function toolData(item: AgentItem, locale: AgentLocale): ToolData | undefined {
  const details = item.details
  if (!details) return undefined
  switch (details.type) {
    case 'commandExecution': return { command: details.command, cwd: details.cwd, exitCode: details.exitCode, durationMs: details.durationMs }
    case 'fileChange': return { changes: details.changes.map((change) => ({ path: change.path, kind: change.kind.type, movePath: change.kind.movePath, diff: change.diff })) }
    case 'mcpToolCall': {
      const content = details.result?.content ?? []
      let label = locale === 'zh' ? '工具图像' : 'Tool image', observedAt: string | undefined
      for (const block of content) {
        if (block.type !== 'text') continue
        try {
          const meta = JSON.parse(block.text)
          if (typeof meta?.imageTitle?.[locale] === 'string') label = meta.imageTitle[locale].slice(0,256)
          if (typeof meta?.observedAt === 'string' && Number.isFinite(Date.parse(meta.observedAt))) observedAt = meta.observedAt
        } catch { /* Ordinary text content is not image metadata. */ }
      }
      const images = content.filter(block => block.type === 'image').map(block => ({ src: `data:${block.mimeType};base64,${block.data}`, width: block.width, height: block.height, label, observedAt }))
      // Image bytes are rendered as images, never expanded into the JSON log.
      const result = details.error ?? (details.result ? { ...details.result, content: content.map(block => block.type === 'image' ? { type: block.type, mimeType: block.mimeType, width: block.width, height: block.height } : block) } : undefined)
      return { arguments: details.arguments, result, durationMs: details.durationMs, ...(images.length ? { images } : {}) }
    }
    default: return undefined
  }
}

function timelineItem(item: AgentItem, state: StreamState, locale: AgentLocale): TimelineItem {
  // The journal key includes the turn; native item IDs alone are not globally unique.
  const id = JSON.stringify([state.sessionId, item.turnId, item.id])
  if (item.role === 'user' || item.role === 'assistant') {
    return { kind: 'message', id, role: item.role, text: item.text, createdAt: item.createdAt, updatedAt: item.updatedAt,
      streaming: item.role === 'assistant' && state.worker === 'running' && state.activeTurnId === item.turnId && ['running', 'inProgress'].includes(item.status),
      displayTruncated: item.truncated }
  }
  if (item.role === 'plan') return { kind: 'plan', id, title: item.title || copy[locale].plan, text: item.text, displayTruncated: item.truncated }
  const data = toolData(item, locale)
  const failed = ['failed', 'error'].includes(item.status)
    || item.details?.type === 'commandExecution' && item.details.exitCode !== undefined && item.details.exitCode !== 0
    || item.details?.type === 'mcpToolCall' && Boolean(item.details.error)
  const progress = ['running', 'inProgress', 'started', 'pending'].includes(item.status)
  const stalled = progress && (Boolean(item.turnStatus) || state.activeTurnId !== item.turnId || ['disconnected', 'closed'].includes(state.worker))
  // A thought has no result for the operator to confirm. Only a tool can be unconfirmed.
  const unconfirmed = item.role === 'tool' && stalled
  const status = unconfirmed ? copy[locale].unconfirmed : item.role !== 'tool' && progress ? undefined : item.status
  return { kind: 'work', id, title: item.title || (item.role === 'tool' ? copy[locale].tool : copy[locale].activity),
    detail: item.text, ...(status ? { status } : {}), tone: failed ? 'error' : unconfirmed ? 'warning' : item.role === 'tool' ? 'tool' : 'info',
    command: data?.command, toolData: data, displayTruncated: item.truncated || item.details?.truncated }
}

function approvalKind(request: AgentRequest): string {
  if (request.sourceMethod === 'item/commandExecution/requestApproval') return 'command'
  if (request.sourceMethod === 'item/fileChange/requestApproval') return 'file-change'
  return request.kind
}

export function nativeRequestPresentation(request: AgentRequest & { submitted: boolean }, locale: AgentLocale = 'en'):
  { approval: PendingApproval; userInput?: never } | { userInput: PendingUserInput; approval?: never } {
  if (request.kind === 'question') {
    return { userInput: { requestId: request.id, createdAt: request.createdAt, submitted: request.submitted, title: request.title,
      questions: request.questions.map((question) => ({ id: question.id, header: question.header ?? question.text,
        question: question.text, multiSelect: question.multiple, allowCustomAnswer: question.freeText,
        options: question.options.map((option) => ({ id: option.id, value: option.id, label: option.label, description: option.description })) })) } }
  }
  // Only the backend's offered choices are actionable. No default session-wide grant.
  return { approval: { requestId: request.id, requestKind: approvalKind(request), createdAt: request.createdAt,
    submitted: request.submitted, title: request.title === 'Native operation approval' && request.details?.command
      ? locale === 'zh' ? '执行命令' : 'Run command' : request.title || copy[locale].request,
    detail: approvalDetail(request),
    truncated: request.details?.truncated,
    options: request.options.map((option) => ({ decision: option.id, label: option.label, warning: option.description })) } }
}

export function nativeConversationModel(state: StreamState, locale: AgentLocale = 'en'): T3ConversationModel {
  const approvals: PendingApproval[] = [], userInputs: PendingUserInput[] = []
  for (const request of Object.values(state.pending)) {
    const mapped = nativeRequestPresentation(request, locale)
    if (mapped.approval) approvals.push(mapped.approval)
    else userInputs.push(mapped.userInput)
  }
  const items = state.items.map((item) => timelineItem(item, state, locale))
  if (state.lastTurnStatus && state.lastTurnStatus !== 'completed' && !(state.lastTurnStatus === 'failed' && Object.keys(state.turnFailures ?? {}).length)) items.push({ kind: 'work', id: `${state.sessionId}:last-turn`, title: copy[locale].lastTurn, status: state.lastTurnStatus,
    tone: ['failed', 'unknown', 'incomplete', 'blocked'].includes(state.lastTurnStatus) ? 'warning' : 'info' })
  return { sessionKey: state.sessionId, items, approvals, userInputs,
    isRunning: ['running', 'awaiting-input', 'cancelling'].includes(state.worker) }
}

function approvalDetail(request: AgentRequest): string | undefined {
  const details = request.details
  const parts = [request.text?.trim(), details?.command?.trim(),
    details?.target ? `Target: ${details.target}` : '', details?.preview,
    details?.cwd ? `Working directory: ${details.cwd}` : '',
    details?.grantRoot ? `Requested access: ${details.grantRoot}` : '',
    details?.network ? `Network: ${details.network.protocol} ${details.network.host}` : '', details?.reason?.trim(),
    details?.truncated ? 'Preview truncated. Review the complete operation before allowing.' : '']
  return [...new Set(parts.filter((part): part is string => Boolean(part)))].join('\n') || undefined
}

function pendingRequest(state: StreamState, requestId: string): AgentRequest {
  const request = state.pending[requestId]
  if (!request || request.submitted) throw new Error('This request is no longer awaiting an answer.')
  return request
}

export function nativeApprovalAnswer(state: StreamState, requestId: string, decision: string): AgentAnswer {
  const request = pendingRequest(state, requestId)
  if (request.kind === 'question' || !request.options.some((option) => option.id === decision)) throw new Error('This decision was not offered by the native session.')
  return { optionId: decision }
}

export function nativeQuestionAnswer(state: StreamState, requestId: string, answers: Record<string, string[]>): AgentAnswer {
  const request = pendingRequest(state, requestId)
  if (request.kind !== 'question' || Object.keys(answers).length !== request.questions.length) throw new Error('Answer each question in this request.')
  const result: Record<string, string[]> = {}
  for (const question of request.questions) {
    const values = answers[question.id]
    if (!Array.isArray(values) || !values.length || (!question.multiple && values.length !== 1)
      || values.some((value) => typeof value !== 'string' || !value.trim()) || new Set(values).size !== values.length
      || (!question.freeText && values.some((value) => !question.options.some((option) => option.id === value)))) {
      throw new Error('The answer does not match the choices offered by this request.')
    }
    result[question.id] = [...values]
  }
  return { answers: result }
}

export function nativeCancelAnswer(state: StreamState, requestId: string): AgentAnswer {
  pendingRequest(state, requestId)
  return { cancel: true }
}
