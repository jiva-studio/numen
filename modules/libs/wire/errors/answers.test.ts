/**
 * The file an answer came out of, and what an error is called.
 *
 * The window carries the file as one string and never reads into it, so the
 * only thing that matters about it is that the parts come back out as they went
 * in — a path with spaces in it included.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { Code, ConnectError } from '@connectrpc/connect'
import { ErrorCode as ProtoErrorCode } from '@numen/protocol'
import {
  getAssetUrl,
  ERROR_CODE,
  findErrorCode,
  parseFingerprint,
  getBytesQuery,
  retryWhileBusy,
  isStaleAnswer,
  formatFingerprint,
  MAX_BUSY_RETRIES,
} from './answers'

describe('the file an answer came out of', () => {
  it('arrives back out as it went in', () => {
    const fingerprint = { path: 'physics/Entropy.md', size: 1234n, mtime: 1_700_000_000n }
    expect(parseFingerprint(formatFingerprint(fingerprint) as string)).toEqual(fingerprint)
  })

  // A path is the one part that may hold a space, so it is what is left after
  // the two numbers and not one field among several.
  it('arrives back out whole where the path holds spaces', () => {
    const fingerprint = { path: 'heat/Maxwell’s demon and the rest.md', size: 7n, mtime: 9n }
    expect(parseFingerprint(formatFingerprint(fingerprint) as string)).toEqual(fingerprint)
  })

  it('is nothing where the answer named no file', () => {
    expect(formatFingerprint(undefined)).toBeUndefined()
  })

  it('is nothing at all where there is nothing to read', () => {
    expect(parseFingerprint('')).toEqual({ path: '', size: 0n, mtime: 0n })
  })
})

describe('what an error is called', () => {
  // The table is keyed by the schema, so the compiler asks for every error code.
  // What it cannot ask is which of them the silence belongs to.
  it('leaves a file that moved past the caller the one error with no word', () => {
    expect(ERROR_CODE[ProtoErrorCode.STALE]).toBeNull()
    expect(Object.values(ERROR_CODE).filter((word) => word === null)).toHaveLength(1)
  })

  it('reads a file that moved past the caller off the answer that says so', () => {
    expect(isStaleAnswer({ error: ProtoErrorCode.STALE })).toBe(true)
    expect(isStaleAnswer({ error: ProtoErrorCode.MISSING })).toBe(false)
    expect(isStaleAnswer({})).toBe(false)
  })

  it('reads an error off an answer that carries one', () => {
    expect(findErrorCode({ error: ProtoErrorCode.MISSING })).toBe('missing')
    expect(findErrorCode({ error: ProtoErrorCode.TOO_LARGE })).toBe('tooLarge')
  })

  it('says nothing about an answer with no error', () => {
    expect(findErrorCode({})).toBeNull()
    expect(findErrorCode({ error: undefined })).toBeNull()
  })

  it('calls an error it has no word of its own for unreadable', () => {
    expect(findErrorCode({ error: ProtoErrorCode.UNSPECIFIED })).toBe('unreadable')
  })
})

describe('the address a file is requested about at', () => {
  it('escapes characters in the path', () => {
    expect(getAssetUrl('notes/Maxwell’s demon.md')).toBe(
      '/assets/notes%2FMaxwell%E2%80%99s%20demon.md',
    )
  })
})

describe('how an address names the bytes of a file', () => {
  it('encodes the size and modification time', () => {
    expect(getBytesQuery('1024 1700000000000000000 notes/Doc.pdf')).toBe(
      'size=1024&mtime=1700000000000000000',
    )
  })
})

describe('waiting for an answer', () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  const busy = () => new ConnectError('the document is busy', Code.Unavailable)

  it('asks again after a second where the document is busy', async () => {
    vi.useFakeTimers()
    const ask = vi.fn<() => Promise<string>>()
    ask.mockRejectedValueOnce(busy()).mockResolvedValueOnce('ok')

    const answer = retryWhileBusy(ask)
    await vi.advanceTimersByTimeAsync(999)
    expect(ask).toHaveBeenCalledTimes(1)
    await vi.advanceTimersByTimeAsync(1)

    await expect(answer).resolves.toBe('ok')
    expect(ask).toHaveBeenCalledTimes(2)
  })

  it('gives up on a document that stays busy after three more asks', async () => {
    vi.useFakeTimers()
    const error = busy()
    const ask = vi.fn<() => Promise<string>>(() => Promise.reject(error))

    const answer = retryWhileBusy(ask)
    const rejection = expect(answer).rejects.toBe(error)
    await vi.advanceTimersByTimeAsync(10_000)

    await rejection
    expect(ask).toHaveBeenCalledTimes(MAX_BUSY_RETRIES + 1)
  })

  it('defines the maximum retry limit as 3', () => {
    expect(MAX_BUSY_RETRIES).toBe(3)
  })

  it('does not ask again after any other error', async () => {
    const error = new ConnectError('no such note', Code.NotFound)
    const ask = vi.fn<() => Promise<string>>(() => Promise.reject(error))

    await expect(retryWhileBusy(ask)).rejects.toBe(error)
    expect(ask).toHaveBeenCalledTimes(1)
  })

  it('returns the answer when the call succeeds', async () => {
    const answer = await retryWhileBusy(async () => 'ok')
    expect(answer).toBe('ok')
  })
})
