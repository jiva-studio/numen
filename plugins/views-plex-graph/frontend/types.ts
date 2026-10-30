/**
 * Type declarations for the plex tab domain.
 */
import type { Component, ComputedRef, Ref } from 'vue'
import type { MenuItem, MenuTrigger } from '@numen/ui'
import type { PlexNeighbourhood, PlexHeading, PlexRelation, PlexDestination } from './plex'
import type { PathRename, PluginContext } from '@numen/plugin-sdk'

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

/** One neighbour joined to the focus. */
export interface Neighbour {
  readonly path: string
  readonly title: string
  readonly relation: PlexRelation
  readonly type: NoteType
  readonly label: string
  readonly reverseLabel?: string
  readonly through: string
  readonly isMutual: boolean
}

/** A neighbourhood of a note: the note in focus, and the notes joined to it. */
export interface Neighbourhood {
  readonly focus: Focus
  readonly focusType: NoteType
  readonly related: readonly Neighbour[]
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
  /** The node the menu was requested on, or null when requested outside every node. */
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

/** Which of the two notes a link is written in: the first (forward) or the second (reverse). */
export type LinkDirectionMode = 'forward' | 'reverse'

/** One link between two notes in the inspector, written in the note it runs from. */
export interface DirectionalLinkItem {
  readonly id: string
  readonly from: string
  readonly to: string
  readonly role: PlexRelation
  readonly direction: LinkDirectionMode
  readonly label: string
}

export interface LinkInspectorNode {
  readonly id: string
  readonly title: string
  readonly path: string
}

/** Request to open the link inspector for a connection between two nodes. */
export interface LinkInspectorRequest {
  readonly nodeA: LinkInspectorNode
  readonly nodeB: LinkInspectorNode
  readonly links: readonly DirectionalLinkItem[]
  readonly at: { readonly x: number; readonly y: number }
}

/** Creating a note from the plex, and linking two notes already in it. */
export interface PlexEditor {
  createInRelation(from: string, relation: PlexRelation): Promise<unknown>
  createWithTitle(
    title: string,
    from: string,
    relation: PlexRelation,
  ): Promise<{ path: string } | null>
  join(from: string, to: string, relation: PlexRelation, label?: string): Promise<boolean>
  replaceLinks(
    first: string,
    second: string,
    links: readonly { from: string; to: string; relation: PlexRelation; label: string }[],
  ): Promise<boolean>
}

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
  /** A requested note whose result has not arrived, or that the vault does not hold yet; empty when none. */
  readonly nextPath: Ref<string>
  readonly isLoading: Ref<boolean>
  go(path: string): Promise<void>
  followMoves(renames: readonly PathRename[]): void
  close(): void
}

/** Dependencies for plex tab state. */
export interface PlexTabDeps {
  readonly editor: PlexEditor
  readonly isReady: Readonly<Ref<boolean>>
  readonly isNesting: Readonly<Ref<boolean>>
  readonly headings: Readonly<Ref<number>>
  openNote(path: string, options?: OpenNoteOptions): void
  onSelectPath?(path: string, title?: string, state?: PlexTabState): void
  readHeadings(paths: readonly string[]): Promise<ReadonlyMap<string, readonly NoteHeading[]>>
  runCommand(id: string, path: string, title: string): void
  /** The commands the plugins declare over one note. */
  getNoteCommands(): readonly { readonly id: string; readonly text: string }[]
  /** The icon the window draws a command with, and nothing for one it draws none for. */
  getCommandIcon(id: string): Component | null
  /** The icon the window draws a type of note with. */
  getNoteIcon(type: NoteType): Component | null
  readonly openingPath: Readonly<Ref<string>>
  readonly dragged: Readonly<Ref<readonly string[]>>
  showMessage(text: string): void
  createUntitledNote(): Promise<string>
  readOpeningPath(): Promise<string>
  readonly creatable: readonly PlexRelation[]
  searchNames?(query: string, limit?: number): Promise<readonly { path: string; title: string }[]>
  readonly isNavigatingOnCreate?: Readonly<Ref<boolean>>
}

export interface PlexViewState {
  readonly view: PlexView
  readonly picture: ComputedRef<PlexNeighbourhood | null>
  readonly empty: ComputedRef<boolean>
  readonly dragged: ComputedRef<readonly string[]>
  readonly creatable: readonly PlexRelation[]
}

export interface PlexMenuState {
  readonly menu: Ref<MenuRequest | null>
  /** What the menu offers where it is open: on a node, or off every node. */
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
  createNote(): Promise<void>
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
  openLinkInspector(pair: string, position: { x: number; y: number }): void
  dismissLinkInspector(): void
  saveLinkInspector(rows: readonly DirectionalLinkItem[]): Promise<void>
}

/** What one plex tab holds. */
export type PlexTabState = PlexViewState &
  PlexMenuState &
  PlexHeadingsState &
  PlexNodeActions &
  PlexFailureReporter &
  PlexQuickLinkState &
  PlexLinkInspectorState & {
    readonly pluginId?: string
  }

export interface PlexGraphViewProps {
  readonly state: PlexTabState
  readonly context?: PluginContext
}
