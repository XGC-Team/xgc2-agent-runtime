import { test } from 'node:test'
import assert from 'node:assert/strict'
import { createAgentClient, isStaleRequest, XrpcError, AGENT_CALL_DEADLINES } from '../dist/client.js'
import { AGENT_RUNTIME_SCHEMA } from '../dist/state.js'

const root = 'https://core.test/api/agent-runtime'
const scope = () => ({ profileId: 'codex', context: { kind: 'fixture', id: 'e1' }, workspace: { id: 'debug', revision: 'reviewed-v1' }, accessConfirmed: true })
const session = (value) => ({ schemaVersion: AGENT_RUNTIME_SCHEMA, id: 's_test', scope: value, provider: 'codex', state: 'ready', createdAt: '2026-09-06T00:00:00Z', lastSeq: 0, title: 'Review', archived: false, metadataRevision: 1 })
const json = (body, status = 200) => new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })
const failure = (code, message, status, details) => json({ error: { code, message, ...(details ? { details } : {}) } }, status)

test('complete inventories follow server cursors while page callers retain explicit pagination', async () => {
  const calls = []
  const client = createAgentClient({ basePath: root, fetch: async url => {
    calls.push(String(url))
    return json(String(url).includes('after=') ? { sessions: [{ ...session(scope()), id: 's_second' }] } : { sessions: [session(scope())], nextCursor: 'next' })
  } })
  assert.deepEqual((await client.getNativeSessions()).map(value => value.id), ['s_test', 's_second'])
  assert.deepEqual(calls, [`${root}/sessions`, `${root}/sessions?after=next`])
  assert.equal((await client.getNativeSessionPage(undefined, { limit: 1 })).nextCursor, 'next')
})

test('pending inputs carry no policy scope and re-evaluation sends no body or answer', async () => {
  const calls = []
  const client = createAgentClient({ basePath: root, fetch: async (url, init) => {
    calls.push({ url: String(url), init })
    return json(String(url).endsWith('evaluate-inputs') ? { requested: true } : [{ request: { id: 'request', kind: 'permission', title: 'Review', options: [], questions: [] }, submitted: false }], 202)
  } })
  const inputs = await client.getNativeInputs('s_test')
  assert.deepEqual(Object.keys(inputs[0]).sort(), ['request', 'submitted'])
  await client.evaluateNativeInputs('s_test')
  assert.equal(calls[1].url, `${root}/sessions/s_test/evaluate-inputs`)
  assert.equal(calls[1].init.method, 'POST')
  assert.equal(calls[1].init.body, undefined)
})

test('mutations are XRPC calls: the key is the request id, deadlines follow the work, headers come from the host', async () => {
  const calls = []
  const client = createAgentClient({ basePath: `${root}/`, headers: { Authorization: 'Bearer host-token' }, fetch: async (url, init) => {
    calls.push({ url: String(url), init, headers: init.headers })
    return json(session(JSON.parse(init.body)), 202)
  } })
  assert.equal((await client.createNativeSession(scope(), 'create-once')).id, 's_test')
  const [create] = calls
  assert.equal(create.url, `${root}/sessions`)
  assert.equal(create.init.method, 'POST')
  assert.equal(create.headers.get('X-Request-ID'), 'create-once')
  assert.equal(create.headers.get('Idempotency-Key'), null)
  assert.equal(create.headers.get('X-XGC-Agent-Client'), null)
  assert.equal(create.headers.get('Content-Type'), 'application/json')
  assert.equal(create.headers.get('Authorization'), 'Bearer host-token')
  assert.ok(Number(create.headers.get('X-Xrpc-Timeout-Ms')) > AGENT_CALL_DEADLINES.long - 1000, 'creating waits for the native client')
  assert.deepEqual(JSON.parse(create.init.body), scope())
  await client.getNativeSession('s_test').catch(() => {})
  assert.ok(Number(calls[1].headers.get('X-Xrpc-Timeout-Ms')) <= AGENT_CALL_DEADLINES.short, 'reads are short')
})

