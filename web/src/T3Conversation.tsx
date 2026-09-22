import { useCallback, useRef, useState, type ReactNode } from 'react';
import { T3IdentityScope, useT3Identity } from './upstream/t3/identity.js';
import { ComposerSurface } from './upstream/t3/ComposerSurface.js';
import { ComposerBanner } from './upstream/t3/ComposerBanner.js';
import { ComposerPrimaryActions } from './upstream/t3/ComposerPrimaryActions.js';
import { ComposerPromptEditor } from './upstream/t3/ComposerPromptEditor.js';
import { DecisionCard } from './DecisionCard.js';
import { StaleApprovalNotice } from './StaleApprovalNotice.js';
import type { AgentTimelineState } from './timelineState.js';
import { ComposerPendingApprovalActions } from './upstream/t3/ComposerPendingApprovalActions.js';
import { ComposerPendingUserInputPanel } from './upstream/t3/ComposerPendingUserInputPanel.js';
import { MessagesTimeline } from './upstream/t3/MessagesTimeline.js';
import { Button } from './upstream/t3/ui/button.js';
import { T3PortalContainer, TooltipProvider } from './upstream/t3/ui/tooltip.js';
import { buildPendingUserInputAnswers, derivePendingUserInputProgress, setPendingUserInputCustomAnswer, togglePendingUserInputOptionSelection, type PendingUserInputDraftAnswer } from './upstream/t3/pendingUserInput.js';
import type { PendingUserInput, T3ConversationModel } from './upstream/t3/types.js';
export type { T3ConversationModel, TimelineItem, PendingApproval, PendingUserInput, ToolData } from './upstream/t3/types.js';

export interface T3RequestCallbacks {
  onApproval: (requestId: string, decision: string) => Promise<unknown>;
  onUserInput: (requestId: string, answers: Record<string, string[]>) => Promise<unknown>;
  onCancelRequest: (requestId: string) => Promise<unknown>;
}
export interface T3PendingRequestsProps extends T3RequestCallbacks {
  model: Pick<T3ConversationModel, 'sessionKey' | 'approvals' | 'userInputs'>;
  active: boolean;
  disabled?: boolean;
  locale?: 'en' | 'zh';
  renderApprovalControls?: (requestId:string) => ReactNode;
  onRefreshRequests?: () => Promise<unknown>;
  staleRequestIds?: readonly string[];
}
export interface T3ConversationProps extends T3PendingRequestsProps {
  model: T3ConversationModel;
  sendDisabled?: boolean;
  sendDisabledReason?: string;
  queueEnabled?: boolean;
  error?: string;
  composerEnabled?: boolean;
  composerControls?: ReactNode;
  dock?: ReactNode;
  additionalPendingRequests?: ReactNode;
  emptyState?: ReactNode;
  interruptEnabled?: boolean;
  draft?: string;
  onDraftChange?: (draft: string) => void;
  clearDraftOnSend?: boolean;
  timelineState?: AgentTimelineState;
  onTimelineStateChange?: (state: AgentTimelineState) => void;
  onSend: (text: string) => Promise<unknown>;
  onInterrupt: () => Promise<unknown>;
}

function ErrorBanner({ error }: { error: string }) {
  return error ? <ComposerBanner.Root variant="error" placement="floating"><ComposerBanner.Row><ComposerBanner.Content><span role="alert">{error}</span></ComposerBanner.Content></ComposerBanner.Row></ComposerBanner.Root> : null;
}
function messageOf(error: unknown) { return error instanceof Error ? error.message : String(error); }

