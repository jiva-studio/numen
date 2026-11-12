import type { PlexGraph, PlexNode } from '../../types'

/**
 * Check the two invariants and return the focus. Throws: a neighborhood that
 * breaks either is a caller with a bug, not a state to be rendered.
 */
export function assertNeighborhood(neighborhood: PlexGraph): PlexNode {
  const focused = neighborhood.nodes.filter((node) => node.relation === 'focus')
  const focus = focused[0]

  if (focus === undefined) {
    throw new Error('A plex neighborhood needs a node with the focus relation')
  }
  if (focused.length > 1) {
    throw new Error(
      `A plex neighborhood has one focus, not ${focused.length}: ` +
        focused.map((node) => node.id).join(', '),
    )
  }

  // A hierarchy with several parents can reach the same node two ways. Which
  // relation it takes is the caller's decision; drawn twice, the edges would route
  // to whichever copy was placed last, and a movement matched by identifier
  // would be ambiguous.
  const observed = new Set<string>()
  const repeated = new Set<string>()
  for (const node of neighborhood.nodes) {
    if (observed.has(node.id)) repeated.add(node.id)
    observed.add(node.id)
  }
  if (repeated.size > 0) {
    throw new Error(`A plex node appears more than once: ${[...repeated].join(', ')}`)
  }

  return focus
}
