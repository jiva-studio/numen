/**
 * Tree state and operations for the files tree.
 */
import { computed, ref, shallowRef, type Ref, type ShallowRef } from 'vue'
import {
  formatErrorMessage,
  getRenamedPath,
  type PathRename,
  type JsonValue,
} from '@numen/plugin-sdk'
import type { Entry, FileTree, Folders, ListingRow } from '../types'

export const ROOT = ''

export const getFolderPath = (path: string): string => {
  const slashIndex = path.lastIndexOf('/')
  // Stryker disable next-line EqualityOperator: a path starting with slash produces ROOT empty string either way.
  return slashIndex < 0 ? ROOT : path.slice(0, slashIndex)
}

/**
 * The folder a dropped row lands in: the one it went into, or the one holding
 * the row it came before. Into no row at all is the root.
 */
export const resolveDropFolder = (target: { into: string | null } | { before: string }): string =>
  'into' in target ? (target.into ?? ROOT) : getFolderPath(target.before)

/** The folders above a path, from the root down. */
export const getParentFolders = (path: string): readonly string[] => {
  const parts = path.split('/').slice(0, -1)
  return parts.map((_, deep) => parts.slice(0, deep + 1).join('/'))
}

/** How the entries of a folder are ordered in the rows of the tree. */
export type SortBy = 'name' | 'modified'

export const readSortBy = (value: JsonValue | undefined): SortBy =>
  value === 'modified' ? 'modified' : 'name'

const getModifiedAt = (entry: Entry): number => entry.modifiedAt ?? Number.NEGATIVE_INFINITY

const compareModifiedDescending = (one: Entry, other: Entry): number => {
  const first = getModifiedAt(one)
  const second = getModifiedAt(other)
  if (first === second) return 0
  return second > first ? 1 : -1
}

/** Folders first, then files; with `modified` each group has the newest first and the undated last. */
export const sortEntries = (entries: readonly Entry[], sortBy: SortBy): readonly Entry[] => {
  if (sortBy === 'name') return entries
  return entries
    .map((entry, index) => ({ entry, index }))
    .sort(
      (one, other) =>
        Number(other.entry.isFolder) - Number(one.entry.isFolder) ||
        compareModifiedDescending(one.entry, other.entry) ||
        one.index - other.index,
    )
    .map(({ entry }) => entry)
}

const isSameEntry = (one: Entry, other: Entry): boolean =>
  one.path === other.path &&
  one.name === other.name &&
  one.isFolder === other.isFolder &&
  one.isNote === other.isNote &&
  one.type === other.type &&
  one.modifiedAt === other.modifiedAt

const isSameEntries = (one: readonly Entry[] | undefined, other: readonly Entry[]): boolean =>
  one !== undefined &&
  one.length === other.length &&
  one.every((entry, index) => isSameEntry(entry, other[index]!))

function getRowsInFolder(
  entriesByFolder: ReadonlyMap<string, readonly Entry[]>,
  isFolderOpen: (folder: string) => boolean,
  folder: string,
  sortBy: SortBy,
): readonly ListingRow[] {
  const entries = sortEntries(entriesByFolder.get(folder) ?? [], sortBy)
  return entries.map((entry) => ({
    entry,
    rows:
      entry.isFolder && isFolderOpen(entry.path)
        ? getRowsInFolder(entriesByFolder, isFolderOpen, entry.path, sortBy)
        : [],
  }))
}

