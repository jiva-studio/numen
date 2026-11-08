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
} from '@numen/plugin-sdk'
import { resolveDropFolder, ROOT } from './useFileTree'
import { TRASH_NOTE } from '../lib/menu'
import { resolveRenamePath } from './rename'
import { WORDS as words } from '../words'
import type {
  DropPosition,
  Entry,
  FilesTabDeps,
  FilesTabState,
  FileTree,
  MenuRequest,
} from '../types'
import * as create from './create'

export type { DropPosition, FilesTabDeps, FilesTabState, MenuRequest }

const createFailureReporter = (deps: FilesTabDeps) => (error: unknown) => {
  deps.showError(formatErrorMessage(error))
}

function createFileCreation(
  list: FileTree,
  deps: FilesTabDeps,
  renamingPath: ReturnType<typeof ref<string | null>>,
) {
  const createFolder = async (path: string | null) => {
    const createdItem = await create.createFolder(list, deps, path)
    if (createdItem) renamingPath.value = createdItem
  }

  const createNote = async (path: string | null) => {
    const createdItem = await create.createNote(list, deps, path)
    if (createdItem) renamingPath.value = createdItem
  }

  return { createFolder, createNote }
}

function createFileMoves(list: FileTree, deps: FilesTabDeps) {
  const move = async (paths: readonly string[], target: DropPosition) => {
    if (paths.length === 0) return

    const into = resolveDropFolder(target)
    await list.loadFolder(into)
    const existingNames = new Set(list.getEntriesInFolder(into).map((one) => one.name))
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
      await deps.movePath(path, to)
    }

    if (conflictedNames.length > 0) deps.showError(`${words.taken} ${conflictedNames.join(', ')}`)
    await list.refresh()
  }

  return { move }
}

const getMenuEntry = (list: FileTree, path: string): Entry =>
  list.getEntryAt(path) ?? {
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

/** The words of the rows the menu writes itself. */
const MENU_WORDS = { loadingText: words.loadingItems }

/** What the menu of the tree is over: where it was requested, and the target of its commands. */
interface FileMenuTarget {
  readonly request: MenuRequest
  readonly target: CommandTarget
}

function useFileMenu(list: FileTree, deps: FilesTabDeps) {
  const reportFailure = createFailureReporter(deps)
  const slotMenu = useSlotMenu<FileMenuTarget>({
    listItems: (command, { target }) => deps.listItems(command, target),
    words: MENU_WORDS,
    onError: reportFailure,
  })
  const menu = computed(() => slotMenu.target.value?.request ?? null)

  const getOverPaths = (path: string): readonly string[] =>
    list.selectedPaths.value.includes(path) ? list.selectedPaths.value : [path]

  const getTarget = (path: string | null): CommandTarget =>
    getMenuTarget(list, path, path === null ? [] : getOverPaths(path))

  /** Opens the menu the request asks for, over the row it was opened on. */
  const openMenu = (request: MenuRequest) => {
    const target = getTarget(request.path)
    slotMenu.open(() => ({ target: { request, target }, entries: deps.getMenuEntries(target) }))
  }

  /** The rows of the open menu for its row; for another row, its rows before any list is read. */
  const getMenuItems = (path: string | null) => {
    const openMenuTree = slotMenu.menu.value
    if (openMenuTree && menu.value?.path === path) return openMenuTree.items
    return createSlotMenu(deps.getMenuEntries(getTarget(path)), new Map(), MENU_WORDS).items
  }

  const getMenuIcon = (id: string): Component | null =>
    (slotMenu.getIcon(id) as Component | undefined) ?? null

  const chooseMenuItem = (id: string) => {
    const choice = slotMenu.pick(id)
    if (!choice) return
    deps.runCommand(choice.command, choice.target.target, choice.itemId).catch(reportFailure)
  }

  return {
    menu,
    openMenu,
    dismissMenu: slotMenu.close,
    chooseMenuItem,
    getOverPaths,
    getMenuItems,
    getMenuIcon,
  }
}

function createFileActions(list: FileTree, deps: FilesTabDeps) {
  const getNameOf = (path: string): string => list.getEntryAt(path)?.name ?? getFileName(path)

  const activate = (path: string) => {
    const entry = list.getEntryAt(path)
    if (!entry) return
    list.selectPaths([path])
    if (!entry.isFolder) deps.openEntry(entry)
  }

  const rename = async (path: string, name: string) => {
    const to = resolveRenamePath(path, name, list.getEntryAt(path)?.isFolder ?? false)
    if (!to) return
    await deps.movePath(path, to)
    await list.refresh()
  }

  const drag = (paths: readonly string[]) => {
    deps.setDraggedPaths(
      paths.filter((path) => {
        const entry = list.getEntryAt(path)
        return !!entry && !entry.isFolder && entry.isNote
      }),
    )
  }

  const remove = (paths: readonly string[]) => {
    const first = paths[0]
    if (first === undefined) return
    deps
      .runCommand(TRASH_NOTE, getPathTarget(list, first, paths.slice(1)))
      .catch(createFailureReporter(deps))
  }

  return { getNameOf, activate, rename, drag, remove }
}

export function useFilesTab(list: FileTree, deps: FilesTabDeps): FilesTabState {
  const renamingPath = ref<string | null>(null)
  const setRenamingPath = (path: string | null) => {
    renamingPath.value = path
  }

  const actions = createFileActions(list, deps)
  const creators = createFileCreation(list, deps, renamingPath)
  const { move } = createFileMoves(list, deps)
  const fileMenu = useFileMenu(list, deps)

  const reportFailure = createFailureReporter(deps)

  const tabState: FilesTabState = {
    ...actions,
    reportFailure,
    list,
    menu: fileMenu.menu,
    renamingPath,
    setRenamingPath,
    getOverPaths: fileMenu.getOverPaths,
    getFolderFor: (path) => create.getFolderFor(list, path),
    open: (path) => {
      list.openFolder(path).catch(reportFailure)
    },
    close: (path) => list.closeFolder(path),
    select: (paths) => list.selectPaths(paths),
    move,
    drop: () => deps.setDraggedPaths([]),
    createFolder: creators.createFolder,
    createNote: creators.createNote,
    canDropData: (types) => deps.canDropData(types),
    dropData: (data) =>
      deps.dropData(create.getFolderFor(list, list.selectedPaths.value[0] ?? null), data),
    openMenu: fileMenu.openMenu,
    dismissMenu: fileMenu.dismissMenu,
    chooseMenuItem: fileMenu.chooseMenuItem,
    getMenuItems: fileMenu.getMenuItems,
    getMenuIcon: fileMenu.getMenuIcon,
    getEntryIcon: (entry, isOpen) => deps.getEntryIcon(entry, isOpen),
  }

  return tabState
}
