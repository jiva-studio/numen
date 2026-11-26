import { describe, expect, it } from 'vitest'
import { create } from '@bufbuild/protobuf'
import {
  CancelLinkResponseSchema,
  ConfirmJoinResponseSchema,
  CreateIdentityResponseSchema,
  DecideLinkResponseSchema,
  DeleteIdentityResponseSchema,
  GenerateRecoveryWordsResponseSchema,
  GetIdentityResponseSchema,
  GetLinkResponseSchema,
  ErrorCode,
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
    expect(await client.account.get()).toStrictEqual(EXPECTED)
  })

  it('reads no identity when the core answers none', async () => {
    const client = createClient({ getIdentity: () => create(GetIdentityResponseSchema, {}) })
    expect(await client.account.get()).toBeUndefined()
  })

  it('reads the twelve new recovery words', async () => {
    const client = createClient({
      generateRecoveryWords: () => create(GenerateRecoveryWordsResponseSchema, { words: WORDS }),
    })
    expect(await client.account.generateWords()).toStrictEqual(WORDS)
  })

  it('sends the words to create and reads the identity', async () => {
    let sent: readonly string[] = []
    const client = createClient({
      createIdentity: (request) => {
        sent = [...request.words]
        return create(CreateIdentityResponseSchema, { identity: INFO })
      },
    })
    const answer = await client.account.create(WORDS)
    expect(answer).toStrictEqual({ ok: true, value: EXPECTED })
    expect(sent).toStrictEqual(WORDS)
  })

  it('reads each served error code as a failure', async () => {
    const codes: Array<[ErrorCode, string]> = [
      [ErrorCode.INVALID_WORDS, 'invalidWords'],
      [ErrorCode.ALREADY_EXISTS, 'identityExists'],
      [ErrorCode.KEY_STORE_FAILED, 'keyStoreFailed'],
    ]
    for (const [code, expected] of codes) {
      const client = createClient({
        createIdentity: () =>
          create(CreateIdentityResponseSchema, { error: code, detail: 'refused' }),
      })
      expect(await client.account.create(WORDS)).toStrictEqual({
        ok: false,
        error: { error: expected, message: 'refused' },
      })
    }
  })

  it('fails when the core answers neither an identity nor an error', async () => {
    const client = createClient({ createIdentity: () => create(CreateIdentityResponseSchema, {}) })
    const answer = await client.account.create(WORDS)
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
    await client.account.delete()
    expect(calls).toBe(1)
  })

  it('sends the words and the confirmation of the person to start a restore and reads that it started', async () => {
    let sent: readonly string[] = []
    let isConfirmed = false
    const client = createClient({
      startRestore: (request) => {
        sent = [...request.words]
        isConfirmed = request.confirmUnmarked
        return create(StartRestoreResponseSchema, {})
      },
    })
    const answer = await client.account.startRestore(WORDS, true)
    expect(answer).toStrictEqual({ ok: true, value: undefined })
    expect(sent).toStrictEqual(WORDS)
    expect(isConfirmed).toBe(true)
  })

  it('reads each served error code of a restore as a failure', async () => {
    const codes: Array<[ErrorCode, string]> = [
      [ErrorCode.INVALID_WORDS, 'invalidWords'],
      [ErrorCode.ALREADY_EXISTS, 'identityExists'],
      [ErrorCode.KEY_STORE_FAILED, 'keyStoreFailed'],
    ]
    for (const [code, expected] of codes) {
      const client = createClient({
        startRestore: () => create(StartRestoreResponseSchema, { error: code, detail: 'refused' }),
      })
      expect(await client.account.startRestore(WORDS, true)).toStrictEqual({
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
    expect(await client.account.saveKit({ words: WORDS })).toStrictEqual({ ok: true, value: true })
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
    expect(await client.account.saveKit({ sheets: [SHEET_ONE, SHEET_TWO] })).toStrictEqual({
      ok: true,
      value: false,
    })
    expect(sent).toMatchObject({
      case: 'sheets',
      value: { sheets: [{ words: SHEET_ONE }, { words: SHEET_TWO }] },
    })
  })

  it('reads each served error code of a kit as a failure', async () => {
    const codes: Array<[ErrorCode, string]> = [
      [ErrorCode.INVALID_WORDS, 'invalidWords'],
      [ErrorCode.INVALID_SHEETS, 'invalidSheets'],
      [ErrorCode.NOT_ENOUGH_SHEETS, 'notEnoughSheets'],
      [ErrorCode.OTHER_ACCOUNT, 'otherAccount'],
      [ErrorCode.NO_IDENTITY, 'noIdentity'],
    ]
    for (const [code, expected] of codes) {
      const client = createClient({
        saveRecoveryKit: () =>
          create(SaveRecoveryKitResponseSchema, { error: code, detail: 'refused' }),
      })
      expect(await client.account.saveKit({ words: WORDS })).toStrictEqual({
        ok: false,
        error: { error: expected, message: 'refused' },
      })
    }
  })

  it('sends the sheets to start a restore and reads that it started', async () => {
    let sent: readonly string[][] = []
    let isConfirmed = false
    const client = createClient({
      startRestoreFromKit: (request) => {
        sent = request.sheets.map((sheet) => [...sheet.words])
        isConfirmed = request.confirmUnmarked
        return create(StartRestoreFromKitResponseSchema, {})
      },
    })
    expect(await client.account.startRestoreFromKit([SHEET_TWO, SHEET_THREE], true)).toStrictEqual({
      ok: true,
      value: undefined,
    })
    expect(sent).toStrictEqual([SHEET_TWO, SHEET_THREE])
    expect(isConfirmed).toBe(true)
  })

  it('reads a refused restore from sheets as a failure with the detail', async () => {
    const client = createClient({
      startRestoreFromKit: () =>
        create(StartRestoreFromKitResponseSchema, {
          error: ErrorCode.NOT_ENOUGH_SHEETS,
          detail: 'two sheets are needed',
        }),
    })
    expect(await client.account.startRestoreFromKit([SHEET_ONE], true)).toStrictEqual({
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
  peerName: 'Phone',
  peerFingerprint: 'AAAA BBBB CCCC DDDD 0000 1111 2222 3333',
  accountFingerprint: INFO.accountFingerprint,
  reason: LinkReason.UNSPECIFIED,
  deviceName: 'Laptop',
  deviceFingerprint: INFO.deviceFingerprint,
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
    const answer = await client.link.startLink('host', 'scan', 'numen://link?v=1')
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
        reason: undefined,
      },
    })
  })

  it('reads the reason a link failed', async () => {
    const client = createClient({
      getLink: () =>
        create(GetLinkResponseSchema, {
          link: { ...LINK, phase: LinkPhase.FAILED, reason: LinkReason.STALE_CHAIN },
        }),
    })
    expect(await client.link.getLink()).toMatchObject({
      ok: true,
      value: { phase: 'failed', reason: 'staleChain' },
    })
  })

  it('reads the refusal of a call that needs a link', async () => {
    const client = createClient({
      getLink: () => create(GetLinkResponseSchema, { error: ErrorCode.NO_LINK, detail: 'none' }),
      decideLink: () =>
        create(DecideLinkResponseSchema, { error: ErrorCode.NO_LINK, detail: 'none' }),
      confirmJoin: () =>
        create(ConfirmJoinResponseSchema, { error: ErrorCode.NO_LINK, detail: 'none' }),
      cancelLink: () =>
        create(CancelLinkResponseSchema, { error: ErrorCode.NO_LINK, detail: 'none' }),
    })
    const noLink = { ok: false, error: { error: 'noLink', message: 'none' } }
    expect(await client.link.getLink()).toStrictEqual(noLink)
    expect(await client.link.decideLink(true)).toStrictEqual(noLink)
    expect(await client.link.confirmJoin(true)).toStrictEqual(noLink)
    expect(await client.link.cancelLink()).toStrictEqual(noLink)
  })

  it('sends the answers of the person', async () => {
    const sent: string[] = []
    const client = createClient({
      decideLink: (request) => {
        sent.push(`decide ${request.approve}`)
        return create(DecideLinkResponseSchema, {})
      },
      confirmJoin: (request) => {
        sent.push(`join ${request.approve}`)
        return create(ConfirmJoinResponseSchema, {})
      },
    })
    await client.link.decideLink(false)
    await client.link.confirmJoin(true)
    expect(sent).toStrictEqual(['decide false', 'join true'])
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
    expect(await client.devices.listDevices()).toStrictEqual({
      ok: true,
      value: [
        { name: 'Laptop', fingerprint: 'A', addedAt: 'x', isThis: true, isRevoked: false },
        { name: 'Phone', fingerprint: 'B', addedAt: 'y', isThis: false, isRevoked: true },
      ],
    })
  })

  it('sends the fingerprint to revoke and reads each refusal', async () => {
    let sent = ''
    const codes: Array<[ErrorCode, string]> = [
      [ErrorCode.LAST_DEVICE, 'lastDevice'],
      [ErrorCode.REVOKE_RACED, 'revokeRaced'],
      [ErrorCode.NOT_ALLOWED_THIS_DEVICE, 'notAllowedThisDevice'],
      [ErrorCode.NOT_FOUND, 'unknownDevice'],
    ]
    for (const [code, expected] of codes) {
      const client = createClient({
        revokeDevice: (request) => {
          sent = request.fingerprint
          return create(RevokeDeviceResponseSchema, { error: code, detail: 'refused' })
        },
      })
      expect(await client.devices.revokeDevice('B')).toStrictEqual({
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
    expect(await client.account.get()).toMatchObject({ isRevoked: true, keyEpoch: 2 })
  })
})