function UserInputCard({ request, active, disabled, onUserInput, onCancelRequest }: {
  request: PendingUserInput; active: boolean; disabled: boolean;
  onUserInput: T3RequestCallbacks['onUserInput']; onCancelRequest: T3RequestCallbacks['onCancelRequest'];
}) {
  const identity = useT3Identity();
  const [answers, setAnswers] = useState<Record<string, PendingUserInputDraftAnswer>>({});
  const [questionIndex, setQuestionIndex] = useState(0);
  const [busy, setBusy] = useState(false);
  const [acknowledged, setAcknowledged] = useState(false);
  const [error, setError] = useState('');
  const activeRef = useRef(active); activeRef.current = active;
  const inFlight = useRef(false);
  const progress = derivePendingUserInputProgress(request.questions, answers, questionIndex);
  const locked = disabled || !active || busy || acknowledged || Boolean(request.submitted);
  const submit = async (cancel = false) => {
    if (locked || inFlight.current || !activeRef.current) return;
    const resolved = buildPendingUserInputAnswers(request.questions, answers);
    if (!cancel && !resolved) return;
    inFlight.current = true; setBusy(true); setError('');
    try {
      if (cancel) await onCancelRequest(request.requestId);
      else await onUserInput(request.requestId, Object.fromEntries(Object.entries(resolved!).map(([id, answer]) => [id, Array.isArray(answer) ? answer : [answer]])));
      setAcknowledged(true);
    } catch (cause) { setError(messageOf(cause)); }
    finally { inFlight.current = false; setBusy(false); }
  };
  const advance = () => {
    if (locked || !activeRef.current || !progress.canAdvance) return;
    // Single-option selection auto-advances between questions, never submits a
    // final answer implicitly. The operator keeps an explicit final action.
    if (!progress.isLastQuestion) setQuestionIndex(value => value + 1);
  };
  return <ComposerBanner.Root placement="floating" aria-busy={busy || acknowledged || request.submitted}>
    {request.title ? <ComposerBanner.Row><ComposerBanner.Content>{request.title}</ComposerBanner.Content></ComposerBanner.Row> : null}
    <ComposerPendingUserInputPanel pendingUserInputs={[request]} active={active && !locked}
      respondingRequestIds={locked ? [request.requestId] : []} answers={answers} questionIndex={questionIndex}
      onToggleOption={(id, value) => { const question = request.questions.find(item => item.id === id); if (question && !locked) setAnswers(current => ({ ...current, [id]: togglePendingUserInputOptionSelection(question, current[id], value) })); }}
      onAdvance={advance}
    />
    {progress.activeQuestion?.allowCustomAnswer !== false ? <ComposerBanner.Body>
      {progress.activeQuestion?.isSecret ? <input type="password" autoComplete="off" disabled={locked} aria-label="Your answer"
        data-xgc-role="agent-secret-answer" data-xgc-id={`${identity}:${request.requestId}:${progress.activeQuestion.id}`}
        className="w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm text-foreground"
        value={progress.customAnswer} onChange={event => { const id = progress.activeQuestion?.id; if (id) setAnswers(current => ({ ...current, [id]: setPendingUserInputCustomAnswer(current[id], event.target.value) })); }}
      /> : <ComposerPromptEditor identityId={`${request.requestId}:${progress.activeQuestion?.id ?? "answer"}`} value={progress.customAnswer} disabled={locked} placeholder="Your answer"
        onChange={value => { const id = progress.activeQuestion?.id; if (id) setAnswers(current => ({ ...current, [id]: setPendingUserInputCustomAnswer(current[id], value) })); }}
      />}
    </ComposerBanner.Body> : null}
    <ComposerBanner.Row><ComposerBanner.Actions>
      <Button size="sm" variant="ghost" disabled={locked} data-xgc-role="agent-request-cancel" data-xgc-id={`${identity}:${request.requestId}`} onClick={() => void submit(true)}>Cancel request</Button>
      {questionIndex > 0 ? <Button size="sm" variant="outline" className="rounded-full" disabled={locked} data-xgc-role="agent-question-previous" data-xgc-id={`${identity}:${request.requestId}`} onClick={() => setQuestionIndex(value => value - 1)}>Previous</Button> : null}
      <Button type="button" size="sm" className="rounded-full bg-message-action text-message-action-foreground hover:bg-message-action-hover px-4"
        disabled={locked || (progress.isLastQuestion ? !progress.isComplete : !progress.canAdvance)}
        data-xgc-role="agent-question-submit" data-xgc-id={`${identity}:${request.requestId}`}
        onClick={() => { if (progress.isLastQuestion) void submit(); else advance(); }}
      >{progress.isLastQuestion ? 'Submit answer' : 'Next question'}</Button>
    </ComposerBanner.Actions></ComposerBanner.Row>
    <ErrorBanner error={error} />
  </ComposerBanner.Root>;
}

