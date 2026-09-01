import { describe, expect, it } from 'vitest'
import { Refusal } from '@numen/protocol'

import { said } from './core'
import { REFUSED } from './words'

describe('why a note has no text', () => {
  // A panel of titles with no prose under any of them says nothing about why.
  it('is said in words a person reads, for every refusal the schema has', () => {
    for (const refusal of Object.values(Refusal)) {
      if (typeof refusal !== 'number') continue
      const words = said(refusal)
      expect(words, `refusal ${refusal}`).toBeTruthy()
      expect(Object.values(REFUSED)).toContain(words)
    }
  })

  it('is nothing where the text is here', () => {
    expect(said(undefined)).toBe('')
  })

  it('names the two a person meets most', () => {
    expect(said(Refusal.MISSING)).toBe(REFUSED.missing)
    expect(said(Refusal.TOO_LARGE)).toBe(REFUSED.tooLarge)
  })

  // The file is a note. What it is not is a preset, and a person told the one
  // goes looking for the other.
  it('says a note that is not a preset is not a preset', () => {
    expect(said(Refusal.NOT_A_PRESET)).toBe('that note is not a preset')
  })
})
