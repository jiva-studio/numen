/**
 * Turn a neighbourhood into a frame: admit, place, route.
 *
 * Pure — the same inputs give the same numbers on any machine. Nothing here
 * reads the clock, measures text or touches the DOM.
 */
import {
  assertNeighbourhood,
  extentOf,
  RELATED_SEATS,
  type PlacedNode,
  type PlexFrame,
  type PlexNeighbourhood,
  type PlexNode,
  type PlexRelatedSeat,
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

  const counts = countSeats(neighbourhood.nodes)
  const limits = limitsFor(resolved, counts)

  const { seating, overflow } = admit(neighbourhood.nodes, limits)
  const nodes = [focus, ...placement.place(seating, focus, resolved, limits)]

  const byId = new Map(nodes.map((node) => [node.id, node]))
  const edges = routeEdges(neighbourhood.edges, byId, routingFor(resolved))

  return { nodes, edges, extent: extentOf(nodes), overflow }
}

function countSeats(nodes: readonly PlexNode[]): Record<PlexRelatedSeat, number> {
  const counts = Object.fromEntries(RELATED_SEATS.map((seat) => [seat, 0])) as Record<
    PlexRelatedSeat,
    number
  >
  for (const node of nodes) {
    if (node.seat !== 'focus') counts[node.seat] += 1
  }
  return counts
}

/**
 * How much of each seat the picture holds. Settled before a strategy is
 * called, so the overflow count means the same thing whichever one draws it.
 */
function admit(
  nodes: readonly PlexNode[],
  limits: Limits,
): { seating: Seating; overflow: Partial<Record<PlexRelatedSeat, number>> } {
  const seating: Partial<Record<PlexRelatedSeat, readonly PlexNode[]>> = {}
  const overflow: Partial<Record<PlexRelatedSeat, number>> = {}

  for (const seat of RELATED_SEATS) {
    const ofSeat = nodes.filter((node) => node.seat === seat)
    if (ofSeat.length === 0) continue

    const capacity = limits[seat].perLine * limits[seat].lines
    const shown = ofSeat.slice(0, capacity)
    if (ofSeat.length > shown.length) overflow[seat] = ofSeat.length - shown.length
    seating[seat] = shown
  }

  return { seating, overflow }
}
