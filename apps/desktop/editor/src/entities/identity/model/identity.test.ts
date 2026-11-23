import { afterEach, describe, expect, it, vi } from 'vitest'
import type { IdentityClient } from '@numen/core-client'
import type { Result } from '@numen/result'
import type { IdentityFailure, IdentityInfo } from '../lib/types'
import { IDENTITY_WORDS } from '../words'
import { createIdentityStore } from './identity'
import { wrapIdentityClient } from '@/testing/identityClient'

const asValue = <T>(value: T): Result<T, IdentityFailure> => ({ ok: true, value })
const asFailure = (error: IdentityFailure): Result<never, IdentityFailure> => ({ ok: false, error })
const asStarted = (): Result<undefined, IdentityFailure> => ({ ok: true, value: undefined })
const asRefusal = (error: IdentityFailure['error']): Result<never, IdentityFailure> =>
  asFailure({ error, message: 'refused' })

const WORDS = [
  'legal',
  'winner',
  'thank',
  'year',
  'wave',
  'sausage',
  'worth',
  'useful',
  'legal',
  'winner',
  'thank',
  'yellow',
]

const createSampleIdentity = (overrides: Partial<IdentityInfo> = {}): IdentityInfo => ({
  accountFingerprint: '3F2A 91C0 0000 1111 2222 3333 4444 5555',
  deviceFingerprint: 'AAAA BBBB CCCC DDDD 0000 1111 2222 3333',
  deviceName: 'Laptop',
  deviceAddedAt: '2026-11-22T12:00:00Z',
  isRevoked: false,
  keyEpoch: 0,
  ...overrides,
})

const createClient = (overrides: Partial<IdentityClient> = {}): IdentityClient => ({
  get: () => Promise.resolve(undefined),
  generateWords: () => Promise.resolve(WORDS),
  create: () => Promise.resolve(asValue(createSampleIdentity())),
  startRestore: () => Promise.resolve(asStarted()),
  delete: () => Promise.resolve(),
  saveKit: () => Promise.resolve(asValue(true)),
  startRestoreFromKit: () => Promise.resolve(asStarted()),
  startLink: () => Promise.resolve(asRefusal('noLink')),
  getLink: () => Promise.resolve(asRefusal('noLink')),
  decideLink: () => Promise.resolve(asStarted()),
  enterLinkCode: () => Promise.resolve(asStarted()),
  confirmJoin: () => Promise.resolve(asStarted()),
  cancelLink: () => Promise.resolve(asStarted()),
  listDevices: () => Promise.resolve(asValue([])),
  revokeDevice: () => Promise.resolve(asStarted()),
  ...overrides,
})

const createStore = (client: IdentityClient) => {
  const notify = vi.fn()
  return { notify, store: createIdentityStore(wrapIdentityClient(client), notify) }
}

