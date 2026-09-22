import type { ReactNode } from "react";
/** Presentation contract for the pinned T3 Code migration. No transport or runtime owner. */
export type ProviderApprovalDecision = string;
export interface ProviderApprovalOption { decision: ProviderApprovalDecision; label: string; warning?: string }
export interface PendingApproval {
  requestId: string;
  requestKind: string;
  title?: string;
  submitted?: boolean;
  createdAt?: string | null;
  detail?: string;
  truncated?: boolean;
  appName?: string;
  options: readonly ProviderApprovalOption[];
}
export interface UserInputQuestion {
  id: string;
  header: string;
  question: string;
  options: readonly { id?: string; value?: string; label: string; description?: string }[];
  multiSelect?: boolean;
  allowCustomAnswer?: boolean;
  isSecret?: boolean;
}
export interface PendingUserInput { title?: string; submitted?: boolean; requestId: string; createdAt?: string | null; questions: readonly UserInputQuestion[] }
export interface ToolData {
  images?: readonly { src: string; width: number; height: number; label: string; observedAt?: string }[];
  command?: string;
  cwd?: string;
  exitCode?: number | null;
  durationMs?: number | null;
  changes?: readonly { path: string; kind?: string | { type: string; movePath?: string }; movePath?: string; diff?: string }[];
  arguments?: unknown;
  result?: unknown;
}
export interface TimelineMessage {
  kind: 'message'; id: string; displayTruncated?: boolean; role: 'user' | 'assistant'; text: string;
  createdAt?: string | null; updatedAt?: string | null; streaming?: boolean;
  delivery?: {state:'sending'|'sent'|'failed';label:string;detail?:string};
  optimistic?: boolean;
  retry?: { label: string; run: () => Promise<unknown> };
}
export interface TimelineWork {
  kind: 'work'; id: string; displayTruncated?: boolean; title: string; detail?: string; command?: string;
  status?: string; tone?: 'tool' | 'info' | 'error' | 'warning'; toolData?: ToolData;
}
export interface TimelinePlan { kind: 'plan'; id: string; displayTruncated?: boolean; title?: string; text: string }
export interface TimelineCustom {
  kind: 'custom'; id: string; content: ReactNode; createdAt?: string | null; displayTruncated?: boolean;
  /** Interactive portal owners must not unmount merely because the row is offscreen. */
  keepMounted?: boolean;
}
export type TimelineItem = TimelineMessage | TimelineWork | TimelinePlan | TimelineCustom;
export interface T3ConversationModel {
  sessionKey: string;
  items: readonly TimelineItem[];
  approvals: readonly PendingApproval[];
  userInputs: readonly PendingUserInput[];
  isRunning: boolean;
  isSending?: boolean;
}
