/**
 * The runs a list of choices is drawn in.
 *
 * A shelf is a stretch of the list, not a bucket: the order the caller offered
 * the choices in is the order they are drawn in.
 */
import { describe, expect, it } from 'vitest'
import { shelved } from './shelves'

describe('the shelves a list of choices makes', () => {
  it('is one run for a list naming no shelf at all', () => {
    expect(shelved([{ id: 'a', text: 'A' }, { id: 'b', text: 'B' }])).toStrictEqual([
      { choices: [{ id: 'a', text: 'A' }, { id: 'b', text: 'B' }] },
    ])
  })

  it('is one run for each stretch naming the same shelf', () => {
    const shelves = shelved([
      { id: 'a', text: 'A', group: 'Ours' },
      { id: 'b', text: 'B', group: 'Ours' },
      { id: 'c', text: 'C', group: 'Yours' },
    ])

    expect(shelves.map((one) => one.label)).toStrictEqual(['Ours', 'Yours'])
    expect(shelves.map((one) => one.choices.map((choice) => choice.id))).toStrictEqual([
      ['a', 'b'],
      ['c'],
    ])
  })

  it('opens a run again where a shelf comes back after another', () => {
    const shelves = shelved([
      { id: 'a', text: 'A', group: 'Ours' },
      { id: 'b', text: 'B', group: 'Yours' },
      { id: 'c', text: 'C', group: 'Ours' },
    ])

    expect(shelves.map((one) => one.label)).toStrictEqual(['Ours', 'Yours', 'Ours'])
  })

  it('keeps a choice naming no shelf out of the one before it', () => {
    const shelves = shelved([
      { id: 'a', text: 'A', group: 'Ours' },
      { id: 'b', text: 'B' },
    ])

    expect(shelves.map((one) => one.label)).toStrictEqual(['Ours', undefined])
  })

  it('is nothing at all for no choices', () => {
    expect(shelved([])).toStrictEqual([])
  })
})
