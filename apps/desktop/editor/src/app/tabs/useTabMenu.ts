/**
 * The menu of a tab, which the tab strip draws: the rows that link, detach and move the tab, then
 * the entries of the slot `tabStrip.context` over the file of the tab.
 */
import { computed, type Component } from 'vue'
import type { MenuItem } from '@numen/ui'
import type { TabMenu } from '@numen/shell'
import { SLOT, type SlotHost, type SlotItem, type SlotTarget } from '@numen/plugin-host'
import type { useWindowTabs } from '@/entities/tab'
import { AppWindow, ExternalLink, Link2Off } from '@lucide/vue'
import {
  createLinkMenuItems,
  DETACH_ID,
  getMoveTarget,
  MOVE_ID,
  UNLINK_ID,
  useTabLinkMenu,
  type TabLinkMenuDeps,
} from '@/features/tab-linking'

/** The prefix of the group of the entries of the slot, kept apart from the groups of the rows. */
const SLOT_GROUP = 'slot:'

export interface TabMenuDeps {
  readonly link: TabLinkMenuDeps
  /** The slots of the window; without them the menu holds the rows of linking alone. */
  readonly slots?: SlotHost | undefined
  /** The file a tab shows, which the commands of the entries run over. */
  getTabTarget(tab: string): SlotTarget
  getTabIcon(tab: string): Component | null
  getKindIcon(kind: string): Component | null
  /** Receives the failure of a command that runs after the menu closed. */
  reportError?(error: unknown): void
}

type SlotRowEntry = Exclude<SlotItem, { kind: 'separator' }>

/** The rows of the entries of the slot; the rows between two separators share a group. */
const createSlotItems = (entries: readonly SlotItem[], group = 0): readonly MenuItem[] => {
  const items: MenuItem[] = []
  for (const entry of entries) {
    if (entry.kind === 'separator') {
      group += 1
      continue
    }
    const groupName = `${SLOT_GROUP}${group}`
    if (entry.kind === 'view') {
      const component = entry.component as Component
      items.push({ id: entry.key, text: entry.view, group: groupName, component })
      continue
    }
    items.push({
      id: entry.key,
      text: entry.title,
      group: groupName,
      ...(entry.kind === 'submenu' ? { children: createSlotItems(entry.entries, group) } : {}),
    })
  }
  return items
}

/** The entry drawn as the row `id`, inside a submenu too. */
const findSlotEntry = (entries: readonly SlotItem[], id: string): SlotRowEntry | undefined => {
  for (const entry of entries) {
    if (entry.kind === 'separator') continue
    if (entry.key === id) return entry
    if (entry.kind === 'submenu') {
      const found = findSlotEntry(entry.entries, id)
      if (found) return found
    }
  }
  return undefined
}

export function useTabMenu(deps: TabMenuDeps) {
  const getEntries = (tab: string): readonly SlotItem[] =>
    deps.slots?.getEntries(SLOT.tabStripContext, deps.getTabTarget(tab)) ?? []

  const link = useTabLinkMenu({
    ...deps.link,
    hasOtherRows: (tab) => getEntries(tab).length > 0,
  })

  const entries = computed(() => (link.tabId.value === null ? [] : getEntries(link.tabId.value)))

  const menu = computed<TabMenu | null>(() => {
    const tab = link.tabId.value
    if (tab === null) return null
    const rows = createLinkMenuItems({
      eligibleTabs: link.eligibleTabs.value,
      isLinked: link.isLinked.value,
      linkedTargetTitle: link.linkedTargetTitle.value,
      canDetach: link.canDetach.value,
      windows: link.windows.value,
      isAsking: link.isAsking.value,
    })
    return { tab, at: link.at.value, items: [...rows, ...createSlotItems(entries.value)] }
  })

  /** Runs the command of an entry over the file of the tab, or the row of linking chosen. */
  const chooseRow = async (tab: string, id: string): Promise<void> => {
    if (link.tabId.value !== tab) return
    const entry = findSlotEntry(entries.value, id)
    if (entry) {
      link.close()
      if (entry.kind === 'command')
        await deps.slots?.runCommand(entry.command, deps.getTabTarget(tab))
      return
    }
    const windowId = getMoveTarget(id)
    if (windowId !== null) link.move(windowId)
    else if (id === DETACH_ID) link.detach()
    else if (id === UNLINK_ID) link.unlink()
    else if (id !== MOVE_ID) link.link(id)
  }

  /** The icon a row is drawn with: its command's, the window's, or the tab's it links to. */
  const getIcon = (id: string): Component | null => {
    const entry = findSlotEntry(entries.value, id)
    if (entry) return entry.kind === 'view' ? null : ((entry.icon as Component | undefined) ?? null)
    if (id === DETACH_ID) return ExternalLink
    if (id === UNLINK_ID) return Link2Off
    if (id === MOVE_ID) return AppWindow
    const windowId = getMoveTarget(id)
    if (windowId !== null) {
      const kind = link.windows.value.find((one) => one.id === windowId)?.activeKind
      return kind ? deps.getKindIcon(kind) : null
    }
    const tab = link.eligibleTabs.value.find((one) => one.id === id)
    return tab ? (deps.getTabIcon(id) ?? deps.getKindIcon(tab.kind)) : null
  }

  return {
    menu,
    open: link.open,
    /** Chooses the row `id` of the menu requested on `tab`; a menu since closed runs nothing. */
    choose: (tab: string, id: string): void => {
      chooseRow(tab, id).catch((error: unknown) => deps.reportError?.(error))
    },
    dismiss: link.close,
    highlight: (id: string | null) => link.highlight(getMoveTarget(id)),
    getIcon,
  }
}

/** The tabs of a window, and what is done to them from the menu of a tab. */
export interface WindowTabMenuDeps extends Pick<
  TabLinkMenuDeps,
  'tabLinks' | 'windowId' | 'isAsking' | 'onDetach' | 'onMove' | 'onHighlight'
> {
  readonly windowTabs: Pick<ReturnType<typeof useWindowTabs>, 'tabs' | 'getTab'>
  readonly slots?: SlotHost | undefined
  getTabIcon(tab: string): Component | null | undefined
  getKindIcon(kind: string): Component | null | undefined
}

/** The file a tab of the window shows, as the commands of the menu of the tab run over it. */
export const getTabFileTarget = (
  windowTabs: WindowTabMenuDeps['windowTabs'],
  id: string,
): SlotTarget => {
  const tab = windowTabs.getTab(id)
  const on = tab ? tab.kind.getTarget?.(tab.state) : undefined
  return { path: on?.file || on?.path || '', title: on?.title ?? '', isFolder: false, others: [] }
}

/** The menu of a tab of the window: a tab whose kind navigates can be linked. */
export function useWindowTabMenu(deps: WindowTabMenuDeps) {
  const { windowTabs, slots, getTabIcon, getKindIcon, ...link } = deps
  return useTabMenu({
    link: {
      ...link,
      getTabs: () => windowTabs.tabs.value,
      getKind: (id) => windowTabs.getTab(id)?.kind.kind,
      isLinkable: (id) => windowTabs.getTab(id)?.kind.navigate !== undefined,
    },
    slots,
    getTabTarget: (id) => getTabFileTarget(windowTabs, id),
    getTabIcon: (id) => getTabIcon(id) ?? null,
    getKindIcon: (kind) => getKindIcon(kind) ?? null,
  })
}
