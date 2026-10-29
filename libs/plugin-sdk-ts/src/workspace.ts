import type { Component, Ref } from 'vue'
import type { Selector } from './texts'
import type { TextRange } from './types'
import type { Severity } from './notices'

// NoteRef is one note of the vault: where it is filed, and what it is called.
export interface NoteRef {
  readonly path: string
  readonly title: string
}

// NoteType is what a note of the vault is: a note, or a type a plugin declares in its manifest.
export type NoteType = string

// NOTE_TYPE is the NoteType of a note of the vault.
export const NOTE_TYPE: NoteType = 'note'

// NoteHeading is one heading inside a note.
export interface NoteHeading {
  readonly text: string
  readonly level: number
  readonly line: number
}

// NoteRelation is how a neighbouring note is linked to the focused note.
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
  createInRelation(from: string, relation: NoteRelation): Promise<NoteRef | null>
  createWithTitle(title: string, from: string, relation: NoteRelation): Promise<NoteRef | null>
  // Whether the link was written.
  join(from: string, to: string, relation: NoteRelation, label?: string): Promise<boolean>
  // Whether the links between two notes are now exactly the ones given.
  replaceLinks(first: string, second: string, links: readonly NoteLink[]): Promise<boolean>
  // The prose of a note below its frontmatter, and why it could not be read: empty where it was.
  read(path: string): Promise<{ readonly body: string; readonly error?: string }>
  // Prose written into a note, keeping its frontmatter. Writes to one path are made in order.
  write(path: string, body: string): Promise<void>
  // The relations a note can be made in.
  readonly creatableRelations: readonly NoteRelation[]
}

// FilePlacement is where a file opened from a tab is drawn: in the current pane, or in a split beside it.
export type FilePlacement = 'current' | 'split'

// OpenFileOptions is what a file is opened with.
export interface OpenFileOptions {
  readonly title?: string
  readonly placement?: FilePlacement
  readonly line?: number
  // The place the reader of the file shows, such as the selector of a search hit.
  readonly selector?: Selector
}

// MessageWriter writes one message under a name of its own, replacing what that name wrote last.
export type MessageWriter = (text: string, severity?: Severity) => void

// WorkspaceHost is the window a plugin's tabs are drawn in.
export interface WorkspaceHost {
  // Whether the vault can be read; false while the window reports it unreadable.
  readonly isReady: Readonly<Ref<boolean>>
  // The note the vault opens with, and empty where it names none.
  readonly openingPath: Readonly<Ref<string>>
  // The note the vault opens with, read from the vault again.
  readOpeningPath(): Promise<string>
  // The vault paths being dragged, from any tab of the window.
  readonly draggedPaths: Readonly<Ref<readonly string[]>>
  setDraggedPaths(paths: readonly string[]): void
  openFile(path: string, options?: OpenFileOptions): void
  // A file opened with the runs of its text marked.
  openFileAt(path: string, ranges: readonly TextRange[]): void
  // The note the most recently active tool tab depends on; empty strings when none.
  getNoteInFocus(): NoteRef
  // The tabs linked to tab `tabId` move to the path. False where none is linked to it.
  syncPath(tabId: string, path: string, title?: string): boolean
  // A command of the window, run over one note.
  runCommand(id: string, target: NoteRef): void
  // The commands the plugins declare over one note, in the words the palette offers them by.
  getNoteCommands(): readonly MenuEntry[]
  // Sends a question to the agent and shows it to the user.
  ask(text: string): void
  showError(text: string): void
  // A writer of messages under the name given.
  getMessageWriter(name: string): MessageWriter
  // The current day, with the day boundary set to the configured start hour.
  getToday(): string
  // The window's commands over one row of the vault, in the order a menu draws them.
  getPathCommands(row: PathMenuRow): readonly MenuEntry[]
  // A command of the window, run over a file and the others selected with it.
  runPathCommand(id: string, target: PathCommandTarget): void
  // The icon the window draws a command with, and nothing for a command it draws none for.
  getCommandIcon(id: string): Component | null
  // The icon the window draws an entry of the vault with: a folder open or shut, a note by its type.
  getEntryIcon(entry: VaultEntry | null, isOpen: boolean): Component
}

export interface VaultEntry {
  readonly path: string
  readonly name: string
  readonly isFolder: boolean
  readonly isNote: boolean
  readonly type: NoteType
}

// MenuEntry is one item a menu offers, in the group it is drawn in.
export interface MenuEntry {
  readonly id: string
  readonly text: string
  readonly detail?: string
  readonly group: string
  readonly isDisabled?: boolean
  readonly parentId?: string
  readonly children?: readonly MenuEntry[]
}

// PathMenuRow is the vault row a menu is requested for.
export interface PathMenuRow {
  readonly entry: VaultEntry
  // Whether the row is one of several selected, which the menu is over together.
  readonly isSeveral: boolean
}

// PathCommandTarget is the file or folder a command runs over, and the others selected with it.
export interface PathCommandTarget {
  readonly paths: readonly string[]
  readonly name: string
  readonly isNote: boolean
  readonly isFolder: boolean
}

// VaultFiles is what a plugin lists of the folders of the vault, and makes and moves in them.
export interface VaultFiles {
  list(folder: string): Promise<readonly VaultEntry[]>
  move(from: string, to: string): Promise<void>
  createFolder(path: string): Promise<void>
  // Each answers with the path of what was made, and empty where nothing was.
  createNote(folder: string): Promise<string>
}

// PathRename is a file move: the previous path and the new path.
export interface PathRename {
  readonly from: string
  readonly to: string
}

// getFileName returns the file name: the last segment of its vault path.
export function getFileName(path: string): string {
  return path.split('/').pop() ?? path
}

// getFileStem returns the file name without its extension, which is what a link writes.
export function getFileStem(path: string): string {
  const file = getFileName(path)
  const dot = file.lastIndexOf('.')
  return dot > 0 ? file.slice(0, dot) : file
}

// getRenamedPath returns the new path of a file, or empty when no rename moved it.
export function getRenamedPath(renames: readonly PathRename[], path: string): string {
  return renames.find((one) => one.from === path)?.to ?? ''
}

// isNotePath reports whether a vault path is a note: a markdown file, or a file name without an extension.
export function isNotePath(path: string): boolean {
  return path.endsWith('.md') || !getFileName(path).includes('.')
}
