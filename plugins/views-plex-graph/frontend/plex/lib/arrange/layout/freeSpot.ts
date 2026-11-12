/** Where a box of a cluster goes: the free spot nearest the middle of its group. */
import type { Size } from '@numen/ui'
import type { Position } from '../../node'

export type Vector = Position

/** An upright box: its center and its size. */
export interface Spot extends Vector, Size {}

export interface Gaps {
  readonly gap: number
  readonly lineGap: number
}

export interface SpotRequest {
  /** The nearest point the box may take: no member sits nearer the focus. */
  readonly start: Vector
  /** Where the group is centered; the box goes to the free spot nearest it. */
  readonly middle: Vector
  readonly direction: Vector
  readonly size: Size
  /** The boxes of the same group that are already placed. */
  readonly group: readonly Spot[]
  readonly gaps: Gaps
}

/**
 * Candidates are built exactly one gap from a neighbor. Without a tolerance,
 * floating-point error makes such a box count as overlapping, every touching
 * spot is refused and the groups degrade into lines.
 */
export const TOLERANCE = 0.5

/** How many quarter-box steps the walk outward takes before it gives up. */
const WALK_STEPS = 400

/** The offsets, in cells across the direction, tried at each step of the walk. */
const WALK_OFFSETS = [0, 1, -1, 2, -2, 3, -3] as const

export function isFree(spot: Spot, taken: readonly Spot[], gaps: Gaps): boolean {
  return taken.every(
    (other) =>
      Math.abs(spot.x - other.x) >= (spot.width + other.width) / 2 + gaps.gap - TOLERANCE ||
      Math.abs(spot.y - other.y) >= (spot.height + other.height) / 2 + gaps.lineGap - TOLERANCE,
  )
}

export function findFreeSpot(taken: readonly Spot[], request: SpotRequest): Spot {
  const { middle, size, gaps } = request
  const toSpot = (point: Vector): Spot => ({ x: point.x, y: point.y, ...size })

  // Candidates cost the same only in a tie, and a sort that is stable keeps their order.
  const ranked = listCandidates(request)
    .map((point) => ({ point, cost: measureCost(point, request) }))
    .filter((each) => Number.isFinite(each.cost))
    .sort((a, b) => a.cost - b.cost)

  for (const { point } of ranked) {
    const spot = toSpot(point)
    if (isFree(spot, taken, gaps)) return spot
  }
  return findFreeSpotOutward(taken, request) ?? toSpot(middle)
}

function listCandidates(request: SpotRequest): Vector[] {
  const { start, middle, size, group, gaps } = request
  const candidates: Vector[] = [middle, start]

  for (const box of group) {
    const right = box.x + box.width / 2 + gaps.gap + size.width / 2
    const left = box.x - box.width / 2 - gaps.gap - size.width / 2
    const below = box.y + box.height / 2 + gaps.lineGap + size.height / 2
    const above = box.y - box.height / 2 - gaps.lineGap - size.height / 2
    candidates.push(
      { x: right, y: box.y },
      { x: left, y: box.y },
      { x: box.x, y: below },
      { x: box.x, y: above },
      { x: (box.x + right) / 2, y: below },
      { x: (box.x + left) / 2, y: below },
      { x: (box.x + right) / 2, y: above },
      { x: (box.x + left) / 2, y: above },
    )
  }
  return candidates
}

/** The distance to the middle counted in boxes, or infinity for a point nearer the focus than the start. */
function measureCost(point: Vector, request: SpotRequest): number {
  const { start, middle, direction, size, gaps } = request
  const along = (point.x - start.x) * direction.x + (point.y - start.y) * direction.y
  if (along < -TOLERANCE) return Infinity
  return Math.hypot(
    (point.x - middle.x) / (size.width + gaps.gap),
    (point.y - middle.y) / (size.height + gaps.lineGap),
  )
}

/** Nothing beside the group is free: walk outward from the middle until something is. */
function findFreeSpotOutward(taken: readonly Spot[], request: SpotRequest): Spot | undefined {
  const { middle, direction, size, gaps } = request
  const cell = { width: size.width + gaps.gap, height: size.height + gaps.lineGap }
  const stride = size.height / 4

  for (let step = 1; step <= WALK_STEPS; step++) {
    for (const offset of WALK_OFFSETS) {
      const spot: Spot = {
        x: middle.x + direction.x * step * stride - direction.y * offset * cell.width,
        y: middle.y + direction.y * step * stride + direction.x * offset * cell.height,
        ...size,
      }
      if (isFree(spot, taken, gaps)) return spot
    }
  }
  return undefined
}
