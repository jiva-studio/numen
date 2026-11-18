import type { PlacedEdge } from './edge'
import type { PlacedNode } from './node'
import type { PlexRelation } from '../../types'

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
  readonly overflow: Readonly<Partial<Record<PlexRelation, number>>>
}

export function getExtent(nodes: readonly PlacedNode[]): Extent {
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

/** Whether a frame holds other nodes than another, or any node of it lies more than a pixel from where the other has it, in place or in size. */
export function isFrameMoved(from: PlexFrame, to: PlexFrame): boolean {
  if (from.nodes.length !== to.nodes.length) return true
  const before = new Map(from.nodes.map((node) => [node.id, node]))
  return to.nodes.some((node) => {
    const was = before.get(node.id)
    return (
      !was ||
      Math.abs(was.x - node.x) > 1 ||
      Math.abs(was.y - node.y) > 1 ||
      Math.abs(was.width - node.width) > 1 ||
      Math.abs(was.height - node.height) > 1
    )
  })
}
