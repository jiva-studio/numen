/**
 * Tab state, tree interactions, and window tab registration for the files tab.
 */
import { computed, ref, type Component } from 'vue'
import {
  createSlotMenu,
  formatErrorMessage,
  getFileName,
  NOTE_TYPE,
  useSlotMenu,
  type CommandTarget,
  type Caught,
} from '@numen/plugin-sdk'
import { resolveDropFolder, ROOT } from './useFileTree'
import { CREATE_FOLDER, CREATE_NOTE, TRASH_FILE } from '../lib/menu'
import { resolveRenamePath } from './rename'
import { WORDS as words } from '../words'
import type {
  DropPosition,
  Entry,
  FilesTabDeps,
  FilesTabEntries,
  FilesTabMenu,
  FilesTabState,
  FileTree,
  MenuRequest,
} from '../types'
import * as create from './create'

export type { FilesTabDeps, FilesTabState }

const createFailureReporter = (deps: FilesTabDeps) => (error: Caught) => {
  deps.window.showError(formatErrorMessage(error))
}

function createFileMoves(list: FileTree, deps: FilesTabDeps) {
  const move = async (paths: readonly string[], target: DropPosition) => {
    if (paths.length === 0) return

    const into = resolveDropFolder(target)
    await list.sync.loadFolder(into)
    const existingNames = new Set(list.queries.getEntriesInFolder(into).map((one) => one.name))
    const conflictedNames: string[] = []

    for (const path of paths) {
      const name = getFileName(path)
      const to = into === ROOT ? name : `${into}/${name}`
      if (to === path) continue
      if (existingNames.has(name)) {
        conflictedNames.push(name)
        continue
      }
      existingNames.add(name)
      await deps.transfer.movePath(path, to)
    }

    if (conflictedNames.length > 0)
      deps.window.showError(`${words.taken} ${conflictedNames.join(', ')}`)
    await list.sync.refresh()
  }

  return { move }
}

const getMenuEntry = (list: FileTree, path: string): Entry =>
  list.queries.getEntryAt(path) ?? {
    path,
    name: getFileName(path),
    isFolder: false,
    isNote: false,
    type: NOTE_TYPE,
  }

const getPathTarget = (list: FileTree, path: string, others: readonly string[]): CommandTarget => {
  const entry = getMenuEntry(list, path)
  const target = { path, title: entry.name, isFolder: entry.isFolder, others, tab: '' }
  return entry.isNote ? { ...target, noteType: entry.type } : target
}

const getMenuTarget = (
  list: FileTree,
  path: string | null,
  overPaths: readonly string[],
): CommandTarget =>
  path === null
    ? { path: ROOT, title: '', isFolder: true, others: [], tab: '' }
    : getPathTarget(
        list,
        path,
        overPaths.filter((one) => one !== path),
      )

const CREATED_COMMANDS: ReadonlySet<string> = new Set([CREATE_NOTE, CREATE_FOLDER])

/** The words of the rows the menu writes itself. */
const MENU_WORDS = { loadingText: words.loadingItems }

/** What the menu of the tree is over: where it was requested, and the target of its commands. */
interface FileMenuTarget {
  readonly request: MenuRequest
  readonly target: CommandTarget
}

