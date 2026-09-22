import assert from 'node:assert/strict'
import { mkdir, writeFile } from 'node:fs/promises'
import { fileURLToPath } from 'node:url'
import { createServer } from 'vite'
import { chromium } from 'playwright'

const root = fileURLToPath(new URL('../../', import.meta.url))
const output = new URL('../../test-results/streaming-chat/', import.meta.url)
await mkdir(output, { recursive: true })
const server = await createServer({ root, configFile: false, esbuild: { jsx: 'automatic' }, server: { host: '127.0.0.1', port: 0 } })
await server.listen()
const address = server.httpServer.address()
assert(address && typeof address !== 'string')
const browser = await chromium.launch({ headless: true })
const page = await browser.newPage({ viewport: { width: 1100, height: 850 } })
const errors = []
page.on('pageerror', error => errors.push(String(error)))
const metrics = { inputRows: 240 }
try {
  await page.goto(`http://127.0.0.1:${address.port}/test/browser/streaming-chat.html`)
  await page.locator('[data-xgc-role="native-agent-item"]').first().waitFor()
  await page.evaluate(() => {
    let node = document.querySelector('[data-xgc-role="native-agent-item"]')?.parentElement
    while (node && !/auto|scroll/.test(getComputedStyle(node).overflowY)) node = node.parentElement
    if (!node) throw new Error('LegendList did not create a scrollport')
    node.dataset.fixtureScrollport = 'true'
  })
  const scrollport = '[data-fixture-scrollport="true"]'
  await page.waitForFunction(selector => {
    const node = document.querySelector(selector)
    return node && node.scrollHeight - node.scrollTop - node.clientHeight < 5
  }, scrollport)
  metrics.initialMountedRows = await page.locator('[data-xgc-role="native-agent-item"]').count()
  assert(metrics.initialMountedRows < 100, `Not virtualized: ${metrics.initialMountedRows} mounted rows`)
  await page.locator('[data-fixture-live="true"]').waitFor({ state: 'attached' })
  const fixedRects = () => page.evaluate(() => ['[data-fixture-fixed="approval"]', '[data-fixture-fixed="dock"]', '[data-xgc-role="native-agent-composer"]']
    .map(selector => { const rect = document.querySelector(selector).getBoundingClientRect(); return { top: rect.top, height: rect.height } }))
  const beforeFixed = await fixedRects()
  await page.locator('[data-fixture-action="append"]').click()
  await page.waitForFunction(selector => {
    const node = document.querySelector(selector)
    return node.scrollHeight - node.scrollTop - node.clientHeight < 5
  }, scrollport)
  // Scroll away like a real user and assert the gesture actually landed:
  // bottom following must release the viewport instead of yanking it back.
  const scrollportBox = await page.locator(scrollport).boundingBox()
  await page.mouse.move(scrollportBox.x + scrollportBox.width / 2, scrollportBox.y + scrollportBox.height / 2)
  await page.mouse.wheel(0, -350)
  await page.waitForFunction(selector => {
    const node = document.querySelector(selector)
    return node.scrollHeight - node.scrollTop - node.clientHeight > 100
  }, scrollport)
  await page.evaluate(() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))))
  const beforeAppend = await page.locator(scrollport).evaluate(node => node.scrollTop)
  await page.locator('[data-fixture-action="append"]').click()
  await page.evaluate(() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))))
  const afterAppend = await page.locator(scrollport).evaluate(node => node.scrollTop)
  assert(Math.abs(afterAppend - beforeAppend) < 5, `Append stole scroll: ${beforeAppend} -> ${afterAppend}`)
  const afterFixed = await fixedRects()
  beforeFixed.forEach((rect, index) => {
    assert(Math.abs(rect.top - afterFixed[index].top) < 2, `Fixed area ${index} moved with timeline`)
    assert(Math.abs(rect.height - afterFixed[index].height) < 2, `Fixed area ${index} resized with timeline`)
  })
  const samples = await page.locator(scrollport).evaluate(async node => {
    const durations = []
    let maxMountedRows = 0
    for (let i = 0; i < 100; i++) {
      const started = performance.now()
      node.scrollTop = (node.scrollHeight - node.clientHeight) * ((i % 40) / 40)
      await new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve)))
      durations.push((performance.now() - started) / 2)
      maxMountedRows = Math.max(maxMountedRows, document.querySelectorAll('[data-xgc-role="native-agent-item"]').length)
    }
    return { durations, maxMountedRows }
  })
  const sorted = samples.durations.slice().sort((a, b) => a - b)
  metrics.frames = sorted.length * 2
  metrics.p50FrameMs = sorted[Math.floor(sorted.length * 0.50)]
  metrics.p95FrameMs = sorted[Math.floor(sorted.length * 0.95)]
  metrics.maxMountedRows = samples.maxMountedRows
  metrics.scrollRetained = Math.abs(afterAppend - beforeAppend) < 5
  metrics.fixedAreasRetained = true
  assert(samples.maxMountedRows < 100, `Mounted row budget exceeded: ${samples.maxMountedRows}`)
  assert(metrics.p95FrameMs < 50, `Long-list p95 frame interval exceeds 50ms: ${metrics.p95FrameMs}`)
  assert.deepEqual(errors, [])
  await page.screenshot({ path: fileURLToPath(new URL('timeline.png', output)) })
  console.log(JSON.stringify(metrics, null, 2))
} finally {
  await writeFile(new URL('metrics.json', output), JSON.stringify({ ...metrics, errors }, null, 2))
  await browser.close()
  await server.close()
}
