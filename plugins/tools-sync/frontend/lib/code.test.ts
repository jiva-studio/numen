import { describe, expect, it } from 'vitest'
import { getCodeText } from './code'

describe('the confirmation code', () => {
  it('reads in two groups of three digits', () => {
    expect(getCodeText('482913')).toBe('482 913')
  })

  it('keeps a code that is not six digits as it is', () => {
    expect(getCodeText('')).toBe('')
  })
})
