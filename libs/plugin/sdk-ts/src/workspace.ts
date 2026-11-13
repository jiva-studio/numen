import type { Component, Ref } from 'vue'
import type { JsonValue, TextRange } from './types'
import type { Selector } from './texts'
import type { Severity } from './notices'
import type { ErrorCode } from './errors'
import type { Result } from './result'

// NoteRef is one note of the vault: where it is filed, and what it is called.
export interface NoteRef {
  readonly path: string
  readonly title: string
}

// NoteType is what a note of the vault is: a note, or a type a plugin declares in its manifest.
export type NoteType = string

export const NOTE_TYPE: NoteType = 'note'

export interface NoteHeading {
  readonly text: string
  readonly level: number
  readonly line: number
}

// NoteRelation is how a related note is linked to the focused note.
export type NoteRelation = 'parent' | 'child' | 'jump' | 'sibling' | 'reference'

// LinkDirection tells which note wrote a text link: the note in focus ('outgoing'), the related note ('incoming'), or each of them ('both'). For a note found by depth 2 the note in focus is the one in its relatedTo.
export type LinkDirection = 'outgoing' | 'incoming' | 'both'

// LinkMarker is what a link record draws at the end that points at its target. A link without one draws nothing there.
export type LinkMarker = 'arrow'

// RelatedNote is one note joined to the note in focus. Its direction is present for a 'reference' only. Its relatedTo is the path of the related note it belongs to, set for a note found by depth 2 (its relation and direction are then relative to that note, not to the focus) and absent for a note related to the focus. A note found by depth 2 has no connection record, it may also be related to the focus, and its path can repeat in NoteGraph.related, once under each related note it belongs to. The records that join notes are in NoteGraph.connections.
export interface RelatedNote {
  readonly path: string
  readonly title: string
  readonly relation: NoteRelation
  readonly type: NoteType
  readonly tabKind?: string
  readonly direction?: LinkDirection
  readonly relatedTo?: string
}

// NoteGraph is a note in focus, the notes joined to it, and one connection for every line drawn between
// them. A line joins the note in focus to a related note, or two related notes to each other. A connection of the
// hierarchy has the parent as its first note and the child as its second; a connection of jumps has the note drawn
// later as its first note.
export interface NoteGraph {
  readonly focus: NoteRef
  readonly focusType: NoteType
  readonly focusTabKind?: string
  readonly related: readonly RelatedNote[]
  readonly connections: readonly Connection[]
}

// VaultNotes is what a plugin reads of the notes of the vault, and writes to them.
export interface VaultNotes {
  listRelatedNotes(path: string, depth?: 1 | 2): Promise<NoteGraph>
  getHeadings(paths: readonly string[]): Promise<ReadonlyMap<string, readonly NoteHeading[]>>
  findNotes(query: string, limit?: number): Promise<readonly NoteRef[]>
  // A note made untitled at the root of the vault: its path, and empty where none was made.
  createUntitledNote(): Promise<string>
  // The prose of a note below its frontmatter, and why it could not be read: empty where it was.
  read(path: string): Promise<{ readonly body: string; readonly error?: string }>
  // Prose written into a note, keeping its frontmatter. Writes to one path are made in order.
  write(path: string, body: string): Promise<void>
  // The frontmatter of the note at the path by its keys, and empty for a path holding no note.
  readProperties(path: string): Promise<Readonly<Record<string, JsonValue>>>
}

// ConnectionRole is what the second note of a connection is to the first: its parent, its child, or a jump.
// The role of the other note is derived by the core.
export type ConnectionRole = 'parent' | 'child' | 'jump'

// LinkRole is the role of a link as written: a ConnectionRole, a mention in prose, or an attachment.
export type LinkRole = ConnectionRole | 'ref' | 'attachment'

// ConnectionSide is the record one note writes about the other in a connection.
export interface ConnectionSide {
  readonly label: string
  readonly marker: LinkMarker | null
  readonly reason: string
}

