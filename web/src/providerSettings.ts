import { PROVIDERS, decodeNativeTurnOptions, type AgentTurnOptions, type AgentProvider } from './state.js'

export type AgentValueOption = { id: string; label: string }
export type AgentModelOption = AgentValueOption & { efforts: AgentValueOption[]; defaultEffort?: string }
export type AgentPermissionOption = AgentValueOption & { description: string }
export type AgentProviderConfiguration = {
  id: string
  provider: AgentProvider
  enabled: boolean
  binaryPath: string
  available: boolean
  version: string
  detail?: string
  login: { status: 'unknown' | 'authenticated' | 'unauthenticated'; detail: string }
  defaults: AgentTurnOptions
  models: AgentModelOption[]
  permissions: AgentPermissionOption[]
}
export type AgentSettings = { revision: string; providers: AgentProviderConfiguration[] }
export type AgentProviderSettingsUpdate = {
  revision: string
  provider: Pick<AgentProviderConfiguration, 'id' | 'provider' | 'enabled' | 'defaults'> & { binaryPath?: string }
}

function record(value: unknown): Record<string, unknown> {
  if (!value || typeof value !== 'object' || Array.isArray(value)) throw new Error('Invalid provider settings.')
  return value as Record<string, unknown>
}
function text(value: unknown, maximum = 4096): string {
  if (typeof value !== 'string' || value.length > maximum || /\0/.test(value)) throw new Error('Invalid provider setting text.')
  return value
}
function id(value: unknown): string {
  const result = text(value, 256)
  if (!result || /[\r\n]/.test(result)) throw new Error('Missing provider setting identity.')
  return result
}
function bool(value: unknown): boolean {
  if (typeof value !== 'boolean') throw new Error('Invalid provider setting state.')
  return value
}
function array<T>(value: unknown, read: (value: unknown) => T, limit: number): T[] {
  if (!Array.isArray(value) || value.length > limit) throw new Error('Invalid provider settings collection.')
  return value.map(read)
}
function unique<T extends { id: string }>(values: T[]): T[] {
  if (new Set(values.map(value => value.id)).size !== values.length) throw new Error('Duplicate provider setting identity.')
  return values
}
function option(value: unknown): AgentValueOption {
  const row = record(value)
  return { id: id(row.id), label: text(row.label, 512) }
}
export function decodeNativeSettings(value: unknown): AgentSettings {
  const settings = record(value)
  const providers = unique(array(settings.providers, value => {
    const row = record(value), login = record(row.login)
    if (!PROVIDERS.includes(row.provider as AgentProvider)) throw new Error('Unknown native provider.')
    if (!['unknown', 'authenticated', 'unauthenticated'].includes(String(login.status))) throw new Error('Unknown provider login state.')
    const models = unique(array(row.models, value => {
      const model = record(value)
      const efforts = unique(array(model.efforts, option, 64))
      const defaultEffort = model.defaultEffort === undefined ? undefined : id(model.defaultEffort)
      if (defaultEffort && !efforts.some(item => item.id === defaultEffort)) throw new Error('Provider default effort is unavailable.')
      return { ...option(model), efforts, ...(defaultEffort === undefined ? {} : { defaultEffort }) }
    }, 2048))
    const permissions = unique(array(row.permissions, value => ({ ...option(value), description: text(record(value).description) }), 32))
    return {
      id: id(row.id), provider: row.provider as AgentProvider, enabled: bool(row.enabled), binaryPath: text(row.binaryPath),
      available: bool(row.available), version: text(row.version, 256), ...(row.detail === undefined ? {} : { detail: text(row.detail) }),
      login: { status: login.status as AgentProviderConfiguration['login']['status'], detail: text(login.detail) },
      defaults: decodeNativeTurnOptions(row.defaults), models, permissions,
    }
  }, 16))
  return { revision: id(settings.revision), providers }
}
