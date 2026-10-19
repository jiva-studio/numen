/**
 * Type declarations for the plex tab domain.
 */
import type { ComputedRef, Ref } from 'vue'
import type {
  MenuTrigger,
  PlexNeighbourhood,
  PlexHeading,
  PlexRelation,
  PlexDestination,
} from '@numen/ui'
import type { PathRename, PluginContext } from '@numen/plugin-sdk'

/** Which of four types a note can be. */
export type NoteType = 'note' | 'deck' | 'stencil' | 'preset'

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

/** The one question a plex asks of the vault: what is around a note. */
export interface Neighbours {
  neighbourhood(path: string): Promise<Neighbourhood>
}

/** Where the menu stands, and the node it was asked for on. */
export interface MenuRequest {
  /** The node it was asked for on, or null when asked off every node. */
  readonly node: string | null
  readonly at: { x: number; y: number }
  readonly opening: MenuTrigger
}

/** Where the quick link popover stands, and what relation it connects. */
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

/** Making a note from the picture, and joining two that are already on it. */
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
  askAgent(text: string): void
  runCommand(id: string, path: string, title: string): void
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
  openMenu(asked: MenuRequest): void
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
  openLinkInspector(pair: string, at: { x: number; y: number }): void
  dismissLinkInspector(): void
  saveLinkInspector(rows: readonly DirectionalLinkItem[]): Promise<void>
}

/** What one plex tab holds. */
export type PlexTabState = PlexViewState &
  PlexMenuState &
  PlexHeadingsState &
  PlexNodeActions &
  PlexQuickLinkState &
  PlexLinkInspectorState & {
    readonly pluginId?: string
  }

export interface PlexGraphViewProps {
  readonly state: PlexTabState
  readonly context?: PluginContext
}
