import type { PlacedNode } from '../../node'
import type { PlexOptions } from '../options'
import type { PlexNode, PlexRelation } from '../../../../types'

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

/** What a node keeps around its pill, as numbers the caller measured. */
export type Rooms = (node: PlexNode) => NodeRoom

/** The space a note keeps over and under its pill, and the widest row set there. */
export interface NodeRoom {
  readonly top: number
  readonly bottom: number
  readonly width: number
}

/** A note that keeps no room around its pill. */
export const NO_ROOM: NodeRoom = { top: 0, bottom: 0, width: 0 }

/** What a node keeps around its pill, as numbers the caller measured. */
export type Rooms = (node: PlexNode) => NodeRoom

export interface PlacementContext {
  readonly options: PlexOptions
  readonly width: Widths
  readonly room: Rooms
  readonly room: Rooms
  readonly shouldStop?: ((node: PlacedNode) => boolean) | undefined
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
