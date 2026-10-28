import { test } from 'node:test'
import assert from 'node:assert/strict'
import { spawnSync } from 'node:child_process'

test('large distinct display items do not retain the source event strings after truncation', () => {
  const script = `
    import { AGENT_RUNTIME_SCHEMA,applyEvent,emptyStream } from './dist/state.js'
    global.gc()
    const before = process.memoryUsage().heapUsed
    function project() {
      let state = emptyStream('s_test','codex')
      for (let seq = 1; seq <= 64; seq++) {
        const text = 'x'.repeat(3 * 1024 * 1024 - 1024) + String(seq)
        state = applyEvent(state,{schemaVersion:AGENT_RUNTIME_SCHEMA,sessionId:'s_test',provider:'codex',seq,kind:'item.snapshot',turnId:'t_a',itemId:'m' + seq,role:'assistant',text})
      }
      return state
    }
    const state = project()
    global.gc()
    process.stdout.write(JSON.stringify({
      retainedHeap:process.memoryUsage().heapUsed - before,
      digestBytes:Object.values(state.recent).reduce((sum,digest) => sum + digest.buffer.byteLength,0),
      displayChars:state.items.reduce((sum,item) => sum + item.text.length,0),
      items:state.items.length,
    }))
  `
  const run = spawnSync(process.execPath,['--expose-gc','--input-type=module','-e',script],{
    cwd:new URL('..',import.meta.url),encoding:'utf8',timeout:30000,maxBuffer:1024 * 1024,
  })
  assert.equal(run.status,0,run.stderr)
  const work = JSON.parse(run.stdout)
  assert.equal(work.items,64)
  assert.equal(work.displayChars,64 * 262144)
  assert.equal(work.digestBytes,2048)
  // The displayed 16 MiB of text and fixed reducer overhead remain. The
  // original 64 near-3 MiB source strings and JSON fingerprints must not.
  assert.ok(work.retainedHeap < 64 * 1024 * 1024,`retained ${work.retainedHeap} heap bytes`)
})
