import { describe, expect, it } from 'vitest'
import { ErrorCode as ProtoErrorCode } from '@numen/protocol'
import { formatErrorCodeMessage } from '@numen/wire'

import { ERROR_CODE } from '@/shared/answers'
import { ERRORS, getErrorWords } from './errors'

describe('the words of an error code', () => {
  it('are the words the wire gives the code of the schema', () => {
    for (const [proto, code] of Object.entries(ERROR_CODE)) {
      if (code === null || Number(proto) === ProtoErrorCode.UNSPECIFIED) continue
      expect(getErrorWords(code), code).toBe(formatErrorCodeMessage(Number(proto)))
    }
  })

  it('are said for every code the window has', () => {
    expect(Object.values(ERRORS).every((one) => one !== '')).toBe(true)
  })

  it('say that the vault could not be reached where it did not answer', () => {
    expect(getErrorWords('unreachable')).toBe('the vault could not be reached')
  })
})
