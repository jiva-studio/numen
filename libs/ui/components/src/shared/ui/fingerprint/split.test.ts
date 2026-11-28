import { describe, expect, it } from 'vitest'
import { splitFingerprint } from './split'

describe('splitting a fingerprint', () => {
  it('keeps the last two groups in the tail', () => {
    expect(splitFingerprint('F9D2 40ED 59AF 4B0D 55BB 09D1 D750 C8AD')).toEqual({
      head: 'F9D2 40ED 59AF 4B0D 55BB 09D1',
      tail: ' D750 C8AD',
    })
  })

  it('puts a short value wholly in the tail', () => {
    expect(splitFingerprint('D750 C8AD')).toEqual({ head: '', tail: 'D750 C8AD' })
    expect(splitFingerprint('')).toEqual({ head: '', tail: '' })
  })
})
