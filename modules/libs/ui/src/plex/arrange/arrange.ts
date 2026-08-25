/**
 * Turn a neighbourhood into a frame: admit, place, route.
 *
 * Pure — the same inputs give the same numbers on any machine. Nothing here
 * reads the clock, measures text or touches the DOM: how wide a title is
 * arrives as a number, from whoever is drawing it.
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
import { resolveOptions, type PlexOptions, type PlexOptionsInput } from './options'
import { rowsAndColumns, type Placement, type Seating, type Widths } from './placement'
import { routeEdges, routingFor } from './routing'

export interface ArrangeInput {
  readonly options?: PlexOptionsInput | undefined
  /** Defaults to rows and columns — the arrangement a plex is recognised by. */
  readonly placement?: Placement | undefined
  /**
   * The width a node's box needs to hold its title, padding included. Text is
   * measured where the plex is drawn; here it arrives as a number. Without one
   * every box is drawn at its widest.
   */
  readonly measure?: ((node: PlexNode) => number) | undefined
  /**
   * The width a label's words need on their line, measured where the plex is
   * drawn. Without one a label is written at whatever length it has.
   */
  readonly measureLabel?: ((label: string) => number) | undefined
}

export function arrangePlex(
  neighbourhood: PlexNeighbourhood,
  { options, placement = rowsAndColumns, measure, measureLabel }: ArrangeInput = {},
): PlexFrame {
  const resolved = resolveOptions(options)
  const focusNode = assertNeighbourhood(neighbourhood)
  const widthOf = widthsFor(resolved, measure)

  const focus: PlacedNode = {
    ...focusNode,
    x: 0,
    y: 0,
    width: widthOf(focusNode),
    height: resolved.focusSize.height,
    order: 0,
    opacity: 1,
  }

  const counts = countSeats(neighbourhood.nodes)
  const limits = limitsFor(resolved, counts)

  const { seating, overflow } = admit(neighbourhood.nodes, limits)
  const nodes = [focus, ...placement.place(seating, focus, resolved, limits, widthOf)]

  const byId = new Map(nodes.map((node) => [node.id, node]))
  const edges = routeEdges(neighbourhood.edges, byId, routingFor(resolved, measureLabel))

  return { nodes, edges, extent: extentOf(nodes), overflow }
}

/**
 * How wide each box is drawn: what the measurer asks for, held between the
 * minimum and the widest the node's seat allows. The focus is measured as
 * every other node is, against its own width.
 */
function widthsFor(
  options: PlexOptions,
  measure: ((node: PlexNode) => number) | undefined,
): Widths {
  return (node) => {
    const widest =
      node.seat === 'focus' ? options.focusSize.width : options.nodeSize.width
    if (!measure) return widest
    return Math.min(widest, Math.max(options.minWidth, measure(node)))
  }
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
