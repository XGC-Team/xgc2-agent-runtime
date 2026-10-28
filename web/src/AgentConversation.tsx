import { useMemo, useRef, type ReactNode } from 'react'
import { T3Conversation, T3PendingRequests } from './T3Conversation.js'
import { DecisionCard, type DecisionCardState } from './DecisionCard.js'
import { StaleApprovalNotice } from './StaleApprovalNotice.js'
import { optimisticTimelineItems, type AgentOptimisticMessage } from './optimisticMessages.js'
import type { AgentTimelineState } from './timelineState.js'
import type { AgentAnswer, AgentRequest, AgentDecisionRecord, StreamState } from './state.js'
import { emptyStream } from './state.js'
import { createNativeTimelineProjector, nativeApprovalAnswer, nativeCancelAnswer, nativeConversationModel, nativeQuestionAnswer, nativeRequestPresentation, type AgentLocale } from './agentPresentation.js'
import type { TimelineItem } from './upstream/t3/types.js'

export type { AgentLocale } from './agentPresentation.js'
export type AgentConversationProps = {
  state: StreamState
  active?: boolean
  disabled?: boolean
  /** Hosts may prepare a durable conversation on send, before a runtime exists. */
  sendDisabled?: boolean
  sendDisabledReason?: string
  queueEnabled?: boolean
  error?: string
  renderApprovalControls?: (requestId:string) => ReactNode
  onSend?: (text: string) => Promise<unknown>
  onInterrupt?: () => Promise<unknown>
  onAnswer: (requestId: string, answer: AgentAnswer) => Promise<unknown>
  additionalItems?: readonly TimelineItem[]
  /** A projection of the existing host queue/outbox, never a second journal. */
  optimisticMessages?: readonly AgentOptimisticMessage[]
  onRetryOptimisticMessage?: (id: string) => Promise<unknown>
  onRefreshRequests?: () => Promise<unknown>
  staleRequestIds?: readonly string[]
  timelineState?: AgentTimelineState
  onTimelineStateChange?: (state: AgentTimelineState) => void
  /** Persistent hosts clear their captured draft only when admission succeeds. */
  clearDraftOnSend?: boolean
  /** Host decisions share the bounded request area; resolved receipts belong in the timeline. */
  additionalPendingRequests?: ReactNode
  draft?: string
  onDraftChange?: (draft: string) => void
  locale?: AgentLocale
  composerControls?: ReactNode
  dock?: ReactNode
  emptyState?: ReactNode
}

const EMPTY_TIMELINE_ITEMS: readonly TimelineItem[] = []
const EMPTY_OPTIMISTIC_MESSAGES: readonly AgentOptimisticMessage[] = []

