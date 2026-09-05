/**
 * What one files tab holds: the tree of the vault, and what a gesture in it
 * does. The tree reports the shape of a gesture, and what it comes to is
 * decided here.
 */
import { ref } from 'vue'
import type { Entry, Move, Source } from '../core'
import type { Landing } from '../finding'
import { folderOf, landedIn, type Listing, ROOT } from './listing'
import {
  NEW_DECK,
  NEW_FOLDER,
  NEW_NOTE,
  NEW_PRESET,
  NEW_STENCIL,
  OFFERED,
  RENAME,
  type CanRun,
} from './menu'
import type { Host, Kind } from '../windowing'
import { FILES } from '../workspace'
import FilesTab from './FilesTab.vue'
import { WORDS as words } from './words'

/** Where the menu stands, and what it was asked for on. */
export interface MenuRequest {
  /** The row it was asked for on, and nothing where it was asked off every row. */
  readonly path: string | null
  readonly at: { x: number; y: number }
}

/** Where rows let go of landed, as the tree reports it. */
export type Dropped = { readonly into: string } | { readonly before: string }

/** What a files tab asks of the window it is drawn in. */
export interface FilesTabDeps {
  /** Somewhere chosen, taken. Nothing chosen takes the person nowhere. */
  lands(landing: Landing | null): void
  /**
   * A command asked for on the files the rows stand for, under what the vault
   * holds at the first of them. One that needs something asks for it in the
   * palette; the rest happen where they stand.
   */
  runs(id: string, paths: readonly string[], name: string, source: Source): void
  /** A file or a folder filed somewhere else, under the name the path ends in. */
  moves(from: string, to: string): Promise<void>
  /**
   * The notes the tree is carrying over the rest of the window, and none once
   * it has let go.
   */
  carries(paths: readonly string[]): void
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
  cuts(folder: string, name: string): Promise<string>
  /** A stencil made the same way. */
  stencils(folder: string, name: string): Promise<string>
  /** A preset made the same way, naming none of its settings. */
  presets(folder: string, name: string): Promise<string>
  /** What could not be done, in words a person reads. */
  says(text: string): void
  /**
   * Whether this build can do a run at all, which decides whether the menu on
   * a row offers it. A window that says nothing offers every run.
   */
  canRun?: CanRun
}

/** One of the three files the vault names itself, made in a folder. */
type Cut = (folder: string, name: string) => Promise<string>

/**
 * Where a row activated takes the person: the file the row stands for, under
 * the name it is filed as. A folder is somewhere to go nowhere, and what the
 * file opens in is not decided here.
 */
export const landingOf = (entry: Entry): Landing | null =>
  entry.folder ? null : { at: 'file', path: entry.path, title: entry.displayName }

/** What a file is filed as, which is the last segment of the path. */
const fileOf = (path: string): string => path.split('/').pop() ?? path

/**
 * Where a name's ending begins, and nowhere for a name carrying none. An ending
 * is the last dot and what follows it, and what follows it holds no space.
 */
const endingAt = (name: string): number => {
  const cut = name.lastIndexOf('.')
  return cut >= 0 && !/\s/u.test(name.slice(cut)) ? cut : -1
}

/**
 * The ending a name carries, the dot with it, and nothing where it carries
 * none. A name that is a dot and an ending carries none: that is its whole
 * name, and it has none to lend.
 */
const endingOf = (name: string): string => {
  const at = endingAt(name)
  return at > 0 ? name.slice(at) : ''
}

/**
 * A name typed over a row, as the path the file is filed under from now on.
 * A name carrying no ending keeps the one the file has, so a note typed over
 * stays a note. A folder keeps whatever was typed.
 *
 * A name that is the one it carries, or that names a folder of its own, moves
 * nothing.
 */
export const renamedTo = (path: string, name: string, folder = false): string => {
  const typed = name.trim()
  if (!typed || typed.includes('/')) return ''

  const carries = endingAt(typed) >= 0
  const called = folder || carries ? typed : `${typed}${endingOf(fileOf(path))}`
  if (called === fileOf(path)) return ''

  const under = folderOf(path)
  return under === ROOT ? called : `${under}/${called}`
}

/** What one files tab holds. */
export type FilesTabState = ReturnType<typeof filing>

