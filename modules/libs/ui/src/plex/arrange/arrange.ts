/**
 * Turn a neighbourhood into a frame: admit, place, route.
 *
 * Pure — the same inputs give the same numbers on any machine. Nothing here
 * reads the clock, measures text or touches the DOM.
 */
import {
  assertNeighbourhood,
  extentOf,
  RELATED_ROLES,
  type PlacedNode,
  type PlexFrame,
  type PlexNeighbourhood,
  type PlexNode,
  type PlexRelatedRole,
} from '../model'
import { limitsFor, type Limits } from './limits'
import { resolveOptions, type PlexOptionsInput } from './options'
import { rowsAndColumns, type Placement, type Seating } from './placement'
import { routeEdges, routingFor } from './routing'

export interface ArrangeInput {
  readonly options?: PlexOptionsInput | undefined
  /** Defaults to rows and columns — the arrangement a plex is recognised by. */
  readonly placement?: Placement | undefined
}

export function arrangePlex(
  neighbourhood: PlexNeighbourhood,
  { options, placement = rowsAndColumns }: ArrangeInput = {},
): PlexFrame {
  const resolved = resolveOptions(options)
  const focusNode = assertNeighbourhood(neighbourhood)

  const focus: PlacedNode = {
    ...focusNode,
    x: 0,
    y: 0,
    width: resolved.focusSize.width,
    height: resolved.focusSize.height,
    order: 0,
    opacity: 1,
  }

  const counts = countRoles(neighbourhood.nodes)
  const limits = limitsFor(resolved, counts)

  const { seating, overflow } = admit(neighbourhood.nodes, limits)
  const nodes = [focus, ...placement.place(seating, focus, resolved, limits)]

  const byId = new Map(nodes.map((node) => [node.id, node]))
  const edges = routeEdges(neighbourhood.edges, byId, routingFor(resolved))

  return { nodes, edges, extent: extentOf(nodes), overflow }
}

function countRoles(nodes: readonly PlexNode[]): Record<PlexRelatedRole, number> {
  const counts = Object.fromEntries(RELATED_ROLES.map((role) => [role, 0])) as Record<
    PlexRelatedRole,
    number
  >
  for (const node of nodes) {
    if (node.role !== 'focus') counts[node.role] += 1
  }
  return counts
}

/**
 * How much of each role the picture holds. Settled before a strategy is
 * called, so the overflow count means the same thing whichever one draws it.
 */
function admit(
  nodes: readonly PlexNode[],
  limits: Limits,
): { seating: Seating; overflow: Partial<Record<PlexRelatedRole, number>> } {
  const seating: Partial<Record<PlexRelatedRole, readonly PlexNode[]>> = {}
  const overflow: Partial<Record<PlexRelatedRole, number>> = {}

  for (const role of RELATED_ROLES) {
    const ofRole = nodes.filter((node) => node.role === role)
    if (ofRole.length === 0) continue

    const capacity = limits[role].perLine * limits[role].lines
    const shown = ofRole.slice(0, capacity)
    if (ofRole.length > shown.length) overflow[role] = ofRole.length - shown.length
    seating[role] = shown
  }

  return { seating, overflow }
}
