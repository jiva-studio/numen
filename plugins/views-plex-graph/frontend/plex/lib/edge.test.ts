/** The curve of a line: how it is keyed, which way it is read, and where its arrowhead is aimed. */
import { describe, expect, it } from 'vitest'
import {
  ARROW_LENGTH,
  createPointOnCurve,
  getArrow,
  edgeKey,
  getHeading,
  getLength,
  type EdgeCurve,
} from './edge'

const toDegrees = (radians: number) => (radians * 180) / Math.PI

/** A curve with no sideways run: it goes up and down the page along x = 0. */
const axisColumn = (from: number, control1: number, control2: number, to: number): EdgeCurve => ({
  fromPoint: { x: 0, y: from },
  control1: { x: 0, y: control1 },
  control2: { x: 0, y: control2 },
  toPoint: { x: 0, y: to },
})

/** A curve with no vertical run: it goes across the page along y = 0. */
const axisRow = (from: number, control1: number, control2: number, to: number): EdgeCurve => ({
  fromPoint: { x: from, y: 0 },
  control1: { x: control1, y: 0 },
  control2: { x: control2, y: 0 },
  toPoint: { x: to, y: 0 },
})

describe('the key of an edge', () => {
  it('is the same whichever end is which', () => {
    expect(edgeKey({ from: 'a', to: 'b' })).toBe('a b')
    expect(edgeKey({ from: 'b', to: 'a' })).toBe('a b')
  })
})

describe('the reading direction of a curve with no sideways run', () => {
  // Every curve here has x = 0 throughout, so the tangent has no x component
  // and the heading is read off the vertical one at the middle.
  it('is along where the middle of the curve runs down the page', () => {
    expect(getHeading(axisColumn(0, 100, 40, 140), 0.5)).toBe('along')
    expect(getHeading(axisColumn(300, 280, 310, 290), 0.5)).toBe('along')
  })

  it('is against where the middle of the curve runs up the page', () => {
    expect(getHeading(axisColumn(300, 290, 190, 180), 0.5)).toBe('against')
  })

  it('is along for a straight run down the page and against for one up it', () => {
    expect(getHeading(axisColumn(0, 10, 20, 30), 0.5)).toBe('along')
    expect(getHeading(axisColumn(30, 20, 10, 0), 0.5)).toBe('against')
  })

  it('is along for a curve that never leaves its point', () => {
    expect(getHeading(axisColumn(5, 5, 5, 5), 0.5)).toBe('along')
  })
})

describe('the arrowhead of a curve shorter than the head', () => {
  // A head has no piece of such a curve to be aimed along, so the tangent at
  const short: EdgeCurve = {
    fromPoint: { x: 0, y: 1 },
    control1: { x: 3, y: 3 },
    control2: { x: 5, y: 6 },
    toPoint: { x: 8, y: 8 },
  }

  it('is aimed along the tangent at the end it arrives at', () => {
    expect(getLength(short)).toBeLessThan(ARROW_LENGTH)
    expect(getArrow(short, 'to').angle).toBeCloseTo(toDegrees(Math.atan2(6, 9)), 5)
  })

  it('is aimed back against the tangent at the end it leaves from', () => {
    expect(getArrow(short, 'from').angle).toBeCloseTo(toDegrees(Math.atan2(-6, -9)), 5)
  })

  it('is aimed along the tangent, and not the chord, where the curve turns', () => {
    // Ends at (6, 3) arriving square-on across the page; the chord leans down.
    const turning: EdgeCurve = {
      fromPoint: { x: 0, y: 0 },
      control1: { x: 0, y: 3 },
      control2: { x: 3, y: 3 },
      toPoint: { x: 6, y: 3 },
    }
    expect(getLength(turning)).toBeLessThan(ARROW_LENGTH)
    expect(getArrow(turning, 'to').angle).toBeCloseTo(0, 5)
  })
})

describe('the arrowhead of a curve whose head covers no distance', () => {
  // The head's length is a vanishing part of this curve, so the point behind
  // the head is the end itself and the tangent aims it.
  const vast: EdgeCurve = axisColumn(0, 3e17, 6e17, 1e18)

  it('is aimed along the tangent at the end', () => {
    expect(getLength(vast)).toBeGreaterThan(1e17)
    expect(getArrow(vast, 'to').angle).toBeCloseTo(90, 5)
  })
})

describe('the arrowhead of a curve that turns back within the head', () => {
  // Rises to a peak slightly past its end and returns to it, so the part of the
  // curve under the arrowhead runs in the direction the line had before it turned.
  const turnsBackDown = axisColumn(0, 300, 300, 295)
  const turnsBackAcross = axisRow(0, 300, 300, 295)

  it('is aimed along the piece it covers when that piece runs down the page', () => {
    expect(getLength(turnsBackDown)).toBeGreaterThan(ARROW_LENGTH)
    expect(getArrow(turnsBackDown, 'to').angle).toBeCloseTo(90, 5)
  })

  it('is aimed along the piece it covers when that piece runs across the page', () => {
    expect(getLength(turnsBackAcross)).toBeGreaterThan(ARROW_LENGTH)
    expect(getArrow(turnsBackAcross, 'to').angle).toBeCloseTo(0, 5)
  })
})

describe('a point on a curve whose start does not move', () => {
  // The first steps of this curve are too small to register, so the curve
  // has length while its first chord has none.
  const stalled: EdgeCurve = {
    fromPoint: { x: 0, y: 0 },
    control1: { x: 0, y: 0 },
    control2: { x: 0, y: 0 },
    toPoint: { x: 1e-320, y: 0 },
  }

  it('is the start at the start of the line', () => {
    expect(getLength(stalled)).toBeGreaterThan(0)
    expect(createPointOnCurve(stalled)(0)).toEqual({ x: 0, y: 0 })
  })
})
