/**
 * The menu of a tab, which the tab strip draws: the rows that link, detach and move the tab, then
 * the entries of the slot `tabStrip.context` over the file of the tab.
 */
import { computed, ref, shallowRef, type Component } from 'vue'
import type { TabMenu } from '@numen/shell'
import {
  SLOT,
  type CommandChoice,
  type CommandsHost,
  type CommandTarget as RunTarget,
  type SlotHost,
  type SlotItem,
} from '@numen/plugin-host'
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
import { WORDS } from '@/shared/words'
import {
  createSlotItems,
  findSlotEntry,
  listChoiceCommands,
  parseChoiceRowId,
  type SlotRowEntry,
} from './tabMenuItems'

export interface TabMenuDeps {
  readonly link: TabLinkMenuDeps
  /** The slots of the window; without them the menu holds the rows of linking alone. */
  readonly slots?: SlotHost | undefined
  /** Runs a command of an entry over the tab; with `choice`, the one picked. */
  runCommand?(command: string, target: RunTarget, choice?: string): Promise<void>
  /** The choices a command of an entry offers over the tab. */
  getChoices?(command: string, target: RunTarget): Promise<readonly CommandChoice[]>
  /** The file a tab shows, which the commands of the entries run over. */
  getTabTarget(tab: string): RunTarget
  getTabIcon(tab: string): Component | null
  getKindIcon(kind: string): Component | null
  /** Receives the failure of a command that runs after the menu closed. */
  reportError?(error: unknown): void
}

export function useTabMenu(deps: TabMenuDeps) {
  const getEntries = (tab: string): readonly SlotItem[] =>
    deps.slots?.getEntries(SLOT.tabStripContext, deps.getTabTarget(tab)) ?? []

  const link = useTabLinkMenu({
    ...deps.link,
    hasOtherRows: (tab) => getEntries(tab).length > 0,
  })

  const entries = computed(() => (link.tabId.value === null ? [] : getEntries(link.tabId.value)))
  const choices = useEntryChoices(deps)

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
    const slotItems = createSlotItems(entries.value, {
      byKey: choices.byKey.value,
      loadingText: WORDS.loadingChoices,
      noneText: WORDS.noChoices,
    })
    return { tab, at: link.at.value, items: [...rows, ...slotItems] }
  })

  /** Runs the command of the entry, or of its choice, the row `id` stands for over the tab. */
  const runEntryRow = async (tab: string, entry: SlotRowEntry, choice?: string): Promise<void> => {
    if (entry.kind !== 'command' || (entry.hasChoices && choice === undefined)) return
    link.close()
    await deps.runCommand?.(entry.command, deps.getTabTarget(tab), choice)
  }

  /** Runs the command of an entry over the tab, or the row of linking chosen. */
  const chooseRow = async (tab: string, id: string): Promise<void> => {
    if (link.tabId.value !== tab) return
    const picked = parseChoiceRowId(id)
    const entry = findSlotEntry(entries.value, picked ? picked.key : id)
    if (entry) return runEntryRow(tab, entry, picked?.choice)
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
    open: (tab: string, where: { x: number; y: number }) => {
      link.open(tab, where)
      if (link.tabId.value === tab) choices.read(getEntries(tab), deps.getTabTarget(tab))
    },
    /** Chooses the row `id` of the menu requested on `tab`; a menu since closed runs nothing. */
    choose: (tab: string, id: string): void => {
      chooseRow(tab, id).catch((error: unknown) => deps.reportError?.(error))
    },
    dismiss: link.close,
    highlight: (id: string | null) => link.highlight(getMoveTarget(id)),
    getIcon,
  }
}

/** The choices of the commands of the open menu that offer them, read as the menu opens. */
function useEntryChoices(deps: Pick<TabMenuDeps, 'getChoices' | 'reportError'>) {
  const byKey = shallowRef<ReadonlyMap<string, readonly CommandChoice[] | null>>(new Map())
  const opening = ref(0)
  const set = (key: string, read: readonly CommandChoice[] | null) => {
    byKey.value = new Map([...byKey.value, [key, read]])
  }
  const read = (entries: readonly SlotItem[], target: RunTarget) => {
    const current = ++opening.value
    byKey.value = new Map()
    for (const entry of listChoiceCommands(entries)) {
      set(entry.key, null)
      ;(deps.getChoices?.(entry.command, target) ?? Promise.resolve([]))
        .then((found) => {
          if (current === opening.value) set(entry.key, found)
        })
        .catch((error: unknown) => {
          deps.reportError?.(error)
          if (current === opening.value) set(entry.key, [])
        })
    }
  }
  return { byKey, read }
}

/** The tabs of a window, and what is done to them from the menu of a tab. */
export interface WindowTabMenuDeps extends Pick<
  TabLinkMenuDeps,
  'tabLinks' | 'windowId' | 'isAsking' | 'onDetach' | 'onMove' | 'onHighlight'
> {
  readonly windowTabs: Pick<ReturnType<typeof useWindowTabs>, 'tabs' | 'getTab'>
  readonly slots?: SlotHost | undefined
  readonly commands?: CommandsHost | undefined
  getTabIcon(tab: string): Component | null | undefined
  getKindIcon(kind: string): Component | null | undefined
}

/** The tab of the window and the file it shows, as the commands of its menu run over them. */
export const getTabFileTarget = (
  windowTabs: WindowTabMenuDeps['windowTabs'],
  id: string,
): RunTarget => {
  const tab = windowTabs.getTab(id)
  const on = tab ? tab.kind.getTarget?.(tab.state) : undefined
  return {
    path: on?.file || on?.path || '',
    title: on?.title ?? '',
    isFolder: false,
    others: [],
    tab: id,
  }
}

/** The menu of a tab of the window: a tab whose kind navigates can be linked. */
export function useWindowTabMenu(deps: WindowTabMenuDeps) {
  const { windowTabs, slots, commands, getTabIcon, getKindIcon, ...link } = deps
  return useTabMenu({
    link: {
      ...link,
      getTabs: () => windowTabs.tabs.value,
      getKind: (id) => windowTabs.getTab(id)?.kind.kind,
      isLinkable: (id) => windowTabs.getTab(id)?.kind.navigate !== undefined,
    },
    slots,
    runCommand: (command, target, choice) =>
      commands?.run(command, target, choice) ?? Promise.resolve(),
    getChoices: (command, target) => commands?.getChoices(command, target) ?? Promise.resolve([]),
    getTabTarget: (id) => getTabFileTarget(windowTabs, id),
    getTabIcon: (id) => getTabIcon(id) ?? null,
    getKindIcon: (kind) => getKindIcon(kind) ?? null,
  })
}
