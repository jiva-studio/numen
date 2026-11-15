/**
 * The menu of a tab, which the tab strip draws: the entries of the slot `tabStrip.context` over the
 * tab and the file it shows, the commands of the window over a tab among them.
 */
import { computed, type Component } from 'vue'
import type { TabMenu } from '@numen/workbench'
import {
  isNotePath,
  SLOT,
  useSlotMenu,
  type CommandItem,
  type JsonObject,
  type CommandTarget as RunTarget,
  type SlotHost,
  type SlotItem,
  type SlotMenu,
} from '@numen/plugin-loader'
import { WORDS } from '@/shared/ui/words'
import type { Caught } from '@/shared/api/errors'

export interface TabMenuDeps {
  readonly slots?: SlotHost | undefined
  readonly moveCommand: string
  /** Runs a command of an entry over the tab; with `itemId`, the one picked, and with `values`, those the entry holds. */
  runCommand?(
    command: string,
    target: RunTarget,
    itemId?: string,
    values?: Readonly<Record<string, string>>,
  ): Promise<void>
  /** The items a command of an entry offers over the tab. */
  listItems?(command: string, target: RunTarget): Promise<readonly CommandItem[]>
  /** The tab and the file it shows, which the commands of the entries run over. */
  getTabTarget(tab: string): RunTarget
  /** Turns the highlight of another window on or off while the pointer is over its item. */
  onHighlight?(this: void, windowId: string, isOn: boolean): void
  /** Receives the failure of a command that runs after the menu closed, or of a list read. */
  reportError?(this: void, error: Caught): void
  /** The type and the properties of the note of the path, which the entries are asked for over. */
  readNoteKind?(this: void, path: string): Promise<NoteKind>
}

/** The words of the rows the menu writes itself. */
const MENU_WORDS = { loadingText: WORDS.loadingItems }

/** What the menu of a tab is over: the tab, where it was requested, and the target of its commands. */
interface TabMenuTarget {
  readonly tab: string
  readonly at: { readonly x: number; readonly y: number }
  readonly target: RunTarget
}

const getMoveWindow = (
  tree: SlotMenu | null,
  id: string | null,
  moveCommand: string,
): string | null => {
  const pick = id === null ? undefined : tree?.getPick(id)
  return pick?.command === moveCommand ? (pick.itemId ?? null) : null
}

/** The type the vault files a note as, and its frontmatter where it has any. */
export interface NoteKind {
  readonly noteType: string
  readonly properties?: JsonObject
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
  const getEntries = (target: RunTarget): readonly SlotItem[] =>
    deps.slots?.getEntries(SLOT.tabStripContext, target) ?? []
  const slotMenu = useSlotMenu<TabMenuTarget>({
    listItems: (command, { target }) => deps.listItems?.(command, target) ?? Promise.resolve([]),
    words: MENU_WORDS,
    onError: (error) => deps.reportError?.(error),
  })
  let highlighted: string | null = null

  const menu = computed<TabMenu | null>(() => {
    const visible = slotMenu.target.value
    const tree = slotMenu.menu.value
    if (visible === null || tree === null) return null
    return { tab: visible.tab, at: visible.at, items: tree.items }
  })

  const highlight = (id: string | null): void => {
    const windowId = getMoveWindow(slotMenu.menu.value, id, deps.moveCommand)
    if (windowId === highlighted) return
    if (highlighted !== null) deps.onHighlight?.(highlighted, false)
    highlighted = windowId
    if (windowId !== null) deps.onHighlight?.(windowId, true)
  }

  const dismiss = (): void => {
    highlight(null)
    slotMenu.close()
  }

  /**
   * Opens the menu on a tab, over the type and properties of its note where the menu reads one first; a tab the
   * slot holds no entry for has none.
   */
  const open = (tab: string, position: { x: number; y: number }): void => {
    const target = deps.getTabTarget(tab)
    highlight(null)
    const request = (typed: RunTarget) => ({
      target: { tab, at: position, target: typed },
      entries: getEntries(typed),
    })
    if (!deps.readNoteKind || !isNotePath(target.path)) {
      slotMenu.open(() => request(target))
      return
    }
    const readNoteKind = deps.readNoteKind
    slotMenu.open(async () => request(await readTypedTarget(target, readNoteKind)))
  }

  /** Chooses the row `id` of the menu requested on `tab`; a menu since closed runs nothing. */
  const choose = (tab: string, id: string): void => {
    if (slotMenu.target.value?.tab !== tab) return
    highlight(null)
    const choice = slotMenu.pick(id)
    if (!choice) return
    ;(
      deps.runCommand?.(choice.command, choice.target.target, choice.itemId, choice.values) ??
      Promise.resolve()
    ).catch((error: Caught) => deps.reportError?.(error))
  }

  /** The icon of the command or submenu of the entry a row stands for. */
  const getIcon = (id: string): Component | null => slotMenu.getIcon(id) ?? null

  return { menu, open, choose, dismiss, highlight, getIcon }
}
