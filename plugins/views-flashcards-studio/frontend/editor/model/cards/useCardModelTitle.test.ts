import { describe, expect, it } from 'vitest'
import { useCardModelTitle } from './useCardModelTitle'

const sayUnknown = (cardModel: string | null): string => `No card model called ${cardModel}`

describe('the title of the card model of a card', () => {
  it('is the name of the card model', () => {
    expect(
      useCardModelTitle(
        () => 'Basic',
        () => sayUnknown,
      ).value,
    ).toBe('Basic')
  })

  it('is what the words say where the card is cut by none', () => {
    expect(
      useCardModelTitle(
        () => null,
        () => sayUnknown,
      ).value,
    ).toBe('No card model called null')
  })
})
