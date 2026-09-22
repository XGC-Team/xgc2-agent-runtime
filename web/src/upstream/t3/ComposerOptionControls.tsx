// T3 Code, MIT, Copyright (c) 2026 T3 Tools Inc.
// Upstream ref: bf3be75c400cf605dc0c80de9458458854c84131. See UPSTREAM.md.
// Adapted ChatComposer.ComposerFooterModeControls and TraitsPicker select menu.
// XGC adapter: render only native capability options; no prompt-injected traits.
import { useState } from 'react';
import { LockIcon, LockOpenIcon, PenLineIcon } from 'lucide-react';
import { ComposerControl, ComposerControlChevron, ComposerControlIcon, ComposerSelectControl } from './ComposerControl.js';
import { Select, SelectItem, SelectPopup, SelectValue } from './ui/select.js';
import { Menu, MenuTrigger, MenuPopup, MenuRadioGroup, MenuRadioItem, MenuGroup, MenuGroupLabel } from './ui/menu.js';
import { Tooltip, TooltipTrigger, TooltipPopup } from './ui/tooltip.js';
import type { NativePermissionOption, NativeValueOption } from '../../providerSettings.js';
const permissionIcons = { 'approval-required': LockIcon, 'auto-accept-edits': PenLineIcon, 'full-access': LockOpenIcon };
export function ComposerPermissionControl({ profileId, options, value, disabled, onChange, locale = 'en', identityId }: {
  identityId?: string; locale?: 'en' | 'zh'; profileId: string; options: readonly NativePermissionOption[]; value?: string; disabled?: boolean; onChange: (value: string) => void;
}) {
  const [open, setOpen] = useState(false);
  const selected = options.find(option => option.id === value);
  const label = selected?.label ?? (value ? `${value} · ${locale === 'zh' ? '不可用' : 'Unavailable'}` : (locale === 'zh' ? '执行权限' : 'Permissions'));
  const Icon = permissionIcons[value as keyof typeof permissionIcons] ?? LockIcon;
  return <Tooltip><Select open={open && !disabled} onOpenChange={value => setOpen(!disabled && value)} value={selected?.id ?? null}
    onValueChange={value => { if (!disabled && typeof value === 'string' && options.some(option => option.id === value)) onChange(value); }}>
    <TooltipTrigger render={<ComposerSelectControl type="button" disabled={disabled} aria-label={locale === 'zh' ? '执行权限' : 'Permissions'} aria-description={label} data-xgc-role="native-composer-permission" data-xgc-id={identityId ? `${identityId}:${profileId}` : profileId} className="min-w-0 shrink-0 font-medium" />}>
      <ComposerControlIcon icon={Icon} /><SelectValue data-xgc-permission-label="">{label}</SelectValue>
    </TooltipTrigger>
    <SelectPopup align="start" side="top" alignItemWithTrigger={false}>
      {options.map(option => { const OptionIcon = permissionIcons[option.id as keyof typeof permissionIcons] ?? LockIcon;
        return <SelectItem key={option.id} value={option.id} hideIndicator className="min-w-64 py-2" data-xgc-role="native-composer-permission-option" data-xgc-id={`${identityId ? `${identityId}:` : ''}${profileId}:${option.id}`}>
          <div className="flex min-w-0 items-center gap-3"><div className="grid min-w-0 flex-1 gap-0.5">
            <span className="inline-flex items-center gap-1.5 font-medium text-foreground"><OptionIcon className="size-3.5 shrink-0 text-muted-foreground" />{option.label}</span>
            <span className="text-xs leading-4 text-muted-foreground">{option.description}</span>
          </div></div>
        </SelectItem>; })}
    </SelectPopup>
  </Select><TooltipPopup side="top">{label}{selected?.description ? ` · ${selected.description}` : ''}</TooltipPopup></Tooltip>;
}
export function ComposerEffortControl({ profileId, options, value, disabled, onChange, locale = 'en', identityId }: {
  identityId?: string; locale?: 'en' | 'zh'; profileId: string; options: readonly NativeValueOption[]; value?: string; disabled?: boolean; onChange: (value: string) => void;
}) {
  const [open, setOpen] = useState(false);
  const selected = options.find(option => option.id === value);
  return <Menu open={open && !disabled} onOpenChange={value => setOpen(!disabled && value)}>
    <MenuTrigger render={<ComposerControl type="button" disabled={disabled} aria-label={locale === 'zh' ? '思考强度' : 'Thinking effort'} data-xgc-role="native-composer-effort" data-xgc-id={identityId ? `${identityId}:${profileId}` : profileId}
      className="shrink-0 justify-start whitespace-nowrap" />}>
      <span className="flex w-full min-w-0 items-center gap-1.5"><span className="min-w-0 truncate">{selected?.label ?? (value ? `${value} · ${locale === 'zh' ? '不可用' : 'Unavailable'}` : (locale === 'zh' ? '思考强度' : 'Thinking effort'))}</span><ComposerControlChevron /></span>
    </MenuTrigger>
    <MenuPopup align="start" side="top"><MenuGroup><MenuGroupLabel>{locale === 'zh' ? '思考强度' : 'Thinking effort'}</MenuGroupLabel>
      <MenuRadioGroup value={selected?.id ?? ''} onValueChange={value => { if (!disabled && options.some(option => option.id === value)) onChange(value); }}>
        {options.map(option => <MenuRadioItem key={option.id} value={option.id} closeOnClick data-xgc-role="native-composer-effort-option" data-xgc-id={`${identityId ? `${identityId}:` : ''}${profileId}:${option.id}`}>
          <span className="flex min-w-0 items-center gap-1.5"><span className="min-w-0 truncate">{option.label}</span></span>
        </MenuRadioItem>)}
      </MenuRadioGroup>
    </MenuGroup></MenuPopup>
  </Menu>;
}
