// Extracted from T3 Code MessageCopyButton.tsx; host toast/shell dependencies removed.
// MIT, Copyright (c) 2026 T3 Tools Inc.
import { memo, useState } from 'react';
import { useT3Identity } from './identity.js';
import { CopyIcon, CheckIcon } from 'lucide-react';
import { Button } from './ui/button.js';
import { Tooltip, TooltipPopup, TooltipTrigger } from './ui/tooltip.js';
export const MessageCopyButton = memo(function MessageCopyButton({ text, identityId }: { text: string; identityId: string }) {
  const identity = useT3Identity();
  const [copied, setCopied] = useState(false);
  const [error, setError] = useState('');
  const copy = async () => {
    try { await navigator.clipboard.writeText(text); setCopied(true); setError(''); }
    catch (cause) { setError(cause instanceof Error ? cause.message : 'Unable to copy'); }
  };
  return <Tooltip>
    <TooltipTrigger render={<Button data-xgc-role="native-agent-message-copy" data-xgc-id={`${identity}:${identityId}`} aria-label={error || (copied ? 'Copied' : 'Copy message')} onClick={() => void copy()} type="button" size="xs" variant="ghost" className="text-muted-foreground hover:text-foreground" />}>
      {copied ? <CheckIcon className="size-3 text-primary" /> : <CopyIcon className="size-3" />}
    </TooltipTrigger>
    <TooltipPopup><p>{error || (copied ? 'Copied' : 'Copy to clipboard')}</p></TooltipPopup>
  </Tooltip>;
});
