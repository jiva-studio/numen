import { describe, expect, it } from 'vitest'
import { ErrorCode as ProtoErrorCode } from '@numen/protocol'
import { formatErrorCodeMessage } from './error'

import { ERROR_CODE } from './answers'
import { ERRORS, getErrorWords, getKnownErrorCode } from './codes'

describe('the words of an error code', () => {
  it('are the words the wire gives the code of the schema', () => {
    for (const [proto, code] of Object.entries(ERROR_CODE)) {
      if (code === null || Number(proto) === ProtoErrorCode.UNSPECIFIED) continue
      expect(getErrorWords(code), code).toBe(formatErrorCodeMessage(Number(proto)))
    }
  })

  it('are returned for every code the window has', () => {
    expect(Object.values(ERRORS).every((one) => one !== '')).toBe(true)
  })

  it('say that the vault could not be reached where it did not answer', () => {
    expect(getErrorWords('unreachable')).toBe('the vault could not be reached')
  })
})

describe('a code given as text', () => {
  it('is the code the window words, where it words it', () => {
    expect(getKnownErrorCode('occupied')).toBe('occupied')
    expect(getKnownErrorCode('unreachable')).toBe('unreachable')
  })

  it('is unreadable where the window has no words for it', () => {
    expect(getKnownErrorCode('changed')).toBe('unreadable')
    expect(getKnownErrorCode('')).toBe('unreadable')
  })
})