// Connection is the relationship between two notes as a whole: up to two records, one written in each note.
export interface Connection {
  readonly first: string
  readonly second: string
  readonly role: ConnectionRole | null
  readonly type: string
  readonly firstSide: ConnectionSide | null
  readonly secondSide: ConnectionSide | null
}

// RelationJoin is the record one note is made to write about another.
export interface RelationJoin {
  readonly role: ConnectionRole
  readonly type?: string
  readonly label?: string
  readonly marker?: LinkMarker
  readonly reason?: string
}

// RelatedNoteRequest is a note to make joined to an existing one.
export interface RelatedNoteRequest {
  readonly from: string
  readonly role: ConnectionRole
  readonly title?: string
  readonly type?: string
  readonly marker?: LinkMarker
}

// Backlink is a link written in another note that points at a note.
export interface Backlink {
  readonly from: NoteRef
  readonly role: LinkRole | null
  readonly sourceRole: LinkRole | null
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

export interface OpenFileOptions {
  readonly title?: string
  readonly placement?: FilePlacement
  readonly line?: number
  readonly selector?: Selector
}

// MessageWriter writes one message under a name of its own, replacing what that name wrote last.
export type MessageWriter = (text: string, severity?: Severity) => void

// WorkspaceWindow is the state of the window a plugin's tabs are drawn in.
export interface WorkspaceWindow {
  readonly isReady: Readonly<Ref<boolean>>
  readonly openingPath: Readonly<Ref<string>>
  // The note the vault opens with, read from the vault again.
  readOpeningPath(): Promise<string>
  // The current day, with the day boundary set to the configured start hour.
  getToday(): string
}

// WorkspaceDragged is the paths of the files dragged inside the window.
export interface WorkspaceDragged {
  readonly paths: Readonly<Ref<readonly string[]>>
  setPaths(paths: readonly string[]): void
}

// WorkspaceFiles is how a plugin's tabs open the files of the vault.
export interface WorkspaceFiles {
  open(path: string, options?: OpenFileOptions): void
  // A file opened with the runs of its text marked.
  openAt(path: string, ranges: readonly TextRange[]): void
  // The tabs linked to tab `tabId` move to the path. False where none is linked to it.
  syncPath(tabId: string, path: string, title?: string): boolean
}

// WorkspaceMessages is what a plugin's tabs tell the user.
export interface WorkspaceMessages {
  showError(text: string): void
  // A writer of messages under the name given.
  getWriter(name: string): MessageWriter
}

// WorkspaceIcons is the icons the window draws.
export interface WorkspaceIcons {
  // The icon the window draws a command with, and nothing for a command it draws none for.
  getCommand(id: string): Component | null
  // The icon the window draws an entry of the vault with: a folder open or shut, a note by its type.
  getEntry(entry: VaultEntry | null, isOpen: boolean): Component
}

// WorkspaceHost is the window a plugin's tabs are drawn in.
export interface WorkspaceHost {
  readonly window: WorkspaceWindow
  readonly dragged: WorkspaceDragged
  readonly files: WorkspaceFiles
  readonly messages: WorkspaceMessages
  readonly icons: WorkspaceIcons
  // The note the most recently active tool tab depends on; empty strings when none.
  getNoteInFocus(): NoteRef
  // Sends a question to the agent and shows it to the user.
  ask(text: string): void
}

export interface VaultEntry {
  readonly path: string
  readonly name: string
  readonly isFolder: boolean
  readonly isNote: boolean
  readonly type: NoteType
  readonly tabKind?: string
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

// isNotePath reports whether a vault path is a note: a file whose extension is md, in any case. A
// file name without an extension is no note, as the when vectors of libs/core/protocol/when.json derive it.
export function isNotePath(path: string): boolean {
  const name = getFileName(path)
  const dot = name.lastIndexOf('.')
  return dot > 0 && name.slice(dot + 1).toLowerCase() === 'md'
}
