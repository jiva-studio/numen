/**
 * Tab state, tree interactions, and window tab registration for the files tab.
 */
import { ref } from 'vue'
import type { Entry, Source } from '@/entities/file'
import type { SearchDestination } from '@/features/command-palette'
import { fileOf } from '@/shared/paths'
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
  entry.folder ? null : { at: 'file', path: entry.path, title: entry.name }

export function useFilesTab(list: FileTree, deps: FilesTabDeps): FilesTabState {
  const menu = ref<MenuRequest | null>(null)
  const renamingPath = ref<string | null>(null)

  const getOverPaths = (path: string): readonly string[] =>
    list.selectedPaths.value.includes(path) ? list.selectedPaths.value : [path]

  const activate = (path: string) => {
    const entry = list.getEntryAt(path)
    if (!entry) return
    list.selectPaths([path])
    deps.openDestination(getLandingDestination(entry))
  }

  const open = (path: string) => void list.openFolder(path)
  const close = (path: string) => list.closeFolder(path)
  const select = (paths: readonly string[]) => list.selectPaths(paths)

  const rename = async (path: string, name: string) => {
    const to = resolveRenamePath(path, name, list.getEntryAt(path)?.folder ?? false)
    if (!to) return
    await deps.movePath(path, to)
    await list.refresh()
  }

  const move = async (paths: readonly string[], at: DropPosition) => {
    if (paths.length === 0) return

    const into = resolveDropFolder(at)
    await list.loadFolder(into)
    const taken = new Set(list.getEntriesInFolder(into).map((one) => one.name))
    const refused: string[] = []

    for (const path of paths) {
      const name = fileOf(path)
      const to = into === ROOT ? name : `${into}/${name}`
      if (to === path) continue
      if (taken.has(name)) {
        refused.push(name)
        continue
      }
      taken.add(name)
      await deps.movePath(path, to)
    }

    if (refused.length > 0) deps.showError(`${words.taken} ${refused.join(', ')}`)
    await list.refresh()
  }

  const drag = (paths: readonly string[]) => {
    deps.setDraggedPaths(
      paths.filter((path) => {
        const entry = list.getEntryAt(path)
        return !!entry && !entry.folder && entry.kind === 'note'
      }),
    )
  }

  const drop = () => deps.setDraggedPaths([])

  const remove = (paths: readonly string[]) => {
    const first = paths[0]
    if (first === undefined) return
    deps.runCommand('remove', paths, getNameOf(first), getSourceOf(first))
  }

  const createFolder = async (path: string | null) => {
    const made = await create.createFolder(list, deps, path)
    if (made) renamingPath.value = made
  }

  const createNote = async (path: string | null) => {
    const made = await create.createNote(list, deps, path)
    if (made) renamingPath.value = made
  }

  const createOne = async (path: string | null, createEntry: FileMaker, name: string) => {
    const made = await create.createOne(list, path, createEntry, name)
    if (made) renamingPath.value = made
  }

  const importUrl = (url: string) =>
    deps.importUrl(create.getFolderFor(list, list.selectedPaths.value[0] ?? null), url)

  const setRenamingPath = (path: string | null) => {
    renamingPath.value = path
  }

  const openMenu = (request: MenuRequest) => {
    menu.value = request
  }

  const dismissMenu = () => {
    menu.value = null
  }

  const chooseMenuItem = (id: string) => {
    const request = menu.value
    menu.value = null
    if (!request || !OFFERED.has(id)) return
    if (id === NEW_NOTE) return void createNote(request.path)
    if (id === NEW_DECK) return void createOne(request.path, deps.createDeck, words.newDeck)
    if (id === NEW_STENCIL)
      return void createOne(request.path, deps.createStencil, words.newStencil)
    if (id === NEW_PRESET) return void createOne(request.path, deps.createPreset, words.newPreset)
    if (id === NEW_FOLDER) return void createFolder(request.path)

    const path = request.path
    if (path === null) return
    if (id === RENAME) {
      renamingPath.value = path
      return
    }
    deps.runCommand(id, getOverPaths(path), getNameOf(path), getSourceOf(path))
  }

  const getNameOf = (path: string): string => list.getEntryAt(path)?.name ?? fileOf(path)

  const getSourceOf = (path: string): Source => list.getEntryAt(path)?.kind ?? 'other'

  const canRun: RunGuard = (run) => deps.canRun?.(run) ?? true

  return {
    list,
    menu,
    renamingPath,
    setRenamingPath,
    getOverPaths,
    getFolderFor: (path) => create.getFolderFor(list, path),
    activate,
    open,
    close,
    select,
    rename,
    move,
    drag,
    drop,
    remove,
    createFolder,
    createNote,
    createOne,
    importUrl,
    openMenu,
    dismissMenu,
    chooseMenuItem,
    getNameOf,
    canRun,
  }
}
