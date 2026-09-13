import type { PlexEdge } from './edge'
import type { PlexNode } from './node'

/** One neighbourhood: the node in focus and everything shown around it. */
export interface PlexNeighbourhood {
  readonly nodes: readonly PlexNode[]
  readonly edges: readonly PlexEdge[]
}

/**
 * Check the two invariants and return the focus. Throws: a neighbourhood that
 * breaks either is a caller with a bug, not a state to be rendered.
 */
export function assertNeighbourhood(neighbourhood: PlexNeighbourhood): PlexNode {
  const focused = neighbourhood.nodes.filter((node) => node.seat === 'focus')
  const focus = focused[0]

  if (focus === undefined) {
    throw new Error('A plex neighbourhood needs a node with the focus seat')
  }
  if (focused.length > 1) {
    throw new Error(
      `A plex neighbourhood has one focus, not ${focused.length}: ` +
        focused.map((node) => node.id).join(', '),
    )
  }

  // A hierarchy with several parents can reach the same node two ways. Which
  // seat it takes is the caller's decision; drawn twice, the edges would route
  // to whichever copy was placed last, and a movement matched by identifier
  // would be ambiguous.
  const seen = new Set<string>()
  const repeated = new Set<string>()
  for (const node of neighbourhood.nodes) {
    if (seen.has(node.id)) repeated.add(node.id)
    seen.add(node.id)
  }
  if (repeated.size > 0) {
    throw new Error(`A plex node appears more than once: ${[...repeated].join(', ')}`)
  }

  return focus
}
