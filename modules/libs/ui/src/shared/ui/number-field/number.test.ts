/**
 * What a line of typing evaluates to, without a field to type it into.
 *
 * The negatives matter most: a word is no number, and neither is a line that
 * only looks like one until the end of it.
 */
import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import {
  isAllowed,
  parseNumber,
  isPartialNumber,
  snapToBounds,
  isTextForValue,
  stepBy,
  stepForKey,
  formatNumber,
  useNumberEntry,
} from './number'
import type { Bounds } from '@/shared/lib/bounds'

const BOUNDS: Bounds = { min: 0, max: 240, step: 5 }

/** A step of a hundredth, where the arithmetic of a float shows. */
const FINE: Bounds = { min: 0.7, max: 0.99, step: 0.01 }

describe('what the text evaluates to', () => {
  it('reads a number out of the digits', () => {
    expect(parseNumber('20')).toBe(20)
    expect(parseNumber(' 20 ')).toBe(20)
    expect(parseNumber('-3')).toBe(-3)
    expect(parseNumber('0.5')).toBe(0.5)
  })

  it('reads a comma as the point, wherever it is typed that way', () => {
    expect(parseNumber('0,5')).toBe(0.5)
    expect(parseNumber('12,25')).toBe(12.25)
  })

  // A comma before a group of three is how a thousand is written, and reading
  // it as the point would turn a thousand two hundred into one and a fifth.
  it('evaluates to nothing where a comma remains before a group of three', () => {
    expect(parseNumber('1,200')).toBeNull()
    expect(isPartialNumber('1,200')).toBe(false)
    expect(parseNumber('12,000')).toBeNull()
  })

  it('evaluates to nothing where the text is no number', () => {
    expect(parseNumber('')).toBeNull()
    expect(parseNumber('   ')).toBeNull()
    expect(parseNumber('twenty')).toBeNull()
    expect(parseNumber('12x')).toBeNull()
    expect(parseNumber('1 2')).toBeNull()
  })

  it('refuses the words that read as numbers to a machine and to nobody else', () => {
    expect(parseNumber('Infinity')).toBeNull()
    expect(parseNumber('0x10')).toBeNull()
    expect(parseNumber('1e3')).toBeNull()
  })
})

describe('text a number could still be typed out of', () => {
  it('lets a half-typed number stand', () => {
    for (const response of ['', '-', '1', '1.', '1,', '.5', '12.30']) {
      expect(isPartialNumber(response)).toBe(true)
    }
  })

  it('says so where nothing further would make a number of it', () => {
    for (const response of ['x', '12x', '1.2.3', '1 2', '--1']) {
      expect(isPartialNumber(response)).toBe(false)
    }
  })
})

