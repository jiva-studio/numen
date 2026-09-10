/**
 * Type declarations for the files tab domain.
 */
import type { Ref } from 'vue'
import type { Entry, Move, Source } from '../shared/core'
import type { SearchDestination } from '../shared/command/search'
import type { FileTree } from './listing'
import type { RunGuard } from './menu'

/** Where the menu stands, and what it was asked for on. */
export interface MenuRequest {
  /** The row it was asked for on, or null when asked off every row. */
  readonly path: string | null
  readonly at: { x: number; y: number }
}

/** Where rows let go of landed, as the tree reports it. */
export type DropPosition = { readonly into: string } | { readonly before: string }

/** What a files tab asks of the window it is drawn in. */
export interface FilesTabDeps {
  lands(going: SearchDestination | null): void
  runs(id: string, paths: readonly string[], name: string, source: Source): void
  moves(from: string, to: string): Promise<void>
  drags(paths: readonly string[]): void
  makes(path: string): Promise<void>
  writes(folder: string): Promise<string>
  decks(folder: string, name: string): Promise<string>
  stencils(folder: string, name: string): Promise<string>
  presets(folder: string, name: string): Promise<string>
  imports(folder: string, address: string): Promise<string>
  says(text: string): void
  canRun?: RunGuard
}

/** One of the three files the vault names itself, made in a folder. */
export type FileMaker = (folder: string, name: string) => Promise<string>

/** What one files tab holds. */
export interface FilesTabState {
  readonly list: FileTree
  readonly menu: Ref<MenuRequest | null>
  readonly renaming: Ref<string | null>
  setRenamingPath(path: string | null): void
  over(path: string): readonly string[]
  folderFor(path: string | null): string
  activate(path: string): void
  open(path: string): void
  close(path: string): void
  select(paths: readonly string[]): void
  rename(path: string, name: string): Promise<void>
  move(paths: readonly string[], at: DropPosition): Promise<void>
  drag(paths: readonly string[]): void
  drop(): void
  remove(paths: readonly string[]): void
  makes(path: string | null): Promise<void>
  createFolder(path: string | null): Promise<void>
  writes(path: string | null): Promise<void>
  createNote(path: string | null): Promise<void>
  makesOne(path: string | null, makes: FileMaker, name: string): Promise<void>
  imports(address: string): Promise<string>
  importAddress(address: string): Promise<string>
  openMenu(asked: MenuRequest): void
  asks(asked: MenuRequest): void
  dismiss(): void
  chooseMenuItem(id: string): void
  nameOf(path: string): string
  getName(path: string): string
  canRun: RunGuard
}
