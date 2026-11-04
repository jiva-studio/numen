/**
 * The menu that links a tab to another: the tab it was requested on, its position, and
 * the effect of choosing an item. It lists the tabs of this window and the linkable tabs
 * of every other window.
 */
import { computed, ref } from 'vue'
import { getAllTabs, getEligibleTabs, getLinkedTitle } from '../lib/eligibility'
import type { TabId, TabLinksState } from '../types'

export interface TabLinkMenuDeps {
  readonly tabLinks: TabLinksState
  /** The tabs of this window. */
  getTabs(): readonly { readonly id: TabId; readonly title: string }[]
  getKind(tab: TabId): string | undefined
  /** Whether a tab of this window can be linked. */
  isLinkable(tab: TabId): boolean
  onDetach?(tab: TabId, position: { x: number; y: number }): void
}

export function useTabLinkMenu(deps: TabLinkMenuDeps) {
  const tabId = ref<TabId | null>(null)
  const coords = ref<{ x: number; y: number }>({ x: 0, y: 0 })

  const getTabs = () => getAllTabs(deps.getTabs(), deps.tabLinks.tabs)
  const getKind = (tab: TabId): string | undefined =>
    deps.getKind(tab) ?? deps.tabLinks.tabs.find((one) => one.id === tab)?.kind
  const isLinkable = (tab: TabId): boolean =>
    deps.isLinkable(tab) || deps.tabLinks.tabs.some((one) => one.id === tab)

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

  /** Opens the menu on a tab; a tab that can neither be linked nor detached has none. */
  const open = (tab: TabId, where: { x: number; y: number }): void => {
    if (!deps.isLinkable(tab) && !deps.onDetach) return
    tabId.value = tab
    coords.value = where
  }

  const close = (): void => {
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
    if (tabId.value === null) return
    const target = tabId.value
    const where = coords.value
    close()
    deps.onDetach?.(target, where)
  }

  const api = {
    tabId,
    at: coords,
    eligibleTabs,
    linkedTargetTitle,
    isLinked,
    canDetach,
    open,
    close,
    link,
    unlink,
    detach,
  }
  return api
}
