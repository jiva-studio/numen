import { describe, expect, it } from 'vitest'
import { getDeviceFingerprint, readFingerprint } from './link.js'

describe('getDeviceFingerprint', () => {
  it('writes the first 16 bytes of the hash of the label and the key in groups of four digits', () => {
    expect(getDeviceFingerprint('00'.repeat(32))).toBe('11E0 1148 C9A3 4C32 1D8E A6C2 D982 416A')
  })

  it('differs for another key', () => {
    expect(getDeviceFingerprint('01'.repeat(32))).not.toBe(getDeviceFingerprint('00'.repeat(32)))
  })
})

describe('readFingerprint', () => {
  it('reads eight groups of four hexadecimal digits', () => {
    expect(readFingerprint('Device 11E0 1148 C9A3 4C32 1D8E A6C2 D982 416A added')).toBe(
      '11E0 1148 C9A3 4C32 1D8E A6C2 D982 416A',
    )
  })

  it('refuses a text with seven groups', () => {
    expect(() => readFingerprint('11E0 1148 C9A3 4C32 1D8E A6C2 D982')).toThrow('no fingerprint')
  })
})