test('queue commands and prompts carry their key as the request id and cancel carries none', async () => {
  const calls = []
  const client = createAgentClient({ basePath: root, fetch: async (url, init) => {
    calls.push({ url: String(url), id: init.headers.get('X-Request-ID'), body: init.body && JSON.parse(init.body) })
    return String(url).endsWith('/prompts') ? json({ turnId: 't_00000000000000000000000000000000' }, 202) : json({ revision: 1, paused: false, items: [] }, 202)
  } })
  await client.sendNativePrompt('s1', 'hello', 'prompt-1')
  await client.sendNativePrompt('s1', 'again', 'prompt-2', { model: 'actual', effort: 'high', permission: 'full-access' })
  await client.updateNativePromptQueue('s1', { operation: 'enqueue', text: 'later' }, 'queue-1')
  assert.deepEqual(calls.map(call => [call.id, call.body]), [
    ['prompt-1', { text: 'hello' }],
    ['prompt-2', { text: 'again', options: { model: 'actual', effort: 'high', permission: 'full-access' } }],
    ['queue-1', { operation: 'enqueue', text: 'later' }],
  ])
  assert.ok(calls.every(call => /^https:\/\/core\.test\/api\/agent-runtime\/sessions\/s1\/(prompts|queue)$/.test(call.url)))
})

test('nested scope identity is compared by fields and rejects a different context or revision', async () => {
  for (const field of ['context', 'workspace']) {
    const changed = scope()
    if (field === 'context') changed.context.id = 'other'
    else changed.workspace.revision = 'other'
    const client = createAgentClient({ basePath: root, fetch: async () => json(session(changed), 202) })
    await assert.rejects(client.createNativeSession(scope(), 'key'), /does not match the requested scope/)
  }
})

test('failed sends keep the service error, are not retried and say whether they may have happened', async () => {
  let attempts = 0
  const client = createAgentClient({ basePath: root, fetch: async () => { attempts++; return failure('unavailable', 'agent runtime unavailable', 503) } })
  await assert.rejects(client.sendNativePrompt('s_test', 'inspect', 'prompt-once'), error =>
    error instanceof XrpcError && error.status === 503 && error.code === 'unavailable' && error.message === 'agent runtime unavailable'
      && error.disposition === 'response_received' && error.requestId === 'prompt-once')
  assert.equal(attempts, 1)

  attempts = 0
  const lost = createAgentClient({ basePath: root, fetch: async () => { attempts++; throw new TypeError('connection reset') } })
  await assert.rejects(lost.sendNativePrompt('s_test', 'inspect', 'prompt-twice'), error => error instanceof XrpcError && error.disposition === 'outcome_unknown')
  assert.equal(attempts, 1, 'an unknown outcome is never replayed by the transport')
})

test('a stale request is a conflict the caller can recognize', async () => {
  const client = createAgentClient({ basePath: root, fetch: async () => failure('conflict', 'request is no longer pending', 409, { reason: 'stale' }) })
  const error = await client.answerNativeRequest('s_test', 'q_1', { optionId: 'yes' }).catch(value => value)
  assert.equal(isStaleRequest(error), true)
  assert.equal(isStaleRequest(new XrpcError('conflict', 'revision', 'response_received')), false)
  assert.equal(isStaleRequest(new Error('plain')), false)
})

test('a prompt acknowledgement needs its stable server turn and never invents output', async () => {
  const client = createAgentClient({ basePath: root, fetch: async () => json({ accepted: true }, 202) })
  await assert.rejects(client.sendNativePrompt('s_test', 'inspect', 'prompt-once'), /stable turn identity/)
})

test('the service root is a path of this origin or an http(s) URL, nothing else', () => {
  for (const basePath of ['//elsewhere.test/api', '/api?host=other', '/api/../other', '', 'ftp://elsewhere.test/api', 'https://user:secret@elsewhere.test/api', 'https://elsewhere.test/api#frame']) {
    assert.throws(() => createAgentClient({ basePath }), /API root/)
  }
  assert.equal(createAgentClient({ basePath: '/api/agent-runtime/' }).basePath, '/api/agent-runtime')
  assert.equal(createAgentClient({ basePath: 'https://core.test/api/agent-runtime' }).basePath, 'https://core.test/api/agent-runtime')
})
