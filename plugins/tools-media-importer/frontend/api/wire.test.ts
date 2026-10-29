import { describe, expect, it } from 'vitest'
import { describeError } from './wire'
import { ErrorCode } from './importer'
import { WORDS as words } from '../words'

describe('describeError', () => {
  it('reads not_found as a note without a web address', () => {
    expect(describeError({ code: ErrorCode.notFound, message: 'not a link' })).toBe(words.notALink)
  })

  it('reads failed_precondition as no vault', () => {
    expect(describeError({ code: ErrorCode.failedPrecondition, message: 'no vault' })).toBe(
      words.noVault,
    )
  })

  it('reads any other code as a failure', () => {
    expect(describeError({ code: 'internal', message: 'disk' })).toBe(words.failed)
  })
})