describe('the identity store', () => {
  it('has no identity before the first load', () => {
    const { store } = createStore(createClient())
    expect(store.identity.value).toBeUndefined()
  })

  it('loads the identity of this device', async () => {
    const { store } = createStore(
      createClient({ get: () => Promise.resolve(createSampleIdentity()) }),
    )
    await store.load()
    expect(store.identity.value).toStrictEqual(createSampleIdentity())
  })

  it('holds no identity when the core has none', async () => {
    const { store } = createStore(createClient())
    await store.load()
    expect(store.identity.value).toBeUndefined()
  })

  it('retains the loaded identity and exposes the error when a load fails', async () => {
    let fails = false
    const { store, notify } = createStore(
      createClient({
        get: () =>
          fails
            ? Promise.reject(new Error('the core is gone'))
            : Promise.resolve(createSampleIdentity()),
      }),
    )
    await store.load()
    fails = true
    await store.load()
    expect(store.identity.value).toStrictEqual(createSampleIdentity())
    expect(notify).toHaveBeenLastCalledWith(
      expect.stringContaining('Numen is not responding'),
      'error',
    )
  })

  it('gives the twelve words and keeps no identity', async () => {
    const { store } = createStore(createClient())
    const answer = await store.generateWords()
    expect(answer).toStrictEqual({ ok: true, value: WORDS })
    expect(store.identity.value).toBeUndefined()
  })

  it('reads a failed generation as a text', async () => {
    const { store } = createStore(
      createClient({ generateWords: () => Promise.reject(new Error('no random')) }),
    )
    const answer = await store.generateWords()
    expect(answer.ok).toBe(false)
    expect(!answer.ok && answer.error).toContain('Numen is not responding')
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
        create: () =>
          Promise.resolve(
            asFailure({ error: 'invalidWords', message: 'word 4 is not a recovery word' }),
          ),
      }),
    )
    const answer = await store.create(WORDS)
    expect(answer).toStrictEqual({ ok: false, error: 'Some words are not valid.' })
    expect(store.identity.value).toBeUndefined()
  })

  it('keeps the identity that exists when a second one is refused', async () => {
    const { store } = createStore(
      createClient({
        get: () => Promise.resolve(createSampleIdentity()),
        create: () => Promise.resolve(asFailure({ error: 'identityExists', message: '' })),
      }),
    )
    await store.load()
    const answer = await store.create(WORDS)
    expect(answer.ok).toBe(false)
    expect(store.identity.value).toStrictEqual(createSampleIdentity())
  })

  it('describes each failure with its own words', () => {
    const texts = (['invalidWords', 'identityExists', 'keyStoreFailed'] as const).map((error) =>
      IDENTITY_WORDS.describeFailure({ error, message: '' }),
    )
    expect(new Set(texts).size).toBe(3)
    expect(texts.every((text) => text !== '')).toBe(true)
    expect(texts[0]).toBe('Some words are not valid.')
  })

  it('reads a failed create as a text and holds no identity', async () => {
    const { store } = createStore(
      createClient({ create: () => Promise.reject(new Error('offline')) }),
    )
    const answer = await store.create(WORDS)
    expect(answer.ok).toBe(false)
    expect(store.identity.value).toBeUndefined()
  })

  it('holds no identity after it is deleted', async () => {
    const remove = vi.fn(() => Promise.resolve())
    const { store } = createStore(
      createClient({ get: () => Promise.resolve(createSampleIdentity()), delete: remove }),
    )
    await store.load()
    const answer = await store.delete()
    expect(answer.ok).toBe(true)
    expect(remove).toHaveBeenCalledOnce()
    expect(store.identity.value).toBeUndefined()
  })

  it('keeps the identity when the delete fails', async () => {
    const { store } = createStore(
      createClient({
        get: () => Promise.resolve(createSampleIdentity()),
        delete: () => Promise.reject(new Error('refused')),
      }),
    )
    await store.load()
    const answer = await store.delete()
    expect(answer.ok).toBe(false)
    expect(store.identity.value).toStrictEqual(createSampleIdentity())
  })
})

describe('the identity store restoring from words', () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  const settle = async <T>(answer: Promise<T>): Promise<T> => {
    let isDone = false
    const markDone = () => {
      isDone = true
    }
    answer.then(markDone, markDone)
    for (let second = 0; second < 70 && !isDone; second += 1) {
      await vi.advanceTimersByTimeAsync(1000)
    }
    return answer
  }

  it('starts the restore with the words and stores the identity once the core has it', async () => {
    vi.useFakeTimers()
    const startRestore = vi.fn(() => Promise.resolve(asStarted()))
    const create = vi.fn(() => Promise.resolve(asValue(createSampleIdentity())))
    let reads = 0
    const { store } = createStore(
      createClient({
        startRestore,
        create,
        get: () => {
          reads += 1
          return Promise.resolve(reads < 3 ? undefined : createSampleIdentity())
        },
      }),
    )

    const answer = await settle(store.restore(WORDS))

    expect(answer).toStrictEqual({ ok: true, value: createSampleIdentity() })
    expect(startRestore).toHaveBeenCalledExactlyOnceWith(WORDS)
    expect(create).not.toHaveBeenCalled()
    expect(reads).toBe(3)
    expect(store.identity.value).toStrictEqual(createSampleIdentity())
  })

  it('waits while the core has no identity yet', async () => {
    vi.useFakeTimers()
    let isDone = false
    const { store } = createStore(createClient({ get: () => Promise.resolve(undefined) }))

    store.restore(WORDS).then(
      () => {
        isDone = true
      },
      (error: unknown) => {
        throw error
      },
    )
    await vi.advanceTimersByTimeAsync(10_000)

    expect(isDone).toBe(false)
    expect(store.identity.value).toBeUndefined()
  })

  it('reports that the words are not valid, stores no identity and does not wait', async () => {
    const get = vi.fn(() => Promise.resolve(undefined))
    const { store } = createStore(
      createClient({
        get,
        startRestore: () =>
          Promise.resolve(asFailure({ error: 'invalidWords', message: 'word 4 is wrong' })),
      }),
    )

    const answer = await store.restore(WORDS)

    expect(answer).toStrictEqual({ ok: false, error: 'Some words are not valid.' })
    expect(get).not.toHaveBeenCalled()
    expect(store.identity.value).toBeUndefined()
  })

  it('reports that this device already has an identity and keeps the one it has', async () => {
    const { store } = createStore(
      createClient({
        get: () => Promise.resolve(createSampleIdentity()),
        startRestore: () => Promise.resolve(asFailure({ error: 'identityExists', message: '' })),
      }),
    )
    await store.load()

    const answer = await store.restore(WORDS)

    expect(answer).toStrictEqual({ ok: false, error: 'This device already has an identity.' })
    expect(store.identity.value).toStrictEqual(createSampleIdentity())
  })

  it('reads a failed start as a text and stores no identity', async () => {
    const { store } = createStore(
      createClient({ startRestore: () => Promise.reject(new Error('offline')) }),
    )

    const answer = await store.restore(WORDS)

    expect(answer.ok).toBe(false)
    expect(store.identity.value).toBeUndefined()
  })
})

