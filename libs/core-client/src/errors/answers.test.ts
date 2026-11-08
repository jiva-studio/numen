/**
 * The file an answer came out of, and what an error is called.
 *
 * The window carries the file as one string and never reads into it, so the
 * only thing that matters about it is that the parts are returned as they went
 * in — a path with spaces in it included.
 */
import { describe, expect, it } from 'vitest'
import { NOTE_ERROR_CODE } from '../clients'
import { findErrorCode, isStaleAnswer, parseFingerprint } from './answers'

const formatFingerprint = (stat: { path: string; size: bigint; mtime: bigint }): string =>
  `${stat.size} ${stat.mtime} ${stat.path}`

describe('the file an answer came out of', () => {
  it('arrives back out as it went in', () => {
    const stat = { path: 'physics/Entropy.md', size: 1234n, mtime: 1_700_000_000n }
    expect(parseFingerprint(formatFingerprint(stat))).toEqual(stat)
  })

  // A path is the one part that may hold a space, so it is what is left after
  // the two numbers and not one field among several.
  it('arrives back out whole where the path holds spaces', () => {
    const stat = { path: 'heat/Maxwell’s demon and the rest.md', size: 7n, mtime: 9n }
    expect(parseFingerprint(formatFingerprint(stat))).toEqual(stat)
  })

  it('is nothing at all where there is nothing to read', () => {
    expect(parseFingerprint('')).toEqual({ path: '', size: 0n, mtime: 0n })
  })
})

describe('what an error is called', () => {
  it('reads a file that moved past the caller off the answer that says so', () => {
    expect(isStaleAnswer({ error: NOTE_ERROR_CODE.CONFLICT })).toBe(true)
    expect(isStaleAnswer({ error: NOTE_ERROR_CODE.NOT_FOUND })).toBe(false)
    expect(isStaleAnswer({})).toBe(false)
  })

  it('leaves a file that moved past the caller with no word', () => {
    expect(findErrorCode({ error: NOTE_ERROR_CODE.CONFLICT })).toBeNull()
  })

  it('reads an error off an answer that carries one', () => {
    expect(findErrorCode({ error: NOTE_ERROR_CODE.NOT_FOUND })).toBe('notFound')
    expect(findErrorCode({ error: NOTE_ERROR_CODE.TOO_LARGE })).toBe('tooLarge')
  })

  it('says nothing about an answer with no error', () => {
    expect(findErrorCode({})).toBeNull()
    expect(findErrorCode({ error: undefined })).toBeNull()
  })

  it('calls an error it has no word of its own for unreadable', () => {
    expect(findErrorCode({ error: NOTE_ERROR_CODE.UNSPECIFIED })).toBe('unreadable')
  })
})
