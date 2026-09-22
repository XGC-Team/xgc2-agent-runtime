import { useT3Identity } from './identity.js';
// T3 Code, MIT, Copyright (c) 2026 T3 Tools Inc.
// Extracted Timeline list, user-message disclosure, assistant row and PlainWorkEntryRow.
// Source ranges and intentionally excluded IDE integrations are recorded in UPSTREAM.md.
import { memo, useCallback, useLayoutEffect, useMemo, useRef, useState, type ReactNode } from 'react';
import { LegendList } from '@legendapp/list/react';
import { ChevronRightIcon, CircleAlertIcon, TerminalIcon } from 'lucide-react';
import { Button } from './ui/button.js';
import { ChatMarkdown } from './ChatMarkdown.js';
import { MessageCopyButton } from './MessageCopyButton.js';
import { cn } from './utils.js';
import type { TimelineItem, TimelineMessage, TimelineWork } from './types.js';
import { NativeToolImages } from '../../NativeToolImages.js';
import { NativeTimelineStateProvider, useNativeTimelineState, useTimelineDisclosure,
  isTimelinePinnedToBottom, type NativeTimelineState } from '../../timelineState.js';

const TIMELINE_MAINTAIN_VISIBLE = { data: true, size: true } as const;
// Streamed text lands a paragraph at a time. A smooth scroll to the end turns
// each landing into a short glide instead of a jump; thread switches and
// reduced-motion users keep the instant variant (t3code MessagesTimeline).
const reducedMotion = typeof window !== 'undefined' && typeof window.matchMedia === 'function' ? window.matchMedia('(prefers-reduced-motion: reduce)') : null;
const MAX_COLLAPSED_USER_MESSAGE_LINES = 8;
const MAX_COLLAPSED_USER_MESSAGE_LENGTH = 600;
const COLLAPSED_USER_MESSAGE_FADE_HEIGHT_REM = 1.75;
const COLLAPSED_USER_MESSAGE_FADE_MASK = `linear-gradient(to bottom, black calc(100% - ${COLLAPSED_USER_MESSAGE_FADE_HEIGHT_REM}rem), transparent)`;
function shouldCollapseUserMessage(text: string): boolean {
  if (text.trim().length === 0) return false;
  return text.length > MAX_COLLAPSED_USER_MESSAGE_LENGTH || text.split('\n').length > MAX_COLLAPSED_USER_MESSAGE_LINES;
}

const CollapsibleUserMessageBody = memo(function CollapsibleUserMessageBody({ text, messageId }: { text: string; messageId: string }) {
  const identity = useT3Identity();
  const [expanded, toggleExpanded] = useTimelineDisclosure(messageId);
  const hasVisibleBody = text.trim().length > 0;
  const canCollapse = hasVisibleBody && shouldCollapseUserMessage(text);
  const isCollapsed = canCollapse && !expanded;
  return <div>
    {hasVisibleBody ? <div className={cn('relative', isCollapsed && 'max-h-44 overflow-hidden')}
      data-user-message-body="true" data-user-message-collapsed={isCollapsed ? 'true' : 'false'}
      data-user-message-collapsible={canCollapse ? 'true' : 'false'} data-user-message-fade={isCollapsed ? 'true' : 'false'}
      style={isCollapsed ? { WebkitMaskImage: COLLAPSED_USER_MESSAGE_FADE_MASK, maskImage: COLLAPSED_USER_MESSAGE_FADE_MASK } : undefined}
    ><ChatMarkdown text={text} /></div> : null}
    {canCollapse ? <div className="mt-1.5 flex items-center gap-2 justify-end" data-user-message-footer="true">
      <Button type="button" size="xs" variant="ghost" aria-expanded={expanded} data-scroll-anchor-ignore data-xgc-role="native-agent-message-disclosure" data-xgc-id={`${identity}:${messageId}`}
        onClick={toggleExpanded}
        className="-ml-1 h-6 rounded-md px-1.5 text-secondary-label text-xs hover:bg-muted/55 hover:text-message-foreground"
      >{expanded ? 'Show less' : 'Show full message'}</Button>
    </div> : null}
  </div>;
});

