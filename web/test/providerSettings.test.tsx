import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { AgentProviderSettings } from '../src/AgentProviderSettings.js'
import { AgentComposerControls } from '../src/AgentComposerControls.js'
import type { AgentComposerSelection } from '../src/AgentComposerControls.js'
import type { AgentProviderConfiguration, AgentSettings } from '../src/providerSettings.js'
import { useState } from 'react'

const provider = (id = 'codex', extra: Partial<AgentProviderConfiguration> = {}): AgentProviderConfiguration => ({
  id, provider: id === 'other' ? 'claude' : 'codex', enabled: true, binaryPath: '', available: true, version: '1.2.3',
  login: { status: 'unknown', detail: 'Refresh to check native sign-in.' }, defaults: { model: 'same', effort: 'medium', permission: 'approval-required' },
  models: [{ id: 'same', label: 'Shared model', efforts: [{ id: 'medium', label: 'Medium' }, { id: 'high', label: 'High' }], defaultEffort: 'medium' }],
  permissions: [{ id: 'approval-required', label: 'Supervised', description: 'Ask before commands.' }, { id: 'full-access', label: 'Full access', description: 'Allow native operations without asking.' }], ...extra,
})
const doc = (revision = 'r1'): AgentSettings => ({ revision, providers: [provider(), provider('other')] })
const role = (name: string, id: string) => document.querySelector(`[data-xgc-role="${name}"][data-xgc-id="${id}"]`) as HTMLElement
afterEach(cleanup)

