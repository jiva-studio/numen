/**
 * The menu of a tab, which the tab strip draws: the entries of the slot `tabStrip.context` over the
 * tab and the file it shows, the commands of the window over a tab among them.
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
import { COMMAND } from '@/shared/commandIds'
import { WORDS } from '@/shared/words'
import {
  createSlotItems,
  findSlotEntry,
  listChoiceCommands,
  parseChoiceRowId,
  type SlotRowEntry,
} from './tabMenuItems'

export interface TabMenuDeps {
  /** The slots of the window; without them a tab has no menu. */
  readonly slots?: SlotHost | undefined
  /** Runs a command of an entry over the tab; with `choice`, the one picked. */
  runCommand?(command: string, target: RunTarget, choice?: string): Promise<void>
  /** The choices a command of an entry offers over the tab. */
  getChoices?(command: string, target: RunTarget): Promise<readonly CommandChoice[]>
  /** The tab and the file it shows, which the commands of the entries run over. */
  getTabTarget(tab: string): RunTarget
  /** Turns the highlight of another window on or off while the pointer is over its choice. */
  onHighlight?(windowId: string, isOn: boolean): void
  /** Receives the failure of a command that runs after the menu closed. */
  reportError?(error: unknown): void
}

/** The choices of the commands of the open menu that offer them, read as the menu opens. */
function useEntryChoices(deps: Pick<TabMenuDeps, 'getChoices' | 'reportError'>) {
  const byKey = shallowRef<ReadonlyMap<string, readonly CommandChoice[] | null>>(new Map())
  let opening = 0
  const set = (key: string, read: readonly CommandChoice[] | null) => {
    byKey.value = new Map([...byKey.value, [key, read]])
  }
  const read = (entries: readonly SlotItem[], target: RunTarget) => {
    const current = ++opening
    byKey.value = new Map()
    for (const entry of listChoiceCommands(entries)) {
      set(entry.key, null)
      ;(deps.getChoices?.(entry.command, target) ?? Promise.resolve([]))
        .then((found) => {
          if (current === opening) set(entry.key, found)
        })
        .catch((error: unknown) => {
          deps.reportError?.(error)
          if (current === opening) set(entry.key, [])
        })
    }
  }
  return { byKey, read }
}

/** The window a row stands for when it is a choice of moving the tab; null for any other row. */
const getMoveWindow = (entries: readonly SlotItem[], id: string | null): string | null => {
  const picked = id === null ? null : parseChoiceRowId(id)
  if (!picked) return null
  const entry = findSlotEntry(entries, picked.key)
  return entry?.kind === 'command' && entry.command === COMMAND.moveTabToWindow
    ? picked.choice
    : null
}

export function useTabMenu(deps: TabMenuDeps) {
  const tabId = ref<string | null>(null)
  const position = ref({ x: 0, y: 0 })
  const getEntries = (tab: string): readonly SlotItem[] =>
    deps.slots?.getEntries(SLOT.tabStripContext, deps.getTabTarget(tab)) ?? []
  const entries = computed(() => (tabId.value === null ? [] : getEntries(tabId.value)))
  const choices = useEntryChoices(deps)
  let highlighted: string | null = null

  const menu = computed<TabMenu | null>(() => {
    if (tabId.value === null) return null
    const items = createSlotItems(entries.value, {
      byKey: choices.byKey.value,
      loadingText: WORDS.loadingChoices,
      noneText: WORDS.noChoices,
    })
    return { tab: tabId.value, at: position.value, items }
  })

  const highlight = (id: string | null): void => {
    const windowId = getMoveWindow(entries.value, id)
    if (windowId === highlighted) return
    if (highlighted !== null) deps.onHighlight?.(highlighted, false)
    highlighted = windowId
    if (windowId !== null) deps.onHighlight?.(windowId, true)
  }

  const dismiss = (): void => {
    highlight(null)
    tabId.value = null
  }

  /** Opens the menu on a tab; a tab the slot holds no entry for has none. */
  const open = (tab: string, where: { x: number; y: number }): void => {
    const found = getEntries(tab)
    if (found.length === 0) return
    tabId.value = tab
    position.value = where
    choices.read(found, deps.getTabTarget(tab))
  }

  /** Runs the command of the entry, or of its choice, over the tab. */
  const runEntry = async (tab: string, entry: SlotRowEntry, choice?: string): Promise<void> => {
    if (entry.kind !== 'command' || (entry.hasChoices && choice === undefined)) return
    dismiss()
    await deps.runCommand?.(entry.command, deps.getTabTarget(tab), choice)
  }

  /** Chooses the row `id` of the menu requested on `tab`; a menu since closed runs nothing. */
  const choose = (tab: string, id: string): void => {
    if (tabId.value !== tab) return
    const picked = parseChoiceRowId(id)
    const entry = findSlotEntry(entries.value, picked ? picked.key : id)
    if (!entry) return
    runEntry(tab, entry, picked?.choice).catch((error: unknown) => deps.reportError?.(error))
  }

  /** The icon of the command or submenu of the entry a row stands for. */
  const getIcon = (id: string): Component | null => {
    const entry = findSlotEntry(entries.value, id)
    if (!entry || entry.kind === 'view') return null
    return (entry.icon as Component | undefined) ?? null
  }

  return { menu, open, choose, dismiss, highlight, getIcon }
}

/** The tabs of a window, and how the commands of the menu of a tab are read and run. */
export interface WindowTabMenuDeps {
  readonly windowTabs: Pick<ReturnType<typeof useWindowTabs>, 'tabs' | 'getTab'>
  readonly slots?: SlotHost | undefined
  readonly commands?: CommandsHost | undefined
  onHighlight?(windowId: string, isOn: boolean): void
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

/** The menu of a tab of the window, whose commands run through the commands of the window. */
export function useWindowTabMenu(deps: WindowTabMenuDeps) {
  const { windowTabs, slots, commands } = deps
  return useTabMenu({
    slots,
    runCommand: (command, target, choice) =>
      commands?.run(command, target, choice) ?? Promise.resolve(),
    getChoices: (command, target) => commands?.getChoices(command, target) ?? Promise.resolve([]),
    getTabTarget: (id) => getTabFileTarget(windowTabs, id),
    ...(deps.onHighlight ? { onHighlight: deps.onHighlight } : {}),
  })
}
