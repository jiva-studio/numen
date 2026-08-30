/**
 * What a line of typing comes to, without a field to type it into.
 *
 * The negatives matter most: a word is no number, and neither is a line that
 * only looks like one until the end of it.
 */
import { describe, expect, it } from 'vitest'
import { clamped, numberOf, onItsWay, stepped, written, type Bounds } from './number'

const BOUNDS: Bounds = { min: 0, max: 240, step: 5 }

describe('what the text comes to', () => {
  it('reads a number out of the digits', () => {
    expect(numberOf('20')).toBe(20)
    expect(numberOf(' 20 ')).toBe(20)
    expect(numberOf('-3')).toBe(-3)
    expect(numberOf('0.5')).toBe(0.5)
  })

  it('reads a comma as the point, wherever it is typed that way', () => {
    expect(numberOf('0,5')).toBe(0.5)
  })

  it('comes to nothing where the text is no number', () => {
    expect(numberOf('')).toBeNull()
    expect(numberOf('   ')).toBeNull()
    expect(numberOf('twenty')).toBeNull()
    expect(numberOf('12x')).toBeNull()
    expect(numberOf('1 2')).toBeNull()
  })

  it('refuses the words that read as numbers to a machine and to nobody else', () => {
    expect(numberOf('Infinity')).toBeNull()
    expect(numberOf('0x10')).toBeNull()
    expect(numberOf('1e3')).toBeNull()
  })
})

describe('text a number could still be typed out of', () => {
  it('lets a half-typed number stand', () => {
    for (const said of ['', '-', '1', '1.', '1,', '.5', '12.30']) {
      expect(onItsWay(said)).toBe(true)
    }
  })

  it('says so where nothing further would make a number of it', () => {
    for (const said of ['x', '12x', '1.2.3', '1 2', '--1']) {
      expect(onItsWay(said)).toBe(false)
    }
  })
})

describe('the bounds', () => {
  it('brings a number inside them', () => {
    expect(clamped(-4, BOUNDS)).toBe(0)
    expect(clamped(900, BOUNDS)).toBe(240)
    expect(clamped(20, BOUNDS)).toBe(20)
  })

  it('moves a number one step, and stops at them', () => {
    expect(stepped(20, 1, BOUNDS)).toBe(25)
    expect(stepped(20, -1, BOUNDS)).toBe(15)
    expect(stepped(2, -1, BOUNDS)).toBe(0)
    expect(stepped(238, 1, BOUNDS)).toBe(240)
  })

  it('starts at the floor where no number stands', () => {
    expect(stepped(null, 1, BOUNDS)).toBe(5)
    expect(stepped(null, -1, BOUNDS)).toBe(0)
  })
})

describe('how a number is written back', () => {
  it('writes the number, and nothing where there is none', () => {
    expect(written(20)).toBe('20')
    expect(written(0)).toBe('0')
    expect(written(null)).toBe('')
  })
})
