import type { PlacedNode } from '../../node'
import type { PlexOptions } from '../options'
import type { PlexNode, PlexRelation } from '../../../../types'

/** The nodes included in the graph, grouped by relation. */
export type NodesByRelation = Readonly<Partial<Record<PlexRelation, readonly PlexNode[]>>>

/** How wide a node's box is drawn, clamped before a strategy is called. */
export type Widths = (node: PlexNode) => number

export interface PlacementContext {
  readonly options: PlexOptions
  readonly width: Widths
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
