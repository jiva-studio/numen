import type { PlacedNode, PlexNode, PlexRelatedSeat } from '../model'
import type { Limits, RoleLimits } from './limits'
import { isVertical, type Direction, type PlexOptions } from './options'

/** The nodes admitted to the picture, grouped by the seat they take. */
export type Seating = Readonly<Partial<Record<PlexRelatedSeat, readonly PlexNode[]>>>

/**
 * Where the nodes go. A strategy, so a radial mind map is another
 * implementation rather than a branch inside this one.
 *
 * It decides coordinates and nothing else: how much is admitted is settled
 * before it is called, and routing follows from where the boxes ended up.
 */
export interface Placement {
  readonly name: string
  /** The focus is already at the origin; it is passed in to be cleared. */
  place(
    seating: Seating,
    focus: PlacedNode,
    options: PlexOptions,
    limits: Limits,
  ): PlacedNode[]
}

/** Parents above, children below, jumps left, siblings right. */
export const rowsAndColumns: Placement = {
  name: 'rows-and-columns',

  place(seating, focus, options, limits) {
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
        ...line(nodes, options.direction[seat], options, limits[seat], focus.height / 2),
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
        ...line(nodes, options.direction[seat], options, limits[seat], clearance),
      )
    }

    return placed.slice(1)
  },
}

/**
 * One seat, in lines running away from the focus, each line centred on the
 * focus axis. `clearance` is what the first line has to clear.
 */
function line(
  nodes: readonly PlexNode[],
  direction: Direction,
  options: PlexOptions,
  limits: RoleLimits,
  clearance: number,
): PlacedNode[] {
  const { width, height } = options.nodeSize
  const vertical = isVertical(direction)
  const sign = direction === 'up' || direction === 'left' ? -1 : 1

  const alongStep = (vertical ? width : height) + options.gap
  const awayStep = (vertical ? height : width) + options.lineGap
  const firstOffset =
    clearance + options.focusGap + (vertical ? height : width) / 2

  return nodes.map((node, index) => {
    const row = Math.floor(index / limits.perLine)
    const withinRow = index % limits.perLine
    const rowLength = Math.min(limits.perLine, nodes.length - row * limits.perLine)

    const along = (withinRow - (rowLength - 1) / 2) * alongStep
    const away = sign * (firstOffset + row * awayStep)

    return {
      ...node,
      x: vertical ? along : away,
      y: vertical ? away : along,
      width,
      height,
      order: index,
      opacity: 1,
    }
  })
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