function MessageTimestamp({ value }: { value?: string | null }) {
  if (!value || !Number.isFinite(Date.parse(value))) return null;
  return <time dateTime={value} title={value} className="text-muted-foreground text-xs tabular-nums">{new Date(value).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</time>;
}
function TruncationNotice({ item }: { item: TimelineItem }) {
  return item.displayTruncated ? <p role="note" className="mt-1 text-secondary-label text-xs">Display truncated; some received content was omitted or shortened.</p> : null;
}
function MessageRetry({ retry, identity }: { retry: NonNullable<TimelineMessage['retry']>; identity: string }) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  return <>
    <Button type="button" size="xs" variant="ghost" disabled={busy}
      data-xgc-role="native-agent-message-retry" data-xgc-id={identity}
      onClick={() => {
        if (busy) return;
        setBusy(true); setError('');
        void retry.run().catch(cause => setError(cause instanceof Error ? cause.message : String(cause))).finally(() => setBusy(false));
      }}>{retry.label}</Button>
    {error ? <span role="alert">{error}</span> : null}
  </>;
}
function UserTimelineRow({ message }: { message: TimelineMessage }) {
  return <div className="group flex flex-col items-end gap-1" data-xgc-optimistic={message.optimistic ? 'true' : undefined}
    aria-busy={message.optimistic && message.delivery?.state !== 'failed'}>
    <div className={cn('relative max-w-[80%] rounded-2xl bg-message p-3 text-message-foreground', message.optimistic && message.delivery?.state !== 'failed' && 'opacity-60')}>
      <CollapsibleUserMessageBody text={message.text} messageId={message.id} /><TruncationNotice item={message} />
    </div>
    <div className="flex min-h-6 w-full max-w-[80%] items-center justify-end pe-1 text-xs tabular-nums">
      <div className="flex shrink-0 items-center gap-2">{message.delivery ? <span title={message.delivery.detail} data-xgc-role="native-agent-message-delivery" data-xgc-id={message.id} data-state={message.delivery.state}>{message.delivery.label}</span> : null}
        {message.retry ? <MessageRetry retry={message.retry} identity={message.id} /> : null}
        <MessageTimestamp value={message.createdAt} />{message.text ? <MessageCopyButton text={message.text} identityId={message.id} /> : null}</div>
    </div>
  </div>;
}
function AssistantTimelineRow({ message }: { message: TimelineMessage }) {
  return <div className="group/assistant relative min-w-0 px-1 py-0.5">
    <ChatMarkdown text={message.text || (message.streaming ? '' : '(empty response)')} isStreaming={message.streaming === true} />
    <TruncationNotice item={message} />
    <div className="mt-1 flex items-center gap-2 text-xs tabular-nums">
      {message.text ? <MessageCopyButton text={message.text} identityId={message.id} /> : null}
      {!message.streaming ? <MessageTimestamp value={message.updatedAt ?? message.createdAt} /> : null}
    </div>
  </div>;
}

