import { describe, expect, it } from 'vitest'
import { NOTE_ERROR_CODE } from '../clients'
import { formatErrorCodeMessage } from './error'
import { ERRORS, getKnownErrorCode } from './codes'

describe('the words of an error code', () => {
  it('are the words the wire gives the code of the schema', () => {
    expect(ERRORS.notFound).toBe(formatErrorCodeMessage(NOTE_ERROR_CODE.NOT_FOUND))
    expect(ERRORS.notANote).toBe(formatErrorCodeMessage(NOTE_ERROR_CODE.NOT_A_NOTE))
    expect(ERRORS.notText).toBe(formatErrorCodeMessage(NOTE_ERROR_CODE.NOT_TEXT))
    expect(ERRORS.tooLarge).toBe(formatErrorCodeMessage(NOTE_ERROR_CODE.TOO_LARGE))
    expect(ERRORS.bodyUnwritable).toBe(formatErrorCodeMessage(NOTE_ERROR_CODE.BODY_UNWRITABLE))
    expect(ERRORS.unreadable).toBe(formatErrorCodeMessage(NOTE_ERROR_CODE.UNREADABLE))
    expect(ERRORS.alreadyExists).toBe(formatErrorCodeMessage(NOTE_ERROR_CODE.ALREADY_EXISTS))
    expect(ERRORS.invalidName).toBe(formatErrorCodeMessage(NOTE_ERROR_CODE.INVALID_NAME))
  })

  it('are returned for every code the window has', () => {
    expect(Object.values(ERRORS).every((one) => one !== '')).toBe(true)
  })

  it('say that the vault could not be reached where it did not answer', () => {
    expect(ERRORS.unreachable).toBe('vault unreachable')
  })
})

describe('a code given as text', () => {
  it('is the code the window words, where it words it', () => {
    expect(getKnownErrorCode('alreadyExists')).toBe('alreadyExists')
    expect(getKnownErrorCode('unreachable')).toBe('unreachable')
  })

  it('is unreadable where the window has no words for it', () => {
    expect(getKnownErrorCode('changed')).toBe('unreadable')
    expect(getKnownErrorCode('')).toBe('unreadable')
  })
})
