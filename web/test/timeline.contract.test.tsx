import { useState } from 'react'
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { TimelineItem } from '../src/upstream/t3/types.js'
import { EMPTY_NATIVE_TIMELINE_STATE, AgentTimelineStateProvider, useTimelineDisclosure } from '../src/timelineState.js'
import { MessagesTimeline } from '../src/upstream/t3/MessagesTimeline.js'

// This probe tests the shared virtualizer contract, NOT rendering performance.
// The unmocked 240+ heterogeneous-row browser check is test/browser/streaming-chat.mjs.
const probe = vi.hoisted(() => vi.fn())
vi.mock('@legendapp/list/react', () => ({ LegendList: (props: unknown) => {
  probe(props)
  return <div data-testid="legend-scrollport" />
} }))
afterEach(() => { cleanup(); probe.mockClear() })
const items: TimelineItem[] = Array.from({ length: 150 }, (_, index) => index % 3 === 0
  ? { kind: 'custom', id: `domain-${index}`, content: <div>Decision or remote card {index}</div>, keepMounted: index === 0 }
  : { kind: 'message', id: `message-${index}`, role: index % 2 ? 'user' : 'assistant', text: `Message ${index}` })
function latest() { return probe.mock.calls.at(-1)![0] }

describe('one shared LegendList for the complete timeline', () => {
  it('receives 100+ mixed rows, a stable identity and only live portal keepalives', () => {
    render(<MessagesTimeline items={items} active />)
    const props = latest()
    expect(props.data).toBe(items)
    expect(props.data).toHaveLength(150)
    expect(props.keyExtractor(items[0])).toBe('domain-0')
    expect(props.alwaysRender.keys).toEqual(['domain-0'])
    expect(props.maintainVisibleContentPosition).toEqual({ data: true, size: true })
    // Bottom following is owned by the wrapper, never delegated to the
    // virtualizer (its bottom-aligned scroll target clamps user scroll-aways).
    expect(props.maintainScrollAtEnd ?? undefined).toBeUndefined()
    expect(props.initialScrollAtEnd ?? undefined).toBeUndefined()
  })
  it('stops following the end when the operator scrolls up and resumes at the end', () => {
    function Host({ rows }: { rows: TimelineItem[] }) {
      const [state, change] = useState(EMPTY_NATIVE_TIMELINE_STATE)
      return <MessagesTimeline items={rows} active timelineState={state} onTimelineStateChange={change} />
    }
    const view = render(<Host rows={items} />)
    const viewport = screen.getByTestId('legend-scrollport')
    Object.defineProperties(viewport, { scrollHeight: { value: 10000 }, clientHeight: { value: 600 }, scrollTop: { value: 8000, writable: true } })
    fireEvent.scroll(viewport)
    const grown = [...items, { kind: 'message', id: 'message-new', role: 'assistant', text: 'New token' } as TimelineItem]
    view.rerender(<Host rows={grown} />)
    expect(viewport.scrollTop).toBe(8000)
    viewport.scrollTop = 9400
    fireEvent.scroll(viewport)
    view.rerender(<Host rows={[...grown, { kind: 'message', id: 'message-next', role: 'user', text: 'Again' } as TimelineItem]} />)
    expect(viewport.scrollTop).toBe(9400)
  })
  it('keeps disclosure state when its virtual row is unmounted', () => {
    function Row() {
      const [expanded, toggle] = useTimelineDisclosure('long-user-message')
      return <button onClick={toggle} aria-expanded={expanded}>Disclosure</button>
    }
    const view = render(<AgentTimelineStateProvider><Row /></AgentTimelineStateProvider>)
    fireEvent.click(screen.getByRole('button'))
    view.rerender(<AgentTimelineStateProvider>{null}</AgentTimelineStateProvider>)
    view.rerender(<AgentTimelineStateProvider><Row /></AgentTimelineStateProvider>)
    expect(screen.getByRole('button').getAttribute('aria-expanded')).toBe('true')
  })
})