describe('shared provider settings', () => {
  it('localizes shared Settings and composer labels while preserving native model and option labels', () => {
    render(<><AgentProviderSettings settings={doc()} onSave={vi.fn()} onRefresh={vi.fn()} locale="zh" />
      <AgentComposerControls providers={doc().providers} value={{ profileId: 'codex' }} onChange={vi.fn()} locale="zh" /></>)
    expect(screen.getByLabelText('CLI 路径')).toBeTruthy()
    expect(screen.getByRole('button', { name: '保存更改' })).toBeTruthy()
    expect(screen.getByText(/登录状态未知/)).toBeTruthy()
    expect(role('agent-composer-model', 'codex').getAttribute('aria-label')).toBe('工作者与模型')
    expect(role('agent-composer-effort', 'codex').getAttribute('aria-label')).toBe('思考强度')
    expect(role('agent-composer-permission', 'codex').getAttribute('aria-label')).toBe('执行权限')
    expect(role('agent-composer-permission', 'codex').getAttribute('aria-description')).toBe('Supervised')
    expect(role('agent-composer-model', 'codex').textContent).toContain('Shared model')
    expect(role('agent-composer-permission', 'codex').textContent).toContain('Supervised')
  })
  it('uses only the configuration whitelist and retains truthful unknown login', () => {
    render(<AgentProviderSettings settings={doc()} onSave={vi.fn()} onRefresh={vi.fn()} />)
    expect(screen.getByLabelText('CLI path')).toBeTruthy()
    expect(screen.getByText(/Login status unknown/)).toBeTruthy()
    expect(document.querySelector('input[type="password"]')).toBeNull()
    expect(screen.queryByText(/billing|credential|environment|shell command/i)).toBeNull()
    expect(document.querySelectorAll('[data-xgc-role="agent-provider-field-input"][data-xgc-id="codex-binaryPath"]')).toHaveLength(1)
  })
  it('keeps the edit baseline across an external revision and preserves failed drafts for retry', async () => {
    const onSave = vi.fn().mockRejectedValueOnce(new Error('The settings changed. Refresh first.')).mockResolvedValueOnce(undefined)
    const view = render(<AgentProviderSettings settings={doc()} onSave={onSave} />)
    fireEvent.change(screen.getByLabelText('CLI path'), { target: { value: '/chosen/codex' } })
    view.rerender(<AgentProviderSettings settings={doc('r2')} onSave={onSave} />)
    fireEvent.click(screen.getByRole('button', { name: 'Save changes' }))
    await screen.findByRole('alert')
    expect(onSave.mock.calls[0]?.[0]).toEqual({ revision: 'r1', provider: { id: 'codex', provider: 'codex', enabled: true, binaryPath: '/chosen/codex', defaults: doc().providers[0]!.defaults } })
    expect((screen.getByLabelText('CLI path') as HTMLInputElement).value).toBe('/chosen/codex')
    fireEvent.click(screen.getByRole('button', { name: 'Save changes' }))
    await screen.findByText('Saved')
    expect(onSave).toHaveBeenCalledTimes(2)
    expect(onSave.mock.calls[1]?.[0].revision).toBe('r1')
  })
  it('preserves edits while switching providers and only probes the explicit selection', async () => {
    const refresh = vi.fn().mockResolvedValue(undefined)
    const view = render(<AgentProviderSettings settings={doc()} onSave={vi.fn()} onRefresh={refresh} />)
    fireEvent.change(screen.getByLabelText('CLI path'), { target: { value: '/draft' } })
    fireEvent.click(role('agent-provider-select', 'other'))
    expect((screen.getByLabelText('CLI path') as HTMLInputElement).value).toBe('')
    fireEvent.click(screen.getByRole('button', { name: 'Refresh status' }))
    await waitFor(() => expect(refresh).toHaveBeenCalledWith('other'))
    fireEvent.click(role('agent-provider-select', 'codex'))
    expect((screen.getByLabelText('CLI path') as HTMLInputElement).value).toBe('/draft')
    expect(role('agent-provider-select', 'codex').textContent).not.toContain('1.2.3')
    expect(role('agent-provider-version', 'codex').textContent).toBe('1.2.3')
    view.rerender(<AgentProviderSettings settings={{ revision: 'r1', providers: [provider(), provider('other', { available: false, version: '9.9.9' })] }} onSave={vi.fn()} onRefresh={refresh} />)
    fireEvent.click(role('agent-provider-select', 'other'))
    expect(role('agent-provider-select', 'other').textContent).not.toContain('9.9.9')
    expect(role('agent-provider-version', 'other')).toBeNull()
  })
  it('serializes save callbacks, shows fixed action labels, and honors inactive surfaces', async () => {
    let complete!: () => void
    const save = vi.fn(() => new Promise<void>(resolve => { complete = resolve }))
    const view = render(<AgentProviderSettings settings={doc()} onSave={save} />)
    fireEvent.change(screen.getByLabelText('CLI path'), { target: { value: '/draft' } })
    const button = screen.getByRole('button', { name: 'Save changes' })
    fireEvent.click(button); fireEvent.click(button)
    expect(save).toHaveBeenCalledTimes(1)
    expect(button.textContent).toBe('Save changes')
    complete(); await waitFor(() => expect(screen.getByText('Saved')).toBeTruthy())
    view.rerender(<AgentProviderSettings settings={doc()} onSave={save} active={false} />)
    expect(screen.queryByRole('button', { name: 'Save changes' })).toBeNull()
  })
  it('does not provide fake save or refresh actions without callbacks', () => {
    render(<AgentProviderSettings settings={doc()} />)
    expect(screen.queryByRole('button', { name: 'Save changes' })).toBeNull()
    expect(screen.queryByRole('button', { name: 'Refresh status' })).toBeNull()
  })
})

