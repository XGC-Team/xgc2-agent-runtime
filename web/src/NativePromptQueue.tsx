import { useState } from 'react'
import { GripVertical, Pencil, X, Check, ArrowUp, ArrowDown, Play, Pause } from 'lucide-react'
import { Button } from './upstream/t3/ui/button.js'

export type PromptQueueRow = {id: string; text: string; local?: boolean; sending?: boolean; error?: string}
export function NativePromptQueue({items,paused,locale='en',disabled=false,onEdit,onRemove,onReorder,onPause,onRetry}:{
  items: readonly PromptQueueRow[]; paused: boolean; locale?: 'en'|'zh'; disabled?: boolean
  onEdit:(id:string,text:string)=>Promise<unknown>; onRemove:(id:string)=>Promise<unknown>
  onReorder:(order:string[])=>Promise<unknown>; onPause:(paused:boolean)=>Promise<unknown>; onRetry:(id:string)=>Promise<unknown>
}) {
  const [editing,setEditing]=useState(''),[draft,setDraft]=useState(''),[dragged,setDragged]=useState(''),[busy,setBusy]=useState(false),[error,setError]=useState('')
  const zh=locale==='zh'
  const act=async(action:()=>Promise<unknown>)=>{if(busy||disabled)return;setBusy(true);setError('');try{await action()}catch(e){setError(e instanceof Error?e.message:String(e))}finally{setBusy(false)}}
  if(!items.length)return null
  const move=(id:string,to:string)=>{const order=items.filter(p=>!p.local).map(p=>p.id);const from=order.indexOf(id),index=order.indexOf(to);if(from<0||index<0||from===index)return;order.splice(from,1);order.splice(index,0,id);void act(()=>onReorder(order))}
  return <section className="max-h-48 overflow-y-auto rounded-xl border border-border bg-background p-2 text-xs" data-xgc-role="native-agent-prompt-queue" aria-label={zh?'待发送消息':'Queued messages'}>
    <div className="flex items-center justify-between gap-2 px-1 pb-1"><span>{paused?(zh?'已暂停':'Paused'):(zh?'待发送':'Queued')}</span>
      {items.some(p=>!p.local)?<Button size="icon-xs" variant="ghost" disabled={disabled||busy} aria-label={paused?(zh?'继续发送队列':'Resume queue'):(zh?'暂停队列':'Pause queue')} data-xgc-role="native-agent-queue-pause" onClick={()=>void act(()=>onPause(!paused))}>{paused?<Play className="size-3"/>:<Pause className="size-3"/>}</Button>:null}
    </div>
    {items.map((item,index)=><div key={item.id} className="border-t border-border py-1" data-xgc-role="native-agent-queued-message" data-xgc-id={item.id}
      onDragOver={event=>{if(dragged&&!item.local)event.preventDefault()}} onDrop={event=>{event.preventDefault();move(dragged,item.id);setDragged('')}}>
      <div className="flex items-center gap-1">
        <button type="button" draggable={!disabled&&!busy&&!item.local} disabled={disabled||busy||item.local} aria-label={zh?'拖动排序':'Drag to reorder'} data-xgc-role="native-agent-queue-drag" data-xgc-id={item.id} className="cursor-grab p-1 text-muted-foreground disabled:opacity-30"
          onDragStart={event=>{setDragged(item.id);event.dataTransfer.setData('text/plain',item.id);event.dataTransfer.effectAllowed='move'}} onDragEnd={()=>setDragged('')}><GripVertical className="size-3"/></button>
        <span className="min-w-0 flex-1 truncate" title={item.text}>{item.text}</span>
        {!item.local?<><Button size="icon-xs" variant="ghost" disabled={disabled||busy||index===0||Boolean(items[index-1]?.local)} aria-label={zh?'上移':'Move up'} onClick={()=>move(item.id,items[index-1]!.id)}><ArrowUp className="size-3"/></Button><Button size="icon-xs" variant="ghost" disabled={disabled||busy||index===items.length-1||Boolean(items[index+1]?.local)} aria-label={zh?'下移':'Move down'} onClick={()=>move(item.id,items[index+1]!.id)}><ArrowDown className="size-3"/></Button></>:null}
        {item.error?<Button size="xs" variant="ghost" disabled={disabled||busy} onClick={()=>void act(()=>onRetry(item.id))}>{zh?'重试':'Retry'}</Button>:null}
        <Button size="icon-xs" variant="ghost" disabled={disabled||busy||item.sending} aria-label={zh?'编辑消息':'Edit message'} data-xgc-role="native-agent-queue-edit" data-xgc-id={item.id} onClick={()=>{setEditing(item.id);setDraft(item.text)}}><Pencil className="size-3"/></Button>
        <Button size="icon-xs" variant="ghost" disabled={disabled||busy||item.sending} aria-label={zh?'撤销消息':'Remove message'} data-xgc-role="native-agent-queue-remove" data-xgc-id={item.id} onClick={()=>void act(()=>onRemove(item.id))}><X className="size-3"/></Button>
      </div>
      {item.error?<p role="alert" className="px-1 text-destructive">{item.error}</p>:null}
      {editing===item.id?<div className="flex gap-1 p-1"><textarea autoFocus value={draft} onChange={e=>setDraft(e.target.value)} className="min-h-16 min-w-0 flex-1 rounded-md border border-input bg-background p-2 text-foreground" aria-label={zh?'编辑待发送消息':'Edit queued message'} data-xgc-role="native-agent-queue-editor" data-xgc-id={item.id}/><Button size="icon-xs" variant="ghost" disabled={disabled||busy||!draft.trim()} aria-label={zh?'保存修改':'Save message'} onClick={()=>void act(async()=>{await onEdit(item.id,draft);setEditing('')})}><Check className="size-3"/></Button><Button size="icon-xs" variant="ghost" aria-label={zh?'取消编辑':'Cancel edit'} onClick={()=>setEditing('')}><X className="size-3"/></Button></div>:null}
    </div>)}
    {error?<p role="alert" className="text-destructive">{error}</p>:null}
  </section>
}