/** Shared native presentation adapter. Transport, ownership and authorizations stay with the host. */
export function AgentConversation({ state, active = true, disabled = false, sendDisabled, sendDisabledReason, queueEnabled, error, renderApprovalControls, onSend, onInterrupt, onAnswer, additionalItems = EMPTY_TIMELINE_ITEMS, additionalPendingRequests, draft, onDraftChange, locale = 'en', composerControls, dock, emptyState,
  optimisticMessages = EMPTY_OPTIMISTIC_MESSAGES, onRetryOptimisticMessage, onRefreshRequests, staleRequestIds, timelineState, onTimelineStateChange, clearDraftOnSend }: AgentConversationProps) {
  const projector = useMemo(() => createNativeTimelineProjector(), [state.sessionId,state.provider])
  const empty = useMemo(() => emptyStream(state.sessionId,state.provider), [state.sessionId,state.provider])
  const retained = useRef<{ state:StreamState;locale:AgentLocale;additionalItems:readonly TimelineItem[];optimisticMessages:readonly AgentOptimisticMessage[];
    onRetryOptimisticMessage?: (id:string) => Promise<unknown>;onRefreshRequests?: () => Promise<unknown> } | null>(null)
  if (retained.current?.state.sessionId !== state.sessionId || retained.current?.state.provider !== state.provider) retained.current = null
  if (active) retained.current = { state,locale,additionalItems,optimisticMessages,onRetryOptimisticMessage,onRefreshRequests }
  const presentation: NonNullable<typeof retained.current> = retained.current ?? { state:empty,locale,additionalItems:EMPTY_TIMELINE_ITEMS,optimisticMessages:EMPTY_OPTIMISTIC_MESSAGES }
  const shownState = presentation.state,shownLocale = presentation.locale
  const nativeModel = useMemo(() => nativeConversationModel(shownState, shownLocale, projector), [shownState, shownLocale, projector])
  const optimistic = useMemo(() => presentation.optimisticMessages.length
    ? optimisticTimelineItems(shownState.sessionId, shownState.items, presentation.optimisticMessages, shownLocale, presentation.onRetryOptimisticMessage)
    : EMPTY_TIMELINE_ITEMS,
  [shownState.sessionId, shownState.items, presentation.optimisticMessages, shownLocale, presentation.onRetryOptimisticMessage])
  const failures = useMemo<TimelineItem[]>(() => Object.entries(shownState.turnFailures ?? {}).map(([turnId, failure]) => ({
    kind: 'custom', id: `failure:${shownState.sessionId}:${turnId}`, createdAt: failure.createdAt,
    content: <div role="alert" className="px-1 py-0.5 text-sm text-destructive" data-xgc-role="agent-turn-error" data-xgc-id={`${shownState.sessionId}:${turnId}`}>
      {failure.message || (shownLocale === 'zh' ? '\u6b64\u8f6e\u56de\u590d\u5931\u8d25\uff0c\u4f9b\u5e94\u8005\u672a\u63d0\u4f9b\u5177\u4f53\u539f\u56e0\u3002' : 'This response failed. The provider did not provide a reason.')}
    </div>,
  })), [shownState.turnFailures,shownState.sessionId,shownLocale])
  const decisions = useMemo<TimelineItem[]>(() => Object.values(shownState.decisions).filter((record) => record.status === 'answered' || record.status === 'expired')
    .map((record) => ({ kind: 'custom', id: `decision:${shownState.sessionId}:${record.request.id}`, createdAt: record.request.createdAt,
      content: <AgentDecisionHistory sessionId={shownState.sessionId} record={record} locale={shownLocale} onRefresh={presentation.onRefreshRequests} /> })),
  [shownState.decisions,shownState.sessionId,shownLocale,presentation.onRefreshRequests])
  const model = useMemo(() => {
    if (!decisions.length && !failures.length && !presentation.additionalItems.length && !optimistic.length) return nativeModel
    return { ...nativeModel, items: mergeTimelineItems(nativeModel.items, [...decisions, ...failures, ...presentation.additionalItems, ...optimistic]) }
  }, [nativeModel,decisions,failures,presentation.additionalItems,optimistic])
  return <T3Conversation key={state.sessionId} model={model}
    active={active} disabled={disabled} locale={shownLocale} queueEnabled={queueEnabled} sendDisabled={sendDisabled ?? (!onSend || state.worker !== 'ready')} sendDisabledReason={sendDisabledReason} error={error} renderApprovalControls={renderApprovalControls}
    composerControls={composerControls} dock={dock} additionalPendingRequests={additionalPendingRequests} emptyState={emptyState} composerEnabled={Boolean(onSend)} interruptEnabled={Boolean(onInterrupt)} draft={draft} onDraftChange={onDraftChange}
    onRefreshRequests={onRefreshRequests} staleRequestIds={staleRequestIds} timelineState={timelineState} onTimelineStateChange={onTimelineStateChange} clearDraftOnSend={clearDraftOnSend}
    onSend={async (text) => { if (!onSend) throw new Error('No native session is connected.'); return onSend(text) }}
    onInterrupt={async () => { if (!onInterrupt) throw new Error('This session cannot be interrupted here.'); return onInterrupt() }}
    onApproval={(requestId, decision) => onAnswer(requestId, nativeApprovalAnswer(state, requestId, decision))}
    onUserInput={(requestId, answers) => onAnswer(requestId, nativeQuestionAnswer(state, requestId, answers))}
    onCancelRequest={(requestId) => onAnswer(requestId, nativeCancelAnswer(state, requestId))} />
}

