/**
 * What an error of a deck or a stencil is recorded under.
 */
import { describe, expect, it } from 'vitest'
import { getFailureCode } from './failures'

describe('the code an error is recorded under', () => {
  it('is none for a file that moved past the caller', () => {
    expect(getFailureCode('changed')).toBeNull()
  })

  it('is the code of the rejection for any other', () => {
    expect(getFailureCode('notADeck')).toBe('notADeck')
  })
})
