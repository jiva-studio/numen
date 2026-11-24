import { describe, expect, it } from 'vitest'
import { getCodeFromLink, getSharingLink } from './sharingLink'

describe('the sharing link', () => {
  it('wraps an identity code and gives it back', () => {
    expect(getCodeFromLink(getSharingLink('numen-reader-1.abc'))).toBe('numen-reader-1.abc')
  })

  it('takes a bare code and surrounding space as they are', () => {
    expect(getCodeFromLink('  numen-reader-1.abc ')).toBe('numen-reader-1.abc')
  })
})
