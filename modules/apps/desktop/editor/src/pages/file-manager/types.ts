/**
 * Type declarations for the files tab domain.
 */
import type { Ref } from 'vue'
import type { Entry, Source } from '@/shared/file'
import type { PathRename } from '@/shared/paths'
import type { SearchDestination } from '@/features/command-palette'
import type { RunGuard } from './lib/menu'

/** Everything a files tab asks of the file storage/vault. */
export interface Folders {
  /** What one folder holds, in the order to draw it. The root is the empty path. */
  list(folder: string): Promise<readonly Entry[]>
}

/**
 * One line of the tree: what it stands for, and the lines drawn under it.
 */
export interface ListingRow {
  readonly entry: Entry
  readonly rows: readonly ListingRow[]
}

/** Where the menu stands, and what it was asked for on. */
export interface MenuRequest {
  /** The row it was asked for on, or null when asked off every row. */
  readonly path: string | null
  readonly at: { x: number; y: number }
}

/** Where rows let go of landed, as the tree reports it. */
export type DropPosition = { readonly into: string | null } | { readonly before: string }

/** What a files tab asks of the window it is drawn in. */
export interface FilesTabDeps {
  openDestination(destination: SearchDestination | null): void
  runCommand(id: string, paths: readonly string[], name: string, source: Source): void
  movePath(from: string, to: string): Promise<void>
  setDraggedPaths(paths: readonly string[]): void
  createFolder(path: string): Promise<void>
  createNote(folder: string): Promise<string>
  createDeck(folder: string, name: string): Promise<string>
  createStencil(folder: string, name: string): Promise<string>
  createPreset(folder: string, name: string): Promise<string>
  importAddress(folder: string, address: string): Promise<string>
  showError(text: string): void
  canRun?: RunGuard
}

/** One of the custom files the vault makes, made in a folder. */
export type FileMaker = (folder: string, name: string) => Promise<string>

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
  toggleFolder(folder: string): void
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
  select(paths: readonly string[]): void
  rename(path: string, name: string): Promise<void>
  move(paths: readonly string[], at: DropPosition): Promise<void>
  drag(paths: readonly string[]): void
  drop(): void
  remove(paths: readonly string[]): void
  createFolder(path: string | null): Promise<void>
  createNote(path: string | null): Promise<void>
  createOne(path: string | null, createEntry: FileMaker, name: string): Promise<void>
  importAddress(address: string): Promise<string>
  openMenu(asked: MenuRequest): void
  dismissMenu(): void
  chooseMenuItem(id: string): void
  getNameOf(path: string): string
  canRun: RunGuard
}
