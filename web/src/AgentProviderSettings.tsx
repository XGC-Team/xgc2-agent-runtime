import type { AgentLocale } from './agentPresentation.js'
import { useRef, useState } from 'react'
import type { AgentProviderConfiguration, AgentProviderSettingsUpdate, AgentSettings } from './providerSettings.js'
import { ProviderSettingsPanel } from './upstream/t3/settings/ProviderSettingsPanel.js'
import { ProviderSettingsForm, type ProviderSettingsFieldModel } from './upstream/t3/settings/ProviderSettingsForm.js'
import { providerLabels } from './upstream/t3/ProviderModelPicker.js'
import { Button } from './upstream/t3/ui/button.js'
import { T3PortalContainer, TooltipProvider } from './upstream/t3/ui/tooltip.js'

export type AgentProviderSettingsProps = {
  locale?: AgentLocale
  settings: AgentSettings
  onSave?: (update: AgentProviderSettingsUpdate) => Promise<unknown>
  onRefresh?: (profileId: string) => Promise<unknown>
  disabled?: boolean
  active?: boolean
}
type Draft = { revision: string; provider: AgentProviderConfiguration; config: Record<string, unknown> }
function configOf(provider: AgentProviderConfiguration): Record<string, unknown> {
  return { enabled: provider.enabled, binaryPath: provider.binaryPath, ...provider.defaults }
}
function fieldsFor(provider: AgentProviderConfiguration, config: Record<string, unknown>, zh: boolean): ProviderSettingsFieldModel[] {
  const fields: ProviderSettingsFieldModel[] = [
    { key: 'enabled', control: 'switch', label: zh ? '启用' : 'Enabled', clearWhenEmpty: 'persist' },
    { key: 'binaryPath', control: 'text', label: zh ? 'CLI 路径' : 'CLI path', placeholder: zh ? '留空以自动发现已安装的 CLI。' : 'Leave empty to discover the installed CLI on PATH.', clearWhenEmpty: 'persist' },
  ]
  const choice = (key: string, label: string, options: readonly { id: string; label: string }[]): ProviderSettingsFieldModel => ({
    key, label, control: 'select', clearWhenEmpty: 'omit', options: [{ value: '', label: zh ? '供应者默认值' : 'Provider default' }, ...options.map(option => ({ value: option.id, label: option.label }))],
  })
  if (provider.models.length) fields.push(choice('model', zh ? '默认模型' : 'Default model', provider.models))
  const model = provider.models.find(model => model.id === (typeof config.model === 'string' ? config.model : provider.defaults.model))
  if (model?.efforts.length) fields.push(choice('effort', zh ? '默认思考强度' : 'Default thinking effort', model.efforts))
  if (provider.permissions.length) fields.push(choice('permission', zh ? '默认权限' : 'Default permissions', provider.permissions))
  return fields
}
function updateOf(draft: Draft): AgentProviderSettingsUpdate {
  const { config, provider, revision } = draft
  const defaults: AgentProviderConfiguration['defaults'] = {}
  for (const key of ['model', 'effort', 'permission'] as const) if (typeof config[key] === 'string' && config[key]) defaults[key] = config[key]
  return { revision, provider: { id: provider.id, provider: provider.provider, enabled: config.enabled === true, binaryPath: typeof config.binaryPath === 'string' ? config.binaryPath : '', defaults } }
}
/** Shared whitelist and editing baseline. Consumers only fetch and save the settings document. */
export function AgentProviderSettings({ settings, onSave, onRefresh, disabled = false, active = true, locale = 'en' }: AgentProviderSettingsProps) {
  const zh = locale === 'zh'
  const [selected, setSelected] = useState(settings.providers[0]?.id ?? '')
  const [drafts, setDrafts] = useState<Record<string, Draft>>({})
  const [busy, setBusy] = useState<string | null>(null)
  const [messages, setMessages] = useState<Record<string, { error?: string; saved?: boolean }>>({})
  const [portal, setPortal] = useState<HTMLDivElement | null>(null)
  const inFlight = useRef(false)
  const current = useRef({ disabled, active }); current.current = { disabled, active }
  const provider = settings.providers.find(provider => provider.id === selected) ?? settings.providers[0]
  const draft = provider ? drafts[provider.id] : undefined
  const config = draft?.config ?? (provider ? configOf(provider) : {})
  const locked = disabled || !active || busy !== null
  const run = async (kind: 'save' | 'refresh') => {
    if (!provider || inFlight.current || current.current.disabled || !current.current.active || (kind === 'save' ? !onSave || !draft : !onRefresh)) return
    const id = provider.id
    inFlight.current = true; setBusy(id); setMessages(current => ({ ...current, [id]: {} }))
    try {
      if (kind === 'save') {
        await onSave!(updateOf(draft!))
        setDrafts(current => { const next = { ...current }; if (next[id] === draft) delete next[id]; return next })
        setMessages(current => ({ ...current, [id]: { saved: true } }))
      } else await onRefresh!(id)
    } catch (cause) { setMessages(current => ({ ...current, [id]: { error: cause instanceof Error ? cause.message : String(cause) } })) }
    finally { inFlight.current = false; setBusy(null) }
  }
  return <div className="xgc-agent-chat" ref={setPortal} hidden={!active} data-xgc-role="agent-provider-settings" data-xgc-id="providers">
    <T3PortalContainer value={portal}><TooltipProvider>
      <ProviderSettingsPanel locale={locale} providers={settings.providers} selectedId={provider?.id ?? ''} onSelect={setSelected}>
        {provider ? <section key={provider.id} data-xgc-role="agent-provider-editor" data-xgc-id={provider.id}>
          <div className="mb-5">
            <div className="flex items-start justify-between gap-3">
              <h3 className="min-w-0 text-[15px] font-medium tracking-[-0.005em]">{providerLabels[provider.provider]}</h3>
              {onRefresh ? <Button type="button" size="sm" variant="ghost" disabled={locked} onClick={() => void run('refresh')} data-xgc-role="agent-provider-refresh" data-xgc-id={provider.id}>{zh ? '刷新状态' : 'Refresh status'}</Button> : null}
            </div>
            <p className="mt-1 flex min-w-0 items-center gap-1.5 whitespace-nowrap text-xs text-muted-foreground" data-xgc-role="agent-provider-login-status" data-xgc-id={provider.id}>
              <span aria-hidden className={provider.login.status === 'authenticated' ? 'inline-block h-1.5 w-1.5 shrink-0 rounded-full bg-current' : 'inline-block h-1.5 w-1.5 shrink-0 rounded-full border border-current opacity-50'} />
              <span className="min-w-0">
                {provider.login.status === 'authenticated' ? (zh ? '已登录' : 'Signed in') : provider.login.status === 'unauthenticated' ? (zh ? '未登录' : 'Not signed in') : (zh ? '登录状态未知' : 'Login status unknown')}{provider.login.detail ? ` · ${provider.login.detail}` : ''}
                {provider.available && provider.version ? <>{' · '}<span data-xgc-role="agent-provider-version" data-xgc-id={provider.id}>{provider.version}</span></> : null}
              </span>
            </p>
          </div>
          <fieldset disabled={locked || !onSave} className="min-w-0">
            <ProviderSettingsForm key={provider.id} fields={fieldsFor(provider, config, zh)} variant="settings" idPrefix={provider.id} value={config} onChange={next => {
              if (locked || !onSave) return
              const nextConfig = { ...next }
              if (nextConfig.model !== config.model) delete nextConfig.effort
              setDrafts(current => ({ ...current, [provider.id]: { revision: draft?.revision ?? settings.revision, provider: draft?.provider ?? provider, config: nextConfig } }))
              setMessages(current => ({ ...current, [provider.id]: {} }))
            }} />
          </fieldset>
          <div className="mt-5 flex items-center gap-2">
            {onSave ? <><Button type="button" size="sm" disabled={locked || !draft} onClick={() => void run('save')} data-xgc-role="agent-provider-save" data-xgc-id={provider.id}>{zh ? '保存更改' : 'Save changes'}</Button>
              <Button type="button" size="sm" variant="ghost" disabled={locked || !draft} onClick={() => { setDrafts(current => { const next = { ...current }; delete next[provider.id]; return next }); setMessages(current => ({ ...current, [provider.id]: {} })) }} data-xgc-role="agent-provider-discard" data-xgc-id={provider.id}>{zh ? '放弃更改' : 'Discard'}</Button></> : null}
          </div>
          <div className="mt-2 min-h-5 text-xs" data-xgc-role="agent-provider-save-status" data-xgc-id={provider.id}>
            {messages[provider.id]?.error ? <span role="alert" className="text-destructive">{messages[provider.id]?.error}</span> : <span role="status" className="text-muted-foreground">{messages[provider.id]?.saved ? (zh ? '已保存' : 'Saved') : ''}</span>}
          </div>
        </section> : <p className="text-sm text-muted-foreground">{zh ? '尚未配置供应者' : 'No providers configured'}</p>}
      </ProviderSettingsPanel>
    </TooltipProvider></T3PortalContainer>
  </div>
}
