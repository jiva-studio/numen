/**
 * Type declarations for the plex tab domain.
 */
import type { Component, ComputedRef, Ref } from 'vue'
import type { MenuItem, MenuTrigger } from '@numen/ui'
import type { PlexNeighbourhood, PlexHeading, PlexRelation, PlexDestination } from './plex'
import type {
  CommandItem,
  CommandTarget,
  SlotItem,
  Connection,
  ConnectionRole,
  NoteNeighbour,
  PathRename,
  PluginContext,
  VaultRelations,
} from '@numen/plugin-sdk'

/** What a note is: a note, or a type a plugin declares. */
export type NoteType = string

/** One heading inside a note. */
export interface NoteHeading {
  readonly text: string
  readonly level: number
  readonly line: number
}

/** Focus of a neighbourhood. */
export interface Focus {
  readonly path: string
  readonly title: string
}

/** One note joined to the focus. */
export type Neighbour = NoteNeighbour

/**
 * A neighbourhood of a note: the note in focus, the notes joined to it, and the connection of
 * every line drawn between them.
 */
export interface Neighbourhood {
  readonly focus: Focus
  readonly focusType: NoteType
  readonly focusTabKind?: string
  readonly related: readonly Neighbour[]
  readonly connections: readonly Connection[]
}

/** Note relation as stored in the vault. */
export type NoteRelation = PlexRelation

/** A path rename. */
export type { PathRename }

/** The one query a plex makes of the vault: the neighbours of a note. */
export interface Neighbours {
  neighbourhood(path: string): Promise<Neighbourhood>
}

/** Position of the menu and the node it was requested on. */
export interface MenuRequest {
  readonly node: string | null
  readonly at: { x: number; y: number }
  readonly opening: MenuTrigger
}

/** Position of the quick link popover and the relation it connects. */
export interface QuickLinkRequest {
  readonly from: string
  readonly relation: PlexRelation
  readonly at: { readonly x: number; readonly y: number }
}

export interface LinkInspectorNode {
  readonly id: string
  readonly title: string
  readonly path: string
}

/**
 * Request to open the link inspector on the connection between two nodes. The
 * connection runs from the first node to the second.
 */
export interface LinkInspectorRequest {
  readonly nodeA: LinkInspectorNode
  readonly nodeB: LinkInspectorNode
  readonly connection: Connection
  readonly at: { readonly x: number; readonly y: number }
}

/** What a plex requests of the relationships between notes. */
export type PlexEditor = Pick<
  VaultRelations,
  'createRelated' | 'join' | 'getConnection' | 'saveConnection' | 'removeConnection'
>

export interface OpenNoteOptions {
  title?: string
  showing?: PlexDestination
  line?: number
  state?: PlexTabState
}

export interface PlexView {
  readonly neighbourhood: Ref<Neighbourhood | null>
  readonly here: Ref<string>
  readonly error: Ref<string>
  readonly nextPath: Ref<string>
  readonly isLoading: Ref<boolean>
  go(path: string): Promise<void>
  followMoves(renames: readonly PathRename[]): void
  close(): void
}

/** What a plex tab requests of the window: the notes it opens, selects and drags, and the messages it shows. */
export interface PlexWindowDeps {
  readonly isReady: Readonly<Ref<boolean>>
  readonly openingPath: Readonly<Ref<string>>
  readonly dragged: Readonly<Ref<readonly string[]>>
  openNote(path: string, options?: OpenNoteOptions): void
  onSelectPath?(path: string, title?: string, state?: PlexTabState): void
  /** The icon the window draws a type of note with. */
  getNoteIcon(tabKind: string): Component | null
  showMessage(text: string): void
  readOpeningPath(): Promise<string>
}

/** What a plex tab requests for its menu. */
export interface PlexMenuDeps {
  /** The entries of the slot `plexGraph.context` over the note of a node, or over the canvas. */
  getMenuEntries(target: CommandTarget): readonly SlotItem[]
  /** The items a command of the menu offers over the note of a node. */
  listItems(command: string, target: CommandTarget): Promise<readonly CommandItem[]>
  /**
   * Runs a command of the menu over the note of a node; with `itemId`, the item picked. Answers
   * the path of the note it made, empty where none.
   */
  runCommand(
    command: string,
    target: CommandTarget,
    itemId?: string,
    values?: Readonly<Record<string, string>>,
  ): Promise<string>
}

