/**
 * Where the ends leave a value and where a key leaves the handle. Neither
 * answer touches a track.
 */
import { describe, expect, it } from 'vitest'
import type { Bounds } from '@/shared/lib/bounds'
import { isWalkingKey, stepForKey } from './track'

const bounds = (overrides: Partial<Bounds> = {}): Bounds => ({ min: 0, max: 100, step: 1, ...overrides })

describe('the keys the handle walks under', () => {
  it('are the arrows, the page keys and the ends', () => {
    expect(isWalkingKey('ArrowLeft')).toBe(true)
    expect(isWalkingKey('PageUp')).toBe(true)
    expect(isWalkingKey('End')).toBe(true)
    expect(isWalkingKey('a')).toBe(false)
    expect(isWalkingKey('Tab')).toBe(false)
  })

  it('move a step, and the ends take the handle to the ends', () => {
    expect(stepForKey('ArrowRight', 40, { bounds: bounds(), isBigStep: false })).toBe(41)
    expect(stepForKey('ArrowUp', 40, { bounds: bounds(), isBigStep: false })).toBe(41)
    expect(stepForKey('ArrowLeft', 40, { bounds: bounds(), isBigStep: false })).toBe(39)
    expect(stepForKey('ArrowDown', 40, { bounds: bounds(), isBigStep: false })).toBe(39)
    expect(stepForKey('Home', 40, { bounds: bounds(), isBigStep: false })).toBe(0)
    expect(stepForKey('End', 40, { bounds: bounds(), isBigStep: false })).toBe(100)
  })

  it('cover ten steps under a page key, and under a key retained with shift', () => {
    expect(stepForKey('PageUp', 40, { bounds: bounds(), isBigStep: false })).toBe(50)
    expect(stepForKey('PageDown', 40, { bounds: bounds(), isBigStep: false })).toBe(30)
    expect(stepForKey('ArrowRight', 40, { bounds: bounds(), isBigStep: true })).toBe(50)
    expect(stepForKey('ArrowLeft', 40, { bounds: bounds(), isBigStep: true })).toBe(30)
  })

  it('leave a key the handle does not answer to with nothing to say', () => {
    expect(stepForKey('a', 40, { bounds: bounds(), isBigStep: false })).toBeNull()
    expect(stepForKey('Enter', 40, { bounds: bounds(), isBigStep: false })).toBeNull()
  })
})
