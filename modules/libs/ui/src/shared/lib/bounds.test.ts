/**
 * Where the ends leave a value, and where a step leaves it. Neither answer
 * touches a track.
 */
import { describe, expect, it } from 'vitest'
import { clamp, getDecimalPlaces, roundToPlaces, stepBy, type Bounds } from './bounds'

const bounds = (over: Partial<Bounds> = {}): Bounds => ({ min: 0, max: 100, step: 1, ...over })

describe('the ends of the track', () => {
  it('hold a value standing outside them', () => {
    expect(clamp(90, bounds({ max: 50 }))).toBe(50)
    expect(clamp(-20, bounds())).toBe(0)
    expect(clamp(40, bounds())).toBe(40)
  })
})

describe('a step', () => {
  it('moves to the next place the step lays', () => {
    const three = bounds({ max: 10, step: 3 })
    expect(stepBy(0, 1, three)).toBe(3)
    expect(stepBy(6, 1, three)).toBe(9)
    expect(stepBy(6, -1, three)).toBe(3)
  })

  // One step up and one step down come to where they began, so a walk along
  // the track and back leaves the handle where it set out.
  it('gives back at the ceiling what it took, and at the floor', () => {
    const three = bounds({ max: 10, step: 3 })
    expect(stepBy(9, 1, three)).toBe(10)
    expect(stepBy(10, -1, three)).toBe(9)
    expect(stepBy(0, -1, three)).toBe(0)
    expect(stepBy(1, 1, bounds({ min: 1, max: 10, step: 3 }))).toBe(4)
  })

  it('draws a value between two places onto the one it is heading for', () => {
    const three = bounds({ max: 10, step: 3 })
    expect(stepBy(7, 1, three)).toBe(9)
    expect(stepBy(7, -1, three)).toBe(6)
  })

  it('stops at the ends', () => {
    expect(stepBy(100, 1, bounds())).toBe(100)
    expect(stepBy(0, -1, bounds())).toBe(0)
  })

  it('leaves a value written to the places its step is written to', () => {
    const fine = bounds({ min: 0.7, max: 0.99, step: 0.01 })
    expect(stepBy(0.81, 1, fine)).toBe(0.82)
    expect(stepBy(0.82, -1, fine)).toBe(0.81)
  })

  it('leaves a track of no step at its current position', () => {
    expect(stepBy(40, 1, bounds({ step: 0 }))).toBe(40)
  })
})

describe('a step far from the floor', () => {
  it('leaves a whole-numbered track on whole numbers', () => {
    const largeBounds = bounds({ max: 1e17, step: 10 })
    expect(stepBy(1e15, 1, largeBounds)).toBe(1000000000000010)
    expect(stepBy(1e15, -1, largeBounds)).toBe(999999999999990)
  })

  it('leaves a track of hundredths on the hundredths its step is written to', () => {
    expect(stepBy(1e14, 10, bounds({ max: 1e17, step: 0.01 }))).toBe(100000000000000.1)
  })

  it('relies on the place a rounding error falls short of', () => {
    const tenths = bounds({ max: 1, step: 0.1 })
    expect(stepBy(0.3, 1, tenths)).toBe(0.4)
    expect(stepBy(0.3, -1, tenths)).toBe(0.2)
  })

  it('leaves a value at its current position for a walk of no steps', () => {
    expect(stepBy(7, 0, bounds({ max: 10, step: 3 }))).toBe(7)
  })
})

describe('the places a number is written to', () => {
  it('counts the digits after the point', () => {
    expect(getDecimalPlaces(5)).toBe(0)
    expect(getDecimalPlaces(0.25)).toBe(2)
  })

  it('rounds to so many places', () => {
    expect(roundToPlaces(0.7 + 0.1, 1)).toBe(0.8)
    expect(roundToPlaces(1.26, 0)).toBe(1)
  })
})
