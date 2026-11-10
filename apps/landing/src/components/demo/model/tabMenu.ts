import { computed, ref, type Ref } from 'vue'
import type { MenuItem, Position } from '@numen/ui'
import type { Tab } from '@numen/workbench'
import { getKindOfTab, isTabLinkable } from '../lib/tabs'
import type { TabLinks } from './tabLinks'

export const UNLINK_ITEM = '__unlink__'

/** The menu of a tab: to link it with another tab that follows the same note, or to unlink it. */
export function createTabMenu(tabs: Ref<readonly Tab[]>, links: TabLinks) {
  const tab = ref<string | null>(null)
  const at = ref<Position>({ x: 0, y: 0 })

  const eligible = computed(() => {
    const from = tab.value
    if (from === null) return []
    const group = links.getGroupOf(from)
    return tabs.value.filter(
      (one) => one.id !== from && !group?.tabs.includes(one.id) && isTabLinkable(one.id),
    )
  })

  const items = computed<readonly MenuItem[]>(() => {
    const from = tab.value
    if (from === null) return []
    const result: MenuItem[] = []
    if (links.getGroupOf(from) !== null) {
      const titles = links
        .getLinkedTargets(from)
        .map((id) => tabs.value.find((one) => one.id === id)?.title ?? id)
      result.push({
        id: UNLINK_ITEM,
        text: 'Unlink tab',
        ...(titles.length > 0 ? { detail: `Linked with ${titles.join(', ')}` } : {}),
      })
    }
    for (const one of eligible.value) {
      result.push({
        id: one.id,
        text: one.title || getKindOfTab(one.id) || one.id,
        detail: getKindOfTab(one.id) ?? '',
        group: 'Link with tab',
      })
    }
    return result
  })

  const close = () => {
    tab.value = null
  }

  return {
    tab,
    at,
    items,
    isLinkTarget: (id: string): boolean => eligible.value.some((one) => one.id === id),
    open(id: string, position: Position) {
      if (!isTabLinkable(id)) return
      tab.value = id
      at.value = position
    },
    close,
    choose(id: string) {
      const from = tab.value
      if (from === null) return
      if (id === UNLINK_ITEM) links.unlink(from)
      else links.link(from, id)
      close()
    },
  }
}

export type TabMenu = ReturnType<typeof createTabMenu>
