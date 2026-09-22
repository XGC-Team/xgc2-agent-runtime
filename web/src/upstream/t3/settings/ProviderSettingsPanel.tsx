// T3 Code, MIT, Copyright (c) 2026 T3 Tools Inc.
// Upstream ref: bf3be75c400cf605dc0c80de9458458854c84131. See UPSTREAM.md.
// ProviderSettingsPanel list/editor layout + ProviderInstanceCard list-row markup.
// XGC adapter replaces environment stores with display props and callbacks.
import type { ReactNode } from 'react';
import { cn } from '../utils.js';
import { providerIcons, providerLabels } from '../ProviderModelPicker.js';
import type { AgentProviderConfiguration } from '../../../providerSettings.js';
export function ProviderSettingsPanel({ providers, selectedId, onSelect, children, locale = 'en' }: {
  locale?: 'en' | 'zh'; providers: readonly AgentProviderConfiguration[]; selectedId: string; onSelect: (id: string) => void; children: ReactNode;
}) {
  // 去卡片化：列表-编辑两栏靠留白与发丝线分区，不再套边框底卡（编辑排版纪律）
  return <div className="lg:grid lg:grid-cols-[13.5rem_minmax(0,1fr)] lg:gap-0">
    <div className="mb-3 lg:mb-0" role="navigation" aria-label={locale === 'zh' ? '供应者' : 'Providers'}>
      {providers.map(provider => { const Icon = providerIcons[provider.provider]; const selected = selectedId === provider.id; return <button key={provider.id} type="button"
        onClick={() => onSelect(provider.id)} aria-pressed={selected}
        data-xgc-role="agent-provider-select" data-xgc-id={provider.id}
        className={cn('flex w-full items-center gap-2.5 rounded-lg px-2.5 py-2 text-left transition-colors duration-150', selected ? 'bg-muted/45' : 'hover:bg-muted/25')}>
        <Icon className={cn('size-4 shrink-0', selected ? 'text-foreground' : 'text-muted-foreground')} aria-hidden="true" />
        <span className="min-w-0 flex-1">
          <span className="block truncate text-[13px]"><span className={cn(selected ? 'font-medium text-foreground' : 'text-foreground/80')}>{providerLabels[provider.provider]}</span></span>
          <span className="mt-0.5 block text-[11px] text-muted-foreground">{!provider.enabled ? (locale === 'zh' ? '已停用' : 'Disabled') : provider.available ? (locale === 'zh' ? '可用' : 'Available') : provider.detail || (locale === 'zh' ? '不可用' : 'Unavailable')}</span>
        </span>
      </button>; })}
    </div>
    <div className="min-w-0 lg:border-l lg:border-border/50 lg:pl-6">{children}</div>
  </div>;
}
