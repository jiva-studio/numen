/** The territories of the groups: parents and children share the middle, jumps and siblings keep to the strips beside it, and the border between them moves when a group needs more room. */
import type { PlacedNode } from '../../node'
import type { Spot, Vector } from './freeSpot'

/** A box may cross a border by this much, so a box built exactly on the border is inside. */
const BORDER_TOLERANCE = 0.5

/** `centralHalf`: how far from the center line of the focus the boxes of the middle groups reach. `sideStart`: how far from it the boxes of the strips begin. */
export interface Borders {
  readonly centralHalf: number
  readonly sideStart: number
  /** Whether the groups grow as if there were no territories: a group with no neighbour across the focus has none to keep out of. */
  readonly isOpen: boolean
}

/** The test a box of a group must pass: the middle groups stay inside the central half-width, the side groups stay beyond the start of their strip. */
export function createTerritory(
  side: Vector,
  focus: PlacedNode,
  borders: Borders,
): (spot: Spot) => boolean {
  if (side.x === 0) {
    return (spot) =>
      Math.abs(spot.x - focus.x) + spot.width / 2 <= borders.centralHalf + BORDER_TOLERANCE
  }
  return (spot) =>
    side.x * (spot.x - focus.x) - spot.width / 2 >= borders.sideStart - BORDER_TOLERANCE
}

interface BordersRequest {
  readonly columns: number
  readonly cellWidth: number
  readonly hasCentral: boolean
  readonly hasSide: boolean
  readonly groupGap: number
  readonly focusReach: number
}

/** The borders for a middle band `columns` boxes wide. */
export function createBorders(request: BordersRequest): Borders {
  const half = (request.columns * request.cellWidth) / 2
  return {
    isOpen: !(request.hasCentral && request.hasSide),
    centralHalf: request.hasSide ? half : Number.POSITIVE_INFINITY,
    sideStart: request.hasCentral
      ? Math.max(request.focusReach, half + request.groupGap)
      : request.focusReach,
  }
}
