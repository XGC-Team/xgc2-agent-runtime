import { useT3Identity } from './identity.js';
// T3 Code ComposerPrimaryActions send/interrupt branches, MIT, 2026 T3 Tools Inc.
// IDE worktree, artwork, plan implementation and provider selection branches excluded.
import { cn } from './utils.js';
import { Spinner } from './ui/spinner.js';
export function ComposerPrimaryActions({ isRunning, isSending, disabled, disabledReason, hasSendableContent, onInterrupt, interruptEnabled = true, queueEnabled = false }: {
  queueEnabled?: boolean; isRunning: boolean; isSending: boolean; disabled: boolean; disabledReason?: string; hasSendableContent: boolean; onInterrupt: () => void; interruptEnabled?: boolean;
}) {
  const identity = useT3Identity();
  if (isRunning && !interruptEnabled && !queueEnabled) return null;
  const stop = isRunning && interruptEnabled ? <button type="button" disabled={disabled} onClick={onInterrupt} aria-label="Stop generation"
    data-xgc-role="agent-interrupt" data-xgc-id={`${identity}:interrupt`}
    className={cn('flex shrink-0 cursor-pointer items-center justify-center rounded-full bg-destructive/90 text-white shadow-xs shadow-destructive/24 inset-shadow-[0_1px_--theme(--color-white/16%)] transition-all duration-150 hover:bg-destructive hover:scale-105 active:inset-shadow-[0_1px_--theme(--color-black/8%)] active:shadow-none', 'size-8 sm:h-8 sm:w-8')}
  ><svg width="12" height="12" viewBox="0 0 12 12" fill="currentColor" aria-hidden="true"><rect x="2" y="2" width="8" height="8" rx="1.5" /></svg></button> : null;
  if (isRunning && !queueEnabled) return stop;
  const send = <button type="submit" disabled={disabled || isSending || !hasSendableContent} aria-label={isRunning ? "Queue message" : "Send message"} aria-busy={isSending}
    aria-description={disabledReason || undefined} title={disabledReason || undefined}
    data-xgc-role="agent-send" data-xgc-id={`${identity}:send`}
    className="relative isolate flex h-9 w-9 shrink-0 items-center justify-center overflow-hidden rounded-full shadow-xs transition-all duration-150 enabled:cursor-pointer enabled:inset-shadow-[0_1px_--theme(--color-white/16%)] hover:scale-105 active:inset-shadow-[0_1px_--theme(--color-black/8%)] active:shadow-none disabled:pointer-events-none disabled:opacity-30 disabled:shadow-none disabled:hover:scale-100 sm:h-8 sm:w-8 bg-message-action text-message-action-foreground enabled:shadow-message-action/24 hover:bg-message-action-hover"
  >{isSending ? <Spinner className="size-3.5" aria-hidden="true" /> : <svg width="14" height="14" viewBox="0 0 14 14" fill="none" aria-hidden="true"><path d="M7 11.5V2.5M7 2.5L3 6.5M7 2.5L11 6.5" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" /></svg>}</button>;
  return <>{stop}{disabledReason ? <span title={disabledReason} className="inline-flex shrink-0">{send}</span> : send}</>;
}
