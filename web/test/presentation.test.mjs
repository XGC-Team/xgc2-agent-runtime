import { test } from 'node:test'
import assert from 'node:assert/strict'
import { createElement } from 'react'
import { renderToStaticMarkup } from 'react-dom/server'
import { AgentConversation, AgentInput } from '../dist/react.js'
import { nativeApprovalAnswer, nativeConversationModel } from '../dist/agentPresentation.js'
import { applyEvent, emptyStream, AGENT_RUNTIME_SCHEMA } from '../dist/state.js'

const permission = { id: 'q_1', kind: 'permission', title: 'Native operation', options: [
  { id: 'accept', label: 'Allow once', kind: 'allow_once' },
  { id: 'decline', label: 'Decline', kind: 'reject_once' },
], questions: [] }
const event = (seq, extra) => ({ schemaVersion: AGENT_RUNTIME_SCHEMA, sessionId: 's_test', provider: 'codex', seq, turnId: 't_test', ...extra })
const requested = () => applyEvent(emptyStream('s_test', 'codex'), event(1, { kind: 'input.request', itemId: permission.id, request: permission }))
const controlTags = html => Array.from(html.matchAll(/<button\b[^>]*data-xgc-role="agent-(?:approval-option|request-cancel)"[^>]*>/g), match => match[0])
const optionIds = html => Array.from(html.matchAll(/data-xgc-role="agent-approval-option"[^>]*data-xgc-id="([^"]+)"/g), match => match[1])

// Retains the original Node presentation contracts against the compiled T3 adapter.
test('rendering a native decision never answers it and exposes stable operator controls', () => {
  let called = false
  const html = renderToStaticMarkup(createElement(AgentInput, { sessionId: 's_test', request: permission, submitted: false, locale: 'en', onAnswer: async () => { called = true } }))
  assert.match(html, /Allow once/)
  assert.match(html, /Decline/)
  assert.match(html, /Cancel request/)
  assert.deepEqual(optionIds(html), ['notification:s_test:q_1:q_1:accept', 'notification:s_test:q_1:q_1:decline'])
  assert.equal(controlTags(html).length, 3)
  assert.ok(controlTags(html).every(tag => !tag.includes('disabled=""')))
  assert.doesNotMatch(html, /acceptForSession|Always allow|agent-composer-input|agent-send/)
  assert.throws(() => nativeApprovalAnswer(requested(), permission.id, 'acceptForSession'), /not offered/)
  assert.equal(called, false)
})

test('submitted answers remain pending until the native stream resolves them', () => {
  const state = applyEvent(requested(), event(2, { kind: 'input.submitted', itemId: permission.id }))
  const pending = state.pending[permission.id]
  assert.equal(pending.submitted, true)
  assert.equal(nativeConversationModel(state).approvals[0].submitted, true)
  const html = renderToStaticMarkup(createElement(AgentInput, { sessionId: state.sessionId, request: pending, submitted: pending.submitted, onAnswer: async () => { throw new Error('must not run') } }))
  assert.match(html, /aria-busy="true"/)
  assert.deepEqual(optionIds(html), ['notification:s_test:q_1:q_1:accept', 'notification:s_test:q_1:q_1:decline'])
  assert.equal(controlTags(html).length, 3)
  assert.ok(controlTags(html).every(tag => tag.includes('disabled=""')))
  assert.doesNotMatch(html, /agent-composer-input|agent-send|agent-interrupt/)
  assert.throws(() => nativeApprovalAnswer(state, permission.id, 'accept'), /no longer/)
  assert.equal(state.lastTurnStatus, '')
  assert.deepEqual(applyEvent(state, event(3, { kind: 'input.resolved', itemId: permission.id })).pending, {})
})

test('shared conversation projects received text once and scopes item and request identity to its session', () => {
  let state = applyEvent(emptyStream('s_test', 'codex'), event(1, { kind: 'item.delta', itemId: 'message', role: 'assistant', text: 'Recorded reply' }))
  state = applyEvent(state, event(2, { kind: 'input.request', itemId: permission.id, request: permission }))
  let called = false
  const html = renderToStaticMarkup(createElement(AgentConversation, { state, locale: 'en', onAnswer: async () => { called = true } }))
  // LegendList needs a browser viewport to render rows, so SSR does not assert
  // virtualized row markup. The pure mapper preserves the same text/identity
  // contract; actual compiled-row rendering is separately browser-verified in
  // research-consumer/verification/shared-artifact.browser.mjs (file:// only).
  const messages = nativeConversationModel(state).items.filter(item => item.kind === 'message')
  assert.equal(messages.length, 1)
  assert.equal(messages[0].text, 'Recorded reply')
  assert.equal(messages[0].id, JSON.stringify(['s_test', 't_test', 'message']))
  assert.equal(messages[0].createdAt, undefined)
  assert.match(html, /data-xgc-role="agent-conversation" data-xgc-id="s_test"/)
  assert.deepEqual(optionIds(html), ['s_test:q_1:accept', 's_test:q_1:decline'])
  assert.doesNotMatch(html, /agent-composer-input|agent-send|agent-interrupt/)
  assert.equal(called, false)
})
