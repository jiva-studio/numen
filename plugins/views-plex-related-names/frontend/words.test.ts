/** What a name is called to assistive technology: its title and how it is related to its note. */
import { describe, expect, it } from 'vitest'
import { WORDS } from './words'

describe('WORDS.nameLabel', () => {
  it('calls a parent a parent of the note', () => {
    expect(WORDS.nameLabel('Orchard plan', 'Pear rows', 'parent')).toBe(
      'Orchard plan, parent of Pear rows',
    )
  })

  it('calls a child a child of the note', () => {
    expect(WORDS.nameLabel('Rootstock list', 'Pear rows', 'child')).toBe(
      'Rootstock list, child of Pear rows',
    )
  })

  it('calls a jump a jump from the note', () => {
    expect(WORDS.nameLabel('Grafting', 'Pear rows', 'jump')).toBe('Grafting, jump from Pear rows')
  })

  it('calls a note the note links to referenced by the note', () => {
    expect(WORDS.nameLabel('Pollination', 'Pear rows', 'reference', 'outgoing')).toBe(
      'Pollination, referenced by Pear rows',
    )
  })

  it('calls a note that links to the note a reference to the note', () => {
    expect(WORDS.nameLabel('Quince', 'Pear rows', 'reference', 'incoming')).toBe(
      'Quince, reference to Pear rows',
    )
  })

  it('calls a note linked both ways linked both ways with the note', () => {
    expect(WORDS.nameLabel('Bees', 'Pear rows', 'reference', 'both')).toBe(
      'Bees, linked both ways with Pear rows',
    )
  })

  it('calls a reference with no direction linked both ways', () => {
    expect(WORDS.nameLabel('Bees', 'Pear rows', 'reference')).toBe(
      'Bees, linked both ways with Pear rows',
    )
  })
})
