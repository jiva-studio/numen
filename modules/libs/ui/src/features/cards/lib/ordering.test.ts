/**
 * The order entries stand in and the names they may be given, as plain values.
 * No DOM, no measurement.
 */
import { describe, expect, it } from 'vitest'
import {
  directionOf,
  landing,
  getFreeName,
  objection,
  orderNames,
  reorderFields,
  getStepLanding,
} from './order'

describe('orderNames', () => {
  const NAMES = ['a', 'b', 'c']

  it('puts a dragged name before the one it lands on', () => {
    expect(orderNames(NAMES, 'c', 'a')).toEqual(['c', 'a', 'b'])
  })

  it('puts it at the end where it lands on nothing', () => {
    expect(orderNames(NAMES, 'a', null)).toEqual(['b', 'c', 'a'])
  })

  it('takes it out before it puts it back', () => {
    expect(orderNames(NAMES, 'a', 'c')).toEqual(['b', 'a', 'c'])
  })

  it('leaves the order alone where it lands on itself', () => {
    expect(orderNames(NAMES, 'b', 'b')).toEqual(NAMES)
  })

  it('leaves the order alone where it lands on a name that is not there', () => {
    expect(orderNames(NAMES, 'b', 'z')).toEqual(NAMES)
  })

  it('leaves the order alone where what is dragged is not there', () => {
    expect(orderNames(NAMES, 'z', 'a')).toEqual(NAMES)
  })

  it('keeps one name alone where it is', () => {
    expect(orderNames(['only'], 'only', null)).toEqual(['only'])
  })
})

describe('landing', () => {
  const FIELDS = ['Name', 'Height', 'Weight']

  it('lands a field below the first before another below the first', () => {
    expect(landing(FIELDS, 'Weight', 'Height')).toBe(true)
  })

  it('lands a field below the first at the end', () => {
    expect(landing(FIELDS, 'Height', null)).toBe(true)
  })

  it('does not move the first field, which names every card', () => {
    expect(landing(FIELDS, 'Name', 'Weight')).toBe(false)
    expect(landing(FIELDS, 'Name', null)).toBe(false)
  })

  it('lands nothing above the first field', () => {
    expect(landing(FIELDS, 'Weight', 'Name')).toBe(false)
  })

  it('moves nothing let go where it stands', () => {
    expect(landing(FIELDS, 'Height', 'Height')).toBe(false)
  })

  it('moves nothing among no fields', () => {
    expect(landing([], 'Height', null)).toBe(false)
  })

  it('does not move the one field a stencil names', () => {
    expect(landing(['Only'], 'Only', null)).toBe(false)
  })
})

describe('reorderFields', () => {
  const FIELDS = ['Name', 'Height', 'Weight']

  it('reorders the fields below the first', () => {
    expect(reorderFields(FIELDS, 'Weight', 'Height')).toEqual(['Name', 'Weight', 'Height'])
  })

  it('leaves the order alone where the first field is dragged', () => {
    expect(reorderFields(FIELDS, 'Name', null)).toEqual(FIELDS)
  })

  it('leaves the order alone where a field is let go above the first', () => {
    expect(reorderFields(FIELDS, 'Weight', 'Name')).toEqual(FIELDS)
  })
})

describe('getStepLanding', () => {
  const NAMES = ['a', 'b', 'c']

  it('lands what is dragged up before the one above it', () => {
    expect(getStepLanding(NAMES, 'c', 'up')).toBe('b')
    expect(orderNames(NAMES, 'c', getStepLanding(NAMES, 'c', 'up') ?? null)).toEqual(['a', 'c', 'b'])
  })

  it('lands what is dragged down before the one below the one below it', () => {
    expect(getStepLanding(NAMES, 'a', 'down')).toBe('c')
    expect(orderNames(NAMES, 'a', getStepLanding(NAMES, 'a', 'down') ?? null)).toEqual(['b', 'a', 'c'])
  })

  it('lands the last but one at the end', () => {
    expect(getStepLanding(NAMES, 'b', 'down')).toBeNull()
    expect(orderNames(NAMES, 'b', null)).toEqual(['a', 'c', 'b'])
  })

  it('lands nothing above the first, and nothing below the last', () => {
    expect(getStepLanding(NAMES, 'a', 'up')).toBeUndefined()
    expect(getStepLanding(NAMES, 'c', 'down')).toBeUndefined()
  })

  it('lands nothing that is not there', () => {
    expect(getStepLanding(NAMES, 'z', 'up')).toBeUndefined()
  })
})

describe('directionOf', () => {
  it('reads the two arrows along the order', () => {
    expect(directionOf('ArrowUp')).toBe('up')
    expect(directionOf('ArrowDown')).toBe('down')
  })

  it('reads every other key as no direction at all', () => {
    expect(directionOf('ArrowLeft')).toBeNull()
    expect(directionOf('Enter')).toBeNull()
  })
})

describe('objection', () => {
  it('objects to a name with nothing in it', () => {
    expect(objection('   ', [])).toBe('blank')
  })

  it('objects to a name already taken', () => {
    expect(objection('Height', ['Height'])).toBe('taken')
  })

  it('objects to a name taken but for the space around it', () => {
    expect(objection(' Height ', ['Height'])).toBe('taken')
  })

  it('objects to a name holding a brace, which no slot could write', () => {
    expect(objection('a{b', [])).toBe('braced')
    expect(objection('a}b', [])).toBe('braced')
  })

  it('takes a name that is free', () => {
    expect(objection('Weight', ['Height'])).toBeNull()
  })

  it('takes a name that is not Latin', () => {
    expect(objection('Продолжительность жизни', ['Height'])).toBeNull()
  })
})

describe('getFreeName', () => {
  it('numbers from one, so the first carries a number like the rest', () => {
    expect(getFreeName([], 'Field')).toBe('Field 1')
  })

  it('numbers past what is taken', () => {
    expect(getFreeName(['Field 1', 'Field 2'], 'Field')).toBe('Field 3')
  })

  it('fills a gap in the numbering', () => {
    expect(getFreeName(['Field 1', 'Field 3'], 'Field')).toBe('Field 2')
  })
})
