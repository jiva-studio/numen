import { computed, ref, shallowRef, watch } from 'vue'
import type { Position, Row } from '@numen/ui'
import { ROOT, getFileName, getParentFolders } from '../lib/entries'
import { getFileMenu } from '../lib/menus'
import { getDropFolder, getRenamedPath } from '../lib/rename'
import type { Vault } from './vault'

export type DropTarget = { readonly into: string | null } | { readonly before: string }

/**
 * What the tree asks of the window: `openFile` opens the file of a row, and `onSelectNote` is told of a note the
 * user selected, so that the tabs linked to the files tab follow it.
 */
export interface FilesDeps {
  readonly vault: Vault
  readonly openFile: (path: string) => void
  readonly onSelectNote: (path: string) => void
  readonly remove: (path: string, others: readonly string[]) => void
}

/** What the file tree shows and what a user did to it: open folders, the selection, the name being typed, the menu. */
export function createFiles(deps: FilesDeps) {
  const { vault } = deps
  const open = shallowRef<ReadonlySet<string>>(new Set([ROOT]))
  const selected = ref<readonly string[]>([])
  const renaming = ref<string | null>(null)
  const menu = ref<{ readonly path: string | null; readonly at: Position } | null>(null)

  const getRows = (folder: string): Row[] =>
    vault.list(folder).map((one) => ({
      id: one.path,
      name: one.name,
      hasChildren: one.isFolder,
      rows: one.isFolder && open.value.has(one.path) ? getRows(one.path) : [],
    }))

  const rows = computed(() => getRows(ROOT))
  const openRows = computed(() => [...open.value])

  const openFolder = (folder: string) => {
    open.value = new Set(open.value).add(folder)
  }

  const closeFolder = (folder: string) => {
    if (folder === ROOT) return
    const rest = new Set(open.value)
    rest.delete(folder)
    open.value = rest
  }

  function reveal(path: string) {
    getParentFolders(path).forEach(openFolder)
    selected.value = [path]
  }

  function activate(path: string) {
    const entry = vault.getEntryAt(path)
    if (!entry) return
    selected.value = [path]
    if (!entry.isFolder) deps.openFile(path)
  }

  function rename(path: string, name: string) {
    const to = getRenamedPath(path, name, vault.getEntryAt(path)?.isFolder ?? false)
    if (!to) return
    vault.move(path, to)
    selected.value = selected.value.map((one) => (one === path ? to : one))
  }

  function moveRows(paths: readonly string[], target: DropTarget) {
    const into = getDropFolder(target)
    const taken = new Set(vault.list(into).map((one) => one.name))
    for (const path of paths) {
      const name = getFileName(path)
      const to = into === ROOT ? name : `${into}/${name}`
      if (to === path || taken.has(name)) continue
      taken.add(name)
      vault.move(path, to)
    }
  }

  const getMenuItems = (path: string | null) => {
    const over = path !== null && selected.value.includes(path) ? selected.value : path === null ? [] : [path]
    const target = {
      path: path ?? ROOT,
      others: over.filter((one) => one !== path),
      entry: path === null ? undefined : vault.getEntryAt(path),
      isDemoNote: (one: string) => Boolean(vault.getNote(one)),
    }
    return getFileMenu(target)
  }

  /** The rows a command from the menu is run over: the row it was opened on, and the rows selected with it. */
  const getMenuTarget = (): { path: string; others: readonly string[] } => {
    const path = menu.value?.path ?? ROOT
    const over = selected.value.includes(path) ? selected.value : [path]
    return { path, others: over.filter((one) => one !== path) }
  }

  watch(selected, (paths) => {
    const first = paths[0]
    const entry = first === undefined ? undefined : vault.getEntryAt(first)
    if (first !== undefined && entry && !entry.isFolder && entry.type === 'note' && entry.isNote) {
      deps.onSelectNote(first)
    }
  })

  for (const folder of ['physics', 'computation', 'reading', 'lectures', 'decks']) openFolder(folder)
  selected.value = ['physics/Entropy.md']

  return {
    rows,
    openRows,
    selected,
    renaming,
    menu,
    getMenuItems,
    getMenuTarget,
    openFolder,
    closeFolder,
    reveal,
    activate,
    rename,
    moveRows,
    setRenaming: (row: string | null) => {
      renaming.value = row
    },
    openMenu: (path: string | null, at: Position) => {
      menu.value = { path, at }
    },
    closeMenu: () => {
      menu.value = null
    },
    select: (paths: readonly string[]) => {
      selected.value = paths
    },
    remove: (paths: readonly string[]) => {
      const [first, ...others] = paths
      if (first !== undefined) deps.remove(first, others)
    },
  }
}

export type Files = ReturnType<typeof createFiles>
