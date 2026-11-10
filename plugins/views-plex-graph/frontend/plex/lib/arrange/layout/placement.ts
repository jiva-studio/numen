import type { PlacedNode } from '../../node'
import type { Limits, RoleLimits } from './limits'
import { isVertical, type Direction, type PlexOptions } from '../options'
import type { PlexNode, PlexRelation } from '../../../../types'

/** The nodes included in the graph, grouped by relation. */
export type NodesByRelation = Readonly<Partial<Record<PlexRelation, readonly PlexNode[]>>>

/** How wide a node's box is drawn, clamped before a strategy is called. */
export type Widths = (node: PlexNode) => number

interface PlacementContext {
  readonly options: PlexOptions
  readonly limits: Limits
  readonly width: Widths
}

/**
 * Where the nodes go, as a strategy: a radial mind map is another implementation of this.
 * It decides coordinates only: how much is admitted is settled before the call, box widths come in as numbers, and routing follows from where the boxes ended up.
 */
export interface Placement {
  readonly name: string
  /** The focus is already at the origin; it is passed in to be cleared. */
  place(
    nodesByRelation: NodesByRelation,
    focus: PlacedNode,
    context: PlacementContext,
  ): PlacedNode[]
}

/** Parents above, children below, jumps left, siblings right. */
export const rowsAndColumns: Placement = {
  name: 'rows-and-columns',

  place(nodesByRelation, focus, context) {
    const { options, limits, width } = context
    const placedNodes: PlacedNode[] = [focus]

    // Rows first. A row of children is as wide as the plex gets, so a column
    // at a fixed distance would sit on top of it once the row outgrew that
    // distance. The sideways relations give way, because moving parents and
    // children would move the axis the plex is read on.
    const entries = Object.entries(nodesByRelation) as [PlexRelation, readonly PlexNode[]][]
    const rows = entries.filter(([relation]) => isVertical(options.direction[relation]))
    const columns = entries.filter(([relation]) => !isVertical(options.direction[relation]))

    for (const [relation, nodes] of rows) {
      placedNodes.push(
        ...line(nodes, options.direction[relation], {
          options,
          limits: limits[relation],
          width,
          clearance: focus.height / 2,
        }),
      )
    }

    // One clearance for every column, measured against the rows alone.
    //
    // Against the rows, because a column on one side is not something the
    // column on the other side has to clear. One clearance for both: the two
    // sides are read as a pair.
    const rowsPlaced = [...placedNodes]
    const reach = Math.max(
      0,
      ...columns.map(([relation, nodes]) => getReach(nodes.length, options, limits[relation])),
    )
    const clearance = measureWidestWithin(rowsPlaced, reach)

    for (const [relation, nodes] of columns) {
      placedNodes.push(
        ...line(nodes, options.direction[relation], {
          options,
          limits: limits[relation],
          width,
          clearance,
        }),
      )
    }

    return placedNodes.slice(1)
  },
}

interface LineContext {
  readonly options: PlexOptions
  readonly limits: RoleLimits
  readonly width: Widths
  readonly clearance: number
}

interface AxisPlacement {
  readonly sign: number
  readonly near: number
}

function line(
  nodes: readonly PlexNode[],
  direction: Direction,
  context: LineContext,
): PlacedNode[] {
  const sign = direction === 'up' || direction === 'left' ? -1 : 1
  const nearGap = context.clearance + context.options.focusGap
  const placement: AxisPlacement = { sign, near: nearGap }
  return isVertical(direction)
    ? inRows(nodes, placement, context)
    : inColumns(nodes, placement, context)
}

function inRows(
  nodes: readonly PlexNode[],
  placement: AxisPlacement,
  context: LineContext,
): PlacedNode[] {
  const { sign, near } = placement
  const { options, limits, width } = context
  const { height } = options.nodeSize
  const placedNodes: PlacedNode[] = []

  for (const { ofLine, depth, first } of lines(nodes, limits.perLine)) {
    const widths = ofLine.map(width)
    const length = widths.reduce((sum, each) => sum + each + options.gap, -options.gap)
    const away = sign * (near + depth * (height + options.lineGap) + height / 2)

    let edge = -length / 2
    ofLine.forEach((node, index) => {
      const box = widths[index]!
      placedNodes.push({
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

  return placedNodes
}

function inColumns(
  nodes: readonly PlexNode[],
  placement: AxisPlacement,
  context: LineContext,
): PlacedNode[] {
  const { sign, near } = placement
  const { options, limits, width } = context
  const { height } = options.nodeSize
  const step = height + options.gap
  const placedNodes: PlacedNode[] = []
  let edge = near

  for (const { ofLine, first } of lines(nodes, limits.perLine)) {
    const widths = ofLine.map(width)

    ofLine.forEach((node, index) => {
      const box = widths[index]!
      placedNodes.push({
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

  return placedNodes
}

function lines(
  nodes: readonly PlexNode[],
  perLine: number,
): { ofLine: readonly PlexNode[]; depth: number; first: number }[] {
  const count = Math.ceil(nodes.length / perLine)
  const each = Math.ceil(nodes.length / count)

  return Array.from({ length: count }, (_, depth) => {
    const first = depth * each
    return { ofLine: nodes.slice(first, first + each), depth, first }
  })
}

function getReach(count: number, options: PlexOptions, limits: RoleLimits): number {
  const perColumn = Math.min(limits.perLine, count)
  const step = options.nodeSize.height + options.gap
  return ((perColumn - 1) / 2) * step + options.nodeSize.height / 2
}

function measureWidestWithin(nodes: readonly PlacedNode[], reach: number): number {
  let half = 0
  for (const node of nodes) {
    if (Math.abs(node.y) - node.height / 2 >= reach) continue
    half = Math.max(half, Math.abs(node.x) + node.width / 2)
  }
  return half
}