function PendingRequestsContent({ model, active, disabled = false, locale = 'en', renderApprovalControls, onRefreshRequests, staleRequestIds, onApproval, onUserInput, onCancelRequest }: T3PendingRequestsProps) {
  const [busy, setBusy] = useState<Set<string>>(() => new Set());
  const [submitted, setSubmitted] = useState<Set<string>>(() => new Set());
  const [error, setError] = useState('');
  const inFlight = useRef(new Set<string>());
  const respond = async (id: string, decision?: string) => {
    if (!active || disabled || staleRequestIds?.includes(id) || inFlight.current.has(id) || submitted.has(id)) return;
    inFlight.current.add(id); setBusy(new Set(inFlight.current)); setError('');
    try {
      if (decision === undefined) await onCancelRequest(id); else await onApproval(id, decision);
      setSubmitted(current => new Set([...current, id]));
    } catch (cause) { setError(messageOf(cause)); }
    finally { inFlight.current.delete(id); setBusy(new Set(inFlight.current)); }
  };
  return <div className="space-y-2">
    {model.approvals.map(approval => {
      const missing = Boolean(staleRequestIds?.includes(approval.requestId));
      const locked = missing || !active || disabled || busy.has(approval.requestId) || submitted.has(approval.requestId) || Boolean(approval.submitted);
      const awaitingProvider = submitted.has(approval.requestId) || Boolean(approval.submitted);
      return <DecisionCard key={approval.requestId} identity={`${model.sessionKey}:${approval.requestId}`}
        title={approval.title || 'Approval request'} timestamp={approval.createdAt ?? undefined}
        state={awaitingProvider || busy.has(approval.requestId) ? 'submitted' : 'pending'}
        status={awaitingProvider ? locale === 'zh' ? '回复已提交' : 'Response submitted' : busy.has(approval.requestId) ? locale === 'zh' ? '提交回复中' : 'Submitting response' : undefined}
        actions={<>
          <ComposerPendingApprovalActions requestId={approval.requestId} options={approval.options} isResponding={locked} onRespondToApproval={respond} />
          <Button size="sm" variant="ghost" disabled={locked} data-xgc-role="agent-request-cancel" data-xgc-id={`${model.sessionKey}:${approval.requestId}`} onClick={() => void respond(approval.requestId)}>{locale === 'zh' ? '取消' : 'Cancel request'}</Button>
        </>}
        detailsLabel={locale === 'zh' ? '详情与授权选项' : 'Details and approval options'}
        details={<>
        <pre className="xgc-decision-card-review" role="group" aria-label={approval.title || 'Approval request'}
          data-xgc-role="agent-approval-review" data-xgc-id={`${model.sessionKey}:${approval.requestId}`}
          data-approval-detail={approval.truncated ? 'partial' : 'complete'} tabIndex={0}>{approval.detail || approval.title || 'Approval request'}</pre>
          {!locked ? renderApprovalControls?.(approval.requestId) : null}
        </>}>
        <StaleApprovalNotice identity={`${model.sessionKey}:${approval.requestId}`} open={!awaitingProvider}
          createdAt={approval.createdAt} missing={missing} locale={locale} onRefresh={onRefreshRequests} />
        {approval.detail ? <p className="m-0 truncate font-mono text-xs" title={approval.detail.split('\n')[0]}>{approval.detail.split('\n')[0]}</p> : null}
      </DecisionCard>;
    })}
    {model.userInputs.map(request => <UserInputCard key={request.requestId} request={request} active={active} disabled={disabled} onUserInput={onUserInput} onCancelRequest={onCancelRequest} />)}
    <ErrorBanner error={error} />
  </div>;
}

/** Same upstream request surface in conversation and global pending-request entry. */
export function T3PendingRequests(props: T3PendingRequestsProps) {
  const [portal, setPortal] = useState<HTMLDivElement | null>(null);
  return <div className="xgc-agent-chat" ref={setPortal} data-xgc-role="agent-pending-requests" data-xgc-id={props.model.sessionKey}>
    <T3IdentityScope value={props.model.sessionKey}><T3PortalContainer value={portal}><TooltipProvider><PendingRequestsContent key={props.model.sessionKey} {...props} /></TooltipProvider></T3PortalContainer></T3IdentityScope>
  </div>;
}

