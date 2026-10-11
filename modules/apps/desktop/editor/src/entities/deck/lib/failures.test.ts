/**
 * What a refusal of a deck or a stencil is recorded under.
 */
import { describe, expect, it } from 'vitest'
import { getFailureCode } from './failures'

describe('the code a refusal is recorded under', () => {
  it('is none for a file that moved past the caller', () => {
    expect(getFailureCode('changed')).toBeNull()
  })

  it('is the code of the refusal for any other', () => {
    expect(getFailureCode('notADeck')).toBe('notADeck')
  })
})
