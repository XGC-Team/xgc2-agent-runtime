import { useMemo, useState } from 'react'
import { createRoot } from 'react-dom/client'
import { AgentConversation } from '../../src/AgentConversation.js'
import { DecisionCard } from '../../src/DecisionCard.js'
import { emptyStream, type AgentItem } from '../../src/state.js'
import type { TimelineItem } from '../../src/upstream/t3/types.js'
import '../../dist/agent-chat.css'

const start = Date.now() - 5000
function message(index: number): AgentItem {
  const turnId = `t_${String(index).padStart(32, '0')}`
  return { key: JSON.stringify([turnId, 'user']), id: 'user', turnId,
    role: index % 2 ? 'assistant' : 'user', text: `Message ${index}: a bounded streaming timeline with variable-height content. ` + 'Measured content. '.repeat(index % 7),
    title: '', status: 'completed', truncated: false, createdAt: new Date(start + index * 20).toISOString() }
}
function Acceptance() {
  const [items, setItems] = useState(() => Array.from({ length: 120 }, (_, index) => message(index)))
  const [draft, setDraft] = useState('')
  const additionalItems = useMemo<TimelineItem[]>(() => Array.from({ length: 120 }, (_, index) => ({
    kind: 'custom', id: `domain-${index}`, createdAt: new Date(start + index * 20 + 10).toISOString(),
    keepMounted: index === 0,
    content: index % 2 ? <DecisionCard identity={`decision-${index}`} title={`Domain approval ${index}`} state="resolved" status="Completed">
      <p>Frozen domain targets: robot-a, robot-b</p>
    </DecisionCard> : <div data-fixture-domain="remote" data-fixture-live={index === 0 ? 'true' : undefined}>
      Remote controller {index}: robot-a, robot-b {index === 0 ? <button type="button">Fixed live portal owner</button> : '(closed)'}
    </div>,
  })), [])
  const state = { ...emptyStream('fixture-session', 'codex'), worker: 'ready' as const, items }
  return <>
    <button data-fixture-action="append" onClick={() => setItems(current => [...current, { ...message(current.length), createdAt: new Date().toISOString() }])}>Append canonical message</button>
    <div style={{ height: 680, width: 480 }}>
      <AgentConversation state={state} active draft={draft} onDraftChange={setDraft} onSend={async () => undefined} onAnswer={async () => undefined}
        additionalItems={additionalItems} emptyState={null}
        dock={<div data-fixture-fixed="dock">Queue dock</div>}
        additionalPendingRequests={<div data-fixture-fixed="approval"><DecisionCard identity="pending-domain" title="Pending domain approval" state="pending"
          actions={<><button type="button">Approve</button><button type="button">Reject</button></>}><p>Frozen targets remain outside the virtual scrollport.</p></DecisionCard></div>} />
    </div>
  </>
}
const style = document.createElement('style')
style.textContent = `body{margin:12px;font-family:Arial,sans-serif} [data-xgc-role="agent-conversation"]{height:100%} [data-fixture-domain="remote"]{padding:14px;background:#202024;color:white;border-radius:8px} button{cursor:pointer}`
document.head.appendChild(style)
createRoot(document.getElementById('root')!).render(<Acceptance />)