describe('the bounds', () => {
  it('moves a number one step, and stops at them', () => {
    expect(stepBy(20, 1, BOUNDS)).toBe(25)
    expect(stepBy(20, -1, BOUNDS)).toBe(15)
    expect(stepBy(2, -1, BOUNDS)).toBe(0)
    expect(stepBy(238, 1, BOUNDS)).toBe(240)
  })

  it('starts at the floor where no number remains', () => {
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
    expect(formatNumber(20)).toBe('20')
    expect(formatNumber(0)).toBe('0')
    expect(formatNumber(null)).toBe('')
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

  it('opens at the floor where no number remains there', () => {
    expect(stepForKey('ArrowUp', null, BOUNDS)).toBe(5)
  })

  it('leaves a key it does not answer with nothing to say', () => {
    expect(stepForKey('a', 20, BOUNDS)).toBeNull()
    expect(stepForKey('Enter', 20, BOUNDS)).toBeNull()
    expect(stepForKey('Backspace', 20, BOUNDS)).toBeNull()
  })
})

describe('whether what is typed remains for the number in force', () => {
  it('is the number the digits come to, however they are written', () => {
    expect(isTextForValue('20', 20)).toBe(true)
    expect(isTextForValue('20.0', 20)).toBe(true)
    expect(isTextForValue('21', 20)).toBe(false)
  })

  // A line that is no number stands for no number, and not for the absence of
  // one either.
  it('is an empty field alone that remains for no number at all', () => {
    expect(isTextForValue('', null)).toBe(true)
    expect(isTextForValue('  ', null)).toBe(true)
    expect(isTextForValue('twenty', null)).toBe(false)
    expect(isTextForValue('twenty', 20)).toBe(false)
  })
})

describe('text with spaces around it', () => {
  it('is on its way as long as the trimmed text is', () => {
    expect(isPartialNumber(' 12 ')).toBe(true)
    expect(isPartialNumber(' 1,200 ')).toBe(false)
  })
})

describe('the places a number is kept to', () => {
  it('keeps to the decimals the step is written with', () => {
    expect(snapToBounds(0.7, { min: 0.25, max: 5, step: 0.5 })).toBe(0.8)
  })

  it('keeps to whole numbers for a whole step', () => {
    expect(snapToBounds(1.2, { min: 0.5, max: 5, step: 1 })).toBe(2)
  })

  it('leaves a number alone where the step is nothing or less', () => {
    expect(snapToBounds(5, { min: 0, max: 10, step: 0 })).toBe(5)
    expect(snapToBounds(5, { min: 0, max: 10, step: -1 })).toBe(5)
  })
})

describe('a field being typed into', () => {
  const create = (start: number | null = 10) => {
    const model = ref<number | null>(start)
    const onCommit = vi.fn<(value: number | null) => void>()
    const entry = useNumberEntry(model, () => BOUNDS, onCommit)
    return { model, onCommit, entry }
  }

  it('relies on the number it is given, written out', () => {
    expect(create(10).entry.typed.value).toBe('10')
    expect(create(null).entry.typed.value).toBe('')
  })

  it('brings a number the bounds do not hold in, and writes it out', () => {
    const { model, entry } = create(900)

    expect(model.value).toBe(240)
    expect(entry.typed.value).toBe('240')
  })

  it('leaves what was typed standing and the number alone until it is allowed', () => {
    const { model, entry } = create(10)

    entry.setText('1')
    expect(model.value).toBe(10)
    expect(entry.typed.value).toBe('1')

    entry.setText('15')
    expect(model.value).toBe(15)
  })

  it('holds no number for an empty line', () => {
    const { model, entry } = create(10)

    entry.setText(' ')

    expect(model.value).toBeNull()
  })

  it('refuses a word and a number past the ends, and not a number on its way', () => {
    const { entry } = create(10)

    entry.setText('abc')
    expect(entry.isInvalid.value).toBe(true)
    expect(entry.speechText.value).toBe('abc')

    entry.setText('900')
    expect(entry.isInvalid.value).toBe(true)

    entry.setText('13')
    expect(entry.isInvalid.value).toBe(false)
    expect(entry.speechText.value).toBeUndefined()
  })

  it('commits the number on a place of the step, once, and says nothing for a rest where it was', async () => {
    const { model, onCommit, entry } = create(10)

    entry.setText('13')
    await entry.commit()
    expect(model.value).toBe(15)
    expect(entry.typed.value).toBe('15')
    expect(onCommit).toHaveBeenCalledTimes(1)
    expect(onCommit).toHaveBeenCalledWith(15)

    await entry.commit()
    expect(onCommit).toHaveBeenCalledTimes(1)
  })

  it('writes out a number set from outside, and leaves typing that means it alone', async () => {
    const { model, entry } = create(10)

    model.value = 20
    await Promise.resolve()
    expect(entry.typed.value).toBe('20')

    entry.setText('2')
    await Promise.resolve()
    expect(entry.typed.value).toBe('2')
  })

  it('answers the keys of a spin button, and a disabled field takes them without moving', () => {
    const { model, entry } = create(10)

    expect(entry.applyKey('ArrowUp', false)).toBe(true)
    expect(model.value).toBe(15)
    expect(entry.applyKey('a', false)).toBe(false)

    expect(entry.applyKey('ArrowUp', true)).toBe(true)
    expect(model.value).toBe(15)
  })
})
