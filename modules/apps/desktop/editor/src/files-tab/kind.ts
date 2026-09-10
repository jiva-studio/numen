/**
 * What one files tab holds: the tree of the vault, and what a gesture in it
 * does. The tree reports the shape of a gesture, and what it comes to is
 * decided here.
 */
import { ref } from 'vue'
import type { Entry, Move, Source } from '../shared/core'
import type { SearchDestination } from '../shared/command/search'
import { folderOf, landedIn, type FileTree, ROOT } from './listing'
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
import type { TabKind, WindowHandle } from '../shared/tabs/windowTabs'
import { FILES } from '../shared/tabs/workspace'
import FilesTab from './FilesTab.vue'
import { fileOf } from '../shared/paths'
import { WORDS as words } from './words'

/** Where the menu stands, and what it was asked for on. */
export interface MenuRequest {
  /** The row it was asked for on, and nothing where it was asked off every row. */
  readonly path: string | null
  readonly at: { x: number; y: number }
}

/** Where rows let go of landed, as the tree reports it. */
export type DropPosition = { readonly into: string } | { readonly before: string }

/** What a files tab asks of the window it is drawn in. */
export interface FilesTabDeps {
  /** Somewhere chosen, taken. Nothing chosen takes the person nowhere. */
  lands(going: SearchDestination | null): void
  /**
   * A command asked for on the files the rows stand for, under what the vault
   * holds at the first of them. One that needs something asks for it in the
   * palette; the rest happen where they stand.
   */
  runs(id: string, paths: readonly string[], name: string, source: Source): void
  /** A file or a folder filed somewhere else, under the name the path ends in. */
  moves(from: string, to: string): Promise<void>
  /**
   * The notes the tree is dragging over the rest of the window, and none once
   * it has let go.
   */
  drags(paths: readonly string[]): void
  /** An empty folder. The folders above it are made with it. */
  makes(path: string): Promise<void>
  /**
   * A note made in a folder, under a name nothing there carries. The path it
   * landed at, and nothing where none was made.
   */
  writes(folder: string): Promise<string>
  /**
   * A deck made in a folder, under the name it is given. The path it landed at,
   * and nothing where none was made.
   */
  decks(folder: string, name: string): Promise<string>
  /** A stencil made the same way. */
  stencils(folder: string, name: string): Promise<string>
  /** A preset made the same way, naming none of its settings. */
  presets(folder: string, name: string): Promise<string>
  /**
   * The file a web address is kept in, made in a folder, and what is at that
   * address fetched into the store beside it.
   */
  imports(folder: string, address: string): Promise<string>
  /** What could not be done, in words a person reads. */
  says(text: string): void
  /**
   * Whether this build can do a run at all, which decides whether the menu on
   * a row offers it. A window that says nothing offers every run.
   */
  canRun?: RunGuard
}

/** One of the three files the vault names itself, made in a folder. */
type FileMaker = (folder: string, name: string) => Promise<string>

/**
 * Where a row activated takes the person: the file the row stands for, under
 * the name it is filed as. A folder is somewhere to go nowhere, and what the
 * file opens in is not decided here.
 */
export const landingOf = (entry: Entry): SearchDestination | null =>
  entry.folder ? null : { at: 'file', path: entry.path, title: entry.name }

/** What one files tab holds. */
export type FilesTabState = ReturnType<typeof useFilesTab>

/**
 * The files tab of a window. A window shows the vault once, so a second asked
 * for is the tree already open.
 */
export function filesKind(handle: WindowHandle, makes: () => FileTree, deps: FilesTabDeps) {
  const kind: TabKind<FilesTabState, typeof FILES> = {
    kind: FILES,
    opens: () => {
      const state = useFilesTab(makes(), deps)
      void state.list.opens(ROOT)
      return state
    },
    called: () => words.files,
    draws: FilesTab,
    identity: () => FILES,
    shuts: (state) => {
      state.list.close()
      return true
    },
  }

  /** The tree of this window, and nothing while it holds none. */
  const front = (): FilesTabState | null => handle.last<FilesTabState>(FILES)?.state ?? null

  /**
   * The tree put in front of the person, walked down to a path. The window that
   * holds none opens one on it.
   */
  const revealPath = async (path: string) => {
    const id = await handle.opens(FILES)
    await handle.holds<FilesTabState>(FILES, id)?.list.reveals(path)
  }

  /** The vault changed, and every open folder a named path sits in is read again. */
  const refreshChangedPaths = (paths: readonly string[], renamed: readonly Move[] = []) =>
    front()?.list.changed(paths, renamed) ?? Promise.resolve()

  return { kind, revealPath, refreshChangedPaths }
}

