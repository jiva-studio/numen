/**
 * Where each problem the vault reports is drawn, asked without a screen.
 */
import { describe, expect, it } from 'vitest'
import { areMarksEqual, createMarks } from './marks'
import type { DeckProblem } from '../types'

/** One problem as the vault reports one, against nothing in particular. */
const problem = (overrides: Partial<DeckProblem> = {}): DeckProblem => ({
  kind: 'cardWithoutACardModel',
  card: null,
  template: null,
  field: '',
  text: 'a card under no card model',
  ...overrides,
})

describe('where a problem is drawn', () => {
  it('is the card it was read against, counted from the first', () => {
    const marks = createMarks([problem({ card: 1 })], ['c1', 'c2'], [])
    expect(marks.at.get('c2')).toStrictEqual(['a card under no card model'])
    expect(marks.at.has('c1')).toBe(false)
  })

  it('is the identity the card was drawn under, so two of one card id are told apart', () => {
    const marks = createMarks(
      [
        problem({ kind: 'cardIdCarriedTwice', card: 1, text: 'a mark carried twice' }),
        problem({ kind: 'cardIdCarriedTwice', card: 2, text: 'a mark carried twice' }),
      ],
      ['c1', 'c2', 'c3'],
      [],
    )
    expect([...marks.at.keys()]).toStrictEqual(['c2', 'c3'])
  })

  it('remains every problem of one card together, in the order they were read', () => {
    const marks = createMarks(
      [problem({ card: 0, text: 'first' }), problem({ card: 0, text: 'second' })],
      ['c1'],
      [],
    )
    expect(marks.at.get('c1')).toStrictEqual(['first', 'second'])
  })

  it('is the template it was read against', () => {
    const marks = createMarks(
      [problem({ kind: 'templateMissingASide', template: 0, text: 'no back' })],
      [],
      ['f1', 'f2'],
    )
    expect(marks.at.get('f1')).toStrictEqual(['no back'])
  })

  it('is the field of the card where it names both, so it is returned under that value', () => {
    const marks = createMarks(
      [problem({ kind: 'fieldWrittenTwice', card: 1, field: 'Name', text: 'twice' })],
      ['c1', 'c2'],
      [],
    )
    expect(marks.bySection.get('c2')?.get('Name')).toStrictEqual(['twice'])
    expect(marks.at.has('c2')).toBe(false)
  })

  it('is the field at its current position against no card and no template', () => {
    const marks = createMarks(
      [problem({ kind: 'fieldDeclaredTwice', field: 'Height', text: 'declared twice' })],
      [],
      [],
    )
    expect(marks.fields.get('Height')).toStrictEqual(['declared twice'])
    expect(marks.whole).toStrictEqual([])
  })

  it('is the file at its current position against none of the three', () => {
    const marks = createMarks([problem({ text: 'something else' })], ['c1'], [])
    expect(marks.whole).toStrictEqual(['something else'])
  })

  it('is the file where the card it names was not read', () => {
    expect(createMarks([problem({ card: 5 })], ['c1'], []).whole).toStrictEqual([
      'a card under no card model',
    ])
  })
})

describe('where two problems of one card stand', () => {
  it('keeps the fields of one card together, each under its own name', () => {
    const marks = createMarks(
      [
        problem({ card: 0, field: 'Front', text: 'first' }),
        problem({ card: 0, field: 'Back', text: 'second' }),
      ],
      ['c1'],
      [],
    )

    expect(marks.bySection.get('c1')?.get('Front')).toStrictEqual(['first'])
    expect(marks.bySection.get('c1')?.get('Back')).toStrictEqual(['second'])
  })
})

/** The marks of problems read against two cards and a template. */
const createTestMarks = (...problems: DeckProblem[]) => createMarks(problems, ['c1', 'c2'], ['f1'])

