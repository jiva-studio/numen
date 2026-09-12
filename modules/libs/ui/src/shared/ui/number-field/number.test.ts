/**
 * What a line of typing comes to, without a field to type it into.
 *
 * The negatives matter most: a word is no number, and neither is a line that
 * only looks like one until the end of it.
 */
import { describe, expect, it } from 'vitest'
import {
  isAllowed,
  clamp,
  numberOf,
  onItsWay,
  snapToBounds,
  isTextForValue,
  stepBy,
  stepForKey,
  written,
  type Bounds,
} from './number'

const BOUNDS: Bounds = { min: 0, max: 240, step: 5 }

/** A step of a hundredth, where the arithmetic of a float shows. */
const FINE: Bounds = { min: 0.7, max: 0.99, step: 0.01 }

describe('what the text comes to', () => {
  it('reads a number out of the digits', () => {
    expect(numberOf('20')).toBe(20)
    expect(numberOf(' 20 ')).toBe(20)
    expect(numberOf('-3')).toBe(-3)
    expect(numberOf('0.5')).toBe(0.5)
  })

  it('reads a comma as the point, wherever it is typed that way', () => {
    expect(numberOf('0,5')).toBe(0.5)
    expect(numberOf('12,25')).toBe(12.25)
  })

  // A comma before a group of three is how a thousand is written, and reading
  // it as the point would turn a thousand two hundred into one and a fifth.
  it('comes to nothing where a comma stands before a group of three', () => {
    expect(numberOf('1,200')).toBeNull()
    expect(onItsWay('1,200')).toBe(false)
    expect(numberOf('12,000')).toBeNull()
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
    expect(clamp(-4, BOUNDS)).toBe(0)
    expect(clamp(900, BOUNDS)).toBe(240)
    expect(clamp(20, BOUNDS)).toBe(20)
  })

  it('moves a number one step, and stops at them', () => {
    expect(stepBy(20, 1, BOUNDS)).toBe(25)
    expect(stepBy(20, -1, BOUNDS)).toBe(15)
    expect(stepBy(2, -1, BOUNDS)).toBe(0)
    expect(stepBy(238, 1, BOUNDS)).toBe(240)
  })

  it('starts at the floor where no number stands', () => {
    expect(stepBy(null, 1, BOUNDS)).toBe(5)
    expect(stepBy(null, -1, BOUNDS)).toBe(0)
  })

  // A step is written to so many places, and the number it leaves is written
  // to the same, so what a person reads is what the file gets.
  it('leaves a number written to the places its step is written to', () => {
    expect(stepBy(0.81, 1, FINE)).toBe(0.82)
    expect(stepBy(0.83, -1, FINE)).toBe(0.82)
  })

  it('lands on a place the step lays, from wherever it started', () => {
    expect(stepBy(13, 1, BOUNDS)).toBe(20)
    expect(stepBy(13, -1, BOUNDS)).toBe(10)
  })
})

describe('the places a step lays', () => {
  it('is where a number off them is brought', () => {
    expect(snapToBounds(13, BOUNDS)).toBe(15)
    expect(snapToBounds(0.873, FINE)).toBe(0.87)
    expect(snapToBounds(20, BOUNDS)).toBe(20)
  })

  it('holds a number brought to them inside the bounds', () => {
    expect(snapToBounds(900, BOUNDS)).toBe(240)
    expect(snapToBounds(-4, BOUNDS)).toBe(0)
  })

  it('is what the bounds allow, and a number between two of them is not', () => {
    expect(isAllowed('15', BOUNDS)).toBe(true)
    expect(isAllowed('13', BOUNDS)).toBe(false)
    expect(isAllowed('0.87', FINE)).toBe(true)
    expect(isAllowed('0.873', FINE)).toBe(false)
  })
})

describe('how a number is written back', () => {
  it('writes the number, and nothing where there is none', () => {
    expect(written(20)).toBe('20')
    expect(written(0)).toBe('0')
    expect(written(null)).toBe('')
  })
})

describe('the keys a spin button answers', () => {
  it('moves a step under the arrows and ten under the page keys', () => {
    expect(stepForKey('ArrowUp', 20, BOUNDS)).toBe(25)
    expect(stepForKey('ArrowDown', 20, BOUNDS)).toBe(15)
    expect(stepForKey('PageUp', 20, BOUNDS)).toBe(70)
    expect(stepForKey('PageDown', 100, BOUNDS)).toBe(50)
  })

  it('takes the number to the ends under home and end', () => {
    expect(stepForKey('Home', 20, BOUNDS)).toBe(0)
    expect(stepForKey('End', 20, BOUNDS)).toBe(240)
    expect(stepForKey('End', 0.8, FINE)).toBe(0.99)
  })

  it('holds them all inside the bounds', () => {
    expect(stepForKey('PageDown', 20, BOUNDS)).toBe(0)
    expect(stepForKey('PageUp', 200, BOUNDS)).toBe(240)
  })

  it('opens at the floor where no number stands there', () => {
    expect(stepForKey('ArrowUp', null, BOUNDS)).toBe(5)
  })

  it('leaves a key it does not answer with nothing to say', () => {
    expect(stepForKey('a', 20, BOUNDS)).toBeNull()
    expect(stepForKey('Enter', 20, BOUNDS)).toBeNull()
    expect(stepForKey('Backspace', 20, BOUNDS)).toBeNull()
  })
})

describe('whether what is typed stands for the number in force', () => {
  it('is the number the digits come to, however they are written', () => {
    expect(isTextForValue('20', 20)).toBe(true)
    expect(isTextForValue('20.0', 20)).toBe(true)
    expect(isTextForValue('21', 20)).toBe(false)
  })

  // A line that is no number stands for no number, and not for the absence of
  // one either.
  it('is an empty field alone that stands for no number at all', () => {
    expect(isTextForValue('', null)).toBe(true)
    expect(isTextForValue('  ', null)).toBe(true)
    expect(isTextForValue('twenty', null)).toBe(false)
    expect(isTextForValue('twenty', 20)).toBe(false)
  })
})
