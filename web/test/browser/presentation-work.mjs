import assert from 'node:assert/strict'
import { createHash } from 'node:crypto'
import { copyFile,readFile } from 'node:fs/promises'
import { dirname,resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { createServer } from 'vite'

const fixtureDir = dirname(fileURLToPath(import.meta.url))
const consumer = process.env.XGC_NATIVE_CONSUMER_ROOT
if (!consumer) throw new Error('XGC_NATIVE_CONSUMER_ROOT must identify an isolated npm-installed consumer')
const installed = resolve(consumer,'node_modules/@xgc2/agent-runtime/dist')
const sourceDist = resolve(fixtureDir,'../../dist')
const hash = async path => createHash('sha256').update(await readFile(path)).digest('hex')
for (const file of ['state.js','agentPresentation.js','AgentConversation.js','timelineState.js','upstream/t3/MessagesTimeline.js','upstream/t3/ChatMarkdown.js']) {
  assert.equal(await hash(resolve(installed,file)),await hash(resolve(sourceDist,file)),`${file} differs from the built source package`)
}
await copyFile(resolve(fixtureDir,'presentation-work.html'),resolve(consumer,'presentation-work.html'))
await copyFile(resolve(fixtureDir,'presentation-work.jsx'),resolve(consumer,'presentation-work.jsx'))
const replace = (text,needle,insert) => {
  assert.equal(text.split(needle).length,2,`instrumentation drift: ${needle}`)
  return text.replace(needle,needle+insert)
}
const instrumentation = {
  name:'installed-agent-work-counters',
  setup(build) {
    build.onLoad({ filter:/\/@xgc2\/agent-runtime\/dist\/.*\.js$/ },async ({ path:file }) => {
      let code = await readFile(file,'utf8')
      if (file.endsWith('/agentPresentation.js')) {
        code = replace(code,'function timelineItem(item, state, locale, projectTool = toolData) {','\n globalThis.__nativeWork.projected++;')
        code = replace(code,'function toolData(item, locale, metadata = readToolMetadata) {','\n globalThis.__nativeWork.toolData++;')
        code = replace(code,'const meta = JSON.parse(block.text);','\n globalThis.__nativeWork.metadataParses++;')
        code = replace(code,'const projectItem = (item, index) => {','\n globalThis.__nativeWork.itemVisits++;')
      } else if (file.endsWith('/upstream/t3/MessagesTimeline.js')) {
        code = replace(code,'function expandedToolBody(entry) {','\n globalThis.__nativeWork.formatted++;')
        code = replace(code,'const PlainWorkEntryRow = memo(function PlainWorkEntryRow({ workEntry }) {','\n globalThis.__nativeWork.workRows++;')
        code = replace(code,'const TimelineRow = memo(function TimelineRow({ item, identity }) {','\n globalThis.__nativeWork.timelineRows++;')
      } else if (file.endsWith('/upstream/t3/ChatMarkdown.js')) {
        code = replace(code,'export const ChatMarkdown = memo(function ChatMarkdown({ text, isStreaming }) {','\n globalThis.__nativeWork.markdown++;')
      }
      return { contents:code,loader:'js' }
    })
  },
}
const server = await createServer({ root:consumer,configFile:false,logLevel:'error',
  optimizeDeps:{ force:true,include:['@xgc2/agent-runtime/react','@xgc2/agent-runtime/state'],esbuildOptions:{ plugins:[instrumentation] } },
  server:{ host:'127.0.0.1',port:0 } })
await server.listen()
const address = server.httpServer.address()
assert(address && typeof address !== 'string')
const playwright = await import(process.env.XGC_PLAYWRIGHT_MODULE || 'playwright')
const browser = await playwright.chromium.launch({ headless:true,...(process.env.XGC_CHROMIUM_PATH ? { executablePath:process.env.XGC_CHROMIUM_PATH } : {}) })
const samples = []
try {
  for (const history of [1,128]) {
    const page = await browser.newPage({ viewport:{ width:1000,height:800 } })
    page.setDefaultTimeout(10000)
    const errors = []
    page.on('pageerror',error => { errors.push(String(error));console.error(error) })
    page.on('console',message => { if (message.type() === 'error') console.error(message.text()) })
    await page.addInitScript(() => { window.__nativeWork = { projected:0,toolData:0,metadataParses:0,itemVisits:0,formatted:0,workRows:0,timelineRows:0,markdown:0 } })
    await page.goto(`http://127.0.0.1:${address.port}/presentation-work.html?history=${history}`)
    await page.waitForFunction(() => Boolean(window.nativeHarness))
    await page.locator('[data-xgc-role="agent-tool-disclosure"]').last().waitFor()
    const initial = await page.evaluate(() => ({ ...window.__nativeWork,mountedTools:document.querySelectorAll('[data-xgc-role="agent-tool-disclosure"]').length }))
    assert.equal(initial.projected,2*history+1);assert.equal(initial.toolData,history);assert.equal(initial.metadataParses,history)
    assert(initial.mountedTools > 0)
    await page.locator('[data-xgc-role="agent-tool-disclosure"]').last().click()
    await page.evaluate(() => {
      let node = document.querySelector('[data-xgc-role="agent-item"]')?.parentElement
      while (node && !/auto|scroll/.test(getComputedStyle(node).overflowY)) node = node.parentElement
      if (!node) throw new Error('Timeline scrollport is unavailable')
      node.scrollTop = node.scrollHeight
    })
    await page.getByText('Live answer').waitFor({ state:'attached' })
    const reset = () => page.evaluate(() => {
      Object.keys(window.__nativeWork).forEach(key => { window.__nativeWork[key]=0 })
      window.__nativeMutations = { tool:0,all:0 }
      window.__nativeObserver?.disconnect()
      window.__nativeObserver = new MutationObserver(records => {
        for (const record of records) {
          window.__nativeMutations.all++
          const element = record.target instanceof Element ? record.target : record.target.parentElement
          const row = element?.closest('[data-xgc-role="agent-item"]')
          if (row?.querySelector('[data-xgc-role="agent-tool-disclosure"]')) window.__nativeMutations.tool++
        }
      })
      window.__nativeObserver.observe(document.querySelector('[data-xgc-role="agent-conversation"]'),{ subtree:true,childList:true,attributes:true,characterData:true })
    })
    const settle = () => page.evaluate(() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))))
    await settle()
    await reset()
    await page.evaluate(() => window.nativeHarness.delta())
    await settle()
    if (!await page.getByText('Live answer more').count()) {
      console.error('Rendered timeline:',await page.locator('[data-xgc-role="agent-conversation"]').innerText())
      console.error('Work counters:',await page.evaluate(() => window.__nativeWork))
    }
    await page.getByText('Live answer more').waitFor({ state:'attached' })
    const delta = await page.evaluate(() => ({ ...window.__nativeWork,mutations:{ ...window.__nativeMutations } }))
    assert.equal(delta.itemVisits,1);assert.equal(delta.projected,1);assert.equal(delta.metadataParses,0);assert.equal(delta.toolData,0)
    assert.equal(delta.formatted,0);assert.equal(delta.workRows,0);assert.equal(delta.timelineRows,1);assert.equal(delta.markdown,1);assert.equal(delta.mutations.tool,0)
    await reset()
    await page.evaluate(() => window.nativeHarness.duplicate())
    await settle()
    const duplicate = await page.evaluate(() => ({ ...window.__nativeWork,mutations:{ ...window.__nativeMutations } }))
    assert.equal(duplicate.itemVisits,0);assert.equal(duplicate.projected,0);assert.equal(duplicate.metadataParses,0);assert.equal(duplicate.workRows,0);assert.equal(duplicate.markdown,0);assert.equal(duplicate.mutations.all,0)
    await page.evaluate(() => window.nativeHarness.hide())
    await settle();await reset()
    await page.evaluate(() => { for (let i=0;i<5;i+=1) window.nativeHarness.delta() })
    await settle()
    const hidden = await page.evaluate(() => ({ ...window.__nativeWork,mutations:{ ...window.__nativeMutations } }))
    assert.equal(hidden.itemVisits,0);assert.equal(hidden.projected,0);assert.equal(hidden.metadataParses,0);assert.equal(hidden.workRows,0);assert.equal(hidden.markdown,0);assert.equal(hidden.mutations.all,0)
    await reset();await page.evaluate(() => window.nativeHarness.reveal());await settle()
    const reveal = await page.evaluate(() => ({ ...window.__nativeWork,mutations:{ ...window.__nativeMutations } }))
    assert(reveal.itemVisits <= 2*history+1);assert.equal(reveal.projected,1);assert.equal(reveal.metadataParses,0);assert.equal(reveal.workRows,0)
    await reset();await page.evaluate(() => window.nativeHarness.disconnect());await settle()
    const disconnected = await page.evaluate(() => ({ ...window.__nativeWork,mutations:{ ...window.__nativeMutations } }))
    assert.equal(disconnected.projected,1);assert.equal(disconnected.metadataParses,0);assert.equal(disconnected.workRows,0)
    await reset();await page.evaluate(() => window.nativeHarness.locale('zh'));await settle()
    const locale = await page.evaluate(() => ({ ...window.__nativeWork,mutations:{ ...window.__nativeMutations } }))
    assert.equal(locale.projected,history);assert.equal(locale.metadataParses,0)
    await reset();await page.evaluate(() => window.nativeHarness.newTurn());await settle()
    const newTurn = await page.evaluate(() => ({ ...window.__nativeWork,mutations:{ ...window.__nativeMutations } }))
    assert.equal(newTurn.projected,0);assert.equal(newTurn.metadataParses,0);assert.equal(newTurn.workRows,0)
    await reset();await page.evaluate(() => window.nativeHarness.newMessage());await settle()
    await page.getByText('New turn reply').waitFor({ state:'attached' })
    const newMessage = await page.evaluate(() => ({ ...window.__nativeWork,mutations:{ ...window.__nativeMutations } }))
    assert.equal(newMessage.itemVisits,1);assert.equal(newMessage.projected,1);assert.equal(newMessage.metadataParses,0);assert.equal(newMessage.workRows,0)
    assert.deepEqual(errors,[])
    samples.push({ history,initial,delta,duplicate,hidden,reveal,disconnected,locale,newTurn,newMessage })
    await page.close()
  }
  console.log(JSON.stringify({ scope:'npm-installed candidate, real Chromium, React and T3 LegendList; in-memory counters on installed module transforms',
    agentPresentationSha256:await hash(resolve(installed,'agentPresentation.js')),samples },null,2))
} finally {
  await browser.close()
  await server.close()
}