export function AgentInput({ request, submitted, onAnswer, sessionId = '', active = true, locale = 'en', renderApprovalControls, onRefreshRequests }: {
  request: AgentRequest; submitted: boolean; onAnswer: (answer: AgentAnswer) => Promise<unknown>; sessionId?: string; locale?: AgentLocale; active?: boolean
  renderApprovalControls?: (requestId:string) => ReactNode
  onRefreshRequests?: () => Promise<unknown>
}) {
  const state = emptyStream(sessionId, 'codex')
  state.pending = { [request.id]: { ...request, submitted } }
  return <T3PendingRequests key={`${sessionId}:${request.id}`} model={{ ...nativeConversationModel(state, locale), sessionKey: `notification:${sessionId}:${request.id}` }} active={active} locale={locale} renderApprovalControls={renderApprovalControls}
    onRefreshRequests={onRefreshRequests}
    onApproval={(requestId, decision) => onAnswer(nativeApprovalAnswer(state, requestId, decision))}
    onUserInput={(requestId, answers) => onAnswer(nativeQuestionAnswer(state, requestId, answers))}
    onCancelRequest={(requestId) => onAnswer(nativeCancelAnswer(state, requestId))} />
}

function AgentDecisionHistory({ sessionId, record, locale, onRefresh }: { sessionId: string; record: AgentDecisionRecord; locale: AgentLocale; onRefresh?: () => Promise<unknown> }) {
  const approval = nativeRequestPresentation({ ...record.request, submitted: true }, locale).approval
  if (!approval) return null
  const outcome = record.decision?.outcome
  const state: DecisionCardState = record.status === 'expired' ? 'expired' : outcome === 'allow' ? 'allowed' : outcome === 'deny' ? 'denied' : outcome === 'cancel' ? 'canceled' : 'resolved'
  const labels = locale === 'zh'
    ? { expired: '请求已失效', allowed: '已允许', denied: '已拒绝', canceled: '已取消', resolved: '已回复', pending: '等待回复', submitted: '回复已提交' }
    : { expired: 'Request expired', allowed: 'Allowed', denied: 'Declined', canceled: 'Canceled', resolved: 'Response completed', pending: 'Awaiting response', submitted: 'Response submitted' }
  const actor = record.decision?.actor?.label || record.decision?.actor?.id
  const choice = record.request.options.find((option) => option.id === record.decision?.optionId)?.label
  const status = [labels[state], actor, record.status === 'expired' && choice ? `${locale === 'zh' ? '此前回复' : 'Earlier response'}: ${choice}` : ''].filter(Boolean).join(' · ')
  const identity = `${sessionId}:${record.request.id}`
  return <DecisionCard identity={identity} title={approval.title} timestamp={record.updatedAt} state={state} status={status}
    data-xgc-role="agent-decision-history" data-xgc-id={identity}
    detailsLabel={locale === 'zh' ? '详情' : 'Details'}
    details={approval.detail ? <pre className="xgc-decision-card-review" data-xgc-role="agent-approval-review" data-xgc-id={identity}>{approval.detail}</pre> : undefined}>
    <StaleApprovalNotice identity={identity} open={false} missing={record.status === 'expired' && !record.decision}
      locale={locale} onRefresh={onRefresh} />
    {approval.detail ? <p className="m-0 truncate font-mono text-xs" title={approval.detail.split('\n')[0]}>{approval.detail.split('\n')[0]}</p> : null}
  </DecisionCard>
}

/** Merge dated domain facts without reordering the native journal or inventing old timestamps. */
export function mergeTimelineItems(nativeItems: readonly TimelineItem[], additionalItems: readonly TimelineItem[]): TimelineItem[] {
  const result: TimelineItem[] = []
  const dated = additionalItems.map((item, index) => ({ item, index, at: itemTime(item) })).sort((a, b) => {
    if (a.at === undefined || b.at === undefined) return a.at === b.at ? a.index - b.index : a.at === undefined ? 1 : -1
    return a.at - b.at || a.index - b.index
  })
  let cursor = 0
  for (const item of nativeItems) {
    const at = itemTime(item)
    while (at !== undefined && cursor < dated.length && dated[cursor]!.at !== undefined && dated[cursor]!.at! <= at) {
      result.push(dated[cursor++]!.item)
    }
    result.push(item)
  }
  return [...result, ...dated.slice(cursor).map(({ item }) => item)]
}
function itemTime(item: TimelineItem): number | undefined {
  if (!('createdAt' in item) || !item.createdAt) return undefined
  const at = Date.parse(item.createdAt)
  return Number.isFinite(at) ? at : undefined
}
