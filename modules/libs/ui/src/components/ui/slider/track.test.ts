/**
 * Where the ends leave a value and where a key leaves the handle. Neither
 * answer touches a track.
 */
import { describe, expect, it } from 'vitest'
import { clamped, stepped, walked, walks, type Bounds } from './track'

const bounds = (over: Partial<Bounds> = {}): Bounds => ({ min: 0, max: 100, step: 1, ...over })

describe('the ends of the track', () => {
  it('hold a value standing outside them', () => {
    expect(clamped(90, bounds({ max: 50 }))).toBe(50)
    expect(clamped(-20, bounds())).toBe(0)
    expect(clamped(40, bounds())).toBe(40)
  })
})

describe('a step', () => {
  it('moves to the next place the step lays', () => {
    const three = bounds({ max: 10, step: 3 })
    expect(stepped(0, 1, three)).toBe(3)
    expect(stepped(6, 1, three)).toBe(9)
    expect(stepped(6, -1, three)).toBe(3)
  })

  // One step up and one step down come to where they began, so a walk along
  // the track and back leaves the handle where it set out.
  it('gives back at the ceiling what it took, and at the floor', () => {
    const three = bounds({ max: 10, step: 3 })
    expect(stepped(9, 1, three)).toBe(10)
    expect(stepped(10, -1, three)).toBe(9)
    expect(stepped(0, -1, three)).toBe(0)
    expect(stepped(1, 1, bounds({ min: 1, max: 10, step: 3 }))).toBe(4)
  })

  it('draws a value between two places onto the one it is heading for', () => {
    const three = bounds({ max: 10, step: 3 })
    expect(stepped(7, 1, three)).toBe(9)
    expect(stepped(7, -1, three)).toBe(6)
  })

  it('stops at the ends', () => {
    expect(stepped(100, 1, bounds())).toBe(100)
    expect(stepped(0, -1, bounds())).toBe(0)
  })

  it('leaves a value written to the places its step is written to', () => {
    const fine = bounds({ min: 0.7, max: 0.99, step: 0.01 })
    expect(stepped(0.81, 1, fine)).toBe(0.82)
    expect(stepped(0.82, -1, fine)).toBe(0.81)
  })

  it('leaves a track of no step where it stands', () => {
    expect(stepped(40, 1, bounds({ step: 0 }))).toBe(40)
  })
})

describe('the keys the handle walks under', () => {
  it('are the arrows, the page keys and the ends', () => {
    expect(walks('ArrowLeft')).toBe(true)
    expect(walks('PageUp')).toBe(true)
    expect(walks('End')).toBe(true)
    expect(walks('a')).toBe(false)
    expect(walks('Tab')).toBe(false)
  })

  it('move a step, and the ends take the handle to the ends', () => {
    expect(walked('ArrowRight', 40, bounds(), false)).toBe(41)
    expect(walked('ArrowUp', 40, bounds(), false)).toBe(41)
    expect(walked('ArrowLeft', 40, bounds(), false)).toBe(39)
    expect(walked('ArrowDown', 40, bounds(), false)).toBe(39)
    expect(walked('Home', 40, bounds(), false)).toBe(0)
    expect(walked('End', 40, bounds(), false)).toBe(100)
  })

  it('cover ten steps under a page key, and under a key held with shift', () => {
    expect(walked('PageUp', 40, bounds(), false)).toBe(50)
    expect(walked('PageDown', 40, bounds(), false)).toBe(30)
    expect(walked('ArrowRight', 40, bounds(), true)).toBe(50)
    expect(walked('ArrowLeft', 40, bounds(), true)).toBe(30)
  })

  it('leave a key the handle does not answer to with nothing to say', () => {
    expect(walked('a', 40, bounds(), false)).toBeNull()
    expect(walked('Enter', 40, bounds(), false)).toBeNull()
  })
})
