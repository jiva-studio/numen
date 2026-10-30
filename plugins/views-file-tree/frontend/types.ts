/**
 * Type declarations for the files tab domain.
 */
import type { Component, Ref } from 'vue'
import type { DroppedData, MenuEntry, PathMenuRow, PathRename, VaultEntry } from '@numen/plugin-sdk'

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
  /** The row the menu was opened on, or null when opened outside every row. */
  readonly path: string | null
  readonly at: { x: number; y: number }
}

/** Where dropped rows landed, as the tree reports it. */
export type DropPosition = { readonly into: string | null } | { readonly before: string }

export interface FileCommandTarget {
  readonly name: string
  readonly isNote: boolean
  readonly isFolder: boolean
}

/** What a files tab requests from the window that hosts it. */
export interface FilesTabDeps {
  /** Opens the file a row represents. */
  openEntry(entry: Entry): void
  onSelectPath?(path: string, state: FilesTabState): void
  runCommand(id: string, paths: readonly string[], target: FileCommandTarget): void
  movePath(from: string, to: string): Promise<void>
  setDraggedPaths(paths: readonly string[]): void
  createFolder(path: string): Promise<void>
  createNote(folder: string): Promise<string>
  /** Whether a plugin takes data dragged in from outside the window, of one of the types. */
  canDropData(types: readonly string[]): boolean
  /** Hands data dropped into the folder to the plugin that takes it; answers what it made. */
  dropData(folder: string, data: DroppedData): Promise<readonly string[]>
  showError(text: string): void
  /** The window's commands over a row, in the order the menu draws them. */
  getPathCommands(row: PathMenuRow): readonly MenuEntry[]
  getCommandIcon(id: string): Component | null
  getEntryIcon(entry: Entry | null, isOpen: boolean): Component
}

export interface FileTree {
  readonly rows: Ref<readonly ListingRow[]>
  readonly openRows: Ref<readonly string[]>
  readonly selectedPaths: Ref<readonly string[]>
  readonly errorMessage: Ref<string>
  getEntriesInFolder(folder: string): readonly Entry[]
  isFolderOpen(folder: string): boolean
  getEntryAt(path: string): Entry | null
  loadFolder(folder: string): Promise<void>
  openFolder(folder: string): Promise<void>
  closeFolder(folder: string): void
  toggleFolder(folder: string): Promise<void>
  selectPaths(paths: readonly string[]): void
  revealPath(path: string): Promise<void>
  refresh(): Promise<void>
  refreshChanged(paths?: readonly string[], renamed?: readonly PathRename[]): Promise<void>
  getUniqueNameInFolder(folder: string, word: string): string
  close(): void
}

/** What one files tab holds. */
export interface FilesTabState {
  readonly list: FileTree
  readonly menu: Ref<MenuRequest | null>
  readonly renamingPath: Ref<string | null>
  setRenamingPath(path: string | null): void
  getOverPaths(path: string): readonly string[]
  getFolderFor(path: string | null): string
  activate(path: string): void
  open(path: string): void
  close(path: string): void
  /** Shows the failure of an operation that runs in the background. */
  reportFailure(error: unknown): void
  select(paths: readonly string[]): void
  rename(path: string, name: string): Promise<void>
  move(paths: readonly string[], target: DropPosition): Promise<void>
  drag(paths: readonly string[]): void
  drop(): void
  remove(paths: readonly string[]): void
  createFolder(path: string | null): Promise<void>
  createNote(path: string | null): Promise<void>
  /** Whether a plugin takes data dragged in from outside the window, of one of the types. */
  canDropData(types: readonly string[]): boolean
  /** Hands data dropped on the tree to the plugin that takes it, into the folder of the selection. */
  dropData(data: DroppedData): Promise<readonly string[]>
  openMenu(request: MenuRequest): void
  dismissMenu(): void
  chooseMenuItem(id: string): void
  getNameOf(path: string): string
  /** The menu items: for the row the menu was opened on, or for no row. */
  getMenuItems(path: string | null): readonly MenuEntry[]
  getCommandIcon(id: string): Component | null
  getEntryIcon(entry: Entry | null, isOpen: boolean): Component
}
