/** The curves as they are written down, read back out of the strings. */
import { describe, expect, it } from 'vitest'
import {
  arrowTransformOf,
  pathOf,
  readingPathOf,
  threadOf,
  ARROWHEAD_PATH,
} from './paths'
import { ARROW_LENGTH, type EdgeCurve, type PlacedEdge } from '../model'

const CURVE: EdgeCurve = {
  fromPoint: { x: 0, y: 0 },
  control1: { x: 10, y: 0 },
  control2: { x: 20, y: 30 },
  toPoint: { x: 30, y: 30 },
}

const edge = (heading: PlacedEdge['heading']): PlacedEdge => ({
  ...CURVE,
  from: 'a',
  to: 'b',
  opacity: 1,
  heading,
  wordsAt: 0.5,
})

/** The numbers in the order the string names them. */
const numbersIn = (path: string): number[] =>
  [...path.matchAll(/-?\d+(?:\.\d+)?/g)].map((found) => Number(found[0]))

describe('a curve as a path', () => {
  it('leaves at one end, arrives at the other, and takes both controls on the way', () => {
    expect(numbersIn(pathOf(CURVE))).toEqual([0, 0, 10, 0, 20, 30, 30, 30])
  })

  it('is one move and one cubic', () => {
    expect(pathOf(CURVE)).toMatch(/^M [-\d. ]+ C [-\d. ]+$/)
  })
})

describe('the line a title is set along', () => {
  it('runs the way the curve does where the words read along it', () => {
    expect(readingPathOf(edge('along'))).toBe(pathOf(CURVE))
  })

  it('runs the other way where the words would be upside down', () => {
    expect(numbersIn(readingPathOf(edge('against')))).toEqual([30, 30, 20, 30, 10, 0, 0, 0])
  })

  it('ends where the curve begins where it is taken the other way', () => {
    const reversed = numbersIn(readingPathOf(edge('against')))
    expect(reversed.slice(-2)).toEqual([CURVE.fromPoint.x, CURVE.fromPoint.y])
  })
})

describe('the arrowhead', () => {
  it('is drawn about its own tip, reaching back along the x axis', () => {
    expect(numbersIn(ARROWHEAD_PATH)).toEqual([0, 0, -ARROW_LENGTH, 5.5, -ARROW_LENGTH, -5.5])
  })

  it('is closed', () => {
    expect(ARROWHEAD_PATH.endsWith('Z')).toBe(true)
  })

  it('is moved onto its point and turned along the line there', () => {
    expect(arrowTransformOf({ at: { x: 7, y: -3 }, angle: 45 })).toBe(
      'translate(7 -3) rotate(45)',
    )
  })
})

describe('a thread between two loose points', () => {
  it('leaves the first and arrives at the second', () => {
    const numbers = numbersIn(threadOf({ x: 0, y: 0 }, { x: 100, y: 40 }))
    expect(numbers.slice(0, 2)).toEqual([0, 0])
    expect(numbers.slice(-2)).toEqual([100, 40])
  })

  it('leaves and arrives square-on, each control level with its own end', () => {
    const [, , , firstY, , secondY] = numbersIn(threadOf({ x: 0, y: 0 }, { x: 100, y: 40 }))
    expect(firstY).toBe(0)
    expect(secondY).toBe(40)
  })

  it('reaches half the sideways distance from each end', () => {
    const [, , firstX, , secondX] = numbersIn(threadOf({ x: 0, y: 0 }, { x: 100, y: 40 }))
    expect(firstX).toBe(50)
    expect(secondX).toBe(50)
  })

  it('reaches the same way for a point behind it as for one ahead', () => {
    const [, , firstX] = numbersIn(threadOf({ x: 100, y: 0 }, { x: 0, y: 40 }))
    expect(firstX).toBe(150)
  })

  it('is straight between two points standing one above the other', () => {
    const [, , firstX, , secondX] = numbersIn(threadOf({ x: 20, y: 0 }, { x: 20, y: 60 }))
    expect(firstX).toBe(20)
    expect(secondX).toBe(20)
  })
})
