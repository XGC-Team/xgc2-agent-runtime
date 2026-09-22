import type { AgentLocale } from './agentPresentation.js'
import { useRef, useState } from 'react'
import { LockIcon } from 'lucide-react'
import type { AgentTurnOptions } from './state.js'
import type { AgentProviderConfiguration } from './providerSettings.js'
import { ProviderModelPicker, providerIcons } from './upstream/t3/ProviderModelPicker.js'
import { ComposerControl, ComposerControlChevron, ComposerControlIcon } from './upstream/t3/ComposerControl.js'
import { ComposerEffortControl, ComposerPermissionControl } from './upstream/t3/ComposerOptionControls.js'
import { TooltipProvider } from './upstream/t3/ui/tooltip.js'

export type AgentComposerSelection = AgentTurnOptions & { profileId: string }
export type AgentComposerControlsProps = {
  identityId?: string
  locale?: AgentLocale
  providers: readonly AgentProviderConfiguration[]
  value: AgentComposerSelection
  onChange: (selection: AgentComposerSelection) => void | Promise<unknown>
  disabled?: boolean
  disabledReason?: string
  active?: boolean
}
/** A draft selection only. The host submits these exact options with the next native turn. */
export function AgentComposerControls({ providers, value, onChange, disabled = false, disabledReason = '', active = true, locale = 'en', identityId }: AgentComposerControlsProps) {
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const inFlight = useRef(false)
  const current = useRef({ disabled, active, providers }); current.current = { disabled, active, providers }
  const selected = providers.find(provider => provider.id === value.profileId)
  const model = selected?.models.find(model => model.id === (value.model ?? selected.defaults.model))
  const permission = value.permission ?? selected?.defaults.permission
  const effort = value.effort ?? (value.model === undefined || value.model === selected?.defaults.model ? selected?.defaults.effort : undefined) ?? model?.defaultEffort
  const locked = disabled || !active || busy
  const change = async (next: AgentComposerSelection) => {
    const nextProvider = current.current.providers.find(provider => provider.id === next.profileId)
    if (current.current.disabled || !current.current.active || inFlight.current || !nextProvider?.enabled || !nextProvider.available) return
    inFlight.current = true; setBusy(true); setError('')
    try { await onChange(next) } catch (cause) { setError(cause instanceof Error ? cause.message : String(cause)) }
    finally { inFlight.current = false; setBusy(false) }
  }
  const hasModelChoices = providers.some(provider => provider.enabled && provider.available && provider.models.length > 0)
  const providerLocked = locked || !selected?.enabled || !selected.available
  const modelIdentity = identityId ? `${identityId}:${value.profileId || 'unselected'}` : value.profileId || 'unselected'
  const ProviderIcon = selected ? providerIcons[selected.provider] : null
  const unavailable = disabledReason || (locale === 'zh' ? '当前不可选择' : 'Selection unavailable')
  const modelUnavailableReason = selected?.models.length === 0
    ? (locale === 'zh' ? '模型目录未读取' : 'Model catalog not loaded')
    : (disabledReason || (locale === 'zh' ? '当前不可选择模型' : 'Model selection unavailable'))
  const effortLabel = locale === 'zh' ? '思考强度' : 'Thinking effort'
  const permissionLabel = locale === 'zh' ? '执行权限' : 'Permissions'
  return <div className="w-full min-w-0" hidden={!active} data-xgc-role="agent-composer-controls" data-xgc-id={identityId ?? (value.profileId || 'unselected')}>
    <TooltipProvider>
      <div className="flex w-full min-w-0 flex-nowrap items-center gap-1" data-xgc-composer-row="">
        {hasModelChoices && (!selected || selected.models.length > 0) ? <ProviderModelPicker identityId={identityId} locale={locale} providers={providers} profileId={value.profileId} model={value.model} disabled={locked || selected?.enabled === false} onChange={(profileId, modelId) => {
          const provider = providers.find(provider => provider.id === profileId)
          if (!provider?.enabled || !provider.available || !provider.models.some(model => model.id === modelId)) return
          // Switching provider does not copy permissions or effort across runtimes.
          const next: AgentComposerSelection = { profileId, model: modelId }
          if (profileId === value.profileId) {
            const model = provider.models.find(model => model.id === modelId)!
            if (value.effort && model.efforts.some(option => option.id === value.effort)) next.effort = value.effort
            if (value.permission && provider.permissions.some(option => option.id === value.permission)) next.permission = value.permission
          }
          void change(next)
        }} /> : <ComposerControl type="button" disabled className="min-w-0 max-w-48 flex-1 shrink overflow-hidden whitespace-nowrap sm:max-w-56"
          aria-label={locale === 'zh' ? '工作者与模型' : 'Provider and model'} aria-description={modelUnavailableReason} title={modelUnavailableReason}
          data-xgc-role="agent-composer-model" data-xgc-id={modelIdentity}>
          {ProviderIcon ? <ProviderIcon className="size-4 shrink-0" aria-hidden="true" /> : null}
          <span className="min-w-0 truncate">{model?.label ?? value.model ?? selected?.defaults.model ?? (locale === 'zh' ? '供应者默认模型' : 'Provider default model')}</span>
        </ComposerControl>}
        {selected && model?.efforts.length ? <ComposerEffortControl identityId={identityId} locale={locale} profileId={selected.id} options={model.efforts} value={effort} disabled={providerLocked} onChange={effort => void change({ ...value, effort })} /> : <ComposerControl type="button" disabled className="shrink-0 justify-start whitespace-nowrap"
          aria-label={effortLabel} aria-description={unavailable} title={unavailable}
          data-xgc-role="agent-composer-effort" data-xgc-id={modelIdentity}>
          <span className="flex w-full min-w-0 items-center gap-1.5"><span className="min-w-0 truncate">{effortLabel}</span><ComposerControlChevron /></span>
        </ComposerControl>}
        {selected && selected.permissions.length ? <ComposerPermissionControl identityId={identityId} locale={locale} profileId={selected.id} options={selected.permissions} value={permission} disabled={providerLocked} onChange={permission => void change({ ...value, permission })} /> : <ComposerControl type="button" disabled className="min-w-0 shrink-0 font-medium"
          aria-label={permissionLabel} aria-description={unavailable} title={unavailable}
          data-xgc-role="agent-composer-permission" data-xgc-id={modelIdentity}>
          <ComposerControlIcon icon={LockIcon} /><span className="min-w-0 truncate">{permissionLabel}</span>
        </ComposerControl>}
      </div>
      {error ? <p role="alert" className="text-xs text-destructive">{error}</p> : null}
    </TooltipProvider>
  </div>
}
