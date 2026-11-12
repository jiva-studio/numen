/**
 * Type declarations for the files tab domain.
 */
import type { Component, Ref } from 'vue'
import type { MenuItem } from '@numen/ui'
import type {
  CommandItem,
  DroppedData,
  PathRename,
  SlotItem,
  CommandTarget,
  VaultEntry,
  Caught,
} from '@numen/plugin-sdk'

/** One file or folder of the vault, as represented by a row of the tree. */
export type Entry = VaultEntry

/** What the vault holds at a path. */

/** What a files tab requests from the file storage/vault. */
export interface Folders {
  /** What one folder holds, in the order to draw it. The root is the empty path. */
  list(folder: string): Promise<readonly Entry[]>
}

/**
 * One row of the tree: its entry and the child rows below it.
 */
export interface ListingRow {
  readonly entry: Entry
  readonly rows: readonly ListingRow[]
}

/** Where the menu opens, and the row it was opened on. */
export interface MenuRequest {
  readonly path: string | null
  readonly at: { x: number; y: number }
}

/** Where dropped rows landed, as the tree reports it. */
export type DropPosition = { readonly into: string | null } | { readonly before: string }

/** What a files tab requests from the window: the files it opens, selects and shows. */
interface FilesTabWindowDeps {
  /** Opens the file a row represents. */
  openEntry(entry: Entry): void
  onSelectPath?(path: string, state: FilesTabState): void
  setDraggedPaths(paths: readonly string[]): void
  showError(text: string): void
  getEntryIcon(entry: Entry | null, isOpen: boolean): Component
}

/** What a files tab requests for its menu. */
interface FilesTabMenuDeps {
  /** The entries of the menu of a row, over the row it was opened on. */
  getMenuEntries(target: CommandTarget): readonly SlotItem[]
  /** The items a command of the menu offers over the row. */
  listItems(command: string, target: CommandTarget): Promise<readonly CommandItem[]>
  /**
   * Runs a command of the window or of a plugin over the row; with `itemId`, the one picked.
   * Answers the path of the note or folder it made, empty where none.
   */
  runCommand(
    command: string,
    target: CommandTarget,
    itemId?: string,
    values?: Readonly<Record<string, string>>,
  ): Promise<string>
}

/** What a files tab requests to move rows and to take dropped data. */
interface FilesTabTransferDeps {
  movePath(from: string, to: string): Promise<void>
  /** Whether a plugin takes data dragged in from outside the window, of one of the types. */
  canDropData(types: readonly string[]): boolean
  /** Hands data dropped into the folder to the plugin that takes it; answers what it made. */
  dropData(folder: string, data: DroppedData): Promise<readonly string[]>
}

/** What a files tab requests from the window that hosts it. */
export interface FilesTabDeps {
  readonly window: FilesTabWindowDeps
  readonly menu: FilesTabMenuDeps
  readonly transfer: FilesTabTransferDeps
}

/** What a tree shows, as values that change. */
interface FileTreeState {
  readonly selectedPaths: Ref<readonly string[]>
  readonly errorMessage: Ref<string>
  readonly isLoaded: Ref<boolean>
}

/** What a tree answers about the folders it has loaded. */
interface FileTreeQueries {
  readonly rows: Ref<readonly ListingRow[]>
  readonly openRows: Ref<readonly string[]>
  getEntriesInFolder(folder: string): readonly Entry[]
  isFolderOpen(folder: string): boolean
  getEntryAt(path: string): Entry | null
}

/** The operations that open and close folders. */
interface FileTreeFolders {
  openFolder(folder: string): Promise<void>
  closeFolder(folder: string): void
  toggleFolder(folder: string): Promise<void>
}

/** The operations that bring the tree in line with the vault. */
interface FileTreeSync {
  loadFolder(folder: string): Promise<void>
  refresh(): Promise<void>
  refreshChanged(paths?: readonly string[], renamed?: readonly PathRename[]): Promise<void>
}

export interface FileTree {
  readonly state: FileTreeState
  readonly queries: FileTreeQueries
  readonly folders: FileTreeFolders
  readonly sync: FileTreeSync
  selectPaths(paths: readonly string[]): void
  revealPath(path: string): Promise<void>
  /** Ends the tree: nothing it was loading is kept, and the folders open are the root. */
  close(): void
}

/** The operations over the rows of the tree. */
export interface FilesTabEntries {
  activate(path: string): void
  open(path: string): void
  close(path: string): void
  select(paths: readonly string[]): void
  rename(path: string, name: string): Promise<void>
  remove(paths: readonly string[]): void
  getNameOf(path: string): string
  getEntryIcon(entry: Entry | null, isOpen: boolean): Component
}

/** The row whose name is in a field. */
interface FilesTabRenaming {
  readonly path: Ref<string | null>
  setPath(path: string | null): void
}

/** Moving rows within the tree, and dropping rows or outside data on it. */
interface FilesTabTransfer {
  getFolderFor(path: string | null): string
  move(paths: readonly string[], target: DropPosition): Promise<void>
  drag(paths: readonly string[]): void
  drop(): void
  /** Whether a plugin takes data dragged in from outside the window, of one of the types. */
  canDropData(types: readonly string[]): boolean
  /** Hands data dropped on the tree to the plugin that takes it, into the folder of the selection. */
  dropData(data: DroppedData): Promise<readonly string[]>
}

/** The menu of a row, or of no row; request is where the open menu is, null while none is open. */
export interface FilesTabMenu {
  readonly request: Readonly<Ref<MenuRequest | null>>
  getOverPaths(path: string): readonly string[]
  open(request: MenuRequest): void
  dismiss(): void
  choose(id: string): void
  /** The menu items: for the row the menu was opened on, or for no row. */
  getItems(path: string | null): readonly MenuItem[]
  /** The icon of the command or submenu of the open menu drawn as the row `id`. */
  getIcon(id: string): Component | null
}

/** What one files tab holds. */
export interface FilesTabState {
  readonly list: FileTree
  readonly entries: FilesTabEntries
  readonly renaming: FilesTabRenaming
  readonly transfer: FilesTabTransfer
  readonly menu: FilesTabMenu
  /** Shows the failure of an operation that runs in the background. */
  reportFailure(error: Caught): void
}
