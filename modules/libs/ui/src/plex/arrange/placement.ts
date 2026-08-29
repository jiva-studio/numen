import type { PlacedNode, PlexNode, PlexRelatedSeat } from '../model'
import type { Limits, RoleLimits } from './limits'
import { isVertical, type Direction, type PlexOptions } from './options'

/** The nodes admitted to the picture, grouped by the seat they take. */
export type Seating = Readonly<Partial<Record<PlexRelatedSeat, readonly PlexNode[]>>>

/** How wide a node's box is drawn, clamped before a strategy is called. */
export type Widths = (node: PlexNode) => number

/**
 * Where the nodes go. A strategy, so a radial mind map is another
 * implementation rather than a branch inside this one.
 *
 * It decides coordinates and nothing else: how much is admitted is settled
 * before it is called, how wide each box is comes in as a number, and routing
 * follows from where the boxes ended up.
 */
export interface Placement {
  readonly name: string
  /** The focus is already at the origin; it is passed in to be cleared. */
  place(
    seating: Seating,
    focus: PlacedNode,
    options: PlexOptions,
    limits: Limits,
    width: Widths,
  ): PlacedNode[]
}

/** Parents above, children below, jumps left, siblings right. */
export const rowsAndColumns: Placement = {
  name: 'rows-and-columns',

  place(seating, focus, options, limits, width) {
    const placed: PlacedNode[] = [focus]

    // Rows first. A row of children is as wide as the plex gets, so a column
    // at a fixed distance would sit on top of it once the row outgrew that
    // distance. The sideways seats give way, because moving parents and
    // children would move the axis the plex is read on.
    const seated = Object.entries(seating) as [PlexRelatedSeat, readonly PlexNode[]][]
    const rows = seated.filter(([seat]) => isVertical(options.direction[seat]))
    const columns = seated.filter(([seat]) => !isVertical(options.direction[seat]))

    for (const [seat, nodes] of rows) {
      placed.push(
        ...line(
          nodes,
          options.direction[seat],
          options,
          limits[seat],
          width,
          focus.height / 2,
        ),
      )
    }

    // One clearance for every column, measured against the rows alone.
    //
    // Against the rows, because a column on one side is not something the
    // column on the other side has to clear. One clearance, because a short
    // column worked out on its own reach would sit nearer the focus than a
    // tall one, and the plex would be lopsided for a reason no reader could
    // see — the two sides are read as a pair.
    const rowsPlaced = [...placed]
    const reach = Math.max(
      0,
      ...columns.map(([seat, nodes]) => reachOf(nodes.length, options, limits[seat])),
    )
    const clearance = widestWithin(rowsPlaced, reach)

    for (const [seat, nodes] of columns) {
      placed.push(
        ...line(nodes, options.direction[seat], options, limits[seat], width, clearance),
      )
    }

    return placed.slice(1)
  },
}

/**
 * One seat, in lines running away from the focus. `clearance` is what the
 * first line has to clear.
 */
function line(
  nodes: readonly PlexNode[],
  direction: Direction,
  options: PlexOptions,
  limits: RoleLimits,
  width: Widths,
  clearance: number,
): PlacedNode[] {
  const sign = direction === 'up' || direction === 'left' ? -1 : 1
  const near = clearance + options.focusGap
  return isVertical(direction)
    ? inRows(nodes, sign, near, options, limits, width)
    : inColumns(nodes, sign, near, options, limits, width)
}

/**
 * A row is as long as its boxes and the gaps between them, and is centred on
 * the focus. Heights are uniform, so each line stands one box beyond the last.
 */
function inRows(
  nodes: readonly PlexNode[],
  sign: number,
  near: number,
  options: PlexOptions,
  limits: RoleLimits,
  width: Widths,
): PlacedNode[] {
  const { height } = options.nodeSize
  const placed: PlacedNode[] = []

  for (const { ofLine, depth, first } of lines(nodes, limits.perLine)) {
    const widths = ofLine.map(width)
    const length = widths.reduce((sum, each) => sum + each + options.gap, -options.gap)
    const away = sign * (near + depth * (height + options.lineGap) + height / 2)

    let edge = -length / 2
    ofLine.forEach((node, index) => {
      const box = widths[index]!
      placed.push({
        ...node,
        x: edge + box / 2,
        y: away,
        width: box,
        height,
        order: first + index,
        opacity: 1,
      })
      edge += box + options.gap
    })
  }

  return placed
}

/**
 * Every box of a column turns the same edge towards the focus, whatever it
 * measures. A line stands clear of the widest box of the line before it.
 */
function inColumns(
  nodes: readonly PlexNode[],
  sign: number,
  near: number,
  options: PlexOptions,
  limits: RoleLimits,
  width: Widths,
): PlacedNode[] {
  const { height } = options.nodeSize
  const step = height + options.gap
  const placed: PlacedNode[] = []
  let edge = near

  for (const { ofLine, first } of lines(nodes, limits.perLine)) {
    const widths = ofLine.map(width)

    ofLine.forEach((node, index) => {
      const box = widths[index]!
      placed.push({
        ...node,
        x: sign * (edge + box / 2),
        y: (index - (ofLine.length - 1) / 2) * step,
        width: box,
        height,
        order: first + index,
        opacity: 1,
      })
    })

    edge += Math.max(...widths) + options.lineGap
  }

  return placed
}

/**
 * The nodes of a seat cut into lines, nearest the focus first. `depth` counts
 * the lines out from the focus and `first` the nodes before.
 *
 * `perLine` is the longest a line may be, and the lines a seat runs to share
 * its nodes out evenly: every line is within one node of every other.
 */
function* lines(
  nodes: readonly PlexNode[],
  perLine: number,
): Generator<{ ofLine: readonly PlexNode[]; depth: number; first: number }> {
  const count = Math.ceil(nodes.length / perLine)
  const each = Math.ceil(nodes.length / count)

  for (let first = 0; first < nodes.length; first += each) {
    yield {
      ofLine: nodes.slice(first, first + each),
      depth: first / each,
      first,
    }
  }
}

/** How far above and below the focus a column of `count` nodes reaches. */
function reachOf(count: number, options: PlexOptions, limits: RoleLimits): number {
  const perColumn = Math.min(limits.perLine, count)
  const step = options.nodeSize.height + options.gap
  return ((perColumn - 1) / 2) * step + options.nodeSize.height / 2
}

/** How far the placed nodes within `reach` of the axis extend sideways. */
function widestWithin(nodes: readonly PlacedNode[], reach: number): number {
  let half = 0
  for (const node of nodes) {
    if (Math.abs(node.y) - node.height / 2 >= reach) continue
    half = Math.max(half, Math.abs(node.x) + node.width / 2)
  }
  return half
}
