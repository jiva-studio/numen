/**
 * Type declarations for the plex tab domain.
 */
import type { Component, ComputedRef, Ref } from 'vue'
import type { MenuItem, MenuTrigger } from '@numen/ui'
import type {
  CommandItem,
  CommandTarget,
  SlotItem,
  Connection,
  ConnectionRole,
  RelatedNote,
  PathRename,
  PluginContext,
  VaultRelations,
  Caught,
} from '@numen/plugin-sdk'

/** What a note is: a note, or a type a plugin declares. */
export type NoteType = string

/** One heading inside a note. */
export interface NoteHeading {
  readonly text: string
  readonly level: number
  readonly line: number
}

/** Focus of a note graph. */
export interface Focus {
  readonly path: string
  readonly title: string
}

/** One note joined to the focus. */
export type { RelatedNote }

/**
 * A note and the notes related to it: the note in focus, the notes joined to it, and the connection of
 * every line drawn between them.
 */
export interface NoteGraph {
  readonly focus: Focus
  readonly focusType: NoteType
  readonly focusTabKind?: string
  readonly related: readonly RelatedNote[]
  readonly connections: readonly Connection[]
}

/** Note relation as stored in the vault. */
export type NoteRelation = PlexRelation

/** A path rename. */
export type { PathRename }

/** The one query a plex makes of the vault: the related notes of a note. */
export interface RelatedNoteQueries {
  listRelatedNotes(path: string, depth?: 1 | 2): Promise<NoteGraph>
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
  readonly noteGraph: Ref<NoteGraph | null>
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

/** Whether names are shown and how many a note shows, as the plugin settings hold them. */
export interface PlexNamesSettings {
  readonly showNames: Readonly<Ref<boolean>>
  readonly namesPerNote: Readonly<Ref<number>>
}

/** How the plex is drawn, as the plugin settings hold it, followed as they change. */
export interface PlexLook {
  readonly references: Readonly<Ref<ReferenceMode>>
  readonly showPills: Readonly<Ref<boolean>>
  readonly highlightLinks: Readonly<Ref<boolean>>
  readonly pillPaddingSides: Readonly<Ref<number>>
  readonly pillPaddingEnds: Readonly<Ref<number>>
  readonly focusSize: Readonly<Ref<number>>
  readonly animationLength: Readonly<Ref<number>>
  readonly lineThickness: Readonly<Ref<number>>
  readonly outlineThickness: Readonly<Ref<number>>
  readonly lineCurvature: Readonly<Ref<number>>
  readonly noteRoom: Readonly<Ref<number>>
  readonly namesOpacity: Readonly<Ref<number>>
  readonly focusedNamesOpacity: Readonly<Ref<number>>
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
  readonly names: PlexNamesSettings
  readonly look: PlexLook
}

export interface PlexViewState {
  readonly view: PlexView
  readonly picture: ComputedRef<PlexGraph | null>
  readonly empty: ComputedRef<boolean>
  readonly dragged: ComputedRef<readonly string[]>
  readonly creatable: readonly PlexRelation[]
  readonly names: PlexNamesState
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

/**
 * The names of the notes related to the drawn notes. A node with none, and every node while names are off, has none;
 * choosing a name moves the focus to the note it stands for.
 */
export interface PlexNamesState {
  readonly showNames: Readonly<Ref<boolean>>
  getNames(node: string): PlexNames
  choose(name: string): void
}

export interface PlexHeadingsState {
  readonly maxHeadings: Readonly<Ref<number>>
  getType(node: string): NoteType
  getHeadings(node: string): readonly PlexHeading[]
  readHeadings(): Promise<void>
}

/** Shows the failure of an operation that runs in the background. */
export interface PlexFailureReporter {
  reportFailure(error: Caught): void
}

export interface PlexNodeActions {
  activate(node: string): void
  addNode(from: string, relation: PlexRelation, at?: { x: number; y: number }): Promise<void>
  joinNodes(from: string, to: string, relation: PlexRelation): Promise<void>
  dropNodes(dragged: readonly string[], relation: PlexRelation): Promise<void>
  openNode(node: string, showing?: PlexDestination): void
  openHeading(node: string, heading: string): void
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
  readonly look: PlexLook
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

/** Which references to show: the links the note writes, the links to it, both, or none. */
export type ReferenceMode = 'off' | 'outgoing' | 'incoming' | 'both'

/** Every relation a node can take to the one in focus. */
export type PlexRelation = 'parent' | 'child' | 'jump' | 'reference' | 'sibling'

/** Where a node is positioned relative to the one in focus. */
export type PlexNodeRelation = 'focus' | PlexRelation

/** Where an opened node is shown: in the current pane, or in an adjacent split. */
export type PlexDestination = 'current' | 'split'

/**
 * A node as the caller describes it. The id is opaque: the plex cannot query what
 * it addresses, and the relation is a position in the layout, not a claim about
 * a graph.
 */
export interface PlexNode {
  readonly id: string
  readonly title: string
  readonly relation: PlexNodeRelation
}

/** The end of a line an arrowhead is drawn at, pointing out of the line there. */
export type EdgeArrow = 'from' | 'to'

/**
 * Words set along an edge. A line with one label carries it in the middle, on
 * the line, its letters across it. With two whose records both have a head,
 * each is set near the end its head is at, on its own side, and reads toward that end. Otherwise
 * the first, written in the `from` note, is set near the `from` end and reads from `from` to `to`,
 * and the second, written in the `to` note, is set near the `to` end on the other side and reads back.
 */
export interface EdgeLabel {
  readonly text: string
  readonly end?: EdgeArrow
}

export interface PlexEdge {
  readonly from: string
  readonly to: string
  readonly labels?: readonly EdgeLabel[]
  readonly arrows?: readonly EdgeArrow[]
}

/** One graph: the node in focus and everything shown around it. */
export interface PlexGraph {
  readonly nodes: readonly PlexNode[]
  readonly edges: readonly PlexEdge[]
}

/**
 * One heading of a node, as the caller describes it. The identifier is opaque:
 * the plex has no way to ask what it addresses.
 */
export interface PlexHeading {
  readonly id: string
  readonly text: string
  readonly level: number
}

/** The name of a note shown beside a related note. The id is the caller's, returned unchanged when the name is chosen. */
export interface PlexName {
  readonly id: string
  readonly title: string
}

/** The names set over a related note, and those set under it. */
export interface PlexNames {
  readonly above: PlexName[]
  readonly below: PlexName[]
}