export function useFilesTab(list: FileTree, deps: FilesTabDeps) {
  /** The menu on a row, for as long as it stands. */
  const menu = ref<MenuRequest | null>(null)
  /** The row whose name is in a field, and nothing while none is. */
  const renaming = ref<string | null>(null)

  /** The files a gesture on a row is over: the selection it stands in, or it alone. */
  const over = (path: string): readonly string[] =>
    list.chosen.value.includes(path) ? list.chosen.value : [path]

  /**
   * The folder something made on a row lands in, and the folder a file carried
   * in from outside the window is filed in: the folder the row stands for, or
   * the folder the row sits in. A gesture off every row lands at the root.
   */
  const folderFor = (path: string | null): string => {
    if (path === null) return ROOT
    const entry = list.entryAt(path)
    return entry?.folder ? entry.path : folderOf(path)
  }

  /** A row activated: what it stands for is put in front of the person. */
  const activate = (path: string) => {
    const entry = list.entryAt(path)
    if (!entry) return
    list.chooses([path])
    deps.lands(landingOf(entry))
  }

  /** A folder opened or closed, and what it holds read as it opens. */
  const open = (path: string) => void list.opens(path)
  const close = (path: string) => list.closes(path)

  /** The rows the person is standing on. */
  const select = (paths: readonly string[]) => list.chooses(paths)

  /**
   * A row given a different name. What a note calls itself follows where a
   * title and a filename are kept as one name.
   */
  const rename = async (path: string, name: string) => {
    const to = renamedTo(path, name, list.entryAt(path)?.folder ?? false)
    if (!to) return
    await deps.moves(path, to)
    await list.again()
  }

  /**
   * Rows let go of somewhere, each filed in the folder the drop landed in
   * under the name it carries. The tree refuses a drop into one of the rows or
   * into anything under one, so what arrives here is somewhere else. A folder
   * is dragged whole, with everything filed inside it.
   *
   * A row whose name is taken in that folder stays where it is and is said;
   * the rest go. The folders are read again once, when all of them are done.
   */
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

  /**
   * Rows lifted clear of the tree and dragged over the rest of the window.
   *
   * The vault's links are between notes, so the notes among the rows are what
   * is dragged and the rest stay where they are. Rows holding no note at all
   * drag nothing.
   */
  const drag = (paths: readonly string[]) => {
    deps.drags(
      paths.filter((path) => {
        const entry = list.entryAt(path)
        return !!entry && !entry.folder && entry.kind === 'note'
      }),
    )
  }

  /** The rows let go of, wherever that was. */
  const drop = () => deps.drags([])

  /** The rows asked to go, handed to the window as one command over all of them. */
  const remove = (paths: readonly string[]) => {
    const first = paths[0]
    if (first === undefined) return
    deps.runs('remove', paths, nameOf(first), sourceOf(first))
  }

  /**
   * A folder made where the row stands, under a name nothing there carries, and
   * its name put in a field for the person to type over.
   */
  const makes = async (path: string | null) => {
    const into = folderFor(path)
    const name = list.freeIn(into, words.folder)
    const made = into === ROOT ? name : `${into}/${name}`
    await deps.makes(made)
    await list.opens(into)
    // The folder the vault holds is the one to name. Nothing there is a
    // refusal, and it has already been said.
    if (list.entryAt(made)) renaming.value = made
  }

  /**
   * A note made where the row stands, under a name nothing there carries, and
   * its name put in a field for the person to type over.
   */
  const writes = async (path: string | null) => {
    const into = folderFor(path)
    const made = await deps.writes(into)
    if (!made) return
    await list.opens(into)
    renaming.value = made
  }

  /** An address dropped on the tree, made into the file it is kept in. */
  const imports = (address: string) =>
    deps.imports(folderFor(list.chosen.value[0] ?? null), address)

  /**
   * A deck, a stencil or a preset made where the row stands, and its name put
   * in a field for the person to type over. The vault names the file and
   * answers where it stands, so a name already taken there comes back as a
   * refusal.
   */
  const makesOne = async (path: string | null, makes: FileMaker, name: string) => {
    const into = folderFor(path)
    const made = await makes(into, name)
    if (!made) return
    await list.opens(into)
    renaming.value = made
  }

  /** The row whose name the person is typing over, and none once they are done. */
  const setRenamingPath = (path: string | null) => {
    renaming.value = path
  }

  /** A menu asked for on a row or off every row, and one put away. */
  const openMenu = (asked: MenuRequest) => {
    menu.value = asked
  }
  const dismiss = () => {
    menu.value = null
  }

  /** An item chosen in the menu, on the files it was asked for on. */
  const chooseMenuItem = (id: string) => {
    const asking = menu.value
    menu.value = null
    if (!asking || !OFFERED.has(id)) return
    if (id === NEW_NOTE) return void writes(asking.path)
    if (id === NEW_DECK) return void makesOne(asking.path, deps.decks, words.newDeck)
    if (id === NEW_STENCIL) return void makesOne(asking.path, deps.stencils, words.newStencil)
    if (id === NEW_PRESET) return void makesOne(asking.path, deps.presets, words.newPreset)
    if (id === NEW_FOLDER) return void makes(asking.path)

    const path = asking.path
    if (path === null) return
    if (id === RENAME) {
      renaming.value = path
      return
    }
    deps.runs(id, over(path), nameOf(path), sourceOf(path))
  }

  /** What a file is called, which is the last segment of the path it is filed at. */
  const nameOf = (path: string): string =>
    list.entryAt(path)?.name ?? fileOf(path)

  /** What the vault holds at a row, and none of the three where it holds none. */
  const sourceOf = (path: string): Source => list.entryAt(path)?.kind ?? 'other'

  /** Whether this build can do a run at all, as the menu on a row asks it. */
  const canRun: RunGuard = (run) => deps.canRun?.(run) ?? true

  return {
    list,
    menu,
    renaming,
    setRenamingPath,
    over,
    folderFor,
    activate,
    open,
    close,
    select,
    rename,
    move,
    drag,
    drop,
    remove,
    makes,
    writes,
    makesOne,
    imports,
    openMenu,
    dismiss,
    chooseMenuItem,
    nameOf,
    canRun,
  }
}