interface FolderListingResult {
  folder: string
  entries?: readonly Entry[]
  error?: string
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
      error = listing.error
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

interface TreeRefreshDeps {
  core: Folders
  entriesByFolder: ShallowRef<ReadonlyMap<string, readonly Entry[]>>
  open: ShallowRef<ReadonlySet<string>>
  errorMessage: Ref<string>
  isLoaded: Ref<boolean>
  selectedPaths: ShallowRef<readonly string[]>
  lookup: {
    isFolderOpen: (folder: string) => boolean
    getEntryAt: (path: string) => Entry | null
  }
  isAlive: () => boolean
}

function createTreeRefresh(deps: TreeRefreshDeps) {
  const {
    core,
    entriesByFolder,
    open,
    errorMessage,
    isLoaded,
    selectedPaths,
    lookup: { isFolderOpen, getEntryAt },
    isAlive,
  } = deps

  const loadFolders = async (folders: readonly string[]) => {
    const listings = await Promise.all(
      folders.map(async (folder) => {
        try {
          return { folder, entries: await core.list(folder) }
        } catch (err) {
          return { folder, error: formatErrorMessage(err) }
        }
      }),
    )
    if (!isAlive()) return
    const merged = mergeFolderListings(entriesByFolder.value, listings)
    if (merged.isChanged) entriesByFolder.value = merged.next
    errorMessage.value = merged.error
    if (folders.includes(ROOT)) isLoaded.value = true
  }

  const loadFolder = (folder: string) => loadFolders([folder])
  const refresh = async () => loadFolders([...open.value])

  const refreshChanged = async (
    paths: readonly string[] = [],
    renames: readonly PathRename[] = [],
  ) => {
    if (renames.length > 0) {
      selectedPaths.value = selectedPaths.value.map(
        (one: string) => getRenamedPath(renames, one) || one,
      )
    }
    const named = [...paths, ...renames.flatMap((one) => [one.from, one.to])]
    if (named.length === 0) {
      await refresh()
      return
    }
    const folders = new Set(
      named
        .map((p) => findDrawnInFolder(p, isFolderOpen, getEntryAt))
        .filter((one): one is string => one !== null),
    )
    await loadFolders([...folders])
  }

  return { loadFolder, refresh, refreshChanged }
}

function createFolderOperations(
  open: ShallowRef<ReadonlySet<string>>,
  loadFolder: (folder: string) => Promise<void>,
  isAlive: () => boolean,
) {
  const isFolderOpen = (folder: string): boolean => open.value.has(folder)

  const openFolder = async (folder: string) => {
    if (!isAlive()) return
    open.value = new Set(open.value).add(folder)
    await loadFolder(folder)
  }

  const closeFolder = (folder: string) => {
    if (folder === ROOT) return
    const rest = new Set(open.value)
    rest.delete(folder)
    open.value = rest
  }

  const toggleFolder = async (folder: string): Promise<void> => {
    if (isFolderOpen(folder)) closeFolder(folder)
    else await openFolder(folder)
  }

  return { openFolder, closeFolder, toggleFolder }
}

function initTreeState() {
  const entriesByFolder = shallowRef<ReadonlyMap<string, readonly Entry[]>>(new Map())
  const open = shallowRef<ReadonlySet<string>>(new Set([ROOT]))
  const selectedPaths = shallowRef<readonly string[]>([])
  const errorMessage = ref('')
  const isLoaded = ref(false)
  return { entriesByFolder, open, selectedPaths, errorMessage, isLoaded }
}

function createTreeQueries(
  entriesByFolder: ShallowRef<ReadonlyMap<string, readonly Entry[]>>,
  open: ShallowRef<ReadonlySet<string>>,
  sortBy: Readonly<Ref<SortBy>> | undefined,
) {
  const getEntriesInFolder = (folder: string): readonly Entry[] =>
    entriesByFolder.value.get(folder) ?? []

  const isFolderOpen = (folder: string): boolean => open.value.has(folder)

  const rows = computed<readonly ListingRow[]>(() =>
    getRowsInFolder(entriesByFolder.value, isFolderOpen, ROOT, sortBy?.value ?? 'name'),
  )
  const openRows = computed<readonly string[]>(() => [...open.value])

  const getEntryAt = (path: string): Entry | null =>
    getEntriesInFolder(getFolderPath(path)).find((one) => one.path === path) ?? null

  return { getEntriesInFolder, isFolderOpen, rows, openRows, getEntryAt }
}

export function createFileTree(
  core: Folders,
  options?: { readonly sortBy?: Readonly<Ref<SortBy>> },
): FileTree {
  const state = initTreeState()
  let isAlive = true

  const queries = createTreeQueries(state.entriesByFolder, state.open, options?.sortBy)

  const refreshControl = createTreeRefresh({
    core,
    ...state,
    lookup: queries,
    isAlive: () => isAlive,
  })

  const folderOps = createFolderOperations(state.open, refreshControl.loadFolder, () => isAlive)

  const revealPath = async (path: string) => {
    for (const folder of getParentFolders(path)) {
      await folderOps.openFolder(folder)
      if (!isAlive) return
    }
    state.selectedPaths.value = [path]
  }

  const close = () => {
    isAlive = false
    state.entriesByFolder.value = new Map()
    state.open.value = new Set([ROOT])
  }

  return {
    state: {
      selectedPaths: state.selectedPaths,
      errorMessage: state.errorMessage,
      isLoaded: state.isLoaded,
    },
    queries,
    folders: folderOps,
    sync: refreshControl,
    selectPaths: (paths: readonly string[]) => {
      state.selectedPaths.value = paths
    },
    revealPath,
    close,
  }
}