describe('the identity store and the recovery kit', () => {
  const SHEETS = [
    ['academic', 'acid'],
    ['acne', 'acquire'],
  ]

  it('saves the kit for the words and holds no kit and no change of identity', async () => {
    const saveKit = vi.fn(() => Promise.resolve(asValue(true)))
    const { store } = createStore(createClient({ saveKit }))
    const answer = await store.saveKit({ words: WORDS })
    expect(answer).toStrictEqual({ ok: true, value: true })
    expect(saveKit).toHaveBeenCalledExactlyOnceWith({ words: WORDS })
    expect(store.identity.value).toBeUndefined()
    expect(Object.keys(store)).not.toContain('kit')
  })

  it('saves the kit for two sheets of an earlier kit and passes on a closed dialog', async () => {
    const saveKit = vi.fn(() => Promise.resolve(asValue(false)))
    const { store } = createStore(createClient({ saveKit }))
    expect(await store.saveKit({ sheets: SHEETS })).toStrictEqual({ ok: true, value: false })
    expect(saveKit).toHaveBeenCalledExactlyOnceWith({ sheets: SHEETS })
  })

  it('says that words of another account are not those of this account', async () => {
    const { store } = createStore(
      createClient({
        saveKit: () => Promise.resolve(asFailure({ error: 'otherAccount', message: '' })),
      }),
    )
    expect(await store.saveKit({ words: WORDS })).toStrictEqual({
      ok: false,
      error: 'The words belong to another account.',
    })
  })

  it('says that sheets of another account are not those of this account', async () => {
    const { store } = createStore(
      createClient({
        saveKit: () => Promise.resolve(asFailure({ error: 'otherAccount', message: '' })),
      }),
    )
    expect(await store.saveKit({ sheets: SHEETS })).toStrictEqual({
      ok: false,
      error: 'The sheets belong to another account.',
    })
  })

  it('says that words are not valid when the kit is saved for words that are not', async () => {
    const { store } = createStore(
      createClient({
        saveKit: () => Promise.resolve(asFailure({ error: 'invalidWords', message: '' })),
      }),
    )
    expect(await store.saveKit({ words: WORDS })).toStrictEqual({
      ok: false,
      error: 'Some words are not valid.',
    })
  })

  it('says what is wrong with the sheets in the words of the page', async () => {
    const texts: Array<[IdentityFailure['error'], string]> = [
      ['notEnoughSheets', 'One more sheet is needed.'],
      ['invalidSheets', 'Some sheets are not valid.'],
      ['otherAccount', 'The sheets belong to another account.'],
    ]
    for (const [error, text] of texts) {
      const { store } = createStore(
        createClient({
          startRestoreFromKit: () =>
            Promise.resolve(asFailure({ error, message: 'sheet 2 is not valid' })),
        }),
      )
      expect(await store.restoreFromKit(SHEETS)).toStrictEqual({ ok: false, error: text })
    }
  })

  it('describes the failures of a kit with words of their own', () => {
    const texts = (['invalidSheets', 'notEnoughSheets', 'otherAccount', 'noIdentity'] as const).map(
      (error) => IDENTITY_WORDS.describeFailure({ error, message: '' }),
    )
    expect(new Set(texts).size).toBe(4)
    expect(texts.every((text) => text !== '')).toBe(true)
  })

  it('reads a failed kit call as a text', async () => {
    const { store } = createStore(
      createClient({
        saveKit: () => Promise.reject(new Error('offline')),
        startRestoreFromKit: () => Promise.reject(new Error('offline')),
      }),
    )
    expect((await store.saveKit({ words: WORDS })).ok).toBe(false)
    expect((await store.restoreFromKit(SHEETS)).ok).toBe(false)
    expect(store.identity.value).toBeUndefined()
  })

  it('starts the restore with two sheets and holds the identity once the core has it', async () => {
    vi.useFakeTimers()
    const startRestoreFromKit = vi.fn(() => Promise.resolve(asStarted()))
    let reads = 0
    const { store } = createStore(
      createClient({
        startRestoreFromKit,
        get: () => {
          reads += 1
          return Promise.resolve(reads < 2 ? undefined : createSampleIdentity())
        },
      }),
    )
    const pending = store.restoreFromKit(SHEETS)
    await vi.advanceTimersByTimeAsync(2000)
    expect(await pending).toStrictEqual({ ok: true, value: createSampleIdentity() })
    expect(startRestoreFromKit).toHaveBeenCalledExactlyOnceWith(SHEETS)
    expect(store.identity.value).toStrictEqual(createSampleIdentity())
    vi.useRealTimers()
  })

  it('holds no identity when the sheets are refused and does not wait', async () => {
    const get = vi.fn(() => Promise.resolve(undefined))
    const { store } = createStore(
      createClient({
        get,
        startRestoreFromKit: () =>
          Promise.resolve(asFailure({ error: 'notEnoughSheets', message: '' })),
      }),
    )
    expect(await store.restoreFromKit(SHEETS.slice(0, 1))).toStrictEqual({
      ok: false,
      error: 'One more sheet is needed.',
    })
    expect(get).not.toHaveBeenCalled()
    expect(store.identity.value).toBeUndefined()
  })
})

