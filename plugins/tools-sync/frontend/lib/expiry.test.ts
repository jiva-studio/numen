import { describe, expect, it } from 'vitest'
import { getExpiryText } from './expiry'

describe('the moment a link stops working', () => {
  it('names the day and the hour', () => {
    const text = getExpiryText('2026-11-25T12:00:00Z')

    expect(text).toMatch(/2026/)
    expect(text).toMatch(/\d{1,2}[:.]\d{2}/)
  })
})
