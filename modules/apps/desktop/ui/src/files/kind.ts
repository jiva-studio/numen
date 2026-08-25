/**
 * What one files tab holds: the tree of the vault, and what a gesture in it
 * does.
 *
 * The tree reports the shape of a gesture and nothing else. Where a row
 * activated takes the person, what the menu on a row offers, and what a name
 * typed over a row comes to are decided here, so a test can ask them without a
 * screen.
 */
import { ref } from 'vue'
import type { Entry, Went } from '../core'
import type { Landing } from '../finding'
import { folderOf, landedIn, type Listing, ROOT } from './listing'
import { NEW_FOLDER, OFFERED, RENAME } from './menu'
import type { Host, Kind } from '../windowing'
import { FILES } from '../workspace'
import FilesTab from './FilesTab.vue'
import { WORDS as words } from './words'

/** Where the menu on a row stands, and what it was asked for on. */
export interface Asked {
  readonly path: string
  readonly at: { x: number; y: number }
}

/** Where a row let go of landed, as the tree reports it. */
export type Dropped = { readonly into: string } | { readonly before: string }

/** What a files tab asks of the window it is drawn in. */
export interface Filing {
  /** Somewhere chosen, taken. Nothing chosen takes the person nowhere. */
  lands(landing: Landing | null): void
  /**
   * A command asked for on a row, on the file it stands for. One that needs
   * something asks for it in the palette; the rest happen where they stand.
   */
  runs(id: string, path: string, name: string): void
  /** A file or a folder filed somewhere else, under the name the path ends in. */
  moves(from: string, to: string): Promise<void>
  /** An empty folder. The folders above it are made with it. */
  makes(path: string): Promise<void>
}

/**
 * Where a row activated takes the person. A note is read, a book is opened at
 * its first page, and anything else is somewhere to go nowhere.
 */
export const landingOf = (entry: Entry): Landing | null => {
  if (entry.folder) return null
  if (entry.kind === 'note') return { at: 'note', path: entry.path, title: entry.name }
  if (entry.kind === 'book') {
    return { at: 'document', path: entry.path, title: entry.name, start: 0, length: 0 }
  }
  return null
}

/**
 * A name typed over a row, as the path the file is filed under from now on.
 * A name that is the one it carries, or that names a folder of its own, moves
 * nothing.
 */
export const renamedTo = (path: string, name: string): string => {
  const called = name.trim()
  if (!called || called.includes('/') || called === path.split('/').pop()) return ''
  const folder = folderOf(path)
  return folder === ROOT ? called : `${folder}/${called}`
}

/** What one files tab holds. */
export type Held = ReturnType<typeof filing>

/**
 * The files tab of a window. A window shows the vault once, so a second asked
 * for is the tree already open.
 */
export function filesKind(host: Host, makes: () => Listing, deps: Filing) {
  const kind: Kind<Held> = {
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
  const front = (): Held | null => host.last<Held>(FILES)?.held ?? null

  /**
   * The tree put in front of the person, walked down to a path. The window that
   * holds none opens one on it.
   */
  const reveals = async (path: string) => {
    const id = await host.opens(FILES)
    await host.holds<Held>(FILES, id)?.list.reveals(path)
  }

  /** The vault changed, and every open folder a named path sits in is read again. */
  const changed = (paths: readonly string[], renamed: readonly Went[] = []) =>
    front()?.list.changed(paths, renamed) ?? Promise.resolve()

  return { kind, reveals, changed }
}

export function filing(list: Listing, deps: Filing) {
  /** The menu on a row, for as long as it stands. */
  const menu = ref<Asked | null>(null)
  /** The row whose name is in a field, and nothing while none is. */
  const renaming = ref<string | null>(null)

  /** A row activated: what it stands for is put in front of the person. */
  const activate = (path: string) => {
    const entry = list.entryAt(path)
    if (!entry) return
    list.chooses(path)
    deps.lands(landingOf(entry))
  }

  /** A folder opened or closed, and what it holds read as it opens. */
  const open = (path: string) => void list.opens(path)
  const close = (path: string) => list.closes(path)

  /** The row the person is standing on. */
  const select = (path: string) => list.chooses(path)

  /**
   * A row given a different name. Only the file is renamed: what a note calls
   * itself is its own, and stays as it was.
   */
  const rename = async (path: string, name: string) => {
    const to = renamedTo(path, name)
    if (!to) return
    await deps.moves(path, to)
    await list.again()
  }

  /**
   * A row let go of somewhere. The tree refuses a row dropped into itself or
   * into anything under it, so what arrives here is somewhere else. A folder is
   * carried whole, with everything filed inside it.
   */
  const move = async (path: string, at: Dropped) => {
    const into = landedIn(at)
    const name = path.split('/').pop() ?? path
    const to = into === ROOT ? name : `${into}/${name}`
    if (to === path) return
    await deps.moves(path, to)
    await list.again()
  }

  /**
   * A folder made where the row stands, under a name nothing there carries, and
   * its name put in a field for the person to type over.
   */
  const makes = async (path: string) => {
    const entry = list.entryAt(path)
    const into = entry?.folder ? entry.path : folderOf(path)
    const name = list.freeIn(into, words.folder)
    const made = into === ROOT ? name : `${into}/${name}`
    await deps.makes(made)
    await list.opens(into)
    renaming.value = made
  }

  /** A menu asked for on a row, and one put away. */
  const asks = (asked: Asked) => {
    menu.value = asked
  }
  const dismiss = () => {
    menu.value = null
  }

  /** An item chosen in the menu, on the file it was asked for on. */
  const chose = (id: string) => {
    const asking = menu.value
    menu.value = null
    if (!asking || !OFFERED.has(id)) return
    if (id === NEW_FOLDER) return void makes(asking.path)
    if (id === RENAME) {
      renaming.value = asking.path
      return
    }
    deps.runs(id, asking.path, nameOf(asking.path))
  }

  /** What a file is called, which is the last segment of the path it is filed at. */
  const nameOf = (path: string): string =>
    list.entryAt(path)?.name ?? (path.split('/').pop() ?? path)

  return {
    list,
    menu,
    renaming,
    activate,
    open,
    close,
    select,
    rename,
    move,
    makes,
    asks,
    dismiss,
    chose,
    nameOf,
  }
}