function ConversationContent({ model, queueEnabled = false, active, disabled = false, sendDisabled = false, sendDisabledReason, error:externalError, composerEnabled = true, composerControls, dock, additionalPendingRequests, emptyState, interruptEnabled = true, draft, onDraftChange, clearDraftOnSend = true, timelineState, onTimelineStateChange, onSend, onInterrupt, ...callbacks }: T3ConversationProps) {
  const [internalPrompt, setInternalPrompt] = useState('');
  const prompt = draft ?? internalPrompt;
  const setPrompt = useCallback((value: string) => {
    if (draft === undefined) setInternalPrompt(value);
    onDraftChange?.(value);
  }, [draft, onDraftChange]);
  const promptRef = useRef(prompt); promptRef.current = prompt;
  const setPromptRef = useRef(setPrompt); setPromptRef.current = setPrompt;
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const inFlight = useRef(false);
  const locked = !active || disabled;
  const send = useCallback(async () => {
    if (!composerEnabled || locked || sendDisabled || (!queueEnabled && (model.isRunning || model.isSending)) || inFlight.current || !prompt.trim()) return;
    const sentPrompt = prompt;
    inFlight.current = true; setBusy(true); setError('');
    try {
      await onSend(sentPrompt);
      if (clearDraftOnSend && promptRef.current === sentPrompt) setPromptRef.current('');
    }
    catch (cause) { setError(messageOf(cause)); }
    finally { inFlight.current = false; setBusy(false); }
  }, [composerEnabled, queueEnabled, locked, sendDisabled, model.isRunning, model.isSending, onSend, prompt, clearDraftOnSend]);
  const interrupt = async () => {
    if (!interruptEnabled || locked || inFlight.current) return;
    inFlight.current = true; setBusy(true); setError('');
    try { await onInterrupt(); } catch (cause) { setError(messageOf(cause)); }
    finally { inFlight.current = false; setBusy(false); }
  };
  return <div className="flex h-full min-h-0 flex-col">
    <div className="min-h-0 flex-1"><MessagesTimeline items={model.items} active={active} emptyState={emptyState}
      timelineState={timelineState} onTimelineStateChange={onTimelineStateChange} /></div>
    {model.approvals.length || model.userInputs.length || additionalPendingRequests ? <div className="min-h-0 max-h-[45%] shrink overflow-y-auto px-3 pb-2 sm:px-5"
      data-xgc-role="agent-request-queue" data-xgc-id={model.sessionKey}>
      <PendingRequestsContent model={model} active={active} disabled={disabled} {...callbacks} />
      {additionalPendingRequests}
    </div> : null}
    {(composerEnabled || (model.isRunning && interruptEnabled) || dock) ? <div className="flex shrink-0 flex-col gap-2 px-3 pb-3 sm:px-5">
      {dock}
      {(composerEnabled || (model.isRunning && interruptEnabled)) ? <>
      <ErrorBanner error={externalError || error} />
      <ComposerSurface.Shell><ComposerSurface.Host><ComposerSurface.Main>
        <form onSubmit={event => { event.preventDefault(); void send(); }} data-chat-composer-surface="true"
          data-xgc-role="agent-composer" data-xgc-id={model.sessionKey}
          className="rounded-[20px] transition-[background-color] duration-200">
          {composerEnabled ? <div data-chat-composer-body="true" className="relative px-3 pt-3.5 sm:px-4 sm:pt-4">
            <ComposerPromptEditor value={prompt} onChange={setPrompt} disabled={locked} placeholder="Ask anything..."
              onCommandKeyDown={(key, event) => { if (key !== 'Enter' || event.shiftKey) return false; if (event.isComposing || event.keyCode === 229) return true; void send(); return true; }}
            />
          </div> : null}
          <div className="flex items-center justify-end gap-2 px-3 pb-3 pt-2 sm:px-4">{composerEnabled && composerControls ? <div className="mr-auto min-w-0 flex-1">{composerControls}</div> : null}<ComposerPrimaryActions queueEnabled={queueEnabled} isRunning={model.isRunning} interruptEnabled={interruptEnabled} isSending={busy || Boolean(model.isSending)} disabled={locked || busy || (!model.isRunning && sendDisabled)} disabledReason={sendDisabledReason} hasSendableContent={Boolean(prompt.trim())} onInterrupt={() => void interrupt()} /></div>
        </form>
      </ComposerSurface.Main></ComposerSurface.Host></ComposerSurface.Shell>
      </> : null}
    </div> : null}
  </div>;
}

export function T3Conversation(props: T3ConversationProps) {
  const [portal, setPortal] = useState<HTMLDivElement | null>(null);
  return <div className="xgc-agent-chat relative min-h-0" data-xgc-role="agent-conversation" data-xgc-id={props.model.sessionKey}>
    <T3IdentityScope value={props.model.sessionKey}><T3PortalContainer value={portal}><TooltipProvider><ConversationContent key={props.model.sessionKey} {...props} /></TooltipProvider></T3PortalContainer></T3IdentityScope>
    <div ref={setPortal} className="pointer-events-none absolute inset-0 z-[130] overflow-visible [&>*]:pointer-events-auto" />
  </div>;
}
