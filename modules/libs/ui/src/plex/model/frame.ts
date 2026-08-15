import type { PlacedEdge } from './edge'
import type { PlacedNode } from './node'
import type { PlexRelatedRole } from './role'

/** The box every placed node fits inside. */
export interface Extent {
  readonly minX: number
  readonly minY: number
  readonly maxX: number
  readonly maxY: number
}

/**
 * Everything to be drawn, at one moment. A finished arrangement and a frame
 * partway through a movement are the same shape, which is what lets one
 * renderer draw both.
 */
export interface PlexFrame {
  readonly nodes: readonly PlacedNode[]
  readonly edges: readonly PlacedEdge[]
  readonly extent: Extent
  /** Nodes of a role that did not fit. Empty when everything fit. */
  readonly overflow: Readonly<Partial<Record<PlexRelatedRole, number>>>
}

export function extentOf(nodes: readonly PlacedNode[]): Extent {
  let minX = Infinity
  let minY = Infinity
  let maxX = -Infinity
  let maxY = -Infinity

  for (const node of nodes) {
    minX = Math.min(minX, node.x - node.width / 2)
    minY = Math.min(minY, node.y - node.height / 2)
    maxX = Math.max(maxX, node.x + node.width / 2)
    maxY = Math.max(maxY, node.y + node.height / 2)
  }

  return { minX, minY, maxX, maxY }
}
