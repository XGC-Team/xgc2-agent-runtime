// T3 Code, MIT, Copyright (c) 2026 T3 Tools Inc.
// Upstream ref: bf3be75c400cf605dc0c80de9458458854c84131. See UPSTREAM.md.
// Adapted ProviderModelPicker / ModelPickerContent / ModelListRow / ModelPickerSidebar.
// XGC supplies a verified capability catalog; no environment, favorites or IDE store.
import { useRef, useState } from 'react';
import { SearchIcon } from 'lucide-react';
import { ComposerControl, ComposerControlChevron } from './ComposerControl.js';
import { Popover, PopoverPopup, PopoverTrigger } from './ui/popover.js';
import { Combobox, ComboboxInput, ComboboxItem, ComboboxList } from './ui/combobox.js';
import { Tooltip, TooltipPopup, TooltipTrigger } from './ui/tooltip.js';
import { ClaudeAI, CursorIcon, GrokIcon, OpenAI, OpenCodeIcon } from './ProviderIcons.js';
import { scoreModelPickerSearch } from './modelPickerSearch.js';
import type { AgentProvider } from '../../state.js';
import type { AgentProviderConfiguration } from '../../providerSettings.js';

export const providerLabels: Record<AgentProvider, string> = { codex: 'Codex', claude: 'Claude', cursor: 'Cursor', grok: 'Grok', opencode: 'OpenCode' };
export const providerIcons = { codex: OpenAI, claude: ClaudeAI, cursor: CursorIcon, grok: GrokIcon, opencode: OpenCodeIcon };
export function ProviderModelPicker({ providers, profileId, model, disabled = false, onChange, locale = 'en', identityId }: {
  identityId?: string; locale?: 'en' | 'zh'; providers: readonly AgentProviderConfiguration[]; profileId: string; model?: string; disabled?: boolean;
  onChange: (profileId: string, model: string) => void;
}) {
  const [open, setOpen] = useState(false);
  const entry = providers.find(item => item.id === profileId);
  const resolvedModel = model ?? entry?.defaults.model;
  const selectedModel = entry?.models.find(item => item.id === resolvedModel);
  const Icon = entry ? providerIcons[entry.provider] : null;
  const title = selectedModel?.label ?? resolvedModel ?? (locale === 'zh' ? '工作者 / 模型' : 'Provider / model');
  const available = providers.filter(item => item.enabled && item.available && item.models.length > 0);
  return <Popover open={open && !disabled} onOpenChange={value => setOpen(!disabled && value)}>
    <PopoverTrigger render={<ComposerControl type="button" aria-label={locale === 'zh' ? '工作者与模型' : 'Provider and model'} disabled={disabled}
      data-chat-provider-model-picker="true" data-xgc-role="agent-composer-model" data-xgc-id={identityId ? `${identityId}:${profileId || 'unselected'}` : profileId || 'unselected'}
      className="min-w-0 max-w-48 flex-1 shrink justify-between overflow-hidden whitespace-nowrap sm:max-w-56" />}>
      <span className="flex min-w-0 flex-1 items-center gap-1.5">
        {Icon ? <Icon className="size-4 shrink-0" aria-hidden="true" /> : null}
        <Tooltip><TooltipTrigger render={<span className="min-w-0 flex-1 overflow-hidden truncate" data-chat-provider-model-picker-label="true" />}>
          {title}
        </TooltipTrigger><TooltipPopup side="top">{entry ? `${providerLabels[entry.provider]} · ${title}` : title}</TooltipPopup></Tooltip>
        {resolvedModel && (!selectedModel || !entry?.available || !entry.enabled) ? <span className="text-xs text-muted-foreground">{locale === 'zh' ? '不可用' : 'Unavailable'}</span> : null}
      </span><span aria-hidden="true" className="flex items-center"><ComposerControlChevron /></span>
    </PopoverTrigger>
    <PopoverPopup align="start" side="top" className="before:hidden [--viewport-inline-padding:0]"
      viewportClassName="overflow-hidden! rounded-[calc(var(--radius-lg)-1px)] p-0 [clip-path:inset(0_round_calc(var(--radius-lg)-1px))]">
      <ModelPickerContent identityId={identityId} locale={locale} key={open ? 'open' : 'closed'} providers={available} profileId={profileId} model={resolvedModel}
        onClose={() => setOpen(false)} onChange={(id, value) => { if (!disabled) { onChange(id, value); setOpen(false); } }} />
    </PopoverPopup>
  </Popover>;
}
function ModelPickerContent({ providers, profileId, model, onClose, onChange, locale, identityId }: {
  identityId?: string; locale: 'en' | 'zh'; providers: readonly AgentProviderConfiguration[]; profileId: string; model?: string;
  onClose: () => void; onChange: (profileId: string, model: string) => void;
}) {
  const [selectedProfile, setSelectedProfile] = useState(providers.some(item => item.id === profileId) ? profileId : providers[0]?.id);
  const [query, setQuery] = useState('');
  const highlighted = useRef<string | null>(null);
  // Composite identity keeps identical model ids on different providers distinct.
  const rows = providers.flatMap(provider => provider.models.map(model => ({ provider, model, key: JSON.stringify([provider.id, model.id]) })));
  const filtered = rows.map(row => ({ ...row, score: scoreModelPickerSearch({ driverKind: row.provider.provider, providerDisplayName: providerLabels[row.provider.provider], name: row.model.label, shortName: row.model.id }, query) }))
    .filter(row => (query.trim() || row.provider.id === selectedProfile) && row.score !== null)
    .sort((a, b) => a.score! - b.score!);
  const select = (key: string) => { const row = filtered.find(item => item.key === key); if (row) onChange(row.provider.id, row.model.id); };
  return <div className="relative flex h-[min(22rem,var(--available-height,22rem))] w-[min(22.5rem,var(--available-width,22.5rem))] max-h-full max-w-full flex-row overflow-hidden" data-model-picker-content="true">
    <div className="w-11 shrink-0 overflow-hidden bg-muted/30" data-model-picker-sidebar="true">
      <div className="h-full overflow-y-auto overscroll-contain"><div className="relative flex min-h-full flex-col gap-1 p-1">
        {providers.map(provider => { const Icon = providerIcons[provider.provider]; return <Tooltip key={provider.id}>
          <TooltipTrigger render={<button type="button" aria-label={providerLabels[provider.provider]} aria-pressed={selectedProfile === provider.id}
            data-model-picker-provider={provider.id} data-xgc-role="agent-composer-provider-option" data-xgc-id={identityId ? `${identityId}:${provider.id}` : provider.id}
            className="relative isolate flex aspect-square w-full cursor-pointer items-center justify-center rounded-md transition-colors hover:bg-muted focus-visible:bg-muted focus-visible:outline-none aria-pressed:bg-accent"
            onClick={() => { setSelectedProfile(provider.id); setQuery(''); }} />}><Icon className="size-5 shrink-0" aria-hidden="true" /></TooltipTrigger>
          <TooltipPopup side="left" sideOffset={8}>{providerLabels[provider.provider]}</TooltipPopup>
        </Tooltip>; })}
      </div></div>
    </div>
    <Combobox inline items={rows.map(item => item.key)} filteredItems={filtered.map(item => item.key)} filter={null} autoHighlight open
      value={JSON.stringify([profileId, model])} onItemHighlighted={key => { highlighted.current = typeof key === 'string' ? key : null; }}
      onValueChange={key => { if (typeof key === 'string') select(key); }}>
      <div className="flex min-h-0 flex-1 flex-col overflow-hidden border-l border-border/70 bg-muted/40">
        <div className="px-2 pt-2"><div className="border-b border-border/70 pb-2.5 transition-colors focus-within:border-ring">
          <ComboboxInput className="[&_input]:h-6.5 [&_input]:font-sans [&_input]:leading-6.5" inputClassName="rounded-none bg-transparent text-sm"
            placeholder={locale === 'zh' ? '搜索模型…' : 'Search models...'} aria-label={locale === 'zh' ? '搜索模型' : 'Search models'} data-xgc-role="agent-composer-model-search" data-xgc-id={identityId ? `${identityId}:models` : 'models'}
            showTrigger={false} startAddon={<SearchIcon className="size-4 shrink-0 text-muted-foreground" />} value={query} onChange={event => setQuery(event.target.value)}
            onKeyDown={event => { event.stopPropagation(); if (event.key === 'Escape') { event.preventDefault(); onClose(); }
              if (event.key === 'Enter') { event.preventDefault(); (event as typeof event & { preventBaseUIHandler?: () => void }).preventBaseUIHandler?.(); if (!event.nativeEvent.isComposing && highlighted.current) select(highlighted.current); } }}
            onMouseDown={event => event.stopPropagation()} size="sm" unstyled />
        </div></div>
        <ComboboxList className="min-h-0 min-w-0 flex-1 overflow-y-auto p-1">
          {filtered.map((row, index) => <ComboboxItem key={row.key} value={row.key} index={index} hideIndicator
            data-xgc-role="agent-composer-model-option" data-xgc-id={`${identityId ? `${identityId}:` : ''}${row.provider.id}:${row.model.id}`}
            contentClassName="flex w-full items-center gap-3"
            className="group relative w-full !min-w-0 max-w-full cursor-pointer rounded-md px-2 py-2 transition-[background-color,box-shadow,color] hover:bg-muted data-highlighted:bg-muted data-selected:bg-foreground/[0.08] data-selected:text-foreground data-selected:ring-0">
            <div className="min-w-0 flex-1 text-left"><div className="flex min-w-0 items-center gap-2"><div className="min-w-0 truncate text-xs font-medium leading-snug">{row.model.label}</div></div>
              <div className="mt-1 flex items-center gap-1.5"><span className="truncate text-xs font-normal leading-snug text-muted-foreground/70">{providerLabels[row.provider.provider]}</span></div>
            </div>
          </ComboboxItem>)}
        </ComboboxList>
        {!filtered.length ? <p role="status" className="p-3 text-xs text-muted-foreground">{locale === 'zh' ? '暂无可用模型' : 'No available models'}</p> : null}
      </div>
    </Combobox>
  </div>;
}
