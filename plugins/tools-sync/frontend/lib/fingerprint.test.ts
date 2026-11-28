import { describe, expect, it } from 'vitest'
import { getShortFingerprint } from './fingerprint'

describe('getShortFingerprint', () => {
  it('keeps a fingerprint of three groups or fewer whole', () => {
    expect(getShortFingerprint('AAAA BBBB CCCC')).toBe('AAAA BBBB CCCC')
  })

  it('cuts a longer fingerprint after three groups and marks the cut', () => {
    expect(getShortFingerprint('AAAA BBBB CCCC DDDD EEEE')).toBe('AAAA BBBB CCCC…')
  })
})
