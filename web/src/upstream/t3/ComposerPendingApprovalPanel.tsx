// T3 Code, MIT, Copyright (c) 2026 T3 Tools Inc.
// Upstream ref: bf3be75c400cf605dc0c80de9458458854c84131. See UPSTREAM.md for integration changes.
import { memo } from "react";
import { type PendingApproval } from "./types.js";
import { cn } from "./utils.js";

interface ComposerPendingApprovalPanelProps {
  approval: PendingApproval;
  className?: string;
}

export const ComposerPendingApprovalPanel = memo(function ComposerPendingApprovalPanel({
  approval,
  className,
}: ComposerPendingApprovalPanelProps) {
  const fallbackLabel = approval.title || (
    approval.requestKind === "mcp-elicitation"
      ? "App access approval"
      : approval.requestKind === "command"
        ? "Command approval"
        : approval.requestKind === "file-read"
          ? "File read approval"
          : approval.requestKind === "file-change" ? "File change approval" : "Approval request");
  const detailAriaLabel =
    approval.requestKind === "mcp-elicitation"
      ? "App access request"
      : approval.requestKind === "command"
        ? "Command"
        : approval.requestKind === "file-read"
          ? "File to read"
          : approval.requestKind === "file-change" ? "File change" : "Request details";

  return (
    <span
      aria-label={fallbackLabel}
      className={cn("flex min-w-0 flex-1 items-center gap-2", className)}
      role="group"
    >
      {approval.appName || approval.title ? (
        <span className="max-w-32 shrink truncate text-[11px] font-medium text-foreground">
          {approval.appName || approval.title}
        </span>
      ) : null}
      <code
        aria-label={detailAriaLabel}
        className="block max-h-20 min-w-0 flex-1 overflow-auto whitespace-pre font-mono text-[11px] text-foreground/85 [scrollbar-width:thin] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring/70 [&::-webkit-scrollbar]:h-1.5"
        data-approval-detail="complete"
        tabIndex={0}
      >
        {approval.detail || fallbackLabel}
      </code>

    </span>
  );
});
