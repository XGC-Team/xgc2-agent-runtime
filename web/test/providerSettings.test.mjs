import { test } from 'node:test'
import assert from 'node:assert/strict'
import { decodeNativeSettings } from '../dist/providerSettings.js'
import { createNativeAgentClient } from '../dist/client.js'
import { decodeSession, NATIVE_SCHEMA } from '../dist/state.js'
const settings = () => ({ revision: 'r1', providers: [{ id: 'codex', provider: 'codex', enabled: true, binaryPath: '', available: false, version: '', login: { status: 'unknown', detail: 'Not probed\nUse Refresh.' }, defaults: {}, models: [], permissions: [] }] })
const json = (data, status = 200) => new Response(JSON.stringify(data), { status, headers: { 'Content-Type': 'application/json' } })
test('settings preserve real unknown and empty capabilities, reject malformed catalog identities', () => {
  assert.deepEqual(decodeNativeSettings(settings()), settings())
  for (const change of [s => { s.providers[0].login.status = 'ready' }, s => { s.providers[0].provider = 'fake' }, s => { s.providers.push(s.providers[0]) }, s => { s.providers[0].defaults = { env: 'unsafe' } }, s => { s.providers[0].models = [{ id: 'm', label: 'M', efforts: [], defaultEffort: 'imaginary' }] }]) {
    const data = settings(); change(data); assert.throws(() => decodeNativeSettings(data))
  }
})
test('settings routes preserve the explicit revision and refresh only one provider', async () => {
  const calls = []
  const client = createNativeAgentClient({ basePath: '/api/native-agents', fetch: async (url, init) => { calls.push({ url, init }); return json({ data: settings() }) } })
  await client.getNativeSettings()
  const update = { revision: 'editing-baseline', provider: { id: 'codex', provider: 'codex', enabled: true, defaults: { model: 'real' } } }
  await client.updateNativeSettings(update)
  await client.refreshNativeSettings('codex')
  assert.deepEqual(calls.map(call => call.url), ['/api/native-agents/settings', '/api/native-agents/settings', '/api/native-agents/settings/refresh'])
  assert.deepEqual(JSON.parse(calls[1].init.body), update)
  assert.deepEqual(JSON.parse(calls[2].init.body), { id: 'codex' })
})
test('settings conflicts are returned once without automatic refresh or replay', async () => {
  let attempts = 0
  const client = createNativeAgentClient({ basePath: '/api/native-agents', fetch: async () => { attempts++; return json({ error: { code: 'stale', message: 'Changed externally' } }, 409) } })
  await assert.rejects(client.updateNativeSettings({}), error => error.status === 409 && error.message === 'Changed externally')
  assert.equal(attempts, 1)
})
test('resolved session options and requested options retain distinct meanings, while legacy sessions still decode', () => {
  const session = { schemaVersion: NATIVE_SCHEMA, id: 's_test', provider: 'codex', state: 'ready', createdAt: '2026-09-06T00:00:00Z', lastSeq: 0, title: 'Review', archived: false, metadataRevision: 1,
    scope: { profileId: 'codex', context: { kind: 'experiment', id: 'e1' }, workspace: { id: 'debug', revision: 'v1' }, nativeAccessConfirmed: true } }
  assert.equal(decodeSession(session).options, undefined)
  session.scope.options = { permission: 'approval-required' }; session.options = { ...session.scope.options, model: 'real', effort: 'high' }
  assert.deepEqual(decodeSession(session).scope.options, { permission: 'approval-required' })
  assert.deepEqual(decodeSession(session).options, session.options)
})
test('prompt fourth-argument options reach the real request without changing the legacy three-argument call', async () => {
  const calls = []
  const client = createNativeAgentClient({ basePath: '/agents', fetch: async (url, init) => { calls.push(JSON.parse(init.body)); return json({ data: { turnId: 't_00000000000000000000000000000000' } }) } })
  await client.sendNativePrompt('s1', 'hello', 'key1')
  await client.sendNativePrompt('s1', 'hello again', 'key2', { model: 'actual', effort: 'high', permission: 'full-access' })
  assert.deepEqual(calls, [{ text: 'hello' }, { text: 'hello again', options: { model: 'actual', effort: 'high', permission: 'full-access' } }])
})
