import React, { useEffect, useRef, useState } from 'react'
import { createRoot } from 'react-dom/client'
import { AgentConversation } from '@xgc2/agent-runtime/react'
import { AGENT_RUNTIME_SCHEMA, applyEvent, emptyStream } from '@xgc2/agent-runtime/state'
import '@xgc2/agent-runtime/styles.css'

const count = Number(new URLSearchParams(location.search).get('history') || '1')
const image = 'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+a3ZkAAAAASUVORK5CYII='
function initialState() {
  const items = []
  for (let index=0;index<count;index+=1) {
    const turnId = `old-${index}`
    items.push({ key:JSON.stringify([turnId,'message']),turnId,id:'message',role:'user',title:'',text:`Historical message ${index}`,status:'completed',truncated:false })
    items.push({ key:JSON.stringify([turnId,'tool']),turnId,id:'tool',role:'tool',title:`Historical tool ${index}`,text:'',status:'completed',truncated:false,
      details:{ type:'mcpToolCall',server:'fixture',tool:'observe',result:{ content:[
        { type:'text',text:'{"imageTitle":{"en":"Camera","zh":"\\u76f8\\u673a"}}' },
        { type:'image',mimeType:'image/png',data:image,width:1,height:1 },
      ] } } })
  }
  items.push({ key:JSON.stringify(['live','answer']),turnId:'live',id:'answer',role:'assistant',title:'',text:'Live answer',status:'running',truncated:false })
  return { ...emptyStream('fixture-session','codex'),worker:'running',activeTurnId:'live',items }
}
function Fixture() {
  const [state,setState] = useState(initialState)
  const [active,setActive] = useState(true)
  const [locale,setLocale] = useState('en')
  const current = useRef(state);current.current = state
  const last = useRef(null)
  useEffect(() => {
    const send = (values) => {
      const event = { schemaVersion:AGENT_RUNTIME_SCHEMA,sessionId:'fixture-session',provider:'codex',seq:current.current.cursor+1,...values }
      const next = applyEvent(current.current,event);current.current = next;setState(next)
      return event
    }
    window.nativeHarness = {
      delta() {
        last.current = send({ kind:'item.delta',turnId:'live',itemId:'answer',role:'assistant',text:' more' })
      },
      duplicate() { if (last.current) setState(applyEvent(current.current,last.current)) },
      hide() { setActive(false) },reveal() { setActive(true) },
      locale(value) { setLocale(value) },
      disconnect() { send({ kind:'session.state',status:'disconnected' }) },
      newTurn() { send({ kind:'session.state',status:'running',turnId:'next-turn' }) },
      newMessage() { send({ kind:'item.delta',turnId:'next-turn',itemId:'answer',role:'assistant',text:'New turn reply' }) },
    }
    return () => { delete window.nativeHarness }
  },[])
  return <div style={{ height:680,width:620 }}><AgentConversation state={state} active={active} locale={locale} onAnswer={async () => undefined} /></div>
}
document.head.insertAdjacentHTML('beforeend','<style>body{margin:12px;font-family:Arial,sans-serif}[data-xgc-role="agent-conversation"]{height:100%}</style>')
createRoot(document.getElementById('root')).render(<Fixture />)
