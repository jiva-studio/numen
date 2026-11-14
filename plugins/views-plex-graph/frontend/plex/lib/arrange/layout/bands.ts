/** How wide the middle band is and when its border moves: what the attempts of the clusters placement are compared by. */
import { RELATION_NAMES } from '../../relation'
import type { PlacedNode } from '../../node'
import type { PlexOptions } from '../options'
import type { Spot } from './freeSpot'
import { GROUP_GAP_FACTOR } from './territory'
import { getFootprint, type NodesByRelation, type PlacementContext } from './placement'
import type { PlexNode, PlexRelation } from '../../../../types'

/** How often the borders move before the placement settles for the best one it has. */
const MAX_ATTEMPTS = 6

/** One placement at one position of the borders. `failures`: notes that had no place in their territory. `crossings`: notes set in their territory with a line along another line. `outside`: notes outside the window. */
export interface Attempt {
  readonly placed: ReadonlyMap<string, PlacedNode>
  readonly failed: ReadonlySet<PlexRelation>
  readonly failures: number
  readonly crossings: number
  readonly outside: number
  readonly centralReach: number
}

/** The window is centered on the focus, which is at the origin. */
export function isInWindow(spot: Spot, options: PlexOptions): boolean {
  const { viewport, margin } = options
  return (
    !viewport ||
    (Math.abs(spot.x) + spot.width / 2 <= viewport.width / 2 - margin &&
      Math.abs(spot.y) + spot.height / 2 <= viewport.height / 2 - margin)
  )
}

/** An attempt is better with fewer notes outside the window, then with fewer notes out of their territory, then with fewer lines along another. */
export const isBetter = (attempt: Attempt, best: Attempt): boolean =>
  attempt.outside !== best.outside
    ? attempt.outside < best.outside
    : attempt.failures !== best.failures
      ? attempt.failures < best.failures
      : attempt.crossings < best.crossings

/**
 * What the groups need: whether the strips beside the middle band have groups, the cell of the middle band and the size of its larger group,
 * and the same for the strip with more notes (`side`).
 */
export interface MiddleBand {
  readonly hasCentral: boolean
  readonly hasSide: boolean
  readonly cellWidth: number
  readonly cellHeight: number
  readonly count: number
  readonly side: { readonly cellWidth: number; readonly cellHeight: number; readonly count: number }
}

const isSide = (direction: string): boolean => direction === 'left' || direction === 'right'

export function measureMiddleBand(
  nodesByRelation: NodesByRelation,
  context: PlacementContext,
): MiddleBand {
  const { gap, lineGap, direction } = context.options
  const footprintOf = (node: PlexNode) =>
    getFootprint(
      { width: context.width(node), height: context.options.nodeSize.height },
      context.room(node),
    )
  const band = {
    hasCentral: false,
    hasSide: false,
    cellWidth: 0,
    cellHeight: 0,
    count: 0,
    side: { cellWidth: 0, cellHeight: 0, count: 0 },
  }
  const sideCounts = new Map<string, number>()
  for (const relation of RELATION_NAMES) {
    const members = nodesByRelation[relation] ?? []
    if (members.length === 0) continue
    const cellWidth = Math.max(...members.map((node) => footprintOf(node).width + gap))
    const cellHeight = Math.max(...members.map((node) => footprintOf(node).height + lineGap))
    if (isSide(direction[relation])) {
      band.hasSide = true
      const count = (sideCounts.get(direction[relation]) ?? 0) + members.length
      sideCounts.set(direction[relation], count)
      band.side = {
        cellWidth: Math.max(band.side.cellWidth, cellWidth),
        cellHeight: Math.max(band.side.cellHeight, cellHeight),
        count: Math.max(band.side.count, count),
      }
      continue
    }
    band.hasCentral = true
    band.count = Math.max(band.count, members.length)
    band.cellWidth = Math.max(band.cellWidth, cellWidth)
    band.cellHeight = Math.max(band.cellHeight, cellHeight)
  }
  return band
}

/**
 * The columns the middle band starts with. Without a window it is a compact patch of its larger group. In a window each group needs as many columns as its notes
 * take at the rows the window holds; the band takes what it needs when the strips still get theirs, and otherwise the free width is shared in proportion to the need,
 * with at least one column for the band. What the groups do not need is shared evenly.
 */
export function getStartColumns(
  band: MiddleBand,
  context: PlacementContext,
  focus: { readonly width: number; readonly height: number },
): number {
  const compact = Math.max(1, Math.ceil(Math.sqrt(band.count)))
  const { viewport, margin, focusGap } = context.options
  if (!viewport || !band.hasCentral || !band.hasSide) return compact
  const half = viewport.width / 2 - margin
  const centralRows = Math.floor(
    (viewport.height / 2 - margin - focus.height / 2 - focusGap) / band.cellHeight,
  )
  const sideRows = Math.floor((viewport.height - 2 * margin) / band.side.cellHeight)
  const centralWidth = Math.ceil(band.count / Math.max(1, centralRows)) * band.cellWidth
  const sideWidth = Math.ceil(band.side.count / Math.max(1, sideRows)) * band.side.cellWidth
  const free = half - GROUP_GAP_FACTOR * context.options.gap
  const wanted = centralWidth / 2
  const spare = free - sideWidth - wanted
  // What is left over is shared evenly: the lines of a group need room beyond the notes themselves.
  if (spare >= 0) return Math.max(compact, Math.floor((2 * (wanted + spare / 2)) / band.cellWidth))
  return Math.max(1, Math.floor((2 * ((free * wanted) / (wanted + sideWidth))) / band.cellWidth))
}

function getNextColumns(
  attempt: Attempt,
  columns: number,
  band: MiddleBand,
  context: PlacementContext,
): number | undefined {
  const failedDirections = [...attempt.failed].map(
    (relation) => context.options.direction[relation],
  )
  const isCentralFull = failedDirections.some((direction) => !isSide(direction))
  const isSideFull = failedDirections.some(isSide)
  const hasSpare = attempt.centralReach <= ((columns - 1) * band.cellWidth) / 2
  if (isCentralFull && !isSideFull) {
    return Math.max(columns + 1, Math.ceil((2 * attempt.centralReach) / band.cellWidth))
  }
  return isSideFull && !isCentralFull && columns > 1 && hasSpare ? columns - 1 : undefined
}

/** The middle band starts as wide as a compact patch of its larger group; a group with no room in its territory moves the border and the placement is tried again. */
export function findBestAttempt(
  place: (columns: number) => Attempt,
  band: MiddleBand,
  context: PlacementContext,
  startColumns: number,
): Attempt {
  let columns = startColumns
  const tried = new Set([columns])
  let best = place(columns)
  let latest = best
  while (tried.size < MAX_ATTEMPTS && latest.failed.size > 0) {
    const next = getNextColumns(latest, columns, band, context)
    if (next === undefined || tried.has(next)) break
    columns = next
    tried.add(columns)
    latest = place(columns)
    if (isBetter(latest, best)) best = latest
  }
  return best
}
