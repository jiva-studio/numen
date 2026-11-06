/**
 * Tab state, tree interactions, and window tab registration for the files tab.
 */
import { computed, ref, shallowRef, type Component } from 'vue'
import {
  formatErrorMessage,
  getFileName,
  NOTE_TYPE,
  type CommandItem,
  type CommandTarget,
} from '@numen/plugin-sdk'
import { resolveDropFolder, ROOT } from './useFileTree'
import {
  createMenuItems,
  findMenuEntry,
  listItemCommands,
  parseItemRowId,
  TRASH_NOTE,
} from '../lib/menu'
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

/** Reports a failed background operation of the tab through the window's error display. */
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

/** The entry for a path requested from a menu, and a non-note file where the tree has none. */
const getMenuEntry = (list: FileTree, path: string): Entry =>
  list.getEntryAt(path) ?? {
    path,
    name: getFileName(path),
    isFolder: false,
    isNote: false,
    type: NOTE_TYPE,
  }

/** The target of a command over the path and the others selected with it. */
const getPathTarget = (list: FileTree, path: string, others: readonly string[]): CommandTarget => {
  const entry = getMenuEntry(list, path)
  return { path, title: entry.name, isFolder: entry.isFolder, others, tab: '' }
}

/** The target of the menu opened on a row, or on the root where it was opened off every row. */
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

function useFileMenu(list: FileTree, deps: FilesTabDeps) {
  const menu = ref<MenuRequest | null>(null)
  const reportFailure = createFailureReporter(deps)
  const items = shallowRef<ReadonlyMap<string, readonly CommandItem[] | null>>(new Map())

  const setItems = (key: string, read: readonly CommandItem[] | null) => {
    items.value = new Map([...items.value, [key, read]])
  }

  /** How many menus were opened, which tells the menu a read of items was made for. */
  let openings = 0

  /** Reads the items of the commands of the menu opened by the request, while it stays open. */
  const readItems = (request: MenuRequest) => {
    const opening = ++openings
    const isOpen = () => menu.value !== null && opening === openings
    const target = getTarget(request.path)
    for (const entry of listItemCommands(deps.getMenuEntries(target))) {
      setItems(entry.key, null)
      deps
        .listItems(entry.command, target)
        .then((read) => {
          if (isOpen()) setItems(entry.key, read)
        })
        .catch((error: unknown) => {
          reportFailure(error)
          if (isOpen()) setItems(entry.key, [])
        })
    }
  }

  const openMenu = (request: MenuRequest) => {
    menu.value = request
    items.value = new Map()
    readItems(request)
  }

  const dismissMenu = () => {
    menu.value = null
  }

  const getOverPaths = (path: string): readonly string[] =>
    list.selectedPaths.value.includes(path) ? list.selectedPaths.value : [path]

  const getTarget = (path: string | null): CommandTarget =>
    getMenuTarget(list, path, path === null ? [] : getOverPaths(path))

  const getMenuItems = (path: string | null) =>
    createMenuItems(deps.getMenuEntries(getTarget(path)), {
      byKey: items.value,
      loadingText: words.loadingItems,
      noneText: words.noItems,
    })

  const openEntries = computed(() =>
    menu.value ? deps.getMenuEntries(getTarget(menu.value.path)) : [],
  )

  const getMenuIcon = (id: string): Component | null => {
    const entry = findMenuEntry(openEntries.value, id)
    if (!entry || entry.kind === 'view') return null
    return (entry.icon as Component | undefined) ?? null
  }

  const chooseMenuItem = (id: string) => {
    const request = menu.value
    if (!request) return
    const target = getTarget(request.path)
    const picked = parseItemRowId(id)
    const entry = findMenuEntry(openEntries.value, picked ? picked.key : id)
    menu.value = null
    if (entry?.kind !== 'command' || (entry.hasItems && !picked)) return
    deps.runCommand(entry.command, target, picked?.item).catch(reportFailure)
  }

  return {
    menu,
    openMenu,
    dismissMenu,
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