/** Uses structured fields provided by the native adapter; never parses flattened text. */
function expandedToolBody(entry: TimelineWork): string {
  const data = entry.toolData;
  const lines: string[] = [];
  if (data?.command || entry.command) lines.push(data?.command ?? entry.command!);
  if (data?.cwd) lines.push(`Working directory: ${data.cwd}`);
  if (data?.exitCode != null) lines.push(`Exit code: ${data.exitCode}`);
  if (data?.durationMs != null) lines.push(`Duration: ${data.durationMs} ms`);
  if (entry.detail) lines.push(entry.detail);
  if (data?.changes?.length) for (const change of data.changes) {
    const kind = typeof change.kind === 'string' ? change.kind : change.kind?.type;
    const movePath = change.movePath ?? (typeof change.kind === 'object' ? change.kind.movePath : undefined);
    lines.push([kind, change.path, movePath ? `→ ${movePath}` : undefined].filter(Boolean).join(' '));
    if (change.diff) lines.push(change.diff);
  }
  if (data?.arguments !== undefined) lines.push(`Arguments\n${JSON.stringify(data.arguments, null, 2)}`);
  if (data?.result !== undefined) lines.push(`Result\n${typeof data.result === 'string' ? data.result : JSON.stringify(data.result, null, 2)}`);
  return lines.join('\n\n');
}
const PlainWorkEntryRow = memo(function PlainWorkEntryRow({ workEntry }: { workEntry: TimelineWork }) {
  const identity = useT3Identity();
  const [expanded, toggleExpanded] = useTimelineDisclosure(workEntry.id);
  const body = expandedToolBody(workEntry);
  const canExpand = Boolean(body);
  const failed = workEntry.tone === 'error' || workEntry.status === 'failed';
  const warning = workEntry.tone === 'warning';
  const stopRowToggle = (event: { stopPropagation: () => void }) => event.stopPropagation();
  return <div className={cn('flex flex-col rounded-md px-0.5 transition-colors py-0.5', canExpand && 'cursor-pointer hover:bg-accent/20 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring/70')}
    {...(canExpand ? { role: 'button', tabIndex: 0, 'data-xgc-role': 'native-agent-tool-disclosure', 'data-xgc-id': `${identity}:${workEntry.id}`, 'aria-label': workEntry.title, 'aria-expanded': expanded,
      onClick: toggleExpanded, onKeyDown: (event: React.KeyboardEvent<HTMLDivElement>) => { if (event.key === 'Enter' || event.key === ' ') { event.preventDefault(); toggleExpanded(); } } } : {})}
  >
    <div className="flex select-none items-center gap-1.5 transition-[opacity,translate] duration-200">
      <span className={cn('flex size-6 shrink-0 items-center justify-center', warning ? 'text-warning' : failed ? 'text-destructive' : 'text-icon-muted')}>
        {failed || warning ? <CircleAlertIcon className="block size-4 shrink-0 stroke-[1.8]" /> : <TerminalIcon className="block size-4 shrink-0 stroke-[1.8]" />}
      </span>
      <div className="flex min-w-0 flex-1 items-center gap-1.5"><div className="min-w-0 flex-1 overflow-hidden">
        <p className="flex min-w-0 w-full items-baseline gap-1.5 text-sm leading-relaxed">
          <span className={cn('min-w-0 flex-1', expanded ? 'whitespace-pre-wrap break-words select-text' : 'truncate', warning ? 'font-medium text-warning' : failed ? 'font-medium text-destructive' : 'text-secondary-label')}
            onClick={expanded ? stopRowToggle : undefined} onPointerDown={expanded ? stopRowToggle : undefined}
          >{workEntry.title}</span>
          {workEntry.status ? <span className="shrink-0 text-secondary-label text-xs">{workEntry.status}</span> : null}
        </p>
      </div><span className={cn('flex size-4 shrink-0 items-center justify-center', !canExpand && 'invisible')} aria-hidden>
        <ChevronRightIcon className={cn('size-3 shrink-0 text-icon-muted opacity-70 transition-transform duration-200', expanded && 'rotate-90')} />
      </span></div>
    </div>
    {expanded && body ? <div className="mt-1 ms-7 cursor-default rounded-md bg-muted/40 px-3 py-2" onClick={stopRowToggle} onPointerDown={stopRowToggle}>
      <pre className="max-h-80 overflow-auto whitespace-pre-wrap break-words font-mono text-xs text-foreground/80">{body}</pre>
    </div> : null}
    <TruncationNotice item={workEntry} />
    {workEntry.toolData?.images?.length ? <NativeToolImages images={workEntry.toolData.images} identity={`${identity}:${workEntry.id}`} /> : null}
  </div>;
});

const TimelineRow = memo(function TimelineRow({ item, identity }: { item: TimelineItem; identity: string }) {
  return <div className="mx-auto w-full min-w-0 max-w-3xl overflow-x-clip py-2" data-timeline-root="true" data-xgc-role="native-agent-item" data-xgc-id={`${identity}:${item.id}`}>
    {item.kind === 'custom' ? item.content : item.kind === 'message' ? item.role === 'user' ? <UserTimelineRow message={item} /> : <AssistantTimelineRow message={item} /> : item.kind === 'work' ? <section className="-mx-1 space-y-0.5 px-1 py-0.5" aria-label="Activity"><PlainWorkEntryRow workEntry={item} /></section> : <div className="relative min-w-0 px-1 py-0.5"><ChatMarkdown text={item.text} /><TruncationNotice item={item} /></div>}
  </div>;
}, (previous, next) => {
  if (previous.identity !== next.identity) return false;
  const a = previous.item, b = next.item;
  if (a === b) return true;
  // Stream projection can recreate historical message wrappers. Avoid parsing
  // their unchanged markdown again for every token in the active assistant row.
  return a.kind === 'message' && b.kind === 'message' && a.id === b.id && a.role === b.role
    && a.text === b.text && a.createdAt === b.createdAt && a.updatedAt === b.updatedAt
    && a.streaming === b.streaming && a.displayTruncated === b.displayTruncated
    && a.optimistic === b.optimistic && a.delivery === b.delivery && a.retry === b.retry;
});
type MessagesTimelineProps = { items: readonly TimelineItem[]; active: boolean; emptyState?: ReactNode;
  timelineState?: NativeTimelineState; onTimelineStateChange?: (state: NativeTimelineState) => void };
