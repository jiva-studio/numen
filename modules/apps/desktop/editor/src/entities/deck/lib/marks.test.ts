/**
 * Where each problem the vault reports is drawn, asked without a screen.
 */
import { describe, expect, it } from 'vitest'
import { createMarks } from './marks'
import type { DeckProblem } from '../types'

/** One problem as the vault reports one, against nothing in particular. */
const problem = (over: Partial<DeckProblem> = {}): DeckProblem => ({
  fault: 'cardWithoutAStencil',
  card: null,
  face: null,
  field: '',
  text: 'a card under no stencil',
  ...over,
})

describe('where a problem is drawn', () => {
  it('is the card it was read against, counted from the first', () => {
    const marks = createMarks([problem({ card: 1 })], ['c1', 'c2'], [])
    expect(marks.at.get('c2')).toStrictEqual(['a card under no stencil'])
    expect(marks.at.has('c1')).toBe(false)
  })

  it('is the identity the card was drawn under, so two of one mark are told apart', () => {
    const marks = createMarks(
      [
        problem({ fault: 'markCarriedTwice', card: 1, text: 'a mark carried twice' }),
        problem({ fault: 'markCarriedTwice', card: 2, text: 'a mark carried twice' }),
      ],
      ['c1', 'c2', 'c3'],
      [],
    )
    expect([...marks.at.keys()]).toStrictEqual(['c2', 'c3'])
  })

  it('stands every problem of one card together, in the order they were read', () => {
    const marks = createMarks(
      [problem({ card: 0, text: 'first' }), problem({ card: 0, text: 'second' })],
      ['c1'],
      [],
    )
    expect(marks.at.get('c1')).toStrictEqual(['first', 'second'])
  })

  it('is the face it was read against', () => {
    const marks = createMarks(
      [problem({ fault: 'faceMissingASide', face: 0, text: 'no back' })],
      [],
      ['f1', 'f2'],
    )
    expect(marks.at.get('f1')).toStrictEqual(['no back'])
  })

  it('is the field of the card where it names both, so it is said under that value', () => {
    const marks = createMarks(
      [problem({ fault: 'fieldWrittenTwice', card: 1, field: 'Name', text: 'twice' })],
      ['c1', 'c2'],
      [],
    )
    expect(marks.under.get('c2')?.get('Name')).toStrictEqual(['twice'])
    expect(marks.at.has('c2')).toBe(false)
  })

  it('is the field where it stands against no card and no face', () => {
    const marks = createMarks(
      [problem({ fault: 'fieldDeclaredTwice', field: 'Height', text: 'declared twice' })],
      [],
      [],
    )
    expect(marks.fields.get('Height')).toStrictEqual(['declared twice'])
    expect(marks.whole).toStrictEqual([])
  })

  it('is the file where it stands against none of the three', () => {
    const marks = createMarks([problem({ text: 'something else' })], ['c1'], [])
    expect(marks.whole).toStrictEqual(['something else'])
  })

  it('is the file where the card it names was not read', () => {
    expect(createMarks([problem({ card: 5 })], ['c1'], []).whole).toStrictEqual([
      'a card under no stencil',
    ])
  })
})
