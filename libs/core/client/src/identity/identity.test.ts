import { describe, expect, it } from 'vitest'
import { create } from '@bufbuild/protobuf'
import {
  CancelLinkResponseSchema,
  ConfirmJoinResponseSchema,
  CreateIdentityResponseSchema,
  DecideLinkResponseSchema,
  DeleteIdentityResponseSchema,
  EnterLinkCodeResponseSchema,
  GenerateRecoveryWordsResponseSchema,
  GetIdentityResponseSchema,
  GetLinkResponseSchema,
  IdentityErrorCode,
  LinkMode,
  LinkPhase,
  LinkReason,
  LinkRole,
  ListDevicesResponseSchema,
  RevokeDeviceResponseSchema,
  SaveRecoveryKitResponseSchema,
  StartLinkResponseSchema,
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

const EXPECTED: IdentityInfo = { ...INFO, isRevoked: false, keyEpoch: 0 }

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

describe('createIdentityClient recovery kit', () => {
  it('sends the words as the proof and reads that the kit was saved', async () => {
    let sent: unknown
    const client = createClient({
      saveRecoveryKit: (request) => {
        sent = request.proof
        return create(SaveRecoveryKitResponseSchema, { isSaved: true })
      },
    })
    expect(await client.saveKit({ words: WORDS })).toStrictEqual({ ok: true, value: true })
    expect(sent).toMatchObject({ case: 'words', value: { words: WORDS } })
  })

  it('sends two sheets as the proof and reads a closed dialog as not saved', async () => {
    let sent: unknown
    const client = createClient({
      saveRecoveryKit: (request) => {
        sent = request.proof
        return create(SaveRecoveryKitResponseSchema, { isSaved: false })
      },
    })
    expect(await client.saveKit({ sheets: [SHEET_ONE, SHEET_TWO] })).toStrictEqual({
      ok: true,
      value: false,
    })
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
        saveRecoveryKit: () =>
          create(SaveRecoveryKitResponseSchema, { error: code, detail: 'refused' }),
      })
      expect(await client.saveKit({ words: WORDS })).toStrictEqual({
        ok: false,
        error: { error: expected, message: 'refused' },
      })
    }
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

const LINK = {
  phase: LinkPhase.CONFIRM_PEER,
  mode: LinkMode.SCAN,
  role: LinkRole.HOST,
  qrText: '',
  code: '123456',
  peerName: 'Phone',
  peerFingerprint: 'AAAA BBBB CCCC DDDD 0000 1111 2222 3333',
  accountFingerprint: INFO.accountFingerprint,
  triesLeft: 3,
  reason: LinkReason.UNSPECIFIED,
  deviceName: 'Laptop',
  deviceFingerprint: INFO.deviceFingerprint,
  isApproved: false,
}

describe('createIdentityClient link', () => {
  it('sends the role, the mode and the scanned text to start a link and reads the link', async () => {
    let sent: { role: LinkRole; mode: LinkMode; scannedText: string } | undefined
    const client = createClient({
      startLink: (request) => {
        sent = { role: request.role, mode: request.mode, scannedText: request.scannedText }
        return create(StartLinkResponseSchema, { link: LINK })
      },
    })
    const answer = await client.startLink('host', 'scan', 'numen://link?v=1')
    expect(sent).toStrictEqual({
      role: LinkRole.HOST,
      mode: LinkMode.SCAN,
      scannedText: 'numen://link?v=1',
    })
    expect(answer).toMatchObject({
      ok: true,
      value: {
        phase: 'confirmPeer',
        mode: 'scan',
        role: 'host',
        code: '123456',
        reason: undefined,
      },
    })
  })

  it('reads the reason a link failed', async () => {
    const client = createClient({
      getLink: () =>
        create(GetLinkResponseSchema, {
          link: { ...LINK, phase: LinkPhase.FAILED, reason: LinkReason.TOO_MANY_WRONG_CODES },
        }),
    })
    expect(await client.getLink()).toMatchObject({
      ok: true,
      value: { phase: 'failed', reason: 'tooManyWrongCodes' },
    })
  })

  it('reads the refusal of a call that needs a link', async () => {
    const client = createClient({
      getLink: () =>
        create(GetLinkResponseSchema, { error: IdentityErrorCode.NO_LINK, detail: 'none' }),
      decideLink: () =>
        create(DecideLinkResponseSchema, { error: IdentityErrorCode.NO_LINK, detail: 'none' }),
      enterLinkCode: () =>
        create(EnterLinkCodeResponseSchema, {
          error: IdentityErrorCode.CODE_INVALID,
          detail: 'six',
        }),
      confirmJoin: () =>
        create(ConfirmJoinResponseSchema, { error: IdentityErrorCode.NO_LINK, detail: 'none' }),
      cancelLink: () =>
        create(CancelLinkResponseSchema, { error: IdentityErrorCode.NO_LINK, detail: 'none' }),
    })
    const noLink = { ok: false, error: { error: 'noLink', message: 'none' } }
    expect(await client.getLink()).toStrictEqual(noLink)
    expect(await client.decideLink(true)).toStrictEqual(noLink)
    expect(await client.confirmJoin(true)).toStrictEqual(noLink)
    expect(await client.cancelLink()).toStrictEqual(noLink)
    expect(await client.enterLinkCode('12')).toStrictEqual({
      ok: false,
      error: { error: 'codeInvalid', message: 'six' },
    })
  })

  it('sends the answers of the person', async () => {
    const sent: string[] = []
    const client = createClient({
      decideLink: (request) => {
        sent.push(`decide ${request.approve}`)
        return create(DecideLinkResponseSchema, {})
      },
      enterLinkCode: (request) => {
        sent.push(`code ${request.code}`)
        return create(EnterLinkCodeResponseSchema, {})
      },
      confirmJoin: (request) => {
        sent.push(`join ${request.approve}`)
        return create(ConfirmJoinResponseSchema, {})
      },
    })
    await client.decideLink(false)
    await client.enterLinkCode('654321')
    await client.confirmJoin(true)
    expect(sent).toStrictEqual(['decide false', 'code 654321', 'join true'])
  })

  it('lists the devices in the order the core gives them', async () => {
    const client = createClient({
      listDevices: () =>
        create(ListDevicesResponseSchema, {
          devices: [
            { name: 'Laptop', fingerprint: 'A', addedAt: 'x', isThis: true, isRevoked: false },
            { name: 'Phone', fingerprint: 'B', addedAt: 'y', isThis: false, isRevoked: true },
          ],
        }),
    })
    expect(await client.listDevices()).toStrictEqual({
      ok: true,
      value: [
        { name: 'Laptop', fingerprint: 'A', addedAt: 'x', isThis: true, isRevoked: false },
        { name: 'Phone', fingerprint: 'B', addedAt: 'y', isThis: false, isRevoked: true },
      ],
    })
  })

  it('sends the fingerprint to revoke and reads each refusal', async () => {
    let sent = ''
    const codes: Array<[IdentityErrorCode, string]> = [
      [IdentityErrorCode.LAST_DEVICE, 'lastDevice'],
      [IdentityErrorCode.REVOKE_RACED, 'revokeRaced'],
      [IdentityErrorCode.NOT_ALLOWED_THIS_DEVICE, 'notAllowedThisDevice'],
      [IdentityErrorCode.UNKNOWN_DEVICE, 'unknownDevice'],
    ]
    for (const [code, expected] of codes) {
      const client = createClient({
        revokeDevice: (request) => {
          sent = request.fingerprint
          return create(RevokeDeviceResponseSchema, { error: code, detail: 'refused' })
        },
      })
      expect(await client.revokeDevice('B')).toStrictEqual({
        ok: false,
        error: { error: expected, message: 'refused' },
      })
      expect(sent).toBe('B')
    }
  })

  it('reads that this device was revoked', async () => {
    const client = createClient({
      getIdentity: () =>
        create(GetIdentityResponseSchema, { identity: { ...INFO, isRevoked: true, keyEpoch: 2 } }),
    })
    expect(await client.get()).toMatchObject({ isRevoked: true, keyEpoch: 2 })
  })
})
