/**
 * The crossing of a card and a deck between the shapes the window carries and
 * the shapes the schema holds.
 */
import { describe, expect, it } from 'vitest'
import type { Deck as DeckMessage } from '@numen/protocol'
import type { VaultCard } from '../types'
import { deserializeDeck, serializeCardToWire } from './serialize'

/** A card as the window carries it, standing under the section given. */
const cardUnder = (sectionIndex: number | null): VaultCard => ({
  mark: 'a1',
  sectionIndex,
  heading: 'Entropy',
  stencilLink: 'Word',
  stencilPath: 'Word.md',
  preamble: 'a note',
  values: [{ field: 'Front', text: 'Entropy' }],
})

describe('a card on its way out', () => {
  it('names the section it stands under by its number, zero included', () => {
    expect(serializeCardToWire(cardUnder(0))).toHaveProperty('sectionIndex', 0)
    expect(serializeCardToWire(cardUnder(2))).toHaveProperty('sectionIndex', 2)
  })

  it('names no section where it stands under none', () => {
    expect(serializeCardToWire(cardUnder(null))).not.toHaveProperty('sectionIndex')
  })

  it('carries its values as field and text only', () => {
    const card: VaultCard = {
      ...cardUnder(null),
      values: [{ field: 'Front', text: 'Entropy', extra: 1 } as VaultCard['values'][number]],
    }

    expect(serializeCardToWire(card).values).toStrictEqual([{ field: 'Front', text: 'Entropy' }])
  })
})

describe('a deck on its way in', () => {
  it('carries its sections as name and preamble only', () => {
    const message = {
      path: 'Deck.md',
      title: 'Deck',
      preamble: '',
      cards: [],
      sections: [{ name: 'Physics', preamble: 'about', $typeName: 'numen.v1.Section', extra: 1 }],
      tail: '',
      problems: [],
    } as unknown as DeckMessage

    expect(deserializeDeck(message).sections).toStrictEqual([
      { name: 'Physics', preamble: 'about' },
    ])
  })
})
