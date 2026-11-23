import { describe, expect, it } from 'vitest'
import { create } from '@bufbuild/protobuf'
import {
  CheckRecoveryKitResponseSchema,
  CreateIdentityResponseSchema,
  CreateRecoveryKitResponseSchema,
  DeleteIdentityResponseSchema,
  GenerateRecoveryWordsResponseSchema,
  GetIdentityResponseSchema,
  IdentityErrorCode,
  StartRestoreFromKitResponseSchema,
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
      [IdentityErrorCode.INVALID_WORDS, 'invalidWords'],
      [IdentityErrorCode.IDENTITY_EXISTS, 'identityExists'],
      [IdentityErrorCode.KEY_STORE_FAILED, 'keyStoreFailed'],
    ]
    for (const [code, expected] of codes) {
      const client = createClient({
        startRestore: () => create(StartRestoreResponseSchema, { error: code, detail: 'refused' }),
      })
      expect(await client.startRestore(WORDS)).toStrictEqual({
        ok: false,
        error: { error: expected, message: 'refused' },
      })
    }
  })
})

const SHEET_ONE = ['academic', 'acid', 'acne', 'acquire']
const SHEET_TWO = ['actress', 'adapt', 'adequate', 'adjust']
const SHEET_THREE = ['admit', 'adorn', 'adult', 'advance']

const KIT = {
  kitId: '5C21',
  accountFingerprint: INFO.accountFingerprint,
  sheets: [{ words: SHEET_ONE }, { words: SHEET_TWO }, { words: SHEET_THREE }],
}

describe('createIdentityClient recovery kit', () => {
  it('sends the words as the proof and reads the kit', async () => {
    let sent: unknown
    const client = createClient({
      createRecoveryKit: (request) => {
        sent = request.proof
        return create(CreateRecoveryKitResponseSchema, { kit: KIT })
      },
    })
    const answer = await client.createKit({ words: WORDS })
    expect(answer).toStrictEqual({
      ok: true,
      value: {
        id: '5C21',
        accountFingerprint: INFO.accountFingerprint,
        sheets: [SHEET_ONE, SHEET_TWO, SHEET_THREE],
      },
    })
    expect(sent).toMatchObject({ case: 'words', value: { words: WORDS } })
  })

  it('sends two sheets as the proof', async () => {
    let sent: unknown
    const client = createClient({
      createRecoveryKit: (request) => {
        sent = request.proof
        return create(CreateRecoveryKitResponseSchema, { kit: KIT })
      },
    })
    await client.createKit({ sheets: [SHEET_ONE, SHEET_TWO] })
    expect(sent).toMatchObject({
      case: 'sheets',
      value: { sheets: [{ words: SHEET_ONE }, { words: SHEET_TWO }] },
    })
  })

  it('reads each served error code of a kit as a failure', async () => {
    const codes: Array<[IdentityErrorCode, string]> = [
      [IdentityErrorCode.INVALID_WORDS, 'invalidWords'],
      [IdentityErrorCode.INVALID_SHEETS, 'invalidSheets'],
      [IdentityErrorCode.NOT_ENOUGH_SHEETS, 'notEnoughSheets'],
      [IdentityErrorCode.OTHER_ACCOUNT, 'otherAccount'],
      [IdentityErrorCode.NO_IDENTITY, 'noIdentity'],
    ]
    for (const [code, expected] of codes) {
      const client = createClient({
        createRecoveryKit: () =>
          create(CreateRecoveryKitResponseSchema, { error: code, detail: 'refused' }),
      })
      expect(await client.createKit({ words: WORDS })).toStrictEqual({
        ok: false,
        error: { error: expected, message: 'refused' },
      })
    }
  })

  it('fails when the core answers neither a kit nor an error', async () => {
    const client = createClient({
      createRecoveryKit: () => create(CreateRecoveryKitResponseSchema, {}),
    })
    expect((await client.createKit({ words: WORDS })).ok).toBe(false)
  })

  it('sends the sheets to check and reads the account fingerprint', async () => {
    let sent: readonly string[][] = []
    const client = createClient({
      checkRecoveryKit: (request) => {
        sent = request.sheets.map((sheet) => [...sheet.words])
        return create(CheckRecoveryKitResponseSchema, {
          accountFingerprint: INFO.accountFingerprint,
        })
      },
    })
    expect(await client.checkKit([SHEET_ONE, SHEET_THREE])).toStrictEqual({
      ok: true,
      value: INFO.accountFingerprint,
    })
    expect(sent).toStrictEqual([SHEET_ONE, SHEET_THREE])
  })

  it('reads a refused check as a failure with the detail', async () => {
    const client = createClient({
      checkRecoveryKit: () =>
        create(CheckRecoveryKitResponseSchema, {
          error: IdentityErrorCode.INVALID_SHEETS,
          detail: 'sheet 2 is not valid',
        }),
    })
    expect(await client.checkKit([SHEET_ONE, SHEET_TWO])).toStrictEqual({
      ok: false,
      error: { error: 'invalidSheets', message: 'sheet 2 is not valid' },
    })
  })

  it('sends the sheets to start a restore and reads that it started', async () => {
    let sent: readonly string[][] = []
    const client = createClient({
      startRestoreFromKit: (request) => {
        sent = request.sheets.map((sheet) => [...sheet.words])
        return create(StartRestoreFromKitResponseSchema, {})
      },
    })
    expect(await client.startRestoreFromKit([SHEET_TWO, SHEET_THREE])).toStrictEqual({
      ok: true,
      value: undefined,
    })
    expect(sent).toStrictEqual([SHEET_TWO, SHEET_THREE])
  })

  it('reads a refused restore from sheets as a failure with the detail', async () => {
    const client = createClient({
      startRestoreFromKit: () =>
        create(StartRestoreFromKitResponseSchema, {
          error: IdentityErrorCode.NOT_ENOUGH_SHEETS,
          detail: 'two sheets are needed',
        }),
    })
    expect(await client.startRestoreFromKit([SHEET_ONE])).toStrictEqual({
      ok: false,
      error: { error: 'notEnoughSheets', message: 'two sheets are needed' },
    })
  })
})
