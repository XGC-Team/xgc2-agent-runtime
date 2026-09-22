import { test } from 'node:test'
import assert from 'node:assert/strict'
import { createAgentClient, AGENT_CLIENT_HEADER, AgentClientError } from '../dist/client.js'
import { AGENT_RUNTIME_SCHEMA } from '../dist/state.js'

const scope = () => ({ profileId: 'codex', context: { kind: 'experiment', id: 'e1' }, workspace: { id: 'debug', revision: 'reviewed-v1' }, accessConfirmed: true })
const session = (value) => ({ schemaVersion: AGENT_RUNTIME_SCHEMA, id: 's_test', scope: value, provider: 'codex', state: 'ready', createdAt: '2026-09-06T00:00:00Z', lastSeq: 0, title: 'Review', archived: false, metadataRevision: 1 })
const json = (data, status = 200) => new Response(JSON.stringify(data), { status, headers: { 'Content-Type': 'application/json' } })

test('complete inventories follow server cursors while page callers retain explicit pagination', async () => {
  const calls=[]
  const client=createAgentClient({basePath:'/agents',fetch:async url => {
    calls.push(url)
    return json({data:url.includes('after=') ? {sessions:[{...session(scope()),id:'s_second'}]} : {sessions:[session(scope())],nextCursor:'next'}})
  }})
  assert.deepEqual((await client.getNativeSessions()).map(value=>value.id),['s_test','s_second'])
  assert.deepEqual(calls,['/agents/sessions','/agents/sessions?after=next'])
  assert.equal((await client.getNativeSessionPage(undefined,{limit:1})).nextCursor,'next')
})

test('pending policy scopes come from the authority and re-evaluation never submits a browser answer', async () => {
  const facts={operation:'native.shell.execute',experimentId:'e1',conversationId:'s_test',workspace:{id:'debug',revision:'reviewed-v1'},targetId:'local',parametersDigest:'a'.repeat(64)}
  const calls=[]
  const client=createAgentClient({basePath:'/agents',fetch:async (url,init) => {
    calls.push({url,init})
    return json({data:url.endsWith('evaluate-inputs') ? {requested:true} : [{request:{id:'request',kind:'permission',title:'Review',options:[],questions:[]},submitted:false,facts}]})
  }})
  assert.deepEqual((await client.getNativeInputs('s_test'))[0].facts,facts)
  await client.evaluateNativeInputs('s_test')
  assert.equal(calls[1].url,'/agents/sessions/s_test/evaluate-inputs')
  assert.deepEqual(JSON.parse(calls[1].init.body),{})
})

test('consumer route root and native mutation headers reach the real request', async () => {
  const calls = []
  const client = createAgentClient({ basePath: '/api/experiments/native-agents/', fetch: async (url, init) => {
    calls.push({ url, init }); return json({ data: session(JSON.parse(init.body)) }, 202)
  } })
  assert.equal((await client.createNativeSession(scope(), 'create-once')).id, 's_test')
  assert.equal(calls[0].url, '/api/experiments/native-agents/sessions')
  assert.equal(calls[0].init.headers[AGENT_CLIENT_HEADER], '1')
  assert.equal(calls[0].init.headers['Idempotency-Key'], 'create-once')
  assert.deepEqual(JSON.parse(calls[0].init.body), scope())
})

test('nested scope identity is compared by fields and rejects a different context or revision', async () => {
  for (const field of ['context', 'workspace']) {
    const changed = scope()
    if (field === 'context') changed.context.id = 'other'
    else changed.workspace.revision = 'other'
    const client = createAgentClient({ basePath: '/agents', fetch: async () => json({ data: session(changed) }) })
    await assert.rejects(client.createNativeSession(scope(), 'key'), /范围不匹配/)
  }
})

test('failed sends retain the actual server error and are never retried by the transport', async () => {
  let attempts = 0
  const client = createAgentClient({ basePath: '/agents', fetch: async () => {
    attempts++; return json({ error: { code: 'not_ready', message: 'native thread disconnected' } }, 409)
  } })
  await assert.rejects(client.sendNativePrompt('s_test', 'inspect', 'prompt-once'), (error) =>
    error instanceof AgentClientError && error.status === 409 && error.code === 'not_ready' && error.message === 'native thread disconnected')
  assert.equal(attempts, 1)
})

test('a prompt acknowledgement needs its stable server turn and never invents output', async () => {
  const client = createAgentClient({ basePath: '/agents', fetch: async () => json({ data: { accepted: true } }, 202) })
  await assert.rejects(client.sendNativePrompt('s_test', 'inspect', 'prompt-once'), /轮次标识/)
})

test('only explicit same-origin route roots can be configured', () => {
  for (const basePath of ['https://elsewhere.test/api', '//elsewhere.test/api', '/api?host=other', '/api/../other', '']) {
    assert.throws(() => createAgentClient({ basePath }))
  }
})
