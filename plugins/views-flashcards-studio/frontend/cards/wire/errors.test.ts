import { describe, expect, it } from 'vitest'
import { ErrorCode as ProtoErrorCode } from '../../gen/numen/flashcards/v1/errors_pb'
import {
  ERRORS,
  findErrorCode,
  formatErrorCodeMessage,
  getErrorWords,
  isStaleAnswer,
} from './errors'

describe('errors', () => {
  it('translates every note code of the schema into the plugin code', () => {
    expect(findErrorCode({ error: ProtoErrorCode.MISSING })).toBe('missing')
    expect(findErrorCode({ error: ProtoErrorCode.NOT_A_NOTE })).toBe('notANote')
    expect(findErrorCode({ error: ProtoErrorCode.INVALID_NAME })).toBe('invalidName')
    expect(findErrorCode({ error: ProtoErrorCode.UNSPECIFIED })).toBe('unreadable')
  })

  it('translates the codes of a card model, a deck and a preset', () => {
    expect(findErrorCode({ error: ProtoErrorCode.NOT_A_CARD_MODEL })).toBe('notACardModel')
    expect(findErrorCode({ error: ProtoErrorCode.NOT_A_DECK })).toBe('notADeck')
    expect(findErrorCode({ error: ProtoErrorCode.DECK_TOO_LARGE })).toBe('deckTooLarge')
    expect(findErrorCode({ error: ProtoErrorCode.NOT_A_PRESET })).toBe('notAPreset')
  })

  it('returns no code for a success or a stale write', () => {
    expect(findErrorCode({})).toBeNull()
    expect(findErrorCode({ error: ProtoErrorCode.STALE })).toBeNull()
    expect(isStaleAnswer({ error: ProtoErrorCode.STALE })).toBe(true)
    expect(isStaleAnswer({ error: ProtoErrorCode.MISSING })).toBe(false)
  })

  it('formats each schema code in the words of its plugin code', () => {
    expect(formatErrorCodeMessage(undefined)).toBe('')
    expect(formatErrorCodeMessage(ProtoErrorCode.MISSING)).toBe(ERRORS.missing)
    expect(formatErrorCodeMessage(ProtoErrorCode.NOT_A_DECK)).toBe(ERRORS.notADeck)
    expect(formatErrorCodeMessage(ProtoErrorCode.UNSPECIFIED)).toBe('note unreadable')
    expect(formatErrorCodeMessage(ProtoErrorCode.STALE)).toBe('not saved: the note changed on disk')
    expect(formatErrorCodeMessage(99 as ProtoErrorCode)).toBe('internal error')
  })

  it('gives words for every plugin code', () => {
    expect(getErrorWords('unreachable')).toBe('vault unreachable')
    expect(getErrorWords('notAPreset')).toBe('that note is not a preset')
  })
})
