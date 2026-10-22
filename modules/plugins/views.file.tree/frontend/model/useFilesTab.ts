/**
 * Tab state, tree interactions, and window tab registration for the files tab.
 */
import { ref } from 'vue'
import { getFileName, type MenuEntry } from '@numen/plugin-sdk'
import { resolveDropFolder, ROOT } from './useFileTree'
import {
  createMenuItems,
  NEW_DECK,
  NEW_FOLDER,
  NEW_NOTE,
  NEW_PRESET,
  NEW_STENCIL,
  OWN_IDS,
  RENAME,
} from '../lib/menu'
import { resolveRenamePath } from './rename'
import { WORDS as words } from '../words'
import type {
  DropPosition,
  Entry,
  FileMaker,
  FilesTabDeps,
  FilesTabState,
  FileTree,
  MenuRequest,
  Source,
} from '../types'
import * as create from './create'

export type { DropPosition, FileMaker, FilesTabDeps, FilesTabState, MenuRequest }

function useFileCreation(
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

  const createOne = async (path: string | null, createEntry: FileMaker, name: string) => {
    const createdItem = await create.createOne(list, path, { createEntry, name })
    if (createdItem) renamingPath.value = createdItem
  }

  return { createFolder, createNote, createOne }
}

function useFileMoves(list: FileTree, deps: FilesTabDeps) {
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

interface FileMenuDeps {
  list: FileTree
  deps: FilesTabDeps
  renamingPath: ReturnType<typeof ref<string | null>>
  creators: ReturnType<typeof useFileCreation>
  getNameOf: (path: string) => string
  getSourceOf: (path: string) => Source
}

/** The entry a menu asked for on a path stands for, and a file of no source where the tree holds none. */
const getMenuEntry = (list: FileTree, path: string): Entry =>
  list.getEntryAt(path) ?? {
    path,
    name: getFileName(path),
    isFolder: false,
    kind: 'other',
    type: 'note',
  }

function useFileMenu(options: FileMenuDeps) {
  const { list, deps, renamingPath, creators, getNameOf, getSourceOf } = options
  const menu = ref<MenuRequest | null>(null)

  const openMenu = (request: MenuRequest) => {
    menu.value = request
  }

  const dismissMenu = () => {
    menu.value = null
  }

  const getOverPaths = (path: string): readonly string[] =>
    list.selectedPaths.value.includes(path) ? list.selectedPaths.value : [path]

  const getMenuItems = (path: string | null): readonly MenuEntry[] => {
    if (path === null) return createMenuItems(null, false)
    const isSeveral = getOverPaths(path).length > 1
    return createMenuItems(
      deps.getPathCommands({ entry: getMenuEntry(list, path), isSeveral }),
      isSeveral,
    )
  }

  const isOffered = (id: string, path: string | null): boolean =>
    OWN_IDS.has(id) || getMenuItems(path).some((one) => one.id === id)

  const chooseMenuItem = (id: string) => {
    const request = menu.value
    menu.value = null
    if (!request || !isOffered(id, request.path)) return
    if (id === NEW_NOTE) return void creators.createNote(request.path)
    if (id === NEW_DECK)
      return void creators.createOne(request.path, deps.createDeck, words.newDeck)
    if (id === NEW_STENCIL)
      return void creators.createOne(request.path, deps.createStencil, words.newStencil)
    if (id === NEW_PRESET)
      return void creators.createOne(request.path, deps.createPreset, words.newPreset)
    if (id === NEW_FOLDER) return void creators.createFolder(request.path)

    const path = request.path
    if (path === null) return
    if (id === RENAME) {
      renamingPath.value = path
      return
    }
    deps.runCommand(id, getOverPaths(path), { name: getNameOf(path), source: getSourceOf(path) })
  }

  return { menu, openMenu, dismissMenu, chooseMenuItem, getOverPaths, getMenuItems }
}

function createFileActions(list: FileTree, deps: FilesTabDeps) {
  const getNameOf = (path: string): string => list.getEntryAt(path)?.name ?? getFileName(path)
  const getSourceOf = (path: string): Source => list.getEntryAt(path)?.kind ?? 'other'

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
        return !!entry && !entry.isFolder && entry.kind === 'note'
      }),
    )
  }

  const remove = (paths: readonly string[]) => {
    const first = paths[0]
    if (first === undefined) return
    deps.runCommand('remove', paths, { name: getNameOf(first), source: getSourceOf(first) })
  }

  return { getNameOf, getSourceOf, activate, rename, drag, remove }
}

export function useFilesTab(list: FileTree, deps: FilesTabDeps): FilesTabState {
  const renamingPath = ref<string | null>(null)
  const setRenamingPath = (path: string | null) => {
    renamingPath.value = path
  }

  const actions = createFileActions(list, deps)
  const creators = useFileCreation(list, deps, renamingPath)
  const { move } = useFileMoves(list, deps)
  const fileMenu = useFileMenu({
    list,
    deps,
    renamingPath,
    creators,
    getNameOf: actions.getNameOf,
    getSourceOf: actions.getSourceOf,
  })

  const tabState: FilesTabState = {
    ...actions,
    list,
    menu: fileMenu.menu,
    renamingPath,
    setRenamingPath,
    getOverPaths: fileMenu.getOverPaths,
    getFolderFor: (path) => create.getFolderFor(list, path),
    open: (path) => void list.openFolder(path),
    close: (path) => list.closeFolder(path),
    select: (paths) => list.selectPaths(paths),
    move,
    drop: () => deps.setDraggedPaths([]),
    createFolder: creators.createFolder,
    createNote: creators.createNote,
    createOne: creators.createOne,
    importUrl: (url) =>
      deps.importUrl(create.getFolderFor(list, list.selectedPaths.value[0] ?? null), url),
    importUrls: (urls) =>
      deps.importUrls(create.getFolderFor(list, list.selectedPaths.value[0] ?? null), urls),
    openMenu: fileMenu.openMenu,
    dismissMenu: fileMenu.dismissMenu,
    chooseMenuItem: fileMenu.chooseMenuItem,
    getMenuItems: fileMenu.getMenuItems,
    getCommandIcon: (id) => deps.getCommandIcon(id),
    getEntryIcon: (entry, isOpen) => deps.getEntryIcon(entry, isOpen),
  }

  return tabState
}
