/**
 * Tree state and operations for the files tree.
 */
import { computed, ref, shallowRef } from 'vue'
import { formatErrorMessage } from '@numen/wire'
import type { Entry } from '@/entities/file'
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

const isSameEntry = (one: Entry, other: Entry): boolean =>
  one.path === other.path &&
  one.name === other.name &&
  one.isFolder === other.isFolder &&
  one.kind === other.kind &&
  one.type === other.type

/** Whether two listings hold the same entries in the same order. */
const isSameEntries = (one: readonly Entry[] | undefined, other: readonly Entry[]): boolean =>
  one !== undefined &&
  one.length === other.length &&
  one.every((entry, at) => isSameEntry(entry, other[at]!))

/**
 * Generates a name not taken in the folder.
 */
export const generateUniqueName = (names: readonly string[], word: string): string => {
  const existingNames = new Set(names)
  if (!existingNames.has(word)) return word
  for (let count = 2; ; count++) {
    const name = `${word} ${count}`
    if (!existingNames.has(name)) return name
  }
}

function getRowsInFolder(
  entriesByFolder: ReadonlyMap<string, readonly Entry[]>,
  isFolderOpen: (folder: string) => boolean,
  folder: string,
): readonly ListingRow[] {
  const entries = entriesByFolder.get(folder) ?? []
  return entries.map((entry) => ({
    entry,
    rows:
      entry.isFolder && isFolderOpen(entry.path)
        ? getRowsInFolder(entriesByFolder, isFolderOpen, entry.path)
        : [],
  }))
}

interface FolderListingResult {
  folder: string
  entries?: readonly Entry[]
  error?: unknown
}

function mergeFolderListings(
  current: ReadonlyMap<string, readonly Entry[]>,
  listings: readonly FolderListingResult[],
): { next: ReadonlyMap<string, readonly Entry[]>; isChanged: boolean; error: string } {
  const next = new Map(current)
  let isChanged = false
  let error = ''
  for (const listing of listings) {
    if (listing.error !== undefined) {
      error = formatErrorMessage(listing.error)
    } else if (listing.entries && !isSameEntries(next.get(listing.folder), listing.entries)) {
      next.set(listing.folder, listing.entries)
      isChanged = true
    }
  }
  return { next, isChanged, error }
}

function findDrawnInFolder(
  path: string,
  isFolderOpen: (folder: string) => boolean,
  getEntryAt: (path: string) => Entry | null,
): string | null {
  let folder = getFolderPath(path)
  while (!isFolderOpen(folder) && !getEntryAt(folder)) {
    folder = getFolderPath(folder)
  }
  return isFolderOpen(folder) ? folder : null
}

export function useFileTree(core: Folders): FileTree {
  const entriesByFolder = shallowRef<ReadonlyMap<string, readonly Entry[]>>(new Map())
  const open = shallowRef<ReadonlySet<string>>(new Set([ROOT]))
  const selectedPaths = shallowRef<readonly string[]>([])
  const errorMessage = ref('')
  let isAlive = true

  const getEntriesInFolder = (folder: string): readonly Entry[] =>
    entriesByFolder.value.get(folder) ?? []

  const isFolderOpen = (folder: string): boolean => open.value.has(folder)

  const rows = computed<readonly ListingRow[]>(() =>
    getRowsInFolder(entriesByFolder.value, isFolderOpen, ROOT),
  )
  const openRows = computed<readonly string[]>(() => [...open.value])

  const getEntryAt = (path: string): Entry | null =>
    getEntriesInFolder(getFolderPath(path)).find((one) => one.path === path) ?? null

  const loadFolders = async (folders: readonly string[]) => {
    const listings = await Promise.all(
      folders.map(async (folder) => {
        try {
          return { folder, entries: await core.list(folder) }
        } catch (err) {
          return { folder, error: err }
        }
      }),
    )
    if (!isAlive) return
    const merged = mergeFolderListings(entriesByFolder.value, listings)
    if (merged.isChanged) entriesByFolder.value = merged.next
    errorMessage.value = merged.error
  }

  const loadFolder = (folder: string) => loadFolders([folder])

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
    void (isFolderOpen(folder) ? closeFolder(folder) : openFolder(folder))

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
    await loadFolders([...open.value])
  }

  const refreshChanged = async (
    paths: readonly string[] = [],
    renames: readonly PathRename[] = [],
  ) => {
    if (renames.length > 0) {
      selectedPaths.value = selectedPaths.value.map((one) => getRenamedPath(renames, one) || one)
    }
    const named = [...paths, ...renames.flatMap((one) => [one.from, one.to])]
    if (named.length === 0) return void (await refresh())
    const folders = new Set(
      named
        .map((p) => findDrawnInFolder(p, isFolderOpen, getEntryAt))
        .filter((one): one is string => one !== null),
    )
    await loadFolders([...folders])
  }

  const getUniqueNameInFolder = (folder: string, word: string): string =>
    generateUniqueName(
      getEntriesInFolder(folder).map((one) => one.name),
      word,
    )

  const close = () => {
    isAlive = false
    entriesByFolder.value = new Map()
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
