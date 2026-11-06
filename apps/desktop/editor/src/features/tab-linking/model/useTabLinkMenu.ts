/**
 * The menu of a tab: the tab it was requested on, its position, and the effect of choosing an
 * item. Detaches the tab, moves it to another window, or links it to a tab of this window or a
 * linkable tab of another window.
 */
import { computed, ref, type Ref } from 'vue'
import { getAllTabs, getEligibleTabs, getLinkedTitle } from '../lib/eligibility'
import type { TabId, TabLinksState } from '../types'

export interface TabLinkMenuDeps {
  readonly tabLinks: TabLinksState
  /** Returns the tabs of this window. */
  getTabs(): readonly { readonly id: TabId; readonly title: string }[]
  getKind(tab: TabId): string | undefined
  /** Returns whether a tab of this window can be linked. */
  isLinkable(tab: TabId): boolean
  onDetach?(tab: TabId, position: { x: number; y: number }): void
  /** The id of this window, excluded from the move targets. */
  readonly windowId?: string
  /** Whether a tab of this window is asking; an asking tab is not moved or detached. */
  isAsking?(tab: TabId): boolean
  onMove?(tab: TabId, windowId: string): void
  /** Turns the highlight of another window on or off while the pointer is over its item. */
  onHighlight?(windowId: string, isOn: boolean): void
  /** Whether the menu of the tab holds rows besides the rows of linking, detaching and moving. */
  hasOtherRows?(tab: TabId): boolean
}

/** Returns the move targets, the asking state of the tab, and the highlight control. */
function createWindowMoves(deps: TabLinkMenuDeps, tabId: Ref<TabId | null>) {
  /** The other windows, main window first; empty without onMove. */
  const windows = computed(() =>
    deps.onMove ? deps.tabLinks.windows.filter((one) => one.id !== deps.windowId) : [],
  )

  const isAsking = computed(() => tabId.value !== null && (deps.isAsking?.(tabId.value) ?? false))

  let highlightedWindowId: string | null = null

  /** Turns the highlight on for the window and off for the previously highlighted window. */
  const highlight = (windowId: string | null): void => {
    if (windowId === highlightedWindowId) return
    if (highlightedWindowId !== null) deps.onHighlight?.(highlightedWindowId, false)
    highlightedWindowId = windowId
    if (windowId !== null) deps.onHighlight?.(windowId, true)
  }

  return { windows, isAsking, highlight }
}

export function useTabLinkMenu(deps: TabLinkMenuDeps) {
  const tabId = ref<TabId | null>(null)
  const coords = ref<{ x: number; y: number }>({ x: 0, y: 0 })

  const getTabs = () => getAllTabs(deps.getTabs(), deps.tabLinks.tabs)
  const getKind = (tab: TabId): string | undefined =>
    deps.getKind(tab) ?? deps.tabLinks.tabs.find((one) => one.id === tab)?.kind
  const isLinkable = (tab: TabId): boolean =>
    deps.getTabs().some((one) => one.id === tab)
      ? deps.isLinkable(tab)
      : deps.tabLinks.tabs.some((one) => one.id === tab)

  const eligibleTabs = computed(() =>
    tabId.value === null
      ? []
      : getEligibleTabs({
          from: tabId.value,
          tabs: getTabs(),
          getKind,
          isLinkable,
          getGroupOf: deps.tabLinks.getGroupOf,
        }),
  )

  const linkedTargetTitle = computed(() =>
    tabId.value === null
      ? ''
      : getLinkedTitle(deps.tabLinks.getLinkedTargets(tabId.value), getTabs()),
  )

  const isLinked = computed(
    () => tabId.value !== null && deps.tabLinks.getGroupOf(tabId.value) !== null,
  )

  const canDetach = computed(() => deps.onDetach !== undefined)

  const { windows, isAsking, highlight } = createWindowMoves(deps, tabId)

  /** Opens the menu on a tab; a tab with no row to offer has none. */
  const open = (tab: TabId, where: { x: number; y: number }): void => {
    const hasLinkRows = deps.isLinkable(tab) || Boolean(deps.onDetach) || Boolean(deps.onMove)
    if (!hasLinkRows && !deps.hasOtherRows?.(tab)) return
    tabId.value = tab
    coords.value = where
  }

  const close = (): void => {
    highlight(null)
    tabId.value = null
  }

  const link = (target: TabId): void => {
    if (tabId.value === null) return
    deps.tabLinks.linkTabs(tabId.value, target)
    close()
  }

  const unlink = (): void => {
    if (tabId.value === null) return
    deps.tabLinks.unlinkTab(tabId.value)
    close()
  }

  const detach = (): void => {
    const target = tabId.value
    close()
    if (target !== null) deps.onDetach?.(target, coords.value)
  }

  const move = (windowId: string): void => {
    const target = tabId.value
    close()
    if (target !== null) deps.onMove?.(target, windowId)
  }

  const api = {
    tabId,
    at: coords,
    eligibleTabs,
    linkedTargetTitle,
    isLinked,
    canDetach,
    windows,
    isAsking,
    open,
    close,
    link,
    unlink,
    detach,
    move,
    highlight,
  }
  return api
}
