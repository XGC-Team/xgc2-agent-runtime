import {cleanup,fireEvent,render,screen,waitFor} from '@testing-library/react'
import {afterEach,describe,expect,it,vi} from 'vitest'
import {NativePromptQueue} from '../src/NativePromptQueue.js'
afterEach(cleanup)
describe('queued messages',()=>{
 it('edits only after save and exposes removal and accessible reordering',async()=>{
  const edit=vi.fn(async()=>undefined),remove=vi.fn(async()=>undefined),reorder=vi.fn(async()=>undefined)
  render(<NativePromptQueue items={[{id:'a',text:'first'},{id:'b',text:'second'}]} paused onEdit={edit} onRemove={remove} onReorder={reorder} onPause={async()=>undefined} onRetry={async()=>undefined}/>)
  fireEvent.click(screen.getAllByRole('button',{name:'Edit message'})[0]!)
  fireEvent.change(screen.getByRole('textbox',{name:'Edit queued message'}),{target:{value:'edited'}})
  expect(edit).not.toHaveBeenCalled()
  fireEvent.click(screen.getByRole('button',{name:'Save message'}));await waitFor(()=>expect(edit).toHaveBeenCalledWith('a','edited'))
  await waitFor(()=>expect(screen.queryByRole('textbox')).toBeNull())
  fireEvent.click(screen.getAllByRole('button',{name:'Move down'})[0]!);await waitFor(()=>expect(reorder).toHaveBeenCalledWith(['b','a']))
  await waitFor(()=>expect(screen.getAllByRole('button',{name:'Remove message'})[0]?.hasAttribute('disabled')).toBe(false))
  fireEvent.click(screen.getAllByRole('button',{name:'Remove message'})[1]!);await waitFor(()=>expect(remove).toHaveBeenCalledWith('b'))
 })
})
