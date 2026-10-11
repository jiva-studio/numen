import { describe, expect, it } from 'vitest'

import { isAdult } from './age'

describe('the canary', () => {
  describe('an adult', () => {
    it('is eighteen or older', () => {
      expect(isAdult(17)).toBe(false)
      expect(isAdult(18)).toBe(true)
      expect(isAdult(40)).toBe(true)
    })
  })
})