export const MessagesTimeline = memo(function MessagesTimeline({ timelineState, onTimelineStateChange, ...props }: MessagesTimelineProps) {
  return <NativeTimelineStateProvider state={timelineState} onChange={onTimelineStateChange}><VirtualTimeline {...props} /></NativeTimelineStateProvider>;
});
function VirtualTimeline({ items, active, emptyState }: Omit<MessagesTimelineProps, 'timelineState' | 'onTimelineStateChange'>) {
  const identity = useT3Identity();
  const { state, change } = useNativeTimelineState();
  const wrapper = useRef<HTMLDivElement | null>(null);
  const latestFollow = useRef({ active, pinnedToBottom: state.pinnedToBottom, streaming: false });
  const renderItem = useCallback(({ item }: { item: TimelineItem }) => <TimelineRow item={item} identity={identity} />, [identity]);
  const alwaysRender = useMemo(() => ({ keys: items.filter(item => item.kind === 'custom' && item.keepMounted).map(item => item.id) }), [items]);
  const streaming = items.some((item) => item.kind === 'message' && item.streaming);
  latestFollow.current = { active, pinnedToBottom: state.pinnedToBottom, streaming };
  // Bottom following is owned here, not delegated to the virtualizer: a
  // bottom-aligned scroll target kept alive inside LegendList clamps every
  // later scroll-away back to the end whenever estimated and measured row
  // heights differ, so an operator can never scroll up during streaming.
  // Follow the end only while pinned; content growth (append, stream tokens,
  // row measurement settle) re-scrolls, a scrolled-up viewport stays put.
  const follow = useCallback(() => {
    const node = wrapper.current?.firstElementChild;
    if (!(node instanceof HTMLElement)) return;
    const { active: followActive, pinnedToBottom, streaming: smooth } = latestFollow.current;
    if (!followActive || !pinnedToBottom) return;
    const end = node.scrollHeight - node.clientHeight;
    if (Math.abs(node.scrollTop - end) < 1) return;
    if (smooth && reducedMotion?.matches !== true) node.scrollTo({ top: end, behavior: 'smooth' });
    else node.scrollTop = end;
  }, []);
  useLayoutEffect(follow, [follow, items, active]);
  useLayoutEffect(() => {
    const node = wrapper.current?.firstElementChild;
    if (!(node instanceof HTMLElement) || typeof ResizeObserver === 'undefined') return;
    const content = node.firstElementChild;
    if (!(content instanceof HTMLElement)) return;
    const observer = new ResizeObserver(follow);
    observer.observe(content);
    return () => observer.disconnect();
  }, [follow]);
  if (!items.length) return <div className="flex h-full items-center justify-center">{emptyState}</div>;
  return <div ref={wrapper} className="relative h-full min-h-0" onScrollCapture={event => {
    const node = event.target;
    // Nested tool/details scrollports are not the conversation scrollport.
    if (!active || !(node instanceof HTMLElement) || node.clientHeight <= 0 || node.closest('[data-timeline-root]')) return;
    const pinnedToBottom = isTimelinePinnedToBottom(node);
    if (pinnedToBottom !== state.pinnedToBottom) change({ ...state, pinnedToBottom });
  }}>
    <LegendList<TimelineItem> data={items} keyExtractor={timelineItemKey} getItemType={timelineItemKind} renderItem={renderItem}
      estimatedItemSize={90} alwaysRender={alwaysRender}
      maintainVisibleContentPosition={TIMELINE_MAINTAIN_VISIBLE}
      className="scrollbar-gutter-both h-full min-h-0 overflow-x-hidden overscroll-y-contain px-3 [overflow-anchor:none] sm:px-5"
      ListHeaderComponent={TimelineSpacer} ListFooterComponent={TimelineSpacer}
    />
  </div>;
}
function timelineItemKey(item: TimelineItem) { return item.id; }
function timelineItemKind(item: TimelineItem) { return item.kind === 'message' ? `${item.kind}:${item.role}` : item.kind; }
function TimelineSpacer() { return <div className="h-3 sm:h-4" />; }
