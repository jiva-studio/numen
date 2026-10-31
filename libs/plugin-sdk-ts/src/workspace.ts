import type { Component, Ref } from 'vue'
import type { Selector } from './texts'
import type { TextRange } from './types'
import type { Severity } from './notices'
import type { ErrorCode } from './errors/errorCode'
import type { Result } from './result'

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

// LinkMarker is what a link record draws at the end that points at its target. A link without one draws nothing there.
export type LinkMarker = 'arrow'

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
  // What the record the note in focus writes draws at this note.
  readonly marker: LinkMarker | null
  // What the record this note writes draws at the note in focus.
  readonly reverseMarker: LinkMarker | null
}

// NoteNeighbourhood is a note in focus, and the notes joined to it.
export interface NoteNeighbourhood {
  readonly focus: NoteRef
  readonly focusType: NoteType
  readonly related: readonly NoteNeighbour[]
}

// VaultNotes is what a plugin reads of the notes of the vault, and writes to them.
export interface VaultNotes {
  getNeighbourhood(path: string): Promise<NoteNeighbourhood>
  getHeadings(paths: readonly string[]): Promise<ReadonlyMap<string, readonly NoteHeading[]>>
  findNotes(query: string, limit?: number): Promise<readonly NoteRef[]>
  // A note made untitled at the root of the vault: its path, and empty where none was made.
  createUntitledNote(): Promise<string>
  // The prose of a note below its frontmatter, and why it could not be read: empty where it was.
  read(path: string): Promise<{ readonly body: string; readonly error?: string }>
  // Prose written into a note, keeping its frontmatter. Writes to one path are made in order.
  write(path: string, body: string): Promise<void>
}

// ConnectionRole is what the second note of a connection is to the first: its parent, its child, or a jump.
// The role of the other note is derived by the core.
export type ConnectionRole = 'parent' | 'child' | 'jump'

// LinkRole is the role of a link as written: a ConnectionRole, a mention in prose, or an attachment.
export type LinkRole = ConnectionRole | 'ref' | 'attachment'

// ConnectionSide is the record one note writes about the other in a connection.
export interface ConnectionSide {
  // What the person calls the relationship; empty when they call it nothing.
  readonly label: string
  // What the record draws at the other note; null draws nothing.
  readonly marker: LinkMarker | null
  // Why the link exists, in the person's words.
  readonly reason: string
}

// Connection is the relationship between two notes as a whole: up to two records, one written in each note.
export interface Connection {
  readonly first: string
  readonly second: string
  // What the second note is to the first; null when neither note writes a record of a role a connection carries.
  readonly role: ConnectionRole | null
  // The type of the records; empty for an ordinary link.
  readonly type: string
  // The record the first note writes about the second; null where it writes none.
  readonly firstSide: ConnectionSide | null
  // The record the second note writes about the first; null where it writes none.
  readonly secondSide: ConnectionSide | null
}

// RelationJoin is the record one note is made to write about another.
export interface RelationJoin {
  // What the other note is to the writing one.
  readonly role: ConnectionRole
  readonly type?: string
  readonly label?: string
  readonly marker?: LinkMarker
  readonly reason?: string
}

// RelatedNoteRequest is a note to make joined to an existing one.
export interface RelatedNoteRequest {
  // The existing note.
  readonly from: string
  // What the new note is to the existing one.
  readonly role: ConnectionRole
  // What the new note is called; empty takes the lowest free "Untitled note N".
  readonly title?: string
  // The type of the record the new note writes.
  readonly type?: string
  // What the record the new note writes draws at the existing note.
  readonly marker?: LinkMarker
}

// Backlink is a link written in another note that points at a note.
export interface Backlink {
  readonly from: NoteRef
  // The role as written, relative to the note it is written in.
  readonly role: LinkRole | null
  // The role of the writing note relative to the note asked about; null where the role has no counterpart.
  readonly roleFromHere: LinkRole | null
  readonly type: string
  readonly reason: string
  readonly label: string
  readonly marker: LinkMarker | null
}

// RelationError is why an operation on relationships did not land: a vault error, a connection that is not one
// ('invalid'), a connection that would put a note above itself ('cycle'), or no free untitled name ('nameExhausted').
export type RelationError = ErrorCode | 'invalid' | 'cycle' | 'nameExhausted'

// VaultRelations is what a plugin reads and writes of the relationships between notes. The core derives the
// role each note writes and enforces the rules of the hierarchy.
export interface VaultRelations {
  // The roles a note can be created or joined in.
  readonly roles: readonly ConnectionRole[]
  // The relationship between two notes, restricted to the records of one type.
  getConnection(
    first: string,
    second: string,
    type?: string,
  ): Promise<Result<Connection, RelationError>>
  // Makes the records between two notes exactly the ones given, in both notes or in neither.
  saveConnection(connection: Connection): Promise<Result<void, RelationError>>
  // Removes the record one note writes about another and leaves the other note alone.
  removeSide(from: string, to: string, type?: string): Promise<Result<void, RelationError>>
  // Removes the records two notes write about each other.
  removeConnection(
    first: string,
    second: string,
    type?: string,
  ): Promise<Result<void, RelationError>>
  // Makes one note write a record about another and keeps the record the other note writes.
  join(from: string, to: string, side: RelationJoin): Promise<Result<void, RelationError>>
  // Makes a note joined to an existing one; the value is its path.
  createRelated(request: RelatedNoteRequest): Promise<Result<string, RelationError>>
  // The links written in other notes that point at a note.
  listBacklinks(path: string): Promise<readonly Backlink[]>
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
