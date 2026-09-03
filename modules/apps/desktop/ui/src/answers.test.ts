/**
 * The file an answer came out of, and what a refusal is called.
 *
 * The window carries the file as one string and never reads into it, so the
 * only thing that matters about it is that the parts come back out as they went
 * in — a path with spaces in it included.
 */
import { describe, expect, it } from 'vitest'
import { Refusal } from '@numen/protocol'
import { fingerprint, REFUSAL, refusalIn, stamp } from './answers'

describe('the file an answer came out of', () => {
  it('comes back out as it went in', () => {
    const at = { path: 'physics/Entropy.md', size: 1234n, mtime: 1_700_000_000n }
    expect(fingerprint(stamp(at) as string)).toEqual(at)
  })

  // A path is the one part that may hold a space, so it is what is left after
  // the two numbers rather than one field among several.
  it('comes back out whole where the path holds spaces', () => {
    const at = { path: 'heat/Maxwell’s demon and the rest.md', size: 7n, mtime: 9n }
    expect(fingerprint(stamp(at) as string)).toEqual(at)
  })

  it('is nothing where the answer named no file', () => {
    expect(stamp(undefined)).toBeUndefined()
  })

  it('is nothing at all where there is nothing to read', () => {
    expect(fingerprint('')).toEqual({ path: '', size: 0n, mtime: 0n })
  })
})

describe('what a refusal is called', () => {
  it('has a word for every refusal the schema carries', () => {
    for (const refusal of Object.values(Refusal)) {
      if (typeof refusal !== 'number') continue
      expect(REFUSAL[refusal as Refusal], String(refusal)).toBeDefined()
    }
  })

  it('reads a refusal off an answer that carries one', () => {
    expect(refusalIn({ refusal: Refusal.MISSING })).toBe('missing')
    expect(refusalIn({ refusal: Refusal.TOO_LARGE })).toBe('tooLarge')
  })

  // Nothing refused is not the same as a refusal nobody named, so an answer
  // that was not refused says nothing rather than saying it is unreadable.
  it('says nothing about an answer that was not refused', () => {
    expect(refusalIn({})).toBeNull()
    expect(refusalIn({ refusal: undefined })).toBeNull()
  })

  it('calls a refusal it has no word of its own for unreadable', () => {
    expect(refusalIn({ refusal: Refusal.UNSPECIFIED })).toBe('unreadable')
  })
})
