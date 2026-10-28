import assert from 'node:assert/strict'
import { createHash } from 'node:crypto'
import { readFile,writeFile } from 'node:fs/promises'
import { dirname,resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { createServer } from 'vite'

const fixtureDir = dirname(fileURLToPath(import.meta.url))
const consumer = process.env.XGC_NATIVE_CONSUMER_ROOT
if (!consumer) throw new Error('XGC_NATIVE_CONSUMER_ROOT must identify an isolated npm-installed consumer')
const installed = resolve(consumer,'node_modules/@xgc2/agent-runtime/dist')
const sourceDist = resolve(fixtureDir,'../../dist')
const hash = async path => createHash('sha256').update(await readFile(path)).digest('hex')
for (const file of ['state.js','fingerprint.js']) {
  assert.equal(await hash(resolve(installed,file)),await hash(resolve(sourceDist,file)),`${file} differs from the built source package`)
}
await writeFile(resolve(consumer,'fingerprint-memory.html'),`<!doctype html><meta charset="utf-8"><script type="module">
  import { AGENT_RUNTIME_SCHEMA,applyEvent,emptyStream } from '@xgc2/agent-runtime/state';
  window.projectLargeEvents = () => {
    let state = emptyStream('s_test','codex');
    for (let seq = 1; seq <= 64; seq++) {
      const text = 'x'.repeat(3 * 1024 * 1024 - 1024) + String(seq);
      state = applyEvent(state,{schemaVersion:AGENT_RUNTIME_SCHEMA,sessionId:'s_test',provider:'codex',seq,kind:'item.snapshot',turnId:'t_a',itemId:'m' + seq,role:'assistant',text});
    }
    window.fingerprintState = state;
    return {cursor:state.cursor,items:state.items.length,displayChars:state.items.reduce((n,item) => n + item.text.length,0),
      digestBytes:Object.values(state.recent).reduce((n,digest) => n + digest.buffer.byteLength,0)};
  };
</script>`)
const server = await createServer({root:consumer,configFile:false,logLevel:'error',server:{host:'127.0.0.1',port:0}})
await server.listen()
const address = server.httpServer.address()
assert(address && typeof address !== 'string')
const playwright = await import(process.env.XGC_PLAYWRIGHT_MODULE || 'playwright')
let browser
try {
  browser = await playwright.chromium.launch({headless:true,...(process.env.XGC_CHROMIUM_PATH ? {executablePath:process.env.XGC_CHROMIUM_PATH} : {})})
  const page = await browser.newPage()
  await page.goto(`http://127.0.0.1:${address.port}/fingerprint-memory.html`)
  await page.waitForFunction(() => typeof window.projectLargeEvents === 'function')
  const cdp = await page.context().newCDPSession(page)
  await cdp.send('HeapProfiler.collectGarbage')
  const before = (await cdp.send('Runtime.getHeapUsage')).usedSize
  const projection = await page.evaluate(() => window.projectLargeEvents())
  await cdp.send('HeapProfiler.collectGarbage')
  const retainedHeap = (await cdp.send('Runtime.getHeapUsage')).usedSize - before
  assert.deepEqual(projection,{cursor:64,items:64,displayChars:64 * 262144,digestBytes:2048})
  assert(retainedHeap < 64 * 1024 * 1024,`retained ${retainedHeap} browser heap bytes`)
  process.stdout.write(JSON.stringify({...projection,retainedHeap})+'\n')
} finally {
  await browser?.close()
  await server.close()
}
