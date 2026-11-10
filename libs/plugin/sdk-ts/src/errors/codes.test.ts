import { describe, expect, it } from 'vitest'
import { ErrorCode as ProtoErrorCode } from '@numen/protocol'
import { formatErrorCodeMessage } from './error'
import { getRelationErrorWords } from './codes'

const getErrorWords = (code: Parameters<typeof getRelationErrorWords>[0]): string =>
  getRelationErrorWords(code)

describe('the words of an error code', () => {
  it('are the words the wire gives the code of the schema', () => {
    expect(getErrorWords('notFound')).toBe(formatErrorCodeMessage(ProtoErrorCode.NOT_FOUND))
    expect(getErrorWords('notFound')).toBe('note not in the vault')
    expect(getErrorWords('notANote')).toBe(formatErrorCodeMessage(ProtoErrorCode.NOT_A_NOTE))
    expect(getErrorWords('notText')).toBe(formatErrorCodeMessage(ProtoErrorCode.NOT_TEXT))
    expect(getErrorWords('tooLarge')).toBe(formatErrorCodeMessage(ProtoErrorCode.TOO_LARGE))
    expect(getErrorWords('bodyUnwritable')).toBe(
      formatErrorCodeMessage(ProtoErrorCode.BODY_UNWRITABLE),
    )
    expect(getErrorWords('unreadable')).toBe(formatErrorCodeMessage(ProtoErrorCode.UNREADABLE))
    expect(getErrorWords('alreadyExists')).toBe(
      formatErrorCodeMessage(ProtoErrorCode.ALREADY_EXISTS),
    )
    expect(getErrorWords('invalidName')).toBe(formatErrorCodeMessage(ProtoErrorCode.INVALID_NAME))
  })

  it('say that the vault could not be reached where it did not answer', () => {
    expect(getErrorWords('unreachable')).toBe('vault unreachable')
  })
})

describe('the words of a relationship error', () => {
  it('word the error of a relation, and fall back on the words of an error code', () => {
    expect(getRelationErrorWords('cycle')).toBe('a note cannot sit above itself in the hierarchy')
    expect(getRelationErrorWords('notFound')).toBe(getErrorWords('notFound'))
  })
})
