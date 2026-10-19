import type { Component, Ref } from 'vue'

// NoteRef is one note of the vault: where it is filed, and what it is called.
export interface NoteRef {
  readonly path: string
  readonly title: string
}

// NoteType is which of four things a note of the vault is.
export type NoteType = 'note' | 'deck' | 'stencil' | 'preset'

// NoteHeading is one heading inside a note.
export interface NoteHeading {
  readonly text: string
  readonly level: number
  readonly line: number
}

// NoteRelation is where a note joined to another sits around it.
export type NoteRelation = 'parent' | 'child' | 'jump' | 'sibling'

// NoteNeighbour is one note joined to the note in focus.
export interface NoteNeighbour {
  readonly path: string
  readonly title: string
  readonly relation: NoteRelation
  readonly type: NoteType
  readonly label: string
  readonly reverseLabel?: string
  readonly through: string
  readonly isMutual: boolean
}

// NoteNeighbourhood is a note in focus, and the notes joined to it.
export interface NoteNeighbourhood {
  readonly focus: NoteRef
  readonly focusType: NoteType
  readonly related: readonly NoteNeighbour[]
}

// NoteLink is one link between two notes, written in the note it runs from.
export interface NoteLink {
  readonly from: string
  readonly to: string
  readonly relation: NoteRelation
  readonly label: string
}

// VaultNotes is what a plugin reads of the notes of the vault, and writes to them.
export interface VaultNotes {
  getNeighbourhood(path: string): Promise<NoteNeighbourhood>
  getHeadings(paths: readonly string[]): Promise<ReadonlyMap<string, readonly NoteHeading[]>>
  findNotes(query: string, limit?: number): Promise<readonly NoteRef[]>
  // A note made untitled at the root of the vault: its path, and empty where none was made.
  createUntitledNote(): Promise<string>
  // A note made in a relation of another. The person is told why where none was made.
  createInRelation(from: string, relation: NoteRelation): Promise<NoteRef | null>
  createWithTitle(title: string, from: string, relation: NoteRelation): Promise<NoteRef | null>
  // Whether the link was written.
  join(from: string, to: string, relation: NoteRelation, label?: string): Promise<boolean>
  // Whether the links between two notes are now exactly the ones given.
  replaceLinks(first: string, second: string, links: readonly NoteLink[]): Promise<boolean>
  // The relations a note can be made in.
  readonly creatableRelations: readonly NoteRelation[]
}

// FilePlacement is where a file opened from a tab is drawn: in the pane the person is in, or beside it.
export type FilePlacement = 'here' | 'beside'

// OpenFileOptions is what a file is opened with.
export interface OpenFileOptions {
  readonly title?: string
  readonly placement?: FilePlacement
  readonly line?: number
}

// WorkspaceHost is the window a plugin's tabs are drawn in.
export interface WorkspaceHost {
  // Whether the vault can be read; false while the window says it cannot.
  readonly isReady: Readonly<Ref<boolean>>
  // The note the vault opens with, and empty where it names none.
  readonly openingPath: Readonly<Ref<string>>
  // The note the vault opens with, asked of the vault again.
  readOpeningPath(): Promise<string>
  // Whether a note drawn in a tab shows its headings, and how many of them.
  readonly headingDisplay: {
    readonly isNesting: Readonly<Ref<boolean>>
    readonly count: Readonly<Ref<number>>
  }
  // The vault paths the person is dragging, from any tab of the window.
  readonly draggedPaths: Readonly<Ref<readonly string[]>>
  setDraggedPaths(paths: readonly string[]): void
  openFile(path: string, options?: OpenFileOptions): void
  // The tabs linked to tab `tabId` move to the path. False where none is linked to it.
  syncPath(tabId: string, path: string, title?: string): boolean
  // A command of the window, run over one note.
  runCommand(id: string, target: NoteRef): void
  // A question for the agent, put in front of the person.
  ask(text: string): void
  showError(text: string): void
  // The window's commands over one row of the vault, in the order a menu draws them.
  getPathCommands(row: PathMenuRow): readonly MenuEntry[]
  // A command of the window, run over a file and the others selected with it.
  runPathCommand(id: string, target: PathCommandTarget): void
  // The icon the window draws a command with, and nothing for a command it draws none for.
  getCommandIcon(id: string): Component | null
  // The icon the window draws an entry of the vault with: a folder open or shut, a note by its type.
  getEntryIcon(entry: VaultEntry | null, isOpen: boolean): Component
}

// FileSource is what the vault holds at a path. A file it holds no source for is other.
export type FileSource = 'note' | 'book' | 'recording' | 'url' | 'other'

// VaultEntry is one file or folder, as a listing of the folder it sits in reports it.
export interface VaultEntry {
  readonly path: string
  readonly name: string
  readonly isFolder: boolean
  readonly kind: FileSource
  readonly type: NoteType
}

// MenuEntry is one item a menu offers, in the group it is drawn in.
export interface MenuEntry {
  readonly id: string
  readonly text: string
  readonly group: string
  readonly disabled?: boolean
}

// PathMenuRow is the row of the vault a menu is asked for on.
export interface PathMenuRow {
  readonly entry: VaultEntry
  // Whether the row is one of several selected, which the menu is over together.
  readonly isSeveral: boolean
}

// PathCommandTarget is the file a command runs over, and the others selected with it.
export interface PathCommandTarget {
  readonly paths: readonly string[]
  readonly name: string
  readonly source: FileSource
}

// VaultFiles is what a plugin lists of the folders of the vault, and makes and moves in them.
export interface VaultFiles {
  list(folder: string): Promise<readonly VaultEntry[]>
  move(from: string, to: string): Promise<void>
  createFolder(path: string): Promise<void>
  // Each answers with the path of what was made, and empty where nothing was.
  createNote(folder: string): Promise<string>
  createDeck(folder: string, name: string): Promise<string>
  createStencil(folder: string, name: string): Promise<string>
  createPreset(folder: string, name: string): Promise<string>
  importUrl(folder: string, url: string): Promise<string>
  importUrls(folder: string, urls: readonly string[]): Promise<readonly string[]>
}

// PathRename is a file that is no longer where it was, and where it now is.
export interface PathRename {
  readonly from: string
  readonly to: string
}

// getFileName says what a file is filed as: the last segment of its vault path.
export function getFileName(path: string): string {
  return path.split('/').pop() ?? path
}

// getRenamedPath says where a file went, and empty where none of the renames moved it.
export function getRenamedPath(renames: readonly PathRename[], path: string): string {
  return renames.find((one) => one.from === path)?.to ?? ''
}

// isNotePath says whether a vault path is a note: a markdown file, or a file name without an extension.
export function isNotePath(path: string): boolean {
  return path.endsWith('.md') || !getFileName(path).includes('.')
}
