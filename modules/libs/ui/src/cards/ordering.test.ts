/**
 * The order entries stand in and the names they may be given, as plain values.
 * No DOM, no measurement.
 */
import { describe, expect, it } from 'vitest'
import {
  landing,
  numbered,
  objection,
  ordered,
  reordered,
  stepped,
  wayOf,
} from './order'

describe('ordered', () => {
  const NAMES = ['a', 'b', 'c']

  it('puts a carried name before the one it lands on', () => {
    expect(ordered(NAMES, 'c', 'a')).toEqual(['c', 'a', 'b'])
  })

  it('puts it at the end where it lands on nothing', () => {
    expect(ordered(NAMES, 'a', null)).toEqual(['b', 'c', 'a'])
  })

  it('takes it out before it puts it back', () => {
    expect(ordered(NAMES, 'a', 'c')).toEqual(['b', 'a', 'c'])
  })

  it('leaves the order alone where it lands on itself', () => {
    expect(ordered(NAMES, 'b', 'b')).toEqual(NAMES)
  })

  it('leaves the order alone where it lands on a name that is not there', () => {
    expect(ordered(NAMES, 'b', 'z')).toEqual(NAMES)
  })

  it('leaves the order alone where what is carried is not there', () => {
    expect(ordered(NAMES, 'z', 'a')).toEqual(NAMES)
  })

  it('keeps one name alone where it is', () => {
    expect(ordered(['only'], 'only', null)).toEqual(['only'])
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

describe('reordered', () => {
  const FIELDS = ['Name', 'Height', 'Weight']

  it('reorders the fields below the first', () => {
    expect(reordered(FIELDS, 'Weight', 'Height')).toEqual(['Name', 'Weight', 'Height'])
  })

  it('leaves the order alone where the first field is carried', () => {
    expect(reordered(FIELDS, 'Name', null)).toEqual(FIELDS)
  })

  it('leaves the order alone where a field is let go above the first', () => {
    expect(reordered(FIELDS, 'Weight', 'Name')).toEqual(FIELDS)
  })
})

describe('stepped', () => {
  const NAMES = ['a', 'b', 'c']

  it('lands what is carried up before the one above it', () => {
    expect(stepped(NAMES, 'c', 'up')).toBe('b')
    expect(ordered(NAMES, 'c', stepped(NAMES, 'c', 'up') ?? null)).toEqual(['a', 'c', 'b'])
  })

  it('lands what is carried down before the one below the one below it', () => {
    expect(stepped(NAMES, 'a', 'down')).toBe('c')
    expect(ordered(NAMES, 'a', stepped(NAMES, 'a', 'down') ?? null)).toEqual(['b', 'a', 'c'])
  })

  it('lands the last but one at the end', () => {
    expect(stepped(NAMES, 'b', 'down')).toBeNull()
    expect(ordered(NAMES, 'b', null)).toEqual(['a', 'c', 'b'])
  })

  it('lands nothing above the first, and nothing below the last', () => {
    expect(stepped(NAMES, 'a', 'up')).toBeUndefined()
    expect(stepped(NAMES, 'c', 'down')).toBeUndefined()
  })

  it('lands nothing that is not there', () => {
    expect(stepped(NAMES, 'z', 'up')).toBeUndefined()
  })
})

describe('wayOf', () => {
  it('reads the two arrows along the order', () => {
    expect(wayOf('ArrowUp')).toBe('up')
    expect(wayOf('ArrowDown')).toBe('down')
  })

  it('reads every other key as no way at all', () => {
    expect(wayOf('ArrowLeft')).toBeNull()
    expect(wayOf('Enter')).toBeNull()
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

describe('numbered', () => {
  it('numbers from one, so the first carries a number like the rest', () => {
    expect(numbered([], 'Field')).toBe('Field 1')
  })

  it('numbers past what is taken', () => {
    expect(numbered(['Field 1', 'Field 2'], 'Field')).toBe('Field 3')
  })

  it('fills a gap in the numbering', () => {
    expect(numbered(['Field 1', 'Field 3'], 'Field')).toBe('Field 2')
  })
})
