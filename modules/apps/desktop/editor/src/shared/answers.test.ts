/**
 * The file an answer came out of, and what a refusal is called.
 *
 * The window carries the file as one string and never reads into it, so the
 * only thing that matters about it is that the parts come back out as they went
 * in — a path with spaces in it included.
 */
import { describe, expect, it } from 'vitest'
import { Refusal } from '@numen/protocol'
import { asset, fingerprint, named, REFUSAL, refusalIn, staleIn, stamp, waiting } from './answers'

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
  // The table is keyed by the schema, so the compiler asks for every refusal.
  // What it cannot ask is which of them the silence belongs to.
  it('leaves a file that moved past the caller the one refusal with no word', () => {
    expect(REFUSAL[Refusal.STALE]).toBeNull()
    expect(Object.values(REFUSAL).filter((word) => word === null)).toHaveLength(1)
  })

  it('reads a file that moved past the caller off the answer that says so', () => {
    expect(staleIn({ refusal: Refusal.STALE })).toBe(true)
    expect(staleIn({ refusal: Refusal.MISSING })).toBe(false)
    expect(staleIn({})).toBe(false)
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

describe('the address a file is asked about at', () => {
  it('escapes characters in the path', () => {
    expect(asset('notes/Maxwell’s demon.md')).toBe('/assets/notes%2FMaxwell%E2%80%99s%20demon.md')
  })
})

describe('how an address names the bytes of a file', () => {
  it('encodes the size and modification time', () => {
    expect(named('1024 1700000000000000000 notes/Doc.pdf')).toBe(
      'size=1024&mtime=1700000000000000000',
    )
  })
})

describe('waiting for an answer', () => {
  it('returns the answer when the call succeeds', async () => {
    const answer = await waiting(async () => 'ok')
    expect(answer).toBe('ok')
  })
})

