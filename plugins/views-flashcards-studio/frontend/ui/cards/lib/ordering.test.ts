/**
 * The order entries stand in and the names they may be given, as plain values.
 * No DOM, no measurement.
 */
import { describe, expect, it } from 'vitest'
import {
  parseStepKey,
  canDropField,
  getFreeName,
  checkFieldName,
  checkHeadingName,
  createSealedMap,
  orderNames,
  reorderFields,
  getStepInsertionPoint,
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

  it('leaves the order alone where the first name lands on a name that is not there', () => {
    expect(orderNames(NAMES, 'a', 'z')).toEqual(NAMES)
  })

  it('keeps one name alone where it is', () => {
    expect(orderNames(['only'], 'only', null)).toEqual(['only'])
  })
})

describe('canDropField', () => {
  const FIELDS = ['Name', 'Height', 'Weight']

  it('lands a field below the first before another below the first', () => {
    expect(canDropField(FIELDS, 'Weight', 'Height')).toBe(true)
  })

  it('lands a field below the first at the end', () => {
    expect(canDropField(FIELDS, 'Height', null)).toBe(true)
  })

  it('does not move the first field, which names every card', () => {
    expect(canDropField(FIELDS, 'Name', 'Weight')).toBe(false)
    expect(canDropField(FIELDS, 'Name', null)).toBe(false)
  })

  it('lands nothing above the first field', () => {
    expect(canDropField(FIELDS, 'Weight', 'Name')).toBe(false)
  })

  it('moves nothing let go at its current position', () => {
    expect(canDropField(FIELDS, 'Height', 'Height')).toBe(false)
  })

  it('moves nothing among no fields', () => {
    expect(canDropField([], 'Height', null)).toBe(false)
  })

  it('does not move the one field a card model names', () => {
    expect(canDropField(['Only'], 'Only', null)).toBe(false)
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

describe('getStepInsertionPoint', () => {
  const NAMES = ['a', 'b', 'c']

  it('lands what is dragged up before the one above it', () => {
    expect(getStepInsertionPoint(NAMES, 'c', 'up')).toBe('b')
    expect(orderNames(NAMES, 'c', getStepInsertionPoint(NAMES, 'c', 'up') ?? null)).toEqual([
      'a',
      'c',
      'b',
    ])
  })

  it('lands what is dragged down before the one below the one below it', () => {
    expect(getStepInsertionPoint(NAMES, 'a', 'down')).toBe('c')
    expect(orderNames(NAMES, 'a', getStepInsertionPoint(NAMES, 'a', 'down') ?? null)).toEqual([
      'b',
      'a',
      'c',
    ])
  })

  it('lands the last but one at the end', () => {
    expect(getStepInsertionPoint(NAMES, 'b', 'down')).toBeNull()
    expect(orderNames(NAMES, 'b', null)).toEqual(['a', 'c', 'b'])
  })

  it('lands nothing above the first, and nothing below the last', () => {
    expect(getStepInsertionPoint(NAMES, 'a', 'up')).toBeUndefined()
    expect(getStepInsertionPoint(NAMES, 'c', 'down')).toBeUndefined()
  })

  it('lands nothing that is not there', () => {
    expect(getStepInsertionPoint(NAMES, 'z', 'up')).toBeUndefined()
  })
})

describe('getStepInsertionPoint, a name that is not there', () => {
  it('lands nothing in either direction', () => {
    expect(getStepInsertionPoint(['a', 'b', 'c'], 'z', 'down')).toBeUndefined()
  })
})

describe('parseStepKey', () => {
  it('reads the two arrows along the order', () => {
    expect(parseStepKey('ArrowUp')).toBe('up')
    expect(parseStepKey('ArrowDown')).toBe('down')
  })

  it('reads every other key as no direction at all', () => {
    expect(parseStepKey('ArrowLeft')).toBeNull()
    expect(parseStepKey('Enter')).toBeNull()
  })
})

describe('checkFieldName', () => {
  it('objects to a name with nothing in it', () => {
    expect(checkFieldName('   ', []).objection).toBe('blank')
  })

  it('objects to a name already taken', () => {
    expect(checkFieldName('Height', ['Height']).objection).toBe('taken')
  })

  it('objects to a name taken but for the space around it', () => {
    expect(checkFieldName(' Height ', ['Height']).objection).toBe('taken')
  })

  it('objects to a name holding a brace, which no slot could write', () => {
    expect(checkFieldName('a{b', []).objection).toBe('braced')
    expect(checkFieldName('a}b', []).objection).toBe('braced')
  })

  it('takes a name that is free', () => {
    expect(checkFieldName('Weight', ['Height']).objection).toBeNull()
  })

  it('takes a name that is not Latin', () => {
    expect(checkFieldName('Продолжительность жизни', ['Height']).objection).toBeNull()
  })

  it('answers with the name as it would be written, without the space around it', () => {
    expect(checkFieldName(' Weight ', ['Height']).name).toBe('Weight')
  })
})

describe('names written with space around them', () => {
  it('answers a braced name as it would be written', () => {
    expect(checkFieldName(' a{b ', []).name).toBe('a{b')
  })

  it('objects to a name taken by one written with space around it', () => {
    expect(checkHeadingName('Height', ['Height ']).objection).toBe('taken')
  })

  it('numbers past a name taken with space around it', () => {
    expect(getFreeName(['Field 1 '], 'Field')).toBe('Field 2')
  })
})

describe('createSealedMap', () => {
  it('is a map of nothing', () => {
    expect(createSealedMap<string, string>().size).toBe(0)
  })

  it('refuses every way of putting something into it or taking something out', () => {
    const sealed = createSealedMap<string, string>() as Map<string, string>
    expect(() => sealed.set('a', 'b')).toThrow(TypeError)
    expect(() => sealed.delete('a')).toThrow(TypeError)
    expect(() => sealed.clear()).toThrow(TypeError)
    expect(() => sealed.set('a', 'b')).toThrow('an empty default holds nothing')
    expect(sealed.size).toBe(0)
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
