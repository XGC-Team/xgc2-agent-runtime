import { createContext, useCallback, useContext, useLayoutEffect, useMemo, useRef, useState, useSyncExternalStore, type ReactNode } from 'react'

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
type DisclosureStore = {
  expanded: (id:string) => boolean
  subscribe: (id:string,notify:() => void) => () => void
  toggle: (id:string) => void
  notifyChanges: (previous:Readonly<Record<string,boolean>>,next:Readonly<Record<string,boolean>>) => void
}
const emptyDisclosure: DisclosureStore = { expanded:() => false,subscribe:() => () => undefined,toggle:() => undefined,notifyChanges:() => undefined }
const DisclosureContext = createContext<DisclosureStore>(emptyDisclosure)

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
  const current = useRef({ state,change })
  current.current = { state,change }
  const disclosure = useMemo<DisclosureStore>(() => {
    const listeners = new Map<string,Set<() => void>>()
    return {
      expanded: id => Boolean(current.current.state.expandedItems[id]),
      subscribe(id,notify) {
        const group = listeners.get(id) ?? new Set<() => void>()
        group.add(notify);listeners.set(id,group)
        return () => { group.delete(notify);if (!group.size) listeners.delete(id) }
      },
      toggle(id) {
        const { state:visible,change:update } = current.current
        update({ ...visible,expandedItems:{ ...visible.expandedItems,[id]:!visible.expandedItems[id] } })
      },
      notifyChanges(previous,next) {
        for (const [id,group] of listeners) {
          if (Boolean(previous[id]) !== Boolean(next[id])) for (const notify of group) notify()
        }
      },
    }
  },[])
  const previousExpanded = useRef(state.expandedItems)
  useLayoutEffect(() => {
    disclosure.notifyChanges(previousExpanded.current,state.expandedItems)
    previousExpanded.current = state.expandedItems
  },[disclosure,state.expandedItems])
  const value = useMemo(() => ({ state, change }), [state, change])
  return <TimelineStateContext.Provider value={value}><DisclosureContext.Provider value={disclosure}>{children}</DisclosureContext.Provider></TimelineStateContext.Provider>
}
export function useNativeTimelineState() { return useContext(TimelineStateContext) }
export function useTimelineDisclosure(id: string) {
  const store = useContext(DisclosureContext)
  const subscribe = useCallback((notify:() => void) => store.subscribe(id,notify),[id,store])
  const snapshot = useCallback(() => store.expanded(id),[id,store])
  const expanded = useSyncExternalStore(subscribe,snapshot,snapshot)
  const toggle = useCallback(() => store.toggle(id),[id,store])
  return [expanded, toggle] as const
}
