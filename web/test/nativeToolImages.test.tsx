import { cleanup,render,screen } from '@testing-library/react'
import { afterEach,describe,expect,it } from 'vitest'
import { NativeToolImages } from '../src/NativeToolImages.js'
import { nativeConversationModel } from '../src/nativePresentation.js'
import { applyEvent,emptyStream,NATIVE_SCHEMA } from '../src/state.js'

const data='iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+a3ZkAAAAASUVORK5CYII='
const event={schemaVersion:NATIVE_SCHEMA,sessionId:'s_view',provider:'codex',seq:1,kind:'item.snapshot',turnId:'t_view',itemId:'image-one',role:'tool',status:'completed',text:'',title:'xgc2_view',details:{type:'mcpToolCall',server:'xgc2',tool:'xgc2_view',result:{content:[
 {type:'text',text:JSON.stringify({imageTitle:{en:'Camera source image',zh:'相机源图'},observedAt:'2026-09-10T01:00:00Z'})},
 {type:'image',mimeType:'image/png',data,width:1,height:1},
]}}}
afterEach(cleanup)
describe('MCP image in conversation',()=>{
 it('decodes replay, maps and renders the exact received image without a new capture',()=>{
  const state=applyEvent(emptyStream('s_view','codex'),JSON.parse(JSON.stringify(event)))
  const item=nativeConversationModel(state,'zh').items[0]
  if(item.kind!=='work')throw new Error('expected tool row')
  const images=item.toolData!.images!
  expect(images[0].src).toBe(`data:image/png;base64,${data}`)
  expect(JSON.stringify(item.toolData!.result)).not.toContain(data)
  render(<NativeToolImages images={images} identity={item.id}/>)
  expect(screen.getByAltText('相机源图').getAttribute('src')).toBe(`data:image/png;base64,${data}`)
  expect(document.querySelector('time')?.dateTime).toBe('2026-09-10T01:00:00Z')
  expect(document.querySelector('[data-xgc-role="native-agent-tool-thumbnail"]')).not.toBeNull()
 })
 it('rejects malformed image content while decoding the journal',()=>{
  const bad=JSON.parse(JSON.stringify(event));bad.details.result.content[1].mimeType='image/svg+xml'
  expect(()=>applyEvent(emptyStream('s_view','codex'),bad)).toThrow()
 })
})
