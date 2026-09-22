import { useT3Identity } from './identity.js';
// T3 Code, MIT, Copyright (c) 2026 T3 Tools Inc.
// Upstream ref: bf3be75c400cf605dc0c80de9458458854c84131. See UPSTREAM.md for integration changes.
import {

  type ProviderApprovalDecision,
  type ProviderApprovalOption,
} from "./types.js";
import { memo } from "react";
import { TriangleAlertIcon } from "lucide-react";
import { Button } from "./ui/button.js";
import { Tooltip, TooltipPopup, TooltipTrigger } from "./ui/tooltip.js";

interface ComposerPendingApprovalActionsProps {
  requestId: string;
  isResponding: boolean;
  options: ReadonlyArray<ProviderApprovalOption>;
  onRespondToApproval: (
    requestId: string,
    decision: ProviderApprovalDecision,
  ) => Promise<unknown>;
}

const APPROVAL_ACTION_CLASS_NAME = "font-normal";


export const ComposerPendingApprovalActions = memo(function ComposerPendingApprovalActions({
  requestId,
  isResponding,
  options,
  onRespondToApproval,
}: ComposerPendingApprovalActionsProps) {
  const identity = useT3Identity();
  return (
    <>
      {options.map((option) => {
        const button = (
          <Button
            key={option.decision}
            data-xgc-role="native-agent-approval-option"
            data-xgc-id={`${identity}:${requestId}:${option.decision}`}
            size="sm"
            variant="outline"
            className={APPROVAL_ACTION_CLASS_NAME}
            disabled={isResponding}
            aria-description={option.warning}
            onClick={() => void onRespondToApproval(requestId, option.decision)}
          >
            {option.warning ? <TriangleAlertIcon className="size-3 shrink-0" /> : null}
            <span className="max-w-40 truncate">{option.label}</span>
          </Button>
        );
        // A provider caution, such as a prompt injection warning on "allow
        // always", rides along as a tooltip so the row stays one line.
        return option.warning ? (
          <Tooltip key={option.decision}>
            <TooltipTrigger render={button} />
            <TooltipPopup side="top" className="max-w-72 text-xs leading-snug">
              {option.warning}
            </TooltipPopup>
          </Tooltip>
        ) : (
          button
        );
      })}
    </>
  );
});
