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

/** Float error forgiven when boxes touch and no gap is left to spend. */
const FLOAT_ERROR = 1e-6

/** How many quarter-box steps the walk outward takes before it gives up. */
const WALK_STEPS = 400

/** Costs are compared to this many parts of a box. */
const COST_PRECISION = 1e6

/** The offsets, in cells across the direction, tried at each step of the walk. */
const WALK_OFFSETS = [0, 1, -1, 2, -2, 3, -3] as const

/** The side of a grid cell in the index. */
const CELL = 64
const CELL_OFFSET = 1 << 20
const CELL_STRIDE = 1 << 21

interface Rect {
  readonly minX: number
  readonly maxX: number
  readonly minY: number
  readonly maxY: number
}

/** Placed boxes, bucketed on a grid, so a spot is checked against the boxes near it and not against all of them. */
export class SpotIndex {
  private readonly cells = new Map<number, Spot[]>()
  private extent: Rect = { minX: 0, maxX: 0, minY: 0, maxY: 0 }

  constructor(spots: readonly Spot[] = []) {
    for (const spot of spots) this.add(spot)
  }

  /** The bounds of every box in the index. */
  getExtent(): Rect {
    return this.extent
  }

  add(spot: Spot): void {
    this.extent = {
      minX: Math.min(this.extent.minX, spot.x - spot.width / 2),
      maxX: Math.max(this.extent.maxX, spot.x + spot.width / 2),
      minY: Math.min(this.extent.minY, spot.y - spot.height / 2),
      maxY: Math.max(this.extent.maxY, spot.y + spot.height / 2),
    }
    this.visitCells(
      {
        minX: spot.x - spot.width / 2,
        maxX: spot.x + spot.width / 2,
        minY: spot.y - spot.height / 2,
        maxY: spot.y + spot.height / 2,
      },
      (cell) => {
        const bucket = this.cells.get(cell)
        if (bucket) bucket.push(spot)
        else this.cells.set(cell, [spot])
        return false
      },
    )
  }

  /** Whether the spot keeps the gaps from every box in the index. */
  isFree(spot: Spot, gaps: Gaps): boolean {
    let free = true
    this.visitCells(
      {
        minX: spot.x - spot.width / 2 - gaps.gap,
        maxX: spot.x + spot.width / 2 + gaps.gap,
        minY: spot.y - spot.height / 2 - gaps.lineGap,
        maxY: spot.y + spot.height / 2 + gaps.lineGap,
      },
      (cell) => {
        for (const other of this.cells.get(cell) ?? []) {
          if (!isClear(spot, other, gaps)) {
            free = false
            return true
          }
        }
        return false
      },
    )
    return free
  }

  /** Calls `visit` for each cell the rectangle touches, until it returns true. */
  private visitCells(rect: Rect, visit: (cell: number) => boolean): void {
    const toX = Math.floor(rect.maxX / CELL)
    const fromY = Math.floor(rect.minY / CELL)
    const toY = Math.floor(rect.maxY / CELL)
    for (let x = Math.floor(rect.minX / CELL); x <= toX; x++) {
      for (let y = fromY; y <= toY; y++) {
        if (visit((x + CELL_OFFSET) * CELL_STRIDE + y + CELL_OFFSET)) return
      }
    }
  }
}

/** Whether a candidate beats the best so far: a lower cost, or an equal cost nearer the start. */
function isBetter(cost: number, along: number, best: { cost: number; along: number }): boolean {
  return cost < best.cost || (cost === best.cost && roundAlong(along) < roundAlong(best.along))
}

/** Rounded like the cost, so float noise does not decide which of two candidates is nearer the start. */
function roundAlong(along: number): number {
  return Math.round(along * COST_PRECISION)
}

/**
 * The gap that must be kept: the gap less the tolerance, never below nothing.
 * With no gap boxes may touch but not cross; only float error is forgiven.
 */
function getRequiredGap(gap: number): number {
  return gap > TOLERANCE ? gap - TOLERANCE : -FLOAT_ERROR
}

function isClear(spot: Spot, other: Spot, gaps: Gaps): boolean {
  return (
    Math.abs(spot.x - other.x) >= (spot.width + other.width) / 2 + getRequiredGap(gaps.gap) ||
    Math.abs(spot.y - other.y) >=
      (spot.height + other.height) / 2 + getRequiredGap(gaps.lineGap)
  )
}

export function isFree(spot: Spot, taken: readonly Spot[], gaps: Gaps): boolean {
  return taken.every((other) => isClear(spot, other, gaps))
}

/**
 * The free spot of least cost: the nearest to the middle, with costs rounded
 * before they are compared so floating-point error does not decide a tie. Of
 * equal costs the one nearer the start wins, which fills a group across its
 * axis before it goes deeper; of those the earlier candidate wins.
 */
export function findFreeSpot(taken: readonly Spot[] | SpotIndex, request: SpotRequest): Spot {
  const { middle, size, gaps } = request
  const index = taken instanceof SpotIndex ? taken : new SpotIndex(taken)
  const toSpot = (point: Vector): Spot => ({ x: point.x, y: point.y, ...size })

  let best: { point: Vector; cost: number; along: number } | undefined
  for (const point of listCandidates(request)) {
    const along = measureAlong(point, request)
    if (along < -TOLERANCE) continue
    const cost = Math.round(measureCost(point, request) * COST_PRECISION)
    if (best && !isBetter(cost, along, best)) continue
    if (index.isFree(toSpot(point), gaps)) best = { point, cost, along }
  }
  if (best) return toSpot(best.point)
  return findFreeSpotOutward(index, request) ?? findSpotBeyond(index, request)
}

/** Nothing was free in reach: a spot past every placed box, far out, where a window drops it. */
function findSpotBeyond(taken: SpotIndex, request: SpotRequest): Spot {
  const { middle, direction, size, gaps } = request
  const { minX, maxX, minY, maxY } = taken.getExtent()
  if (Math.abs(direction.x) >= Math.abs(direction.y)) {
    const x =
      direction.x >= 0
        ? maxX + gaps.gap + size.width / 2 + 1
        : minX - gaps.gap - size.width / 2 - 1
    return { x, y: middle.y, ...size }
  }
  const y =
    direction.y >= 0
      ? maxY + gaps.lineGap + size.height / 2 + 1
      : minY - gaps.lineGap - size.height / 2 - 1
  return { x: middle.x, y, ...size }
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

/** How far a point is from the start, along the direction. Below nothing it is nearer the focus than the start. */
function measureAlong(point: Vector, request: SpotRequest): number {
  const { start, direction } = request
  return (point.x - start.x) * direction.x + (point.y - start.y) * direction.y
}

/** The distance to the middle counted in boxes. */
function measureCost(point: Vector, request: SpotRequest): number {
  const { middle, size, gaps } = request
  return Math.hypot(
    (point.x - middle.x) / (size.width + gaps.gap),
    (point.y - middle.y) / (size.height + gaps.lineGap),
  )
}

/** Nothing beside the group is free: walk outward from the middle until something is. */
function findFreeSpotOutward(taken: SpotIndex, request: SpotRequest): Spot | undefined {
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
      if (taken.isFree(spot, gaps)) return spot
    }
  }
  return undefined
}
