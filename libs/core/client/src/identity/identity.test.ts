import { describe, expect, it } from 'vitest'
import { create } from '@bufbuild/protobuf'
import {
  CreateIdentityResponseSchema,
  DeleteIdentityResponseSchema,
  GenerateRecoveryWordsResponseSchema,
  GetIdentityResponseSchema,
  IdentityErrorCode,
  StartRestoreResponseSchema,
} from '@numen/protocol'
import { createIdentityRpc } from '../clients'
import { createCoreTransport, type CoreAnswers } from '../testing'
import { createIdentityClient, type IdentityInfo } from './identity'

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

const INFO = {
  accountFingerprint: '3F2A 91C0 0000 1111 2222 3333 4444 5555',
  deviceFingerprint: 'AAAA BBBB CCCC DDDD 0000 1111 2222 3333',
  deviceName: 'Laptop',
  deviceAddedAt: '2026-11-22T12:00:00Z',
}

const EXPECTED: IdentityInfo = { ...INFO }

const createClient = (answers: CoreAnswers['identity'] = {}) =>
  createIdentityClient(createIdentityRpc(createCoreTransport({ identity: answers })))

describe('createIdentityClient', () => {
  it('reads the identity of this device', async () => {
    const client = createClient({
      getIdentity: () => create(GetIdentityResponseSchema, { identity: INFO }),
    })
    expect(await client.get()).toStrictEqual(EXPECTED)
  })

  it('reads no identity when the core answers none', async () => {
    const client = createClient({ getIdentity: () => create(GetIdentityResponseSchema, {}) })
    expect(await client.get()).toBeUndefined()
  })

  it('reads the twelve new recovery words', async () => {
    const client = createClient({
      generateRecoveryWords: () => create(GenerateRecoveryWordsResponseSchema, { words: WORDS }),
    })
    expect(await client.generateWords()).toStrictEqual(WORDS)
  })

  it('sends the words to create and reads the identity', async () => {
    let sent: readonly string[] = []
    const client = createClient({
      createIdentity: (request) => {
        sent = [...request.words]
        return create(CreateIdentityResponseSchema, { identity: INFO })
      },
    })
    const answer = await client.create(WORDS)
    expect(answer).toStrictEqual({ ok: true, value: EXPECTED })
    expect(sent).toStrictEqual(WORDS)
  })

  it('reads each served error code as a failure', async () => {
    const codes: Array<[IdentityErrorCode, string]> = [
      [IdentityErrorCode.INVALID_WORDS, 'invalidWords'],
      [IdentityErrorCode.IDENTITY_EXISTS, 'identityExists'],
      [IdentityErrorCode.KEY_STORE_FAILED, 'keyStoreFailed'],
    ]
    for (const [code, expected] of codes) {
      const client = createClient({
        createIdentity: () =>
          create(CreateIdentityResponseSchema, { error: code, detail: 'refused' }),
      })
      expect(await client.create(WORDS)).toStrictEqual({
        ok: false,
        error: { error: expected, message: 'refused' },
      })
    }
  })

  it('fails when the core answers neither an identity nor an error', async () => {
    const client = createClient({ createIdentity: () => create(CreateIdentityResponseSchema, {}) })
    const answer = await client.create(WORDS)
    expect(answer.ok).toBe(false)
  })

  it('deletes the identity of this device', async () => {
    let calls = 0
    const client = createClient({
      deleteIdentity: () => {
        calls += 1
        return create(DeleteIdentityResponseSchema, {})
      },
    })
    await client.delete()
    expect(calls).toBe(1)
  })

  it('sends the words to start a restore and reads that it started', async () => {
    let sent: readonly string[] = []
    const client = createClient({
      startRestore: (request) => {
        sent = [...request.words]
        return create(StartRestoreResponseSchema, {})
      },
    })
    const answer = await client.startRestore(WORDS)
    expect(answer).toStrictEqual({ ok: true, value: undefined })
    expect(sent).toStrictEqual(WORDS)
  })

  it('reads each served error code of a restore as a failure', async () => {
    const codes: Array<[IdentityErrorCode, string]> = [
      [IdentityErrorCode.INVALID_WORDS, 'invalid_words'],
      [IdentityErrorCode.IDENTITY_EXISTS, 'identity_exists'],
      [IdentityErrorCode.KEY_STORE_FAILED, 'key_store_failed'],
    ]
    for (const [code, expected] of codes) {
      const client = createClient({
        startRestore: () => create(StartRestoreResponseSchema, { error: code, detail: 'refused' }),
      })
      expect(await client.startRestore(WORDS)).toStrictEqual({
        ok: false,
        error: { errorCode: expected, detail: 'refused' },
      })
    }
  })
})
