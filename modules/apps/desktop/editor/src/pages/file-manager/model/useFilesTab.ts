/**
 * Tab state, tree interactions, and window tab registration for the files tab.
 */
import { ref } from 'vue'
import type { Entry, Source } from '@/entities/file'
import type { SearchDestination } from '@/features/command-palette'
import { getFileName } from '@/shared/paths'
import { resolveDropFolder, ROOT } from './useFileTree'
import {
  NEW_DECK,
  NEW_FOLDER,
  NEW_NOTE,
  NEW_PRESET,
  NEW_STENCIL,
  OFFERED,
  RENAME,
  type RunGuard,
} from '../lib/menu'
import { resolveRenamePath } from './rename'
import { WORDS as words } from '../words'
import type {
  DropPosition,
  FileMaker,
  FilesTabDeps,
  FilesTabState,
  FileTree,
  MenuRequest,
} from '../types'
import * as create from './create'

export type { DropPosition, FileMaker, FilesTabDeps, FilesTabState, MenuRequest }

/** Where a row activated takes the person: the file the row stands for. */
export const getLandingDestination = (entry: Entry): SearchDestination | null =>
  entry.isFolder ? null : { at: 'file', path: entry.path, title: entry.name }

function useFileCreation(
  list: FileTree,
  deps: FilesTabDeps,
  renamingPath: ReturnType<typeof ref<string | null>>,
) {
  const createFolder = async (path: string | null) => {
    const made = await create.createFolder(list, deps, path)
    if (made) renamingPath.value = made
  }

  const createNote = async (path: string | null) => {
    const made = await create.createNote(list, deps, path)
    if (made) renamingPath.value = made
  }

  const createOne = async (path: string | null, createEntry: FileMaker, name: string) => {
    const made = await create.createOne(list, path, { createEntry, name })
    if (made) renamingPath.value = made
  }

  return { createFolder, createNote, createOne }
}

function useFileMoves(list: FileTree, deps: FilesTabDeps) {
  const move = async (paths: readonly string[], at: DropPosition) => {
    if (paths.length === 0) return

    const into = resolveDropFolder(at)
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

  const chooseMenuItem = (id: string) => {
    const request = menu.value
    menu.value = null
    if (!request || !OFFERED.has(id)) return
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

  return { menu, openMenu, dismissMenu, chooseMenuItem, getOverPaths }
}

export function useFilesTab(list: FileTree, deps: FilesTabDeps): FilesTabState {
  const renamingPath = ref<string | null>(null)
  const setRenamingPath = (path: string | null) => {
    renamingPath.value = path
  }

  const getNameOf = (path: string): string => list.getEntryAt(path)?.name ?? getFileName(path)
  const getSourceOf = (path: string): Source => list.getEntryAt(path)?.kind ?? 'other'

  const creators = useFileCreation(list, deps, renamingPath)
  const { move } = useFileMoves(list, deps)
  const fileMenu = useFileMenu({ list, deps, renamingPath, creators, getNameOf, getSourceOf })

  const activate = (path: string) => {
    const entry = list.getEntryAt(path)
    if (!entry) return
    list.selectPaths([path])
    deps.openDestination(getLandingDestination(entry), tabState)
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

  const canRun: RunGuard = (run) => deps.canRun?.(run) ?? true

  const tabState: FilesTabState = {
    list,
    menu: fileMenu.menu,
    renamingPath,
    setRenamingPath,
    getOverPaths: fileMenu.getOverPaths,
    getFolderFor: (path) => create.getFolderFor(list, path),
    activate,
    open: (path) => void list.openFolder(path),
    close: (path) => list.closeFolder(path),
    select: (paths) => list.selectPaths(paths),
    rename,
    move,
    drag,
    drop: () => deps.setDraggedPaths([]),
    remove,
    createFolder: creators.createFolder,
    createNote: creators.createNote,
    createOne: creators.createOne,
    importUrl: (url) =>
      deps.importUrl(create.getFolderFor(list, list.selectedPaths.value[0] ?? null), url),
    openMenu: fileMenu.openMenu,
    dismissMenu: fileMenu.dismissMenu,
    chooseMenuItem: fileMenu.chooseMenuItem,
    getNameOf,
    canRun,
    getArtifactStates: deps.getArtifactStates,
  }

  return tabState
}
