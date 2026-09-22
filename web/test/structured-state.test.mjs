import { test } from 'node:test'
import assert from 'node:assert/strict'
import { applyEvent, emptyStream, NATIVE_SCHEMA } from '../dist/state.js'
const event = (seq, extra = {}) => ({ schemaVersion: NATIVE_SCHEMA, sessionId: 's_test', provider: 'codex', seq, kind: 'item.snapshot', turnId: 't_test', itemId: 'tool-one', role: 'tool', text: 'output', ...extra })
const empty = () => emptyStream('s_test','codex')
const command = { type: 'commandExecution', command: 'rg experiment src', cwd: '/reviewed/workspace', exitCode: 2, durationMs: 17, commandActions: [{ type: 'search',command: 'rg experiment src',query: 'experiment',path: 'src' }] }
test('structured tool metadata, native refs and receipt times survive deltas and final state', () => {
 let state = applyEvent(empty(), event(1,{ sourceMethod: 'item/started',createdAt: '2026-09-06T00:00:00Z',nativeThreadId: 'thread-one',nativeTurnId: 'turn-one',details: command }))
 state = applyEvent(state,event(2,{ kind: 'item.delta',text: ' next',sourceMethod: 'item/commandExecution/outputDelta',createdAt: '2026-09-06T00:00:01Z' }))
 assert.deepEqual(state.items[0].details,command)
 assert.equal(state.items[0].sourceMethod,'item/commandExecution/outputDelta')
 assert.equal(state.items[0].nativeThreadId,'thread-one')
 assert.equal(state.items[0].nativeTurnId,'turn-one')
 assert.equal(state.items[0].createdAt,'2026-09-06T00:00:00Z')
 assert.equal(state.items[0].updatedAt,'2026-09-06T00:00:01Z')
 assert.equal(state.items[0].text,'output next')
})
test('file changes retain structured operation and move destination', () => {
 const details={type:'fileChange',changes:[{path:'old.ts',kind:{type:'update',movePath:'new.ts'},diff:'-old\n+new'}]}
 assert.deepEqual(applyEvent(empty(),event(1,{details})).items[0].details,details)
})
test('request method, source item and human question descriptions survive the pending projection', () => {
 const request={id:'q_one',kind:'question',title:'Input',sourceMethod:'item/tool/requestUserInput',nativeThreadId:'thread-one',nativeTurnId:'turn-one',nativeItemId:'tool-one',createdAt:'2026-09-06T00:00:00Z',options:[],questions:[{id:'scope',header:'Debug scope',text:'Which robot?',options:[{id:'one',label:'One robot',kind:'answer',description:'Keep the debug scope narrow'}],multiple:false,freeText:true}]}
 const state=applyEvent(empty(),event(1,{kind:'input.request',itemId:'q_one',request}))
 assert.deepEqual(state.pending.q_one,{...request,submitted:false})
})
test('unknown detail kinds, role changes, malformed times and oversized structure fail closed', () => {
 assert.throws(()=>applyEvent(empty(),event(1,{details:{type:'reasoning',content:'private'}})))
 assert.throws(()=>applyEvent(empty(),event(1,{createdAt:'not-a-date'})))
 assert.throws(()=>applyEvent(empty(),event(1,{details:{...command,exitCode:1.1}})))
 assert.throws(()=>applyEvent(empty(),event(1,{details:{type:'fileChange',changes:Array(65).fill({path:'x',kind:{type:'add'},diff:''})}})))
 assert.throws(()=>applyEvent(empty(),event(1,{role:'assistant',details:command})))
 const state=applyEvent(empty(),event(1,{details:command,nativeThreadId:'thread-one'}))
 assert.throws(()=>applyEvent(state,event(2,{details:command,nativeThreadId:'thread-other'})))
 assert.throws(()=>applyEvent(state,event(2,{details:{type:'webSearch',query:'changed type'}})))
})
test('MCP content is structured and sensitive RPC/config fields cannot be decoded as tool data', () => {
 const details={type:'mcpToolCall',server:'docs',tool:'read',arguments:{path:'guide.md'},result:{content:[{type:'text',text:'body'}],structuredContent:{title:'Guide'}},truncated:true}
 assert.deepEqual(applyEvent(empty(),event(1,{details})).items[0].details,details)
 for(const key of ['authorization','api_key','_meta','__proto__']) {
  const argumentsValue=Object.fromEntries([[key,'must-not-cross']])
  assert.throws(()=>applyEvent(empty(),event(1,{details:{...details,arguments:argumentsValue}})))
 }
 assert.throws(()=>applyEvent(empty(),event(1,{details:{...details,result:{content:[{type:'image',data:'binary'}]}}})))
})

