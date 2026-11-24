import { describe, expect, it, vi } from 'vitest'
import type { KeyFileApi } from '../api/keyFile'
import type { KeyFileState } from '../lib/types'
import { createKeyFileStore } from './keyFile'

const NO_KEYCHAIN: KeyFileState = {
  isInUse: true,
  mode: 'unset',
  isChoiceNeeded: true,
  isLocked: false,
  warning: 'No keychain.',
}

const createApi = (overrides: Partial<KeyFileApi> = {}): KeyFileApi => ({
  getState: () => Promise.resolve(NO_KEYCHAIN),
  choose: () => Promise.resolve({ ok: true, value: undefined }),
  unlock: () => Promise.resolve({ ok: true, value: undefined }),
  ...overrides,
})

const refused = { ok: false, error: 'not kept' } as const
const kept = { ok: true, value: 'identity' } as const

describe('createKeyFileStore', () => {
  it('returns a call that worked without asking anything', async () => {
    const store = createKeyFileStore(createApi(), vi.fn())

    const answer = await store.retryAfterAsking(() => Promise.resolve(kept))

    expect(answer).toBe(kept)
    expect(store.request.value).toBeUndefined()
  })

  it('asks for a choice when a call is refused and the file needs one, then calls again', async () => {
    const store = createKeyFileStore(createApi(), vi.fn())
    const call = vi.fn().mockResolvedValueOnce(refused).mockResolvedValueOnce(kept)

    const pending = store.retryAfterAsking(call)
    await vi.waitFor(() => expect(store.request.value).toBe('choose'))
    await store.choose('plaintext', '')
    const answer = await pending

    expect(answer).toBe(kept)
    expect(call).toHaveBeenCalledTimes(2)
    expect(store.request.value).toBeUndefined()
  })

  it('asks for the passphrase when the file is locked', async () => {
    const locked = { ...NO_KEYCHAIN, mode: 'passphrase', isChoiceNeeded: false, isLocked: true } as const
    const store = createKeyFileStore(createApi({ getState: () => Promise.resolve(locked) }), vi.fn())

    void store.retryAfterAsking(() => Promise.resolve(refused))

    await vi.waitFor(() => expect(store.request.value).toBe('unlock'))
  })

  it('returns the refusal when the person cancels', async () => {
    const store = createKeyFileStore(createApi(), vi.fn())
    const call = vi.fn().mockResolvedValue(refused)

    const pending = store.retryAfterAsking(call)
    await vi.waitFor(() => expect(store.request.value).toBe('choose'))
    store.cancel()

    expect(await pending).toBe(refused)
    expect(call).toHaveBeenCalledTimes(1)
  })

  it('returns the refusal when the file needs nothing', async () => {
    const ready = { ...NO_KEYCHAIN, isChoiceNeeded: false } as const
    const store = createKeyFileStore(createApi({ getState: () => Promise.resolve(ready) }), vi.fn())

    expect(await store.retryAfterAsking(() => Promise.resolve(refused))).toBe(refused)
    expect(store.request.value).toBeUndefined()
  })

  it('keeps the question open with a plain line when the passphrase is wrong', async () => {
    const store = createKeyFileStore(
      createApi({
        unlock: () =>
          Promise.resolve({ ok: false, error: { error: 'wrongPassphrase', message: 'wrong' } }),
      }),
      vi.fn(),
    )
    store.change()

    await store.unlock('guess')

    expect(store.problem.value).toBe('This passphrase does not open your keys. Try again.')
    expect(store.request.value).toBe('choose')
  })

  it('tells the person when the state cannot be read', async () => {
    const notify = vi.fn()
    const store = createKeyFileStore(
      createApi({ getState: () => Promise.reject(new Error('down')) }),
      notify,
    )

    expect(await store.load()).toBeUndefined()
    expect(notify).toHaveBeenCalledWith(expect.stringContaining('could not be read'), 'error')
  })
})