describe('whether two evaluations are wrong in the same way', () => {
  it('is so for two evaluations of nothing', () => {
    expect(areMarksEqual(createTestMarks(), createTestMarks())).toBe(true)
  })

  it('is so for two evaluations saying the same against the same things', () => {
    const all = () => [
      problem({ text: 'file' }),
      problem({ card: 0, text: 'card' }),
      problem({ template: 0, text: 'template' }),
      problem({ card: 1, field: 'Front', text: 'value' }),
      problem({ field: 'Back', text: 'field' }),
    ]

    expect(areMarksEqual(createTestMarks(...all()), createTestMarks(...all()))).toBe(true)
  })

  it('is not so where the file is told something else', () => {
    expect(
      areMarksEqual(
        createTestMarks(problem({ text: 'a' })),
        createTestMarks(problem({ text: 'b' })),
      ),
    ).toBe(false)
  })

  it('is not so where the file is told more in the same words', () => {
    const one = createTestMarks(problem({ text: 'a' }))
    const other = createTestMarks(problem({ text: 'a' }), problem({ text: 'b' }))

    expect(areMarksEqual(one, other)).toBe(false)
    expect(areMarksEqual(other, one)).toBe(false)
  })

  it('is not so where the file is told the same words in another order', () => {
    const one = createTestMarks(problem({ text: 'a' }), problem({ text: 'b' }))
    const other = createTestMarks(problem({ text: 'b' }), problem({ text: 'a' }))

    expect(areMarksEqual(one, other)).toBe(false)
  })

  it('is not so where only a card or a template is told something else', () => {
    expect(
      areMarksEqual(
        createTestMarks(problem({ card: 0, text: 'a' })),
        createTestMarks(problem({ card: 0, text: 'b' })),
      ),
    ).toBe(false)
    expect(
      areMarksEqual(
        createTestMarks(problem({ card: 0, text: 'a' })),
        createTestMarks(problem({ card: 1, text: 'a' })),
      ),
    ).toBe(false)
  })

  it('is not so where one reading remains against more cards than the other', () => {
    const one = createTestMarks(problem({ card: 0 }))
    const other = createTestMarks(problem({ card: 0 }), problem({ card: 1 }))

    expect(areMarksEqual(one, other)).toBe(false)
    expect(areMarksEqual(other, one)).toBe(false)
  })

  it('is not so where only a value of a card is told something else', () => {
    const one = createTestMarks(problem({ card: 0, field: 'Front', text: 'a' }))

    expect(
      areMarksEqual(one, createTestMarks(problem({ card: 0, field: 'Front', text: 'b' }))),
    ).toBe(false)
    expect(
      areMarksEqual(one, createTestMarks(problem({ card: 0, field: 'Back', text: 'a' }))),
    ).toBe(false)
    expect(
      areMarksEqual(one, createTestMarks(problem({ card: 1, field: 'Front', text: 'a' }))),
    ).toBe(false)
  })

  it('is not so where one reading remains against more cards with values than the other', () => {
    const one = createTestMarks(problem({ card: 0, field: 'Front' }))
    const other = createTestMarks(
      problem({ card: 0, field: 'Front' }),
      problem({ card: 1, field: 'Front' }),
    )

    expect(areMarksEqual(one, other)).toBe(false)
    expect(areMarksEqual(other, one)).toBe(false)
  })

  it('is not so where one reading remains against more values of a card than the other', () => {
    const one = createTestMarks(problem({ card: 0, field: 'Front' }))
    const other = createTestMarks(
      problem({ card: 0, field: 'Front' }),
      problem({ card: 0, field: 'Back' }),
    )

    expect(areMarksEqual(one, other)).toBe(false)
    expect(areMarksEqual(other, one)).toBe(false)
  })

  it('is not so where only a field of the file is told something else', () => {
    expect(
      areMarksEqual(
        createTestMarks(problem({ field: 'Back', text: 'a' })),
        createTestMarks(problem({ field: 'Back', text: 'b' })),
      ),
    ).toBe(false)
    expect(
      areMarksEqual(
        createTestMarks(problem({ field: 'Back', text: 'a' })),
        createTestMarks(problem({ field: 'Front', text: 'a' })),
      ),
    ).toBe(false)
  })
})