describe('migrated native composer choices', () => {
  function Controlled({ providers = doc().providers, onChange = vi.fn(), initial = { profileId: 'codex', model: 'same', effort: 'high', permission: 'full-access' } }: {
    providers?: AgentProviderConfiguration[]; onChange?: (value: AgentComposerSelection) => void; initial?: AgentComposerSelection
  }) {
    const [value, setValue] = useState(initial)
    return <AgentComposerControls providers={providers} value={value} onChange={next => { onChange(next); setValue(next) }} />
  }
  it('keeps independent permission capabilities when the model catalog has not been read', () => {
    const configured = provider('codex', { models: [] })
    const view = render(<AgentComposerControls providers={[configured]} value={{ profileId: 'codex', model: 'current-native-model' }} onChange={vi.fn()} />)
    expect(role('agent-composer-model', 'codex').textContent).toContain('current-native-model')
    expect(role('agent-composer-model', 'codex').getAttribute('aria-description')).toBe('Model catalog not loaded')
    expect(role('agent-composer-model', 'codex').tagName).toBe('BUTTON')
    expect((role('agent-composer-model', 'codex') as HTMLButtonElement).disabled).toBe(true)
    expect(role('agent-composer-model', 'codex').textContent).not.toContain('Model catalog not loaded')
    expect(role('agent-composer-model', 'codex').title).toBe('Model catalog not loaded')
    expect(role('agent-composer-model-status', 'codex')).toBeNull()
    expect(role('agent-composer-permission', 'codex')).toBeTruthy()
    expect((role('agent-composer-permission', 'codex') as HTMLButtonElement).disabled).toBe(false)
    view.rerender(<AgentComposerControls providers={[{ ...configured, enabled: false }]} value={{ profileId: 'codex' }} onChange={vi.fn()} />)
    expect(role('agent-composer-permission', 'codex')).toBeTruthy()
    expect((role('agent-composer-permission', 'codex') as HTMLButtonElement).disabled).toBe(true)
    expect(role('agent-composer-model', 'codex').getAttribute('aria-description')).toBe('Model catalog not loaded')
  })
  it('keeps two real controls for the same provider independently markable', () => {
    render(<><AgentComposerControls providers={doc().providers} value={{ profileId: 'codex' }} onChange={vi.fn()} identityId="connect" />
      <AgentComposerControls providers={doc().providers} value={{ profileId: 'codex' }} onChange={vi.fn()} identityId="current-session" /></>)
    expect(role('agent-composer-model', 'connect:codex')).toBeTruthy()
    expect(role('agent-composer-model', 'current-session:codex')).toBeTruthy()
    expect(role('agent-composer-permission', 'connect:codex')).toBeTruthy()
    expect(role('agent-composer-permission', 'current-session:codex')).toBeTruthy()
    expect(role('agent-composer-model', 'codex')).toBeNull()
  })
  it('uses exact native permission ids without inventing the upstream auto mode', () => {
    const changed = vi.fn()
    render(<Controlled onChange={changed} />)
    fireEvent.click(role('agent-composer-permission', 'codex'))
    const option = role('agent-composer-permission-option', 'codex:approval-required')
    fireEvent.pointerDown(option, { pointerType: 'mouse' }); fireEvent.click(option)
    expect(changed).toHaveBeenCalledWith({ profileId: 'codex', model: 'same', effort: 'high', permission: 'approval-required' })
    expect(role('agent-composer-permission-option', 'codex:auto')).toBeNull()
  })
  it('passes real effort ids and closes the upstream radio menu after selection', () => {
    const changed = vi.fn()
    render(<Controlled onChange={changed} />)
    fireEvent.click(role('agent-composer-effort', 'codex'))
    fireEvent.click(role('agent-composer-effort-option', 'codex:medium'))
    expect(changed).toHaveBeenCalledWith({ profileId: 'codex', model: 'same', effort: 'medium', permission: 'full-access' })
    expect(role('agent-composer-effort', 'codex')?.getAttribute('aria-expanded')).toBe('false')
  })
  it('routes the same model id by provider identity and does not carry old permissions across runtimes', () => {
    const changed = vi.fn()
    render(<Controlled onChange={changed} />)
    fireEvent.click(role('agent-composer-model', 'codex'))
    fireEvent.click(role('agent-composer-provider-option', 'other'))
    const option = role('agent-composer-model-option', 'other:same')
    fireEvent.click(option)
    expect(changed).toHaveBeenCalledWith({ profileId: 'other', model: 'same' })
  })
  it('search uses upstream provider/model ranking, and unavailable profiles cannot be selected', () => {
    const changed = vi.fn()
    render(<Controlled providers={[provider(), provider('other', { available: false })]} onChange={changed} />)
    fireEvent.click(role('agent-composer-model', 'codex'))
    const search = role('agent-composer-model-search', 'models')
    fireEvent.change(search, { target: { value: 'codex shared' } })
    expect(role('agent-composer-model-option', 'codex:same')).toBeTruthy()
    expect(role('agent-composer-provider-option', 'other')).toBeNull()
    fireEvent.change(search, { target: { value: 'claude' } })
    screen.getByText('No available models')
    expect(changed).not.toHaveBeenCalled()
  })
  it('keeps model, effort and permission on one nowrap composer row', () => {
    render(<AgentComposerControls providers={doc().providers} value={{ profileId: 'codex' }} onChange={vi.fn()} />)
    const host = role('agent-composer-controls', 'codex')
    const row = host.querySelector('[data-xgc-composer-row]')
    const model = role('agent-composer-model', 'codex')
    const effort = role('agent-composer-effort', 'codex')
    const permission = role('agent-composer-permission', 'codex')
    expect(row).toBeTruthy()
    expect(row!.classList.contains('flex-nowrap')).toBe(true)
    expect(row!.classList.contains('flex-wrap')).toBe(false)
    expect(row!.contains(model)).toBe(true)
    expect(row!.contains(effort)).toBe(true)
    expect(row!.contains(permission)).toBe(true)
    const css = readFileSync(join(dirname(fileURLToPath(import.meta.url)), '../src/t3-host-tokens.css'), 'utf8')
    const start = css.indexOf(':scope[data-xgc-role="agent-composer-controls"] [data-xgc-composer-row] {')
    expect(start).toBeGreaterThanOrEqual(0)
    const close = css.indexOf('}', css.indexOf('{', start))
    expect(css.slice(start, close)).toContain('flex-wrap: nowrap')
  })
  it('never manufactures effort choices and gates all callbacks while inactive', () => {
    const changed = vi.fn()
    const providers = [provider('codex', { models: [{ id: 'same', label: 'Shared model', efforts: [] }], permissions: [] })]
    const view = render(<AgentComposerControls providers={providers} value={{ profileId: 'codex' }} onChange={changed} />)
    expect((screen.getByRole('button', { name: 'Thinking effort' }) as HTMLButtonElement).disabled).toBe(true)
    expect((screen.getByRole('button', { name: 'Permissions' }) as HTMLButtonElement).disabled).toBe(true)
    expect(screen.queryByRole('combobox', { name: 'Permissions' })).toBeNull()
    expect(screen.queryByRole('menuitemradio')).toBeNull()
    expect(screen.queryByRole('option')).toBeNull()
    view.rerender(<AgentComposerControls providers={providers} value={{ profileId: 'codex' }} onChange={changed} active={false} />)
    expect(screen.queryByRole('button', { name: 'Provider and model' })).toBeNull()
    expect(screen.queryByRole('button', { name: 'Thinking effort' })).toBeNull()
    expect(changed).not.toHaveBeenCalled()
  })
  it('keeps disabled composer controls when no provider catalog is available', () => {
    render(<AgentComposerControls providers={[]} value={{ profileId: '' }} onChange={vi.fn()} disabledReason="Local native companion is not connected" />)
    expect(role('agent-composer-controls', 'unselected')).toBeTruthy()
    expect((screen.getByRole('button', { name: 'Provider and model' }) as HTMLButtonElement).disabled).toBe(true)
    expect((screen.getByRole('button', { name: 'Thinking effort' }) as HTMLButtonElement).disabled).toBe(true)
    expect((screen.getByRole('button', { name: 'Permissions' }) as HTMLButtonElement).disabled).toBe(true)
    expect(screen.getByRole('button', { name: 'Thinking effort' }).getAttribute('title')).toBe('Local native companion is not connected')
    expect(screen.queryByRole('menuitemradio')).toBeNull()
  })
})