/**
 * The files tab of a window. A window shows the vault once, so a second asked
 * for is the tree already open.
 */
export function filesKind(host: Host, makes: () => Listing, deps: FilesTabDeps) {
  const kind: Kind<FilesTabState> = {
    kind: FILES,
    opens: () => {
      const held = filing(makes(), deps)
      void held.list.opens(ROOT)
      return held
    },
    called: () => words.files,
    draws: FilesTab,
    identity: () => FILES,
    shuts: (held) => {
      held.list.close()
      return true
    },
  }

  /** The tree of this window, and nothing while it holds none. */
  const front = (): FilesTabState | null => host.last<FilesTabState>(FILES)?.held ?? null

  /**
   * The tree put in front of the person, walked down to a path. The window that
   * holds none opens one on it.
   */
  const reveals = async (path: string) => {
    const id = await host.opens(FILES)
    await host.holds<FilesTabState>(FILES, id)?.list.reveals(path)
  }

  /** The vault changed, and every open folder a named path sits in is read again. */
  const changed = (paths: readonly string[], renamed: readonly Move[] = []) =>
    front()?.list.changed(paths, renamed) ?? Promise.resolve()

  return { kind, reveals, changed }
}

export function filing(list: Listing, deps: FilesTabDeps) {
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
   * is carried whole, with everything filed inside it.
   *
   * A row whose name is taken in that folder stays where it is and is said;
   * the rest go. The folders are read again once, when all of them are done.
   */
  const move = async (paths: readonly string[], at: Dropped) => {
    if (paths.length === 0) return

    const into = landedIn(at)
    await list.lists(into)
    const taken = new Set(list.entriesIn(into).map((one) => one.displayName))
    const refused: string[] = []

    for (const path of paths) {
      const name = path.split('/').pop() ?? path
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
   * Rows lifted clear of the tree and carried over the rest of the window.
   *
   * The vault's links are between notes, so the notes among the rows are what
   * is carried and the rest stay where they are. Rows holding no note at all
   * carry nothing.
   */
  const carry = (paths: readonly string[]) => {
    deps.carries(
      paths.filter((path) => {
        const entry = list.entryAt(path)
        return !!entry && !entry.folder && entry.kind === 'note'
      }),
    )
  }

  /** The rows let go of, wherever that was. */
  const drop = () => deps.carries([])

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

  /**
   * A deck, a stencil or a preset made where the row stands, and its name put
   * in a field for the person to type over. The vault names the file and
   * answers where it stands, so a name already taken there comes back as a
   * refusal.
   */
  const cuts = async (path: string | null, cut: Cut, name: string) => {
    const into = folderFor(path)
    const made = await cut(into, name)
    if (!made) return
    await list.opens(into)
    renaming.value = made
  }

  /** The row whose name the person is typing over, and none once they are done. */
  const renames = (path: string | null) => {
    renaming.value = path
  }

  /** A menu asked for on a row or off every row, and one put away. */
  const asks = (asked: MenuRequest) => {
    menu.value = asked
  }
  const dismiss = () => {
    menu.value = null
  }

  /** An item chosen in the menu, on the files it was asked for on. */
  const chose = (id: string) => {
    const asking = menu.value
    menu.value = null
    if (!asking || !OFFERED.has(id)) return
    if (id === NEW_NOTE) return void writes(asking.path)
    if (id === NEW_DECK) return void cuts(asking.path, deps.cuts, words.newDeck)
    if (id === NEW_STENCIL) return void cuts(asking.path, deps.stencils, words.newStencil)
    if (id === NEW_PRESET) return void cuts(asking.path, deps.presets, words.newPreset)
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
    list.entryAt(path)?.displayName ?? (path.split('/').pop() ?? path)

  /** What the vault holds at a row, and none of the three where it holds none. */
  const sourceOf = (path: string): Source => list.entryAt(path)?.kind ?? 'other'

  /** Whether this build can do a run at all, as the menu on a row asks it. */
  const canRun: CanRun = (run) => deps.canRun?.(run) ?? true

  return {
    list,
    menu,
    renaming,
    renames,
    over,
    folderFor,
    activate,
    open,
    close,
    select,
    rename,
    move,
    carry,
    drop,
    remove,
    makes,
    writes,
    cuts,
    asks,
    dismiss,
    chose,
    nameOf,
    canRun,
  }
}
