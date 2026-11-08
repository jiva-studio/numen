/**
 * The menu of a tab, which the tab strip draws: the entries of the slot `tabStrip.context` over the
 * tab and the file it shows, the commands of the window over a tab among them.
 */
import { computed, ref, shallowRef, type Component } from 'vue'
import type { TabMenu } from '@numen/shell'
import {
  createSlotMenu,
  readCommandItemLists,
  SLOT,
  type CommandItem,
  type CommandItemLists,
  type CommandsHost,
  type CommandTarget as RunTarget,
  type Disposable,
  type SlotHost,
  type SlotItem,
  type SlotMenu,
} from '@numen/plugin-host'
import type { useWindowTabs } from '@/entities/tab'
import { COMMAND } from '@/shared/commandIds'
import { WORDS } from '@/shared/words'

export interface TabMenuDeps {
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
  /** The type and the properties of the note of the path, which the entries are asked for over. */
  readNoteKind?(path: string): Promise<NoteKind>
}

/** The extension that makes a file a note. */
const NOTE_EXTENSION = '.md'

/** The words of the rows the menu writes itself. */
const MENU_WORDS = { loadingText: WORDS.loadingItems }

function useEntryItems(deps: Pick<TabMenuDeps, 'listItems' | 'reportError'>) {
  const lists = shallowRef<CommandItemLists>(new Map())
  let itemSubscription: Disposable | null = null
  const stop = () => {
    itemSubscription?.dispose()
    itemSubscription = null
  }
  /** Reads the lists of the entries over the target; onShow once the menu may be drawn, onCancel when its open is cancelled. */
  const read = (
    entries: readonly SlotItem[],
    target: RunTarget,
    onShow: () => void,
    onCancel: () => void,
  ) => {
    stop()
    itemSubscription = readCommandItemLists(
      entries,
      (command) => deps.listItems?.(command, target) ?? Promise.resolve([]),
      {
        onChange: (next) => (lists.value = next),
        onShow,
        onCancel,
        onError: (error) => deps.reportError?.(error),
      },
    )
  }
  return { lists, read, stop }
}

const getMoveWindow = (tree: SlotMenu, id: string | null): string | null => {
  const pick = id === null ? undefined : tree.getPick(id)
  return pick?.command === COMMAND.moveTabToWindow ? (pick.argument ?? null) : null
}

/** The type the vault files a note as, and its frontmatter where it has any. */
export interface NoteKind {
  readonly noteType: string
  readonly properties?: Readonly<Record<string, unknown>>
}

const readTypedTarget = async (
  target: RunTarget,
  readNoteKind: (path: string) => Promise<NoteKind>,
): Promise<RunTarget> => {
  try {
    const { noteType, properties } = await readNoteKind(target.path)
    if (noteType === '') return target
    return { ...target, noteType, ...(properties ? { properties } : {}) }
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

  const tree = computed(() => createSlotMenu(entries.value, entryItems.lists.value, MENU_WORDS))

  const menu = computed<TabMenu | null>(() => {
    if (tabId.value === null) return null
    return { tab: tabId.value, at: position.value, items: tree.value.items }
  })

  const highlight = (id: string | null): void => {
    const windowId = getMoveWindow(tree.value, id)
    if (windowId === highlighted) return
    if (highlighted !== null) deps.onHighlight?.(highlighted, false)
    highlighted = windowId
    if (windowId !== null) deps.onHighlight?.(windowId, true)
  }

  const dismiss = (): void => {
    requests += 1
    highlight(null)
    entryItems.stop()
    tabId.value = null
    openTarget.value = null
  }

  const show = (tab: string, where: { x: number; y: number }, target: RunTarget): void => {
    const found = getEntries(target)
    if (found.length === 0) return
    entryItems.read(
      found,
      target,
      () => {
        tabId.value = tab
        position.value = where
        openTarget.value = target
      },
      dismiss,
    )
  }

  /**
   * Opens the menu on a tab, over the type and properties of its note where the menu reads one first; a tab the
   * slot holds no entry for has none.
   */
  const open = (tab: string, where: { x: number; y: number }): void => {
    const target = deps.getTabTarget(tab)
    const request = ++requests
    if (!deps.readNoteKind || !target.path.endsWith(NOTE_EXTENSION)) {
      show(tab, where, target)
      return
    }
    readTypedTarget(target, deps.readNoteKind)
      .then((typed) => {
        if (request === requests) show(tab, where, typed)
      })
      .catch((error: unknown) => deps.reportError?.(error))
  }

  /** Chooses the row `id` of the menu requested on `tab`; a menu since closed runs nothing. */
  const choose = (tab: string, id: string): void => {
    const target = openTarget.value
    if (tabId.value !== tab || !target) return
    const pick = tree.value.getPick(id)
    if (!pick) return
    dismiss()
    ;(deps.runCommand?.(pick.command, target, pick.argument) ?? Promise.resolve()).catch(
      (error: unknown) => deps.reportError?.(error),
    )
  }

  /** The icon of the command or submenu of the entry a row stands for. */
  const getIcon = (id: string): Component | null =>
    (tree.value.getIcon(id) as Component | undefined) ?? null

  return { menu, open, choose, dismiss, highlight, getIcon }
}

/** The tabs of a window, and how the commands of the menu of a tab are read and run. */
export interface WindowTabMenuDeps {
  readonly windowTabs: Pick<ReturnType<typeof useWindowTabs>, 'tabs' | 'getTab'>
  readonly slots?: SlotHost | undefined
  readonly commands?: CommandsHost | undefined
  onHighlight?(windowId: string, isOn: boolean): void
  /** The type and the properties of the note of the path. */
  readNoteKind?(path: string): Promise<NoteKind>
  /** Shows the error of a command of the menu that failed, or of items that could not be read. */
  reportError?(error: unknown): void
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
    ...(deps.readNoteKind ? { readNoteKind: deps.readNoteKind } : {}),
    ...(deps.reportError ? { reportError: deps.reportError } : {}),
  })
}
