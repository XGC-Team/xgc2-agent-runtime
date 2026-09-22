import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from 'react'

export type AgentTimelineState = {
  pinnedToBottom: boolean
  expandedItems: Readonly<Record<string, boolean>>
}
export const EMPTY_NATIVE_TIMELINE_STATE: AgentTimelineState = { pinnedToBottom: true, expandedItems: {} }
export const TIMELINE_END_THRESHOLD = 0.01
export function isTimelinePinnedToBottom({ scrollHeight, scrollTop, clientHeight }: {
  scrollHeight: number; scrollTop: number; clientHeight: number
}): boolean {
  return scrollHeight - scrollTop - clientHeight <= Math.max(2, clientHeight * TIMELINE_END_THRESHOLD)
}
const TimelineStateContext = createContext<{
  state: AgentTimelineState
  change: (state: AgentTimelineState) => void
}>({ state: EMPTY_NATIVE_TIMELINE_STATE, change: () => undefined })

/** Disclosure state survives a virtual row unmount; a host may own this UI layer. */
export function AgentTimelineStateProvider({ state: controlled, onChange, children }: {
  state?: AgentTimelineState; onChange?: (state: AgentTimelineState) => void; children: ReactNode
}) {
  const [local, setLocal] = useState(EMPTY_NATIVE_TIMELINE_STATE)
  const state = controlled ?? local
  const change = useCallback((next: AgentTimelineState) => {
    if (controlled === undefined) setLocal(next)
    onChange?.(next)
  }, [controlled, onChange])
  const value = useMemo(() => ({ state, change }), [state, change])
  return <TimelineStateContext.Provider value={value}>{children}</TimelineStateContext.Provider>
}
export function useNativeTimelineState() { return useContext(TimelineStateContext) }
export function useTimelineDisclosure(id: string) {
  const { state, change } = useNativeTimelineState()
  const expanded = Boolean(state.expandedItems[id])
  const toggle = () => change({ ...state, expandedItems: { ...state.expandedItems, [id]: !expanded } })
  return [expanded, toggle] as const
}
