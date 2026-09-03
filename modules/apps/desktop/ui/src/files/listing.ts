/**
 * What a files tab knows about the vault: what each open folder holds, which
 * folders are open, and which rows are chosen.
 *
 * A folder is read again when it is opened, when something this tab did
 * finished, when a change names a path inside it, and when the window comes
 * back to the front. A file the vault holds no source for is not reported by
 * the watcher, and appears at the next of those.
 */
import { computed, ref, shallowRef } from 'vue'
import { wentTo, type Entry, type Went } from '../core'

/** Everything a files tab asks of the application. */
export interface Folders {
  /** What one folder holds, in the order to draw it. The root is the empty path. */
  list(folder: string): Promise<readonly Entry[]>
}

/** The vault's root, which is the folder every other one is under. */
export const ROOT = ''

/**
 * One line of the tree: what it stands for, and the lines drawn under it. A
 * folder that is closed holds none until it opens.
 */
export interface Row {
  readonly entry: Entry
  readonly rows: readonly Row[]
}

/** The folder a path sits in, and the root for a path at the top of the vault. */
export const folderOf = (path: string): string => {
  const cut = path.lastIndexOf('/')
  return cut < 0 ? ROOT : path.slice(0, cut)
}

/**
 * The folder a row let go of lands in: the one it went into, or the one holding
 * the row it came before. Into no row at all is the root.
 */
export const landedIn = (at: { into: string | null } | { before: string }): string =>
  'into' in at ? (at.into ?? ROOT) : folderOf(at.before)

/** The folders above a path, from the root down. The root itself is in none of them. */
export const above = (path: string): readonly string[] => {
  const parts = path.split('/').slice(0, -1)
  return parts.map((_, deep) => parts.slice(0, deep + 1).join('/'))
}

/**
 * A name nothing in a folder carries. The first is the word itself, and each
 * after it is that word and a count.
 */
export const freeName = (taken: readonly string[], word: string): string => {
  const held = new Set(taken)
  if (!held.has(word)) return word
  for (let count = 2; ; count++) {
    const name = `${word} ${count}`
    if (!held.has(name)) return name
  }
}

export type Listing = ReturnType<typeof listing>

export function listing(core: Folders) {
  /** What each folder that has been read holds, under the path of the folder. */
  const held = shallowRef<ReadonlyMap<string, readonly Entry[]>>(new Map())
  /** The folders drawn open. The root is one of them for as long as the tab is. */
  const open = ref<ReadonlySet<string>>(new Set([ROOT]))
  /** The rows the person is standing on, by the paths they stand for. */
  const chosen = ref<readonly string[]>([])
  /** What the folders could not be read as, in words the window puts up for it. */
  const trouble = ref('')

  /** Whether the tab this tree stands in is still open. */
  let alive = true

  /** What one folder holds, and nothing for a folder that has not been read. */
  const entriesIn = (folder: string): readonly Entry[] => held.value.get(folder) ?? []

  /** Whether a folder is drawn open. */
  const opened = (folder: string): boolean => open.value.has(folder)

  /** One folder as lines, with every folder open under it drawn inside it. */
  const rowsIn = (folder: string): readonly Row[] =>
    entriesIn(folder).map((entry) => ({
      entry,
      rows: entry.folder && opened(entry.path) ? rowsIn(entry.path) : [],
    }))

  /** The tree, in the order the vault gave each folder. */
  const rows = computed<readonly Row[]>(() => rowsIn(ROOT))

  /** The folders drawn open, as the list the tree is handed. */
  const openRows = computed<readonly string[]>(() => [...open.value])

  /** What a path stands for, and nothing where the tree draws no row on it. */
  const entryAt = (path: string): Entry | null =>
    entriesIn(folderOf(path)).find((one) => one.path === path) ?? null

  /** One folder asked for and kept. A folder that could not be read is said. */
  const lists = async (folder: string) => {
    try {
      const entries = await core.list(folder)
      if (!alive) return
      held.value = new Map(held.value).set(folder, entries)
      trouble.value = ''
    } catch (error) {
      if (!alive) return
      trouble.value = String(error)
    }
  }

  /** A folder opened, and what it holds read. */
  const opens = async (folder: string) => {
    if (!alive) return
    open.value = new Set(open.value).add(folder)
    await lists(folder)
  }

  /**
   * A folder closed. What it holds is kept, so the folders open inside it are
   * drawn again where it opens.
   */
  const closes = (folder: string) => {
    if (folder === ROOT) return
    const rest = new Set(open.value)
    rest.delete(folder)
    open.value = rest
  }

  const toggles = (folder: string) => (opened(folder) ? void closes(folder) : opens(folder))

  /** The rows the person is standing on. */
  const chooses = (paths: readonly string[]) => {
    chosen.value = paths
  }

  /**
   * The tree walked down to a path: each folder above it is opened and read
   * before the one under it is, and the path is left the whole of what is
   * chosen.
   */
  const reveals = async (path: string) => {
    for (const folder of above(path)) {
      await opens(folder)
      if (!alive) return
    }
    chosen.value = [path]
  }

  /** Every open folder read again. */
  const again = async () => {
    await Promise.all([...open.value].map(lists))
  }

  /**
   * The folder a change to a path is read in, and none where the tree draws
   * nothing for it. A path under a folder the tree has no row for is that
   * folder arriving, and it appears in the open folder above it.
   */
  const drawnIn = (path: string): string | null => {
    let folder = folderOf(path)
    while (folder !== ROOT && !opened(folder) && !entryAt(folder)) folder = folderOf(folder)
    return opened(folder) ? folder : null
  }

  /**
   * The vault changed. Every folder a named path is read in is read again, and
   * a change naming nothing is the whole tree. A chosen row that moved is
   * chosen at where it went.
   */
  const changed = async (paths: readonly string[] = [], renamed: readonly Went[] = []) => {
    if (renamed.length > 0) {
      chosen.value = chosen.value.map((one) => wentTo(renamed, one) || one)
    }
    const named = [...paths, ...renamed.flatMap((one) => [one.from, one.to])]
    if (named.length === 0) return void (await again())
    const folders = new Set(named.map(drawnIn).filter((one) => one !== null))
    await Promise.all([...folders].map(lists))
  }

  /**
   * A name nothing in a folder is filed under, for something about to be made
   * there.
   */
  const freeIn = (folder: string, word: string): string =>
    freeName(
      entriesIn(folder).map((one) => one.name),
      word,
    )

  /** The tab has closed: nothing is asked for again and nothing is drawn. */
  const close = () => {
    alive = false
    held.value = new Map()
    open.value = new Set([ROOT])
  }

  return {
    rows,
    openRows,
    chosen,
    trouble,
    entriesIn,
    opened,
    entryAt,
    lists,
    opens,
    closes,
    toggles,
    chooses,
    reveals,
    again,
    changed,
    freeIn,
    close,
  }
}
