/**
 * The menu of a tab, which the tab strip draws: the entries of the slot `tabStrip.context` over the
 * tab and the file it shows, the commands of the window over a tab among them.
 */
import { computed, ref, shallowRef, type Component } from 'vue'
import type { TabMenu } from '@numen/shell'
import {
  SLOT,
  type CommandItem,
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
  listItemCommands,
  parseItemRowId,
  type SlotRowEntry,
} from './tabMenuItems'

export interface TabMenuDeps {
  /** The slots of the window; without them a tab has no menu. */
  readonly slots?: SlotHost | undefined
  /** Runs a command of an entry over the tab; with `argument`, the one picked. */
  runCommand?(command: string, target: RunTarget, argument?: string): Promise<void>
  /** The items a command of an entry offers over the tab. */
  listItems?(command: string, target: RunTarget): Promise<readonly CommandItem[]>
  /** The tab and the file it shows, which the commands of the entries run over. */
  getTabTarget(tab: string): RunTarget
  /** Turns the highlight of another window on or off while the pointer is over its item. */
  onHighlight?(windowId: string, isOn: boolean): void
  /** Receives the failure of a command that runs after the menu closed. */
  reportError?(error: unknown): void
  /** The type the vault files the note of the path as, which the entries are asked for over. */
  readNoteType?(path: string): Promise<string>
}

/** The extension that makes a file a note. */
const NOTE_EXTENSION = '.md'

/** The items of the commands of the open menu that offer them, read as the menu opens. */
function useEntryItems(deps: Pick<TabMenuDeps, 'listItems' | 'reportError'>) {
  const byKey = shallowRef<ReadonlyMap<string, readonly CommandItem[] | null>>(new Map())
  let opening = 0
  const set = (key: string, read: readonly CommandItem[] | null) => {
    byKey.value = new Map([...byKey.value, [key, read]])
  }
  const read = (entries: readonly SlotItem[], target: RunTarget) => {
    const current = ++opening
    byKey.value = new Map()
    for (const entry of listItemCommands(entries)) {
      set(entry.key, null)
      ;(deps.listItems?.(entry.command, target) ?? Promise.resolve([]))
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

/** The window a row stands for when it is an item of moving the tab; null for any other row. */
const getMoveWindow = (entries: readonly SlotItem[], id: string | null): string | null => {
  const picked = id === null ? null : parseItemRowId(id)
  if (!picked) return null
  const entry = findSlotEntry(entries, picked.key)
  return entry?.kind === 'command' && entry.command === COMMAND.moveTabToWindow ? picked.item : null
}

/** The target of a tab with the type of its note, where the type is read; without it otherwise. */
const readTypedTarget = async (
  target: RunTarget,
  readNoteType: (path: string) => Promise<string>,
): Promise<RunTarget> => {
  try {
    const noteType = await readNoteType(target.path)
    return noteType === '' ? target : { ...target, noteType }
  } catch {
    return target
  }
}

export function useTabMenu(deps: TabMenuDeps) {
  const tabId = ref<string | null>(null)
  const position = ref({ x: 0, y: 0 })
  const openTarget = shallowRef<RunTarget | null>(null)
  const getEntries = (target: RunTarget): readonly SlotItem[] =>
    deps.slots?.getEntries(SLOT.tabStripContext, target) ?? []
  const entries = computed(() => (openTarget.value === null ? [] : getEntries(openTarget.value)))
  const entryItems = useEntryItems(deps)
  let highlighted: string | null = null
  /** How many menus were requested or closed, which tells a read of a note type it is too late. */
  let requests = 0

  const menu = computed<TabMenu | null>(() => {
    if (tabId.value === null) return null
    const items = createSlotItems(entries.value, {
      byKey: entryItems.byKey.value,
      loadingText: WORDS.loadingItems,
      noneText: WORDS.noItems,
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
    requests += 1
    highlight(null)
    tabId.value = null
    openTarget.value = null
  }

  const show = (tab: string, where: { x: number; y: number }, target: RunTarget): void => {
    const found = getEntries(target)
    if (found.length === 0) return
    tabId.value = tab
    position.value = where
    openTarget.value = target
    entryItems.read(found, target)
  }

  /**
   * Opens the menu on a tab, over the type of its note where the menu reads one first; a tab the
   * slot holds no entry for has none.
   */
  const open = (tab: string, where: { x: number; y: number }): void => {
    const target = deps.getTabTarget(tab)
    const request = ++requests
    if (!deps.readNoteType || !target.path.endsWith(NOTE_EXTENSION)) {
      show(tab, where, target)
      return
    }
    readTypedTarget(target, deps.readNoteType)
      .then((typed) => {
        if (request === requests) show(tab, where, typed)
      })
      .catch((error: unknown) => deps.reportError?.(error))
  }

  /** Runs the command of the entry, with the item picked as its argument, over the tab. */
  const runEntry = async (entry: SlotRowEntry, target: RunTarget, argument?: string) => {
    if (entry.kind !== 'command' || (entry.hasItems && argument === undefined)) return
    dismiss()
    await deps.runCommand?.(entry.command, target, argument)
  }

  /** Chooses the row `id` of the menu requested on `tab`; a menu since closed runs nothing. */
  const choose = (tab: string, id: string): void => {
    const target = openTarget.value
    if (tabId.value !== tab || !target) return
    const picked = parseItemRowId(id)
    const entry = findSlotEntry(entries.value, picked ? picked.key : id)
    if (!entry) return
    runEntry(entry, target, picked?.item).catch((error: unknown) => deps.reportError?.(error))
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
  /** The type the vault files the note of the path as. */
  readNoteType?(path: string): Promise<string>
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
    runCommand: (command, target, argument) =>
      commands?.run(command, target, argument) ?? Promise.resolve(),
    listItems: (command, target) => commands?.listItems(command, target) ?? Promise.resolve([]),
    getTabTarget: (id) => getTabFileTarget(windowTabs, id),
    ...(deps.onHighlight ? { onHighlight: deps.onHighlight } : {}),
    ...(deps.readNoteType ? { readNoteType: deps.readNoteType } : {}),
  })
}
