import { ref, watch } from 'vue'
import type { Position } from '@numen/ui'
import { ROOT, getParentFolders, isNoteRow } from '../lib/entries'
import { getFileMenu, getRowsOver } from '../lib/menus'
import { getDropFolder, getDropMoves, getRenamedPath } from '../lib/rename'
import { createFilesTree } from './filesTree'
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
  const branches = createFilesTree(vault, ['physics', 'computation', 'reading', 'lectures', 'decks'])
  const selected = ref<readonly string[]>([])
  const renaming = ref<string | null>(null)
  const menu = ref<{ readonly path: string | null; readonly at: Position } | null>(null)

  function reveal(path: string) {
    getParentFolders(path).forEach(branches.openFolder)
    selected.value = [path]
  }

  function activate(path: string) {
    const entry = vault.read.getEntryAt(path)
    if (!entry) return
    selected.value = [path]
    if (!entry.isFolder) deps.openFile(path)
  }

  function rename(path: string, name: string) {
    const to = getRenamedPath(path, name, vault.read.getEntryAt(path)?.isFolder ?? false)
    if (!to) return
    vault.edit.move(path, to)
    selected.value = selected.value.map((one) => (one === path ? to : one))
  }

  function moveRows(paths: readonly string[], target: DropTarget) {
    const into = getDropFolder(target)
    const taken = new Set(vault.read.list(into).map((one) => one.name))
    for (const { from, to } of getDropMoves(paths, into, taken)) vault.edit.move(from, to)
  }

  const getMenuItems = (path: string | null) => {
    const over = path === null ? [] : getRowsOver(path, selected.value)
    return getFileMenu({
      path: path ?? ROOT,
      others: over.filter((one) => one !== path),
      entry: path === null ? undefined : vault.read.getEntryAt(path),
      isDemoNote: (one: string) => Boolean(vault.read.getNote(one)),
    })
  }

  /** The rows a command from the menu is run over: the row it was opened on, and the rows selected with it. */
  const getMenuTarget = (): { path: string; others: readonly string[] } => {
    const path = menu.value?.path ?? ROOT
    return { path, others: getRowsOver(path, selected.value).filter((one) => one !== path) }
  }

  watch(selected, ([first]) => {
    if (first !== undefined && isNoteRow(vault.read.getEntryAt(first))) deps.onSelectNote(first)
  })

  selected.value = ['physics/Entropy.md']

  return {
    state: { rows: branches.rows, openRows: branches.openRows, selected, renaming, menu },
    tree: {
      openFolder: branches.openFolder,
      closeFolder: branches.closeFolder,
      reveal,
      activate,
      select: (paths: readonly string[]) => {
        selected.value = paths
      },
    },
    menus: {
      getMenuItems,
      getMenuTarget,
      openMenu: (path: string | null, at: Position) => {
        menu.value = { path, at }
      },
      closeMenu: () => {
        menu.value = null
      },
    },
    edits: {
      rename,
      moveRows,
      setRenaming: (row: string | null) => {
        renaming.value = row
      },
      remove: (paths: readonly string[]) => {
        const [first, ...others] = paths
        if (first !== undefined) deps.remove(first, others)
      },
    },
  }
}

export type Files = ReturnType<typeof createFiles>
