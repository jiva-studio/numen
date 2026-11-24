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
  `${stat.size}:0: ${stat.mtime} ${stat.path}`

const NO_CONTENT = { hash: '', version: 0n }

describe('the file an answer came out of', () => {
  it('arrives back out as it went in', () => {
    const stat = { path: 'physics/Entropy.md', size: 1234n, mtime: 1_700_000_000n }
    expect(parseFingerprint(formatFingerprint(stat))).toEqual({ ...stat, ...NO_CONTENT })
  })

  it('arrives back out with the hash and the version of the vault it carried', () => {
    const hash = 'a'.repeat(64)
    expect(parseFingerprint(`1234:7:${hash} 1700000000 physics/Entropy Notes.md`)).toEqual({
      path: 'physics/Entropy Notes.md',
      size: 1234n,
      mtime: 1_700_000_000n,
      hash,
      version: 7n,
    })
  })

  // A path is the one part that may hold a space, so it is what is left after
  // the two numbers and not one field among several.
  it('arrives back out whole where the path holds spaces', () => {
    const stat = { path: 'heat/Maxwell’s demon and the rest.md', size: 7n, mtime: 9n }
    expect(parseFingerprint(formatFingerprint(stat))).toEqual({ ...stat, ...NO_CONTENT })
  })

  it('is an error where the string is not a fingerprint', () => {
    expect(() => parseFingerprint('')).toThrow()
    expect(() => parseFingerprint('1234 1700000000 physics/Entropy.md')).toThrow()
    expect(() => parseFingerprint('1234:0: 1700000000 ')).toThrow()
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

  it('has no error for an answer that carries none', () => {
    expect(findErrorCode({ error: NOTE_ERROR_CODE.UNSPECIFIED })).toBeNull()
  })
})

describe('what the codes of a file on a phone are called', () => {
  it('words a file with no bytes here as one that is not found', () => {
    expect(findErrorCode({ error: NOTE_ERROR_CODE.FILE_ABSENT })).toBe('notFound')
  })

  it('words a different file at the path as one that exists', () => {
    expect(findErrorCode({ error: NOTE_ERROR_CODE.FILE_PRESENT })).toBe('alreadyExists')
  })

  it('words bytes of another hash as unreadable', () => {
    expect(findErrorCode({ error: NOTE_ERROR_CODE.HASH_MISMATCH })).toBe('unreadable')
  })

  it('leaves a file with changes of its own with no word', () => {
    expect(findErrorCode({ error: NOTE_ERROR_CODE.CHANGED })).toBeNull()
  })
})
