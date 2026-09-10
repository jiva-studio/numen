/**
 * Tab state and tree interactions for the files tab.
 */
import { ref } from 'vue'
import type { Entry, Source } from '../shared/core'
import type { SearchDestination } from '../shared/command/search'
import { fileOf } from '../shared/paths'
import { landedIn, type FileTree, ROOT } from './listing'
import {
  NEW_DECK,
  NEW_FOLDER,
  NEW_NOTE,
  NEW_PRESET,
  NEW_STENCIL,
  OFFERED,
  RENAME,
  type RunGuard,
} from './menu'
import { renamedTo } from './rename'
import { WORDS as words } from './words'
import type { DropPosition, FileMaker, FilesTabDeps, FilesTabState, MenuRequest } from './types'
import { createFolder, createNote, createOne, folderFor } from './create'

/** Where a row activated takes the person: the file the row stands for. */
export const landingOf = (entry: Entry): SearchDestination | null =>
  entry.folder ? null : { at: 'file', path: entry.path, title: entry.name }

export function useFilesTab(list: FileTree, deps: FilesTabDeps): FilesTabState {
  const menu = ref<MenuRequest | null>(null)
  const renaming = ref<string | null>(null)

  const over = (path: string): readonly string[] =>
    list.chosen.value.includes(path) ? list.chosen.value : [path]

  const activate = (path: string) => {
    const entry = list.entryAt(path)
    if (!entry) return
    list.chooses([path])
    deps.lands(landingOf(entry))
  }

  const open = (path: string) => void list.opens(path)
  const close = (path: string) => list.closes(path)
  const select = (paths: readonly string[]) => list.chooses(paths)

  const rename = async (path: string, name: string) => {
    const to = renamedTo(path, name, list.entryAt(path)?.folder ?? false)
    if (!to) return
    await deps.moves(path, to)
    await list.again()
  }

  const move = async (paths: readonly string[], at: DropPosition) => {
    if (paths.length === 0) return

    const into = landedIn(at)
    await list.lists(into)
    const taken = new Set(list.entriesIn(into).map((one) => one.name))
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
      await deps.moves(path, to)
    }

    if (refused.length > 0) deps.says(`${words.taken} ${refused.join(', ')}`)
    await list.again()
  }

  const drag = (paths: readonly string[]) => {
    deps.drags(
      paths.filter((path) => {
        const entry = list.entryAt(path)
        return !!entry && !entry.folder && entry.kind === 'note'
      }),
    )
  }

  const drop = () => deps.drags([])

  const remove = (paths: readonly string[]) => {
    const first = paths[0]
    if (first === undefined) return
    deps.runs('remove', paths, nameOf(first), sourceOf(first))
  }

  const handleCreateFolder = async (path: string | null) => {
    const made = await createFolder(list, deps, path)
    if (made) renaming.value = made
  }

  const handleCreateNote = async (path: string | null) => {
    const made = await createNote(list, deps, path)
    if (made) renaming.value = made
  }

  const handleCreateOne = async (path: string | null, createEntry: FileMaker, name: string) => {
    const made = await createOne(list, path, createEntry, name)
    if (made) renaming.value = made
  }

  const imports = (address: string) =>
    deps.imports(folderFor(list, list.chosen.value[0] ?? null), address)

  const setRenamingPath = (path: string | null) => {
    renaming.value = path
  }

  const openMenu = (asked: MenuRequest) => {
    menu.value = asked
  }

  const dismiss = () => {
    menu.value = null
  }

  const chooseMenuItem = (id: string) => {
    const asking = menu.value
    menu.value = null
    if (!asking || !OFFERED.has(id)) return
    if (id === NEW_NOTE) return void handleCreateNote(asking.path)
    if (id === NEW_DECK) return void handleCreateOne(asking.path, deps.decks, words.newDeck)
    if (id === NEW_STENCIL) return void handleCreateOne(asking.path, deps.stencils, words.newStencil)
    if (id === NEW_PRESET) return void handleCreateOne(asking.path, deps.presets, words.newPreset)
    if (id === NEW_FOLDER) return void handleCreateFolder(asking.path)

    const path = asking.path
    if (path === null) return
    if (id === RENAME) {
      renaming.value = path
      return
    }
    deps.runs(id, over(path), nameOf(path), sourceOf(path))
  }

  const nameOf = (path: string): string =>
    list.entryAt(path)?.name ?? fileOf(path)

  const sourceOf = (path: string): Source => list.entryAt(path)?.kind ?? 'other'

  const canRun: RunGuard = (run) => deps.canRun?.(run) ?? true

  return {
    list,
    menu,
    renaming,
    setRenamingPath,
    over,
    folderFor: (path) => folderFor(list, path),
    activate,
    open,
    close,
    select,
    rename,
    move,
    drag,
    drop,
    remove,
    makes: handleCreateFolder,
    createFolder: handleCreateFolder,
    writes: handleCreateNote,
    createNote: handleCreateNote,
    makesOne: handleCreateOne,
    createOne: handleCreateOne,
    imports,
    importAddress: imports,
    importUrl: imports,
    openMenu,
    asks: openMenu,
    dismiss,
    chooseMenuItem,
    nameOf,
    getName: nameOf,
    canRun,
  }
}
