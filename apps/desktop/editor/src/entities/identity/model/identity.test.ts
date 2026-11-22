import { describe, expect, it, vi } from 'vitest'
import type { IdentityClient } from '@numen/core-client'
import type { Result } from '@numen/result'
import type { IdentityFailure, IdentityInfo } from '../lib/types'
import { createIdentityApi } from '../api/identity'
import { IDENTITY_WORDS } from '../words'
import { createIdentityStore } from './identity'

const asValue = <T>(value: T): Result<T, IdentityFailure> => ({ ok: true, value })
const asFailure = (error: IdentityFailure): Result<never, IdentityFailure> => ({ ok: false, error })

const WORDS = ['legal', 'winner', 'thank', 'year', 'wave', 'sausage', 'worth', 'useful', 'legal', 'winner', 'thank', 'yellow']

const createSampleIdentity = (overrides: Partial<IdentityInfo> = {}): IdentityInfo => ({
  accountFingerprint: '3F2A 91C0 0000 1111 2222 3333 4444 5555',
  deviceFingerprint: 'AAAA BBBB CCCC DDDD 0000 1111 2222 3333',
  deviceName: 'Laptop',
  deviceAddedAt: '2026-11-22T12:00:00Z',
  ...overrides,
})

const createClient = (overrides: Partial<IdentityClient> = {}): IdentityClient => ({
  get: () => Promise.resolve(undefined),
  generateWords: () => Promise.resolve(WORDS),
  create: () => Promise.resolve(asValue(createSampleIdentity())),
  delete: () => Promise.resolve(),
  ...overrides,
})

const createStore = (client: IdentityClient) => {
  const notify = vi.fn()
  return { notify, store: createIdentityStore(createIdentityApi(client), notify) }
}

describe('the identity store', () => {
  it('holds no identity before the core is asked', () => {
    const { store } = createStore(createClient())
    expect(store.identity.value).toBeUndefined()
  })

  it('loads the identity of this device', async () => {
    const { store } = createStore(createClient({ get: () => Promise.resolve(createSampleIdentity()) }))
    await store.load()
    expect(store.identity.value).toStrictEqual(createSampleIdentity())
  })

  it('holds no identity when the core has none', async () => {
    const { store } = createStore(createClient())
    await store.load()
    expect(store.identity.value).toBeUndefined()
  })

  it('keeps what it held and tells the person when a load fails', async () => {
    let fails = false
    const { store, notify } = createStore(
      createClient({
        get: () => (fails ? Promise.reject(new Error('the core is gone')) : Promise.resolve(createSampleIdentity())),
      }),
    )
    await store.load()
    fails = true
    await store.load()
    expect(store.identity.value).toStrictEqual(createSampleIdentity())
    expect(notify).toHaveBeenLastCalledWith(expect.stringContaining('the core is gone'), 'error')
  })

  it('gives the twelve words and keeps no identity', async () => {
    const { store } = createStore(createClient())
    const answer = await store.generateWords()
    expect(answer).toStrictEqual({ ok: true, value: WORDS })
    expect(store.identity.value).toBeUndefined()
  })

  it('reads a failed generation as a text', async () => {
    const { store } = createStore(createClient({ generateWords: () => Promise.reject(new Error('no random')) }))
    const answer = await store.generateWords()
    expect(answer.ok).toBe(false)
    expect(!answer.ok && answer.error).toContain('no random')
  })

  it('holds the identity after it is created with the words', async () => {
    const create = vi.fn(() => Promise.resolve(asValue(createSampleIdentity())))
    const { store } = createStore(createClient({ create }))
    const answer = await store.create(WORDS)
    expect(answer).toStrictEqual({ ok: true, value: createSampleIdentity() })
    expect(create).toHaveBeenCalledExactlyOnceWith(WORDS)
    expect(store.identity.value).toStrictEqual(createSampleIdentity())
  })

  it('says that the words are not valid and holds no identity', async () => {
    const { store } = createStore(
      createClient({
        create: () => Promise.resolve(asFailure({ errorCode: 'invalid_words', detail: 'word 4 is not a recovery word' })),
      }),
    )
    const answer = await store.create(WORDS)
    expect(answer).toStrictEqual({ ok: false, error: 'These are words that are not valid.' })
    expect(store.identity.value).toBeUndefined()
  })

  it('keeps the identity that exists when a second one is refused', async () => {
    const { store } = createStore(
      createClient({
        get: () => Promise.resolve(createSampleIdentity()),
        create: () => Promise.resolve(asFailure({ errorCode: 'identity_exists', detail: '' })),
      }),
    )
    await store.load()
    const answer = await store.create(WORDS)
    expect(answer.ok).toBe(false)
    expect(store.identity.value).toStrictEqual(createSampleIdentity())
  })

  it('describes each failure with its own words', () => {
    const texts = (['invalid_words', 'identity_exists', 'key_store_failed'] as const).map((errorCode) =>
      IDENTITY_WORDS.describeFailure({ errorCode, detail: '' }),
    )
    expect(new Set(texts).size).toBe(3)
    expect(texts.every((text) => text !== '')).toBe(true)
    expect(texts[0]).toBe('These are words that are not valid.')
  })

  it('reads a failed create as a text and holds no identity', async () => {
    const { store } = createStore(createClient({ create: () => Promise.reject(new Error('offline')) }))
    const answer = await store.create(WORDS)
    expect(answer.ok).toBe(false)
    expect(store.identity.value).toBeUndefined()
  })

  it('holds no identity after it is deleted', async () => {
    const remove = vi.fn(() => Promise.resolve())
    const { store } = createStore(createClient({ get: () => Promise.resolve(createSampleIdentity()), delete: remove }))
    await store.load()
    const answer = await store.delete()
    expect(answer.ok).toBe(true)
    expect(remove).toHaveBeenCalledOnce()
    expect(store.identity.value).toBeUndefined()
  })

  it('keeps the identity when the delete fails', async () => {
    const { store } = createStore(
      createClient({ get: () => Promise.resolve(createSampleIdentity()), delete: () => Promise.reject(new Error('refused')) }),
    )
    await store.load()
    const answer = await store.delete()
    expect(answer.ok).toBe(false)
    expect(store.identity.value).toStrictEqual(createSampleIdentity())
  })
})
