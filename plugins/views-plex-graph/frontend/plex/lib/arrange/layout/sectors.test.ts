/** The cells of the grid around the focus that a sector takes. */
import { describe, expect, it } from 'vitest'
import { isInSector, listSectorCells, type GridRequest } from './sectors'

const STEP = { width: 160, height: 54 }
const FIRST_ROW = 93
const sector = { relation: 'child' as const, from: 30, to: 150 }

const request: GridRequest = {
  sector,
  step: STEP,
  firstRow: FIRST_ROW,
  count: 20,
  bounds: { width: 600, height: 400 },
  isUsable: () => true,
}

const isWhole = (value: number): boolean => Math.abs(value - Math.round(value)) < 1e-9

describe('the cells of a sector', () => {
  const cells = listSectorCells(request)

  it('point into the sector from the focus', () => {
    expect(cells.length).toBeGreaterThan(0)
    for (const cell of cells)
      expect(isInSector((Math.atan2(cell.y, cell.x) * 180) / Math.PI, sector)).toBe(true)
  })

  it('stand in columns a step apart and in rows a step apart from the first row', () => {
    for (const cell of cells) {
      expect(isWhole(cell.x / STEP.width), `${cell.x}`).toBe(true)
      expect(isWhole((Math.abs(cell.y) - FIRST_ROW) / STEP.height), `${cell.y}`).toBe(true)
    }
  })

  it('come nearest first', () => {
    cells.slice(1).forEach((cell, index) => {
      expect(cell.distance).toBeGreaterThanOrEqual(cells[index]!.distance)
    })
    expect(cells[0]).toMatchObject({ x: 0, y: FIRST_ROW })
  })

  it('keep inside the bounds and leave out the cells that cannot take a note', () => {
    const usable = listSectorCells({ ...request, isUsable: (cell) => cell.x !== 0 })

    for (const cell of usable) {
      expect(cell.x).not.toBe(0)
      expect(Math.abs(cell.x)).toBeLessThanOrEqual(600)
      expect(Math.abs(cell.y)).toBeLessThanOrEqual(400)
    }
  })

  it('without bounds grow until they are as many as the sector needs', () => {
    const unbounded = listSectorCells({ ...request, bounds: undefined, count: 200 })

    expect(unbounded.length).toBeGreaterThanOrEqual(200)
  })

  it('put a row of the focus beside it, and rows between it and the first row only a step from both', () => {
    const wide = listSectorCells({
      ...request,
      sector: { relation: 'jump', from: 150, to: 210 },
      firstRow: 200,
    })
    const rows = new Set(wide.map((cell) => Math.abs(cell.y)))

    expect(rows.has(0)).toBe(true)
    for (const y of rows) expect(y === 0 || y >= STEP.height).toBe(true)
    expect(rows.has(200 - 2 * STEP.height)).toBe(true)
  })
})