test('cancelled partial items remain terminal when another turn starts streaming', () => {
 let state = applyEvent(empty(), event(1, { kind: 'session.state', status: 'running', turnId: 'old-turn' }))
 state = applyEvent(state, event(2, { turnId: 'old-turn', itemId: 'assistant-one', role: 'assistant', text: 'Partial reply', status: 'inProgress' }))
 state = applyEvent(state, event(3, { turnId: 'old-turn', itemId: 'command-one', status: 'inProgress', details: { type: 'commandExecution', command: 'rg experiment src' } }))
 state = applyEvent(state, event(4, { kind: 'session.state', status: 'awaiting-input', turnId: 'old-turn' }))
 assert.equal(state.activeTurnId, 'old-turn')
 state = applyEvent(state, event(5, { kind: 'session.state', status: 'cancelling', turnId: undefined }))
 assert.equal(state.activeTurnId, 'old-turn')
 state = applyEvent(state, event(6, { kind: 'turn.end', turnId: 'old-turn', status: 'cancelled' }))
 assert.equal(state.activeTurnId, '')
 assert.equal(state.lastTurnStatus, 'cancelled')
 assert.ok(state.items.every(item => item.turnStatus === 'cancelled' && item.status === 'inProgress'))
 state = applyEvent(state, event(7, { kind: 'session.state', status: 'ready', turnId: undefined }))
 state = applyEvent(state, event(8, { kind: 'session.state', status: 'running', turnId: 'new-turn' }))
 state = applyEvent(state, event(9, { turnId: 'new-turn', itemId: 'assistant-one', role: 'assistant', text: 'New reply', status: 'inProgress' }))
 assert.equal(state.activeTurnId, 'new-turn')
 assert.deepEqual(state.items.map(item => [item.turnId, item.turnStatus]), [['old-turn','cancelled'],['old-turn','cancelled'],['new-turn',undefined]])
 assert.deepEqual(state.items.filter(item => item.role === 'assistant' && item.turnId === state.activeTurnId).map(item => item.text), ['New reply'])
 // A later item patch preserves the known turn result without overwriting native item status.
 state = applyEvent(state, event(10, { kind: 'item.patch', turnId: 'old-turn', itemId: 'command-one', status: 'completed' }))
 assert.equal(state.items[1].status, 'completed')
 assert.equal(state.items[1].turnStatus, 'cancelled')
 assert.equal(state.activeTurnId, 'new-turn')
})
test('turn terminal status never invents tool success and terminal detail remains visible', () => {
 for (const status of ['completed', 'failed', 'cancelled', 'unknown', 'incomplete', 'refused', 'blocked']) {
  let state = applyEvent(empty(), event(1, { kind: 'session.state', status: 'running', text: undefined }))
  state = applyEvent(state, event(2, { status: 'inProgress' }))
  state = applyEvent(state, event(3, { kind: 'turn.end', status, text: 'Authoritative terminal detail' }))
  assert.equal(state.activeTurnId, '')
  assert.equal(state.lastTurnStatus, status)
  assert.equal(state.items[0].turnStatus, status)
  assert.equal(state.items[0].status, 'inProgress')
  assert.deepEqual(state.notices, ['Authoritative terminal detail'])
 }
})
test('idle and disconnected sessions have no active turn and unknown results remain unknown', () => {
 for (const status of ['starting', 'ready', 'closed', 'disconnected']) {
  let state = applyEvent(empty(), event(1, { kind: 'session.state', status: 'running', text: undefined }))
  state = applyEvent(state, event(2, { status: 'inProgress' }))
  state = applyEvent(state, event(3, { kind: 'session.state', status, turnId: undefined }))
  assert.equal(state.activeTurnId, '')
  assert.equal(state.items[0].status, 'inProgress')
  assert.equal(state.items[0].turnStatus, undefined)
  assert.equal(state.lastTurnStatus, '')
 }
 assert.equal(applyEvent(empty(), event(1, { status: 'inProgress' })).activeTurnId, '')
 assert.equal(applyEvent(empty(), event(1, { kind: 'session.state', status: 'cancelling', turnId: undefined })).activeTurnId, '')
})
