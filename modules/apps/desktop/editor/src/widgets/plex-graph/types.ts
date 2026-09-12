/**
 * Type declarations for the plex tab domain.
 */
import type { ComputedRef, Ref } from 'vue'
import type {
  MenuOpening,
  PlexNeighbourhood,
  PlexPart,
  PlexRelatedSeat,
  PlexShowing,
} from '@numen/ui'
import type { NoteHeading } from '@/entities/note'
import type { NoteType } from '@/shared/file'
import type { PathRename } from '@/shared/paths'
import type { PlexView } from './model/usePlexView'

/** Where the menu stands, and the node it was asked for on. */
export interface MenuRequest {
  /** The node it was asked for on, or null when asked off every node. */
  readonly node: string | null
  readonly at: { x: number; y: number }
  readonly opening: MenuOpening
}

/** Making a note from the picture, and joining two that are already on it. */
export interface PlexEditor {
  make(from: string, seat: PlexRelatedSeat): Promise<unknown>
  join(from: string, to: string, seat: PlexRelatedSeat): Promise<boolean>
}

/** Dependencies for plex tab state. */
export interface PlexTabDeps {
  readonly makes: PlexEditor
  readonly ready: Readonly<Ref<boolean>>
  readonly hangs: Readonly<Ref<boolean>>
  readonly parts: Readonly<Ref<number>>
  opens(path: string, title: string, showing: PlexShowing, line?: number): void
  inside(paths: readonly string[]): Promise<ReadonlyMap<string, readonly NoteHeading[]>>
  asks(text: string): void
  runs(id: string, path: string, title: string): void
  readonly opening: Readonly<Ref<string>>
  readonly dragged: Readonly<Ref<readonly string[]>>
  says(text: string): void
  writes(): Promise<string>
  first(): Promise<string>
  readonly creatable: readonly PlexRelatedSeat[]
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
  asks(asked: MenuRequest): void
  chose(id: string): void
}

export interface PlexPartsState {
  readonly mostParts: Readonly<Ref<number>>
  typeOf(node: string): NoteType
  partsOf(node: string): readonly PlexPart[]
  reads(): Promise<void>
  readParts(): Promise<void>
  openPart(node: string, part: string): void
  entered(node: string, part: string): void
}

export interface PlexNodeActions {
  activate(node: string): void
  createNode(from: string, seat: PlexRelatedSeat): Promise<void>
  made(from: string, seat: PlexRelatedSeat): Promise<void>
  joinNodes(from: string, to: string, seat: PlexRelatedSeat): Promise<void>
  joined(from: string, to: string, seat: PlexRelatedSeat): Promise<void>
  bringNodes(dragged: readonly string[], seat: PlexRelatedSeat): Promise<void>
  brought(dragged: readonly string[], seat: PlexRelatedSeat): Promise<void>
  openNode(node: string, showing?: PlexShowing): void
  opens(node: string, showing?: PlexShowing): void
  createNote(): Promise<void>
  writes(): Promise<void>
  followMoves(renamed: readonly PathRename[]): void
  follows(renamed: readonly PathRename[]): void
  getName(path: string): string
  nameOf(path: string): string
}

/** What one plex tab holds. */
export type PlexTabState = PlexViewState & PlexMenuState & PlexPartsState & PlexNodeActions