function useFileMenu(
  list: FileTree,
  deps: FilesTabDeps,
  showCreated: (folder: string, path: string) => Promise<void>,
): FilesTabMenu {
  const reportFailure = createFailureReporter(deps)
  const slotMenu = useSlotMenu<FileMenuTarget>({
    listItems: (command, { target }) => deps.menu.listItems(command, target),
    words: MENU_WORDS,
    onError: reportFailure,
  })
  const menu = computed(() => slotMenu.target.value?.request ?? null)

  const getOverPaths = (path: string): readonly string[] =>
    list.state.selectedPaths.value.includes(path) ? list.state.selectedPaths.value : [path]

  const getTarget = (path: string | null): CommandTarget =>
    getMenuTarget(list, path, path === null ? [] : getOverPaths(path))

  /** Opens the menu the request asks for, over the row it was opened on. */
  const openMenu = (request: MenuRequest) => {
    const target = getTarget(request.path)
    slotMenu.open(() => ({
      target: { request, target },
      entries: deps.menu.getMenuEntries(target),
    }))
  }

  /** The rows of the open menu for its row; for another row, its rows before any list is read. */
  const getMenuItems = (path: string | null) => {
    const openMenuTree = slotMenu.menu.value
    if (openMenuTree && menu.value?.path === path) return openMenuTree.items
    return createSlotMenu(deps.menu.getMenuEntries(getTarget(path)), new Map(), MENU_WORDS).items
  }

  const getMenuIcon = (id: string): Component | null =>
    (slotMenu.getIcon(id) as Component | undefined) ?? null

  const chooseMenuItem = (id: string) => {
    const choice = slotMenu.pick(id)
    if (!choice) return
    deps.menu
      .runCommand(choice.command, choice.target.target, choice.itemId, choice.values)
      .then((made) =>
        CREATED_COMMANDS.has(choice.command) && made
          ? showCreated(choice.target.target.path, made)
          : undefined,
      )
      .catch(reportFailure)
  }

  return {
    request: menu,
    open: openMenu,
    dismiss: slotMenu.close,
    choose: chooseMenuItem,
    getOverPaths,
    getItems: getMenuItems,
    getIcon: getMenuIcon,
  }
}

function createFileActions(list: FileTree, deps: FilesTabDeps) {
  const reportFailure = createFailureReporter(deps)

  const getNameOf = (path: string): string =>
    list.queries.getEntryAt(path)?.name ?? getFileName(path)

  const activate = (path: string) => {
    const entry = list.queries.getEntryAt(path)
    if (!entry) return
    list.selectPaths([path])
    if (!entry.isFolder) deps.window.openEntry(entry)
  }

  const rename = async (path: string, name: string) => {
    const to = resolveRenamePath(path, name, list.queries.getEntryAt(path)?.isFolder ?? false)
    if (!to) return
    await deps.transfer.movePath(path, to)
    await list.sync.refresh()
  }

  const drag = (paths: readonly string[]) => {
    deps.window.setDraggedPaths(
      paths.filter((path) => {
        const entry = list.queries.getEntryAt(path)
        return !!entry && !entry.isFolder && entry.isNote
      }),
    )
  }

  const remove = (paths: readonly string[]) => {
    const first = paths[0]
    if (first === undefined) return
    deps.menu
      .runCommand(TRASH_FILE, getPathTarget(list, first, paths.slice(1)))
      .catch(reportFailure)
  }

  const entries: FilesTabEntries = {
    activate,
    open: (path) => {
      list.folders.openFolder(path).catch(reportFailure)
    },
    close: (path) => list.folders.closeFolder(path),
    select: (paths) => list.selectPaths(paths),
    rename,
    remove,
    getNameOf,
    getEntryIcon: (entry, isOpen) => deps.window.getEntryIcon(entry, isOpen),
  }

  return { entries, drag }
}

async function selectDroppedFiles(list: FileTree, paths: readonly string[]) {
  const first = paths[0]
  if (first === undefined) return
  await list.sync.refresh()
  await list.revealPath(first)
  list.selectPaths(paths)
}

export function useFilesTab(list: FileTree, deps: FilesTabDeps): FilesTabState {
  const renamingPath = ref<string | null>(null)
  const setRenamingPath = (path: string | null) => {
    renamingPath.value = path
  }

  const actions = createFileActions(list, deps)
  const { move } = createFileMoves(list, deps)
  const fileMenu = useFileMenu(list, deps, async (folder, path) => {
    await list.folders.openFolder(folder)
    renamingPath.value = path
  })

  const reportFailure = createFailureReporter(deps)

  return {
    list,
    reportFailure,
    entries: actions.entries,
    renaming: { path: renamingPath, setPath: setRenamingPath },
    transfer: {
      getFolderFor: (path) => create.getFolderFor(list, path),
      move,
      drag: actions.drag,
      drop: () => deps.window.setDraggedPaths([]),
      canDropData: (types) => deps.transfer.canDropData(types),
      dropData: async (data) => {
        const folder = create.getFolderFor(list, list.state.selectedPaths.value[0] ?? null)
        const paths = await deps.transfer.dropData(folder, data)
        await selectDroppedFiles(list, paths)
        return paths
      },
    },
    menu: fileMenu,
  }
}
