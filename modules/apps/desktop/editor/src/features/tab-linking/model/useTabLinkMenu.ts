/**
 * The menu that links a tab to another: which tab it was asked on, where, and
 * what choosing an item does.
 */
import { computed, ref } from 'vue'
import { getEligibleTabs, getLinkedTitle } from '../lib/eligibility'
import type { TabId, TabLinksState } from '../types'

export interface TabLinkMenuDeps {
  readonly tabLinks: TabLinksState
  getTabs(): readonly { readonly id: TabId; readonly title: string }[]
  getKind(tab: TabId): string | undefined
  isLinkable(tab: TabId): boolean
}

export function useTabLinkMenu(deps: TabLinkMenuDeps) {
  const tabId = ref<TabId | null>(null)
  const coords = ref<{ x: number; y: number }>({ x: 0, y: 0 })

  const eligibleTabs = computed(() =>
    tabId.value === null
      ? []
      : getEligibleTabs({
          from: tabId.value,
          tabs: deps.getTabs(),
          getKind: deps.getKind,
          isLinkable: deps.isLinkable,
          getGroupOf: deps.tabLinks.getGroupOf,
        }),
  )

  const linkedTargetTitle = computed(() =>
    tabId.value === null
      ? ''
      : getLinkedTitle(deps.tabLinks.getLinkedTargets(tabId.value), deps.getTabs()),
  )

  const isLinked = computed(
    () => tabId.value !== null && deps.tabLinks.getGroupOf(tabId.value) !== null,
  )

  /** The menu asked for on a tab; a tab of a kind that cannot be linked has none. */
  const open = (tab: TabId, where: { x: number; y: number }): void => {
    if (!deps.isLinkable(tab)) return
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

  const api = {
    tabId,
    at: coords,
    eligibleTabs,
    linkedTargetTitle,
    isLinked,
    open,
    close,
    link,
    unlink,
  }
  return api
}