/** What a plex tab reads of the notes: their headings and names, and whether it nests the headings. */
export interface PlexNotesDeps {
  readonly isNesting: Readonly<Ref<boolean>>
  readonly headings: Readonly<Ref<number>>
  readHeadings(paths: readonly string[]): Promise<ReadonlyMap<string, readonly NoteHeading[]>>
  searchNames?(query: string, limit?: number): Promise<readonly { path: string; title: string }[]>
}

/** Dependencies for plex tab state. */
export interface PlexTabDeps {
  readonly tabId: string
  readonly editor: PlexEditor
  readonly creatable: readonly ConnectionRole[]
  readonly isNavigatingOnCreate?: Readonly<Ref<boolean>>
  readonly window: PlexWindowDeps
  readonly menu: PlexMenuDeps
  readonly notes: PlexNotesDeps
}

export interface PlexViewState {
  readonly view: PlexView
  readonly picture: ComputedRef<PlexNeighbourhood | null>
  readonly empty: ComputedRef<boolean>
  readonly dragged: ComputedRef<readonly string[]>
  readonly creatable: readonly PlexRelation[]
}

export interface PlexMenuState {
  readonly request: Readonly<Ref<MenuRequest | null>>
  readonly menuItems: ComputedRef<readonly MenuItem[]>
  getMenuIcon(id: string): Component | null
  /** The icon a node is drawn with, and nothing for an ordinary note. */
  getNodeIcon(node: string): Component | null
  openMenu(request: MenuRequest): void
  dismiss(): void
  chooseMenuItem(id: string): void
}

export interface PlexHeadingsState {
  readonly maxHeadings: Readonly<Ref<number>>
  getType(node: string): NoteType
  getHeadings(node: string): readonly PlexHeading[]
  readHeadings(): Promise<void>
  openHeading(node: string, heading: string): void
}

/** Shows the failure of an operation that runs in the background. */
export interface PlexFailureReporter {
  reportFailure(error: unknown): void
}

export interface PlexNodeActions {
  activate(node: string): void
  createNode(from: string, relation: PlexRelation, at?: { x: number; y: number }): Promise<void>
  joinNodes(from: string, to: string, relation: PlexRelation): Promise<void>
  dropNodes(dragged: readonly string[], relation: PlexRelation): Promise<void>
  openNode(node: string, showing?: PlexDestination): void
  followMoves(renamed: readonly PathRename[]): void
  getName(path: string): string
}

export interface PlexQuickLinkState {
  readonly quickLink: Ref<QuickLinkRequest | null>
  readonly isNavigatingOnCreate: Ref<boolean>
  openQuickLink(request: QuickLinkRequest): void
  dismissQuickLink(): void
  confirmQuickLink(titleOrPath: string, isExisting: boolean): Promise<void>
  searchNotes(query: string): Promise<readonly { path: string; title: string }[]>
}

export interface PlexLinkInspectorState {
  readonly linkInspector: Ref<LinkInspectorRequest | null>
  openLinkInspector(pair: string, position: { x: number; y: number }): Promise<void>
  dismissLinkInspector(): void
  saveLinkInspector(connection: Connection): Promise<void>
  removeLinkInspector(): Promise<void>
}

/** What one plex tab holds. */
export interface PlexTabState {
  readonly canvas: PlexViewState
  readonly menu: PlexMenuState
  readonly headings: PlexHeadingsState
  readonly nodes: PlexNodeActions
  readonly quickLink: PlexQuickLinkState
  readonly inspector: PlexLinkInspectorState
  readonly reportFailure: PlexFailureReporter['reportFailure']
  readonly pluginId?: string
}

export interface PlexGraphViewProps {
  readonly state: PlexTabState
  readonly context?: PluginContext
}
