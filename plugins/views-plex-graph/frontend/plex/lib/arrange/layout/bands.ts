/** How wide the middle band is and when its border moves: what the attempts of the clusters placement are compared by. */
import { RELATION_NAMES } from '../../relation'
import type { PlacedNode } from '../../node'
import type { PlexOptions } from '../options'
import type { Spot } from './freeSpot'
import { getFootprint, type NodesByRelation, type PlacementContext } from './placement'
import type { PlexNode, PlexRelation } from '../../../../types'

/** How often the borders move before the placement settles for the best one it has. */
const MAX_ATTEMPTS = 6

/** One placement at one position of the borders. `failures`: notes that had no place in their territory. `outside`: notes outside the window. */
export interface Attempt {
  readonly placed: ReadonlyMap<string, PlacedNode>
  readonly failed: ReadonlySet<PlexRelation>
  readonly failures: number
  readonly outside: number
  readonly centralReach: number
  readonly isStopped: boolean
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

/** An attempt is better with fewer notes outside the window, and then with fewer notes out of their territory. */
export const isBetter = (attempt: Attempt, best: Attempt): boolean =>
  attempt.outside < best.outside ||
  (attempt.outside === best.outside && attempt.failures < best.failures)

/** What the middle band needs: whether the strips beside it have groups, the width of a cell, and the size of its larger group. */
export interface MiddleBand {
  readonly hasCentral: boolean
  readonly hasSide: boolean
  readonly cellWidth: number
  readonly count: number
}

const isSide = (direction: string): boolean => direction === 'left' || direction === 'right'

export function measureMiddleBand(
  nodesByRelation: NodesByRelation,
  context: PlacementContext,
): MiddleBand {
  const { gap, direction } = context.options
  const widthOf = (node: PlexNode) =>
    getFootprint(
      { width: context.width(node), height: context.options.nodeSize.height },
      context.room(node),
    ).width
  const band = { hasCentral: false, hasSide: false, cellWidth: 0, count: 0 }
  for (const relation of RELATION_NAMES) {
    const members = nodesByRelation[relation] ?? []
    if (members.length === 0) continue
    if (isSide(direction[relation])) {
      band.hasSide = true
      continue
    }
    band.hasCentral = true
    band.count = Math.max(band.count, members.length)
    band.cellWidth = Math.max(band.cellWidth, ...members.map((node) => widthOf(node) + gap))
  }
  return band
}

/** The width, in columns, to try next: the band grows when a middle group had no room, and gives room back when a strip had none and the band has some to spare. */
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
): Attempt {
  let columns = Math.max(1, Math.ceil(Math.sqrt(band.count)))
  const tried = new Set([columns])
  let best = place(columns)
  let latest = best
  while (tried.size < MAX_ATTEMPTS && latest.failures > 0 && !latest.isStopped) {
    const next = getNextColumns(latest, columns, band, context)
    if (next === undefined || tried.has(next)) break
    columns = next
    tried.add(columns)
    latest = place(columns)
    if (isBetter(latest, best)) best = latest
  }
  return best
}
