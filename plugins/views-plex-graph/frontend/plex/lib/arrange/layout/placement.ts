import type { PlacedNode } from '../../node'
import type { PlexOptions } from '../options'
import type { PlexNode, PlexRelation } from '../../../../types'
import type { Spot } from './freeSpot'
import type { Link } from './lines'

/** The nodes included in the graph, grouped by relation. */
export type NodesByRelation = Readonly<Partial<Record<PlexRelation, readonly PlexNode[]>>>

/** How wide a node's box is drawn, clamped before a strategy is called. */
export type Widths = (node: PlexNode) => number

/** The space a note keeps over and under its pill, and the widest row set there. */
export interface NodeRoom {
  readonly top: number
  readonly bottom: number
  readonly width: number
}

/** A note that keeps no room around its pill. */
export const NO_ROOM: NodeRoom = { top: 0, bottom: 0, width: 0 }

/** The box a note takes: its pill and the room it keeps over and under it. */
export function getFootprint(
  pill: { readonly width: number; readonly height: number },
  room: NodeRoom,
): { readonly width: number; readonly height: number } {
  return { width: Math.max(pill.width, room.width), height: pill.height + room.top + room.bottom }
}

export const getFootprintCenterY = (pillY: number, room: NodeRoom): number =>
  pillY + (room.bottom - room.top) / 2

/** The center of the pill of a note whose footprint is centered on `spotY`. */
export const getPillCenterY = (spotY: number, room: NodeRoom): number =>
  spotY + (room.top - room.bottom) / 2

/** What a node keeps around its pill, as numbers the caller measured. */
export type Rooms = (node: PlexNode) => NodeRoom

/**
 * `fits` reports whether a footprint lies inside the window, so a group grows sideways where it has no room to grow deeper.
 * `edges` are the lines between notes: a note is set where its lines do not run along another line or through another note.
 * `isKeptApart` sets a smaller group out of the bounds of a larger one.
 */
export interface PlacementContext {
  readonly options: PlexOptions
  readonly width: Widths
  readonly room: Rooms
  readonly shouldStop?: ((node: PlacedNode) => boolean) | undefined
  readonly fits?: ((footprint: Spot) => boolean) | undefined
  readonly edges?: readonly Link[] | undefined
  readonly isKeptApart?: boolean | undefined
}

/**
 * Where the nodes go, as a strategy: a radial mind map is another implementation of this.
 * It decides coordinates only: how much is admitted is settled by the window, box widths come in as numbers, and routing follows from where the boxes ended up.
 */
export interface Placement {
  readonly name: string
  /** The focus is already at the origin; it is passed in to be cleared. */
  place(
    nodesByRelation: NodesByRelation,
    focus: PlacedNode,
    context: PlacementContext,
  ): PlacedNode[]
}