describe('the devices and the link of the identity store', () => {
  const DEVICES = [
    { name: 'Laptop', fingerprint: 'A', addedAt: 'x', isThis: true, isRevoked: false },
    { name: 'Phone', fingerprint: 'B', addedAt: 'y', isThis: false, isRevoked: true },
  ]

  it('loads the devices of the account', async () => {
    const { store } = createStore(
      createClient({ listDevices: () => Promise.resolve(asValue(DEVICES)) }),
    )
    await store.devices.load()
    expect(store.devices.list.value).toStrictEqual(DEVICES)
  })

  it('reads the list again after a revoke, also when the revoke was refused', async () => {
    let listed = 0
    const { store } = createStore(
      createClient({
        listDevices: () => {
          listed += 1
          return Promise.resolve(asValue(DEVICES))
        },
        revokeDevice: () => Promise.resolve(asRefusal('revokeRaced')),
      }),
    )
    const answer = await store.devices.revoke('B')
    expect(answer).toStrictEqual({
      ok: false,
      error: 'A device was added while you revoked. Check the list.',
    })
    expect(listed).toBe(1)
  })

  it('words the refusal of a revoke of this device', async () => {
    const { store } = createStore(
      createClient({ revokeDevice: () => Promise.resolve(asRefusal('notAllowedThisDevice')) }),
    )
    expect(await store.devices.revoke('A')).toStrictEqual({
      ok: false,
      error: 'This device cannot revoke itself.',
    })
  })

  it('passes the answers of the person to the core', async () => {
    const sent: string[] = []
    const { store } = createStore(
      createClient({
        decideLink: (approve) => {
          sent.push(`decide ${approve}`)
          return Promise.resolve(asStarted())
        },
        enterLinkCode: (code) => {
          sent.push(`code ${code}`)
          return Promise.resolve(asStarted())
        },
      }),
    )
    await store.devices.link.decide(true)
    await store.devices.link.enterCode('123456')
    expect(sent).toStrictEqual(['decide true', 'code 123456'])
  })

  it('words a refused link text', async () => {
    const { store } = createStore(
      createClient({ startLink: () => Promise.resolve(asRefusal('linkTextInvalid')) }),
    )
    expect(await store.devices.link.start('host', 'scan', 'x')).toStrictEqual({
      ok: false,
      error: 'This text is not the link of a QR code.',
    })
  })
})
