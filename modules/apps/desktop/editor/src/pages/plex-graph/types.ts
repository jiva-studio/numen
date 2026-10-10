/**
 * Type declarations for the plex tab domain.
 */
import type { ComputedRef, Ref } from 'vue'
import type {
  MenuOpening,
  PlexNeighbourhood,
  PlexPart,
  PlexRelatedSeat,
  PlexDestination,
} from '@numen/ui'
import type { NoteHeading } from '@/entities/note'
import type { NoteType } from '@/entities/file'
import type { PathRename } from '@/shared/paths'
import type { PlexView } from './model/usePlexView'

/** Where the menu stands, and the node it was asked for on. */
export interface MenuRequest {
  /** The node it was asked for on, or null when asked off every node. */
  readonly node: string | null
  readonly at: { x: number; y: number }
  readonly opening: MenuOpening
}

/** Where the quick link popover stands, and what seat it connects. */
export interface QuickLinkRequest {
  readonly from: string
  readonly seat: PlexRelatedSeat
  readonly at: { readonly x: number; readonly y: number }
}

/** Making a note from the picture, and joining two that are already on it. */
export interface PlexEditor {
  createInSeat(from: string, seat: PlexRelatedSeat): Promise<unknown>
  createWithTitle(
    title: string,
    from: string,
    seat: PlexRelatedSeat,
  ): Promise<{ path: string } | null>
  join(from: string, to: string, seat: PlexRelatedSeat): Promise<boolean>
}

/** Dependencies for plex tab state. */
export interface PlexTabDeps {
  readonly editor: PlexEditor
  readonly isReady: Readonly<Ref<boolean>>
  readonly isHanging: Readonly<Ref<boolean>>
  readonly parts: Readonly<Ref<number>>
  openNote(
    path: string,
    title: string,
    showing: PlexDestination,
    line?: number,
    state?: PlexTabState,
  ): void
  onSelectPath?(path: string, title?: string, state?: PlexTabState): void
  readHeadings(paths: readonly string[]): Promise<ReadonlyMap<string, readonly NoteHeading[]>>
  askAgent(text: string): void
  runCommand(id: string, path: string, title: string): void
  readonly openingPath: Readonly<Ref<string>>
  readonly dragged: Readonly<Ref<readonly string[]>>
  showMessage(text: string): void
  createUntitledNote(): Promise<string>
  readOpeningPath(): Promise<string>
  readonly creatable: readonly PlexRelatedSeat[]
  searchNames?(query: string, limit?: number): Promise<readonly { path: string; title: string }[]>
  readonly isNavigatingOnCreate?: Readonly<Ref<boolean>>
}

export interface PlexViewState {
  readonly view: PlexView
  readonly picture: ComputedRef<PlexNeighbourhood | null>
  readonly empty: ComputedRef<boolean>
  readonly dragged: ComputedRef<readonly string[]>
  readonly creatable: readonly PlexRelatedSeat[]
}

export interface PlexMenuState {
  readonly menu: Ref<MenuRequest | null>
  openMenu(asked: MenuRequest): void
  dismiss(): void
  chooseMenuItem(id: string): void
}

export interface PlexPartsState {
  readonly mostParts: Readonly<Ref<number>>
  typeOf(node: string): NoteType
  partsOf(node: string): readonly PlexPart[]
  readParts(): Promise<void>
  openPart(node: string, part: string): void
}

export interface PlexNodeActions {
  activate(node: string): void
  createNode(from: string, seat: PlexRelatedSeat, at?: { x: number; y: number }): Promise<void>
  joinNodes(from: string, to: string, seat: PlexRelatedSeat): Promise<void>
  dropNodes(dragged: readonly string[], seat: PlexRelatedSeat): Promise<void>
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

/** What one plex tab holds. */
export type PlexTabState = PlexViewState &
  PlexMenuState &
  PlexPartsState &
  PlexNodeActions &
  PlexQuickLinkState
