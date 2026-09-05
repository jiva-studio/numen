/**
 * Turn a neighbourhood into a frame: admit, place, route.
 *
 * Pure — the same inputs give the same numbers on any machine. Nothing here
 * reads the clock, measures text or touches the DOM: how wide a title is
 * arrives as a number, from whoever is drawing it.
 */
import { extentOf, type PlexFrame } from '../frame'
import { assertNeighbourhood, type PlexNeighbourhood } from '../neighbourhood'
import type { PlacedNode, PlexNode } from '../node'
import { RELATED_SEATS, type PlexRelatedSeat } from '../seat'
import { crowdingFor } from './crowding'
import { limitsFor, type Limits } from './limits'
import { resolveOptions, type PlexOptions, type PlexOptionsInput } from './options'
import { rowsAndColumns, type Placement, type Seating, type Widths } from './placement'
import { routeEdges, routingFor } from './routing'
import { spacingFor, type Spacing } from './spacing'
import { settleTitles } from './titles'

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
  /**
   * How deep a label's line stands, across the line it is set on, measured
   * where the plex is drawn. A label is set in a type of its own, and one
   * given no depth takes up none.
   */
  readonly labelDepth?: number | undefined
}

export function arrangePlex(
  neighbourhood: PlexNeighbourhood,
  {
    options,
    placement = rowsAndColumns,
    measure,
    measureLabel,
    labelDepth,
  }: ArrangeInput = {},
): PlexFrame {
  const asked = resolveOptions(options)
  const focusNode = assertNeighbourhood(neighbourhood)
  const counts = countSeats(neighbourhood.nodes)

  // Packed as closely as this window needs and no closer, so a box narrows
  // and a gap closes before a seat is given up.
  const resolved = crowdingFor(asked, counts)
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

  const limits = limitsFor(resolved, counts)

  const { seating, overflow } = admit(neighbourhood.nodes, limits)

  // How much is admitted is settled at that packing, and the gaps then open
  // into the room that is left. The opening is measured against the
  // arrangement itself, so a placement of any shape keeps the window.
  const lay = (spacing: Spacing) =>
    placement.place(seating, focus, { ...resolved, ...spacing }, limits, widthOf)
  const spacing = spacingFor(resolved, (candidate) => within(lay(candidate), resolved))

  const nodes = [focus, ...lay(spacing)]

  const byId = new Map(nodes.map((node) => [node.id, node]))
  const routing = routingFor(resolved, measureLabel, labelDepth)
  const edges = settleTitles(
    routeEdges(neighbourhood.edges, byId, routing),
    nodes,
    routing,
  )

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

/** Whether every node stays inside the window, the margin kept clear. */
function within(nodes: readonly PlacedNode[], options: PlexOptions): boolean {
  const { viewport, margin } = options
  if (!viewport) return true

  const halfWidth = viewport.width / 2 - margin
  const halfHeight = viewport.height / 2 - margin
  return nodes.every(
    (node) =>
      Math.abs(node.x) + node.width / 2 <= halfWidth &&
      Math.abs(node.y) + node.height / 2 <= halfHeight,
  )
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
