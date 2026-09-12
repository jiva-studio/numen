/**
 * Tree state and operations for the files tree.
 */
import { computed, ref, shallowRef } from 'vue'
import { formatErrorMessage } from '@numen/wire'
import type { Entry } from '@/shared/file'
import { getRenamedPath, type PathRename } from '@/shared/paths'
import type { FileTree, Folders, ListingRow } from '../types'

export const ROOT = ''

/** The folder a path sits in, and the root for a path at the top of the vault. */
export const getFolderPath = (path: string): string => {
  const cut = path.lastIndexOf('/')
  return cut < 0 ? ROOT : path.slice(0, cut)
}

/**
 * The folder a dropped row lands in: the one it went into, or the one holding
 * the row it came before. Into no row at all is the root.
 */
export const resolveDropFolder = (at: { into: string | null } | { before: string }): string =>
  'into' in at ? (at.into ?? ROOT) : getFolderPath(at.before)

/** The folders above a path, from the root down. */
export const getParentFolders = (path: string): readonly string[] => {
  const parts = path.split('/').slice(0, -1)
  return parts.map((_, deep) => parts.slice(0, deep + 1).join('/'))
}

/**
 * Generates a name not taken in the folder.
 */
export const generateUniqueName = (taken: readonly string[], word: string): string => {
  const held = new Set(taken)
  if (!held.has(word)) return word
  for (let count = 2; ; count++) {
    const name = `${word} ${count}`
    if (!held.has(name)) return name
  }
}

export function useFileTree(core: Folders): FileTree {
  const held = shallowRef<ReadonlyMap<string, readonly Entry[]>>(new Map())
  const open = shallowRef<ReadonlySet<string>>(new Set([ROOT]))
  const selectedPaths = shallowRef<readonly string[]>([])
  const errorMessage = ref('')

  let isAlive = true

  const getEntriesInFolder = (folder: string): readonly Entry[] => held.value.get(folder) ?? []

  const isFolderOpen = (folder: string): boolean => open.value.has(folder)

  const getRowsInFolder = (folder: string): readonly ListingRow[] =>
    getEntriesInFolder(folder).map((entry) => ({
      entry,
      rows: entry.folder && isFolderOpen(entry.path) ? getRowsInFolder(entry.path) : [],
    }))

  const rows = computed<readonly ListingRow[]>(() => getRowsInFolder(ROOT))
  const openRows = computed<readonly string[]>(() => [...open.value])

  const getEntryAt = (path: string): Entry | null =>
    getEntriesInFolder(getFolderPath(path)).find((one) => one.path === path) ?? null

  const loadFolder = async (folder: string) => {
    try {
      const entries = await core.list(folder)
      if (!isAlive) return
      held.value = new Map(held.value).set(folder, entries)
      errorMessage.value = ''
    } catch (err) {
      if (!isAlive) return
      errorMessage.value = formatErrorMessage(err)
    }
  }

  const openFolder = async (folder: string) => {
    if (!isAlive) return
    open.value = new Set(open.value).add(folder)
    await loadFolder(folder)
  }

  const closeFolder = (folder: string) => {
    if (folder === ROOT) return
    const rest = new Set(open.value)
    rest.delete(folder)
    open.value = rest
  }

  const toggleFolder = (folder: string) =>
    isFolderOpen(folder) ? void closeFolder(folder) : openFolder(folder)

  const selectPaths = (paths: readonly string[]) => {
    selectedPaths.value = paths
  }

  const revealPath = async (path: string) => {
    for (const folder of getParentFolders(path)) {
      await openFolder(folder)
      if (!isAlive) return
    }
    selectedPaths.value = [path]
  }

  const refresh = async () => {
    await Promise.all([...open.value].map(loadFolder))
  }

  const getDrawnInFolder = (path: string): string | null => {
    let folder = getFolderPath(path)
    while (folder !== ROOT && !isFolderOpen(folder) && !getEntryAt(folder)) {
      folder = getFolderPath(folder)
    }
    return isFolderOpen(folder) ? folder : null
  }

  const refreshChanged = async (
    paths: readonly string[] = [],
    renamed: readonly PathRename[] = [],
  ) => {
    if (renamed.length > 0) {
      selectedPaths.value = selectedPaths.value.map((one) => getRenamedPath(renamed, one) || one)
    }
    const named = [...paths, ...renamed.flatMap((one) => [one.from, one.to])]
    if (named.length === 0) return void (await refresh())
    const folders = new Set(named.map(getDrawnInFolder).filter((one): one is string => one !== null))
    await Promise.all([...folders].map(loadFolder))
  }

  const getUniqueNameInFolder = (folder: string, word: string): string =>
    generateUniqueName(
      getEntriesInFolder(folder).map((one) => one.name),
      word,
    )

  const close = () => {
    isAlive = false
    held.value = new Map()
    open.value = new Set([ROOT])
  }

  return {
    rows,
    openRows,
    selectedPaths,
    errorMessage,
    getEntriesInFolder,
    isFolderOpen,
    getEntryAt,
    loadFolder,
    openFolder,
    closeFolder,
    toggleFolder,
    selectPaths,
    revealPath,
    refresh,
    refreshChanged,
    getUniqueNameInFolder,
    close,
  }
}
