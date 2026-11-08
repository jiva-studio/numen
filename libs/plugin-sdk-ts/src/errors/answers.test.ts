/**
 * The file an answer came out of.
 *
 * The window carries the file as one string and never reads into it, so the
 * only thing that matters about it is that the parts are returned as they went
 * in — a path with spaces in it included.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { Code, ConnectError } from '@connectrpc/connect'
import {
  getAssetUrl,
  getBytesQuery,
  retryWhileBusy,
  formatFingerprint,
  MAX_BUSY_RETRIES,
} from './answers'

describe('the file an answer came out of', () => {
  it('arrives back out as the size and modification time it went in with', () => {
    const stat = { path: 'physics/Entropy.md', size: 1234n, mtime: 1_700_000_000n }
    expect(getBytesQuery(formatFingerprint(stat) as string)).toBe('size=1234&mtime=1700000000')
  })

  // A path is the one part that may hold a space, so it is what is left after
  // the two numbers and not one field among several.
  it('arrives back out whole where the path holds spaces', () => {
    const stat = { path: 'heat/Maxwell’s demon and the rest.md', size: 7n, mtime: 9n }
    expect(getBytesQuery(formatFingerprint(stat) as string)).toBe('size=7&mtime=9')
  })

  it('is nothing where the answer named no file', () => {
    expect(formatFingerprint(undefined)).toBeUndefined()
  })

  it('is nothing at all where there is nothing to read', () => {
    expect(getBytesQuery('')).toBe('size=0&mtime=0')
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
