/** Placed boxes on a grid, so a spot is checked against the boxes near it and not against all of them. */
import type { Size } from '@numen/ui'
import type { Position } from '../../node'

export type Vector = Position

/** An upright box: its center and its size. */
export interface Spot extends Vector, Size {}

export interface Gaps {
  readonly gap: number
  readonly lineGap: number
}

/**
 * A spot built exactly one gap from a neighbor is free. Without a tolerance,
 * floating-point error makes such a box count as overlapping.
 */
const TOLERANCE = 0.5

/** The side of a grid cell in the index. */
const CELL = 64
const CELL_OFFSET = 1 << 20
const CELL_STRIDE = 1 << 21

export interface Rect {
  readonly minX: number
  readonly maxX: number
  readonly minY: number
  readonly maxY: number
}

/** Placed boxes, bucketed on a grid, so a spot is checked against the boxes near it and not against all of them. */
export class SpotIndex {
  private readonly cells = new Map<number, Spot[]>()
  private readonly spots: Spot[] = []
  private extent: Rect = { minX: 0, maxX: 0, minY: 0, maxY: 0 }

  constructor(spots: readonly Spot[] = []) {
    for (const spot of spots) this.add(spot)
  }

  /** Every box in the index, in the order it was added. */
  list(): readonly Spot[] {
    return this.spots
  }

  /** The bounds of every box in the index. */
  getExtent(): Rect {
    return this.extent
  }

  add(spot: Spot): void {
    this.spots.push(spot)
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

  /** The boxes in the cells the rectangle touches, so a few more than the ones it overlaps. */
  listNear(rect: Rect): Spot[] {
    const near = new Set<Spot>()
    this.visitCells(rect, (cell) => {
      for (const spot of this.cells.get(cell) ?? []) near.add(spot)
      return false
    })
    return [...near]
  }

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

function getRequiredGap(gap: number): number {
  return Math.max(0, gap - TOLERANCE)
}

/** Whether two boxes keep the gaps from each other. */
export function isClear(spot: Spot, other: Spot, gaps: Gaps): boolean {
  return (
    Math.abs(spot.x - other.x) >= (spot.width + other.width) / 2 + getRequiredGap(gaps.gap) ||
    Math.abs(spot.y - other.y) >= (spot.height + other.height) / 2 + getRequiredGap(gaps.lineGap)
  )
}
