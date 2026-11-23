/**
 * The account identity of this device: its two fingerprints and the name of the device, the link of
 * a device to the account by a QR code and the devices of the account. No secret key reaches the
 * window; the recovery words and the sheets of a kit cross only in the calls that make or take them.
 */
import type { Client } from '@connectrpc/connect'
import {
  IdentityErrorCode as ProtoIdentityErrorCode,
  LinkMode as ProtoLinkMode,
  LinkPhase as ProtoLinkPhase,
  LinkReason as ProtoLinkReason,
  LinkRole as ProtoLinkRole,
  type DeviceInfo as ProtoDeviceInfo,
  type IdentityInfo as ProtoIdentityInfo,
  type IdentityService,
  type LinkInfo as ProtoLinkInfo,
  type RecoveryKit as ProtoRecoveryKit,
} from '@numen/protocol'
import { asFailure, asValue, type Failure, type Result } from '@numen/result'

/** What a person can see of the identity of this device. */
export interface IdentityInfo {
  readonly accountFingerprint: string
  readonly deviceFingerprint: string
  readonly deviceName: string
  readonly deviceAddedAt: string
  readonly isRevoked: boolean
  readonly keyEpoch: number
}

/** Why a call of the identity was refused. */
type IdentityErrorCode =
  | 'invalidWords'
  | 'identityExists'
  | 'keyStoreFailed'
  | 'invalidSheets'
  | 'notEnoughSheets'
  | 'otherAccount'
  | 'linkActive'
  | 'noLink'
  | 'linkTextInvalid'
  | 'lastDevice'
  | 'revokeRaced'
  | 'notAllowedThisDevice'
  | 'noIdentity'
  | 'deviceRevoked'
  | 'unknownDevice'
  | 'codeInvalid'

/** A served refusal with the reason the core gives. */
export type IdentityFailure = Failure<IdentityErrorCode>

/** The device that has the account is the host, a device with none is the joiner. */
export type LinkRole = 'host' | 'joiner'

/** The device that shows the QR code, or the one that reads it. */
export type LinkMode = 'show' | 'scan'

/** Where the link stands. */
export type LinkPhase =
  | 'waitingForChannel'
  | 'waitingForPeer'
  | 'confirmPeer'
  | 'waitingForAccept'
  | 'enterCode'
  | 'publishing'
  | 'confirmAccount'
  | 'linked'
  | 'failed'

/** Why a link failed. */
export type LinkReason =
  | 'declined'
  | 'deviceHasIdentity'
  | 'wrongCode'
  | 'tooManyWrongCodes'
  | 'staleChain'
  | 'cancelled'
  | 'expired'
  | 'invalid'

/** What the window shows of the link in progress. It holds no key. */
export interface LinkInfo {
  readonly phase: LinkPhase
  readonly mode: LinkMode
  readonly role: LinkRole
  readonly qrText: string
  readonly code: string
  readonly peerName: string
  readonly peerFingerprint: string
  readonly accountFingerprint: string
  readonly triesLeft: number
  readonly reason: LinkReason | undefined
  readonly deviceName: string
  readonly deviceFingerprint: string
  readonly isApproved: boolean
}

/** One device of the account. */
export interface DeviceInfo {
  readonly name: string
  readonly fingerprint: string
  readonly addedAt: string
  readonly isThis: boolean
  readonly isRevoked: boolean
}

type Rpc<Name extends keyof Client<typeof IdentityService>> = Pick<
  Client<typeof IdentityService>,
  Name
>

/** The calls of IdentityService that keep the identity of this device. */
type AccountRpc = Rpc<
  | 'getIdentity'
  | 'generateRecoveryWords'
  | 'createIdentity'
  | 'startRestore'
  | 'deleteIdentity'
  | 'createRecoveryKit'
  | 'checkRecoveryKit'
  | 'startRestoreFromKit'
>

/** Three sheets of one recovery kit, each the words of a sheet in order. */
export interface RecoveryKit {
  readonly id: string
  readonly accountFingerprint: string
  readonly sheets: readonly (readonly string[])[]
}

/** What proves the account when a kit is made: the 12 words or two sheets of an earlier kit. */
export type KitProof =
  { readonly words: readonly string[] } | { readonly sheets: readonly (readonly string[])[] }

/** The calls of IdentityService that link a device to an account. */
type LinkRpc = Rpc<
  'startLink' | 'getLink' | 'decideLink' | 'enterLinkCode' | 'confirmJoin' | 'cancelLink'
>

/** The calls of IdentityService that list the devices of the account and revoke one. */
type DevicesRpc = Rpc<'listDevices' | 'revokeDevice'>

/** The calls of IdentityService the identity is managed through. */
type IdentityRpc = AccountRpc & LinkRpc & DevicesRpc

/** The identity of this device. */
export interface AccountClient {
  /** The identity of this device, or undefined when it has none. */
  get(): Promise<IdentityInfo | undefined>
  generateWords(): Promise<readonly string[]>
  create(words: readonly string[]): Promise<Result<IdentityInfo, IdentityFailure>>
  /** Starts to restore the account of the words; the identity exists once the core has completed it. */
  startRestore(words: readonly string[]): Promise<Result<undefined, IdentityFailure>>
  delete(): Promise<void>
  /** Makes a new kit of three sheets once the proof fits the account of this device. */
  createKit(proof: KitProof): Promise<Result<RecoveryKit, IdentityFailure>>
  /** The account fingerprint when two sheets fit the account of this device. */
  checkKit(sheets: readonly (readonly string[])[]): Promise<Result<string, IdentityFailure>>
  /** Starts to restore the account of two sheets of a kit, like startRestore does for the words. */
  startRestoreFromKit(
    sheets: readonly (readonly string[])[],
  ): Promise<Result<undefined, IdentityFailure>>
}

/** The link of this device to an account. */
export interface LinkClient {
  /** Starts a link; scannedText is the text of the QR code in mode scan. */
  startLink(
    role: LinkRole,
    mode: LinkMode,
    scannedText: string,
  ): Promise<Result<LinkInfo, IdentityFailure>>
  /** The link in progress; the failure noLink when there is none. */
  getLink(): Promise<Result<LinkInfo, IdentityFailure>>
  /** The answer of the person on the host to the new device. */
  decideLink(approve: boolean): Promise<Result<undefined, IdentityFailure>>
  enterLinkCode(code: string): Promise<Result<undefined, IdentityFailure>>
  /** The answer of the person on the new device to the account it is offered. */
  confirmJoin(approve: boolean): Promise<Result<undefined, IdentityFailure>>
  cancelLink(): Promise<Result<undefined, IdentityFailure>>
}

/** The devices of the account. */
export interface DevicesClient {
  /** The devices of the account: the active ones, then the revoked ones. */
  listDevices(): Promise<Result<readonly DeviceInfo[], IdentityFailure>>
  revokeDevice(fingerprint: string): Promise<Result<undefined, IdentityFailure>>
}

/** The identity of this device, its link and the devices of its account. */
export type IdentityClient = AccountClient & LinkClient & DevicesClient

const readInfo = (info: ProtoIdentityInfo): IdentityInfo => ({
  accountFingerprint: info.accountFingerprint,
  deviceFingerprint: info.deviceFingerprint,
  deviceName: info.deviceName,
  deviceAddedAt: info.deviceAddedAt,
  isRevoked: info.isRevoked,
  keyEpoch: info.keyEpoch,
})

const ERROR_CODES: Partial<Record<ProtoIdentityErrorCode, IdentityErrorCode>> = {
  [ProtoIdentityErrorCode.INVALID_WORDS]: 'invalidWords',
  [ProtoIdentityErrorCode.IDENTITY_EXISTS]: 'identityExists',
  [ProtoIdentityErrorCode.KEY_STORE_FAILED]: 'keyStoreFailed',
  [ProtoIdentityErrorCode.INVALID_SHEETS]: 'invalidSheets',
  [ProtoIdentityErrorCode.NOT_ENOUGH_SHEETS]: 'notEnoughSheets',
  [ProtoIdentityErrorCode.OTHER_ACCOUNT]: 'otherAccount',
  [ProtoIdentityErrorCode.LINK_ACTIVE]: 'linkActive',
  [ProtoIdentityErrorCode.NO_LINK]: 'noLink',
  [ProtoIdentityErrorCode.LINK_TEXT_INVALID]: 'linkTextInvalid',
  [ProtoIdentityErrorCode.LAST_DEVICE]: 'lastDevice',
  [ProtoIdentityErrorCode.REVOKE_RACED]: 'revokeRaced',
  [ProtoIdentityErrorCode.NOT_ALLOWED_THIS_DEVICE]: 'notAllowedThisDevice',
  [ProtoIdentityErrorCode.NO_IDENTITY]: 'noIdentity',
  [ProtoIdentityErrorCode.DEVICE_REVOKED]: 'deviceRevoked',
  [ProtoIdentityErrorCode.UNKNOWN_DEVICE]: 'unknownDevice',
  [ProtoIdentityErrorCode.CODE_INVALID]: 'codeInvalid',
}

const readErrorCode = (code: ProtoIdentityErrorCode): IdentityErrorCode | undefined =>
  ERROR_CODES[code]

const PHASES: Partial<Record<ProtoLinkPhase, LinkPhase>> = {
  [ProtoLinkPhase.WAITING_FOR_CHANNEL]: 'waitingForChannel',
  [ProtoLinkPhase.WAITING_FOR_PEER]: 'waitingForPeer',
  [ProtoLinkPhase.CONFIRM_PEER]: 'confirmPeer',
  [ProtoLinkPhase.WAITING_FOR_ACCEPT]: 'waitingForAccept',
  [ProtoLinkPhase.ENTER_CODE]: 'enterCode',
  [ProtoLinkPhase.PUBLISHING]: 'publishing',
  [ProtoLinkPhase.CONFIRM_ACCOUNT]: 'confirmAccount',
  [ProtoLinkPhase.LINKED]: 'linked',
}

const readPhase = (phase: ProtoLinkPhase): LinkPhase => PHASES[phase] ?? 'failed'

const readReason = (reason: ProtoLinkReason): LinkReason | undefined => {
  switch (reason) {
    case ProtoLinkReason.UNSPECIFIED:
      return undefined
    case ProtoLinkReason.DECLINED:
      return 'declined'
    case ProtoLinkReason.DEVICE_HAS_IDENTITY:
      return 'deviceHasIdentity'
    case ProtoLinkReason.WRONG_CODE:
      return 'wrongCode'
    case ProtoLinkReason.TOO_MANY_WRONG_CODES:
      return 'tooManyWrongCodes'
    case ProtoLinkReason.STALE_CHAIN:
      return 'staleChain'
    case ProtoLinkReason.CANCELED:
      return 'cancelled'
    case ProtoLinkReason.EXPIRED:
      return 'expired'
    case ProtoLinkReason.INVALID:
      return 'invalid'
  }
}

const readLink = (link: ProtoLinkInfo): LinkInfo => ({
  phase: readPhase(link.phase),
  mode: link.mode === ProtoLinkMode.SCAN ? 'scan' : 'show',
  role: link.role === ProtoLinkRole.HOST ? 'host' : 'joiner',
  qrText: link.qrText,
  code: link.code,
  peerName: link.peerName,
  peerFingerprint: link.peerFingerprint,
  accountFingerprint: link.accountFingerprint,
  triesLeft: link.triesLeft,
  reason: readReason(link.reason),
  deviceName: link.deviceName,
  deviceFingerprint: link.deviceFingerprint,
  isApproved: link.isApproved,
})

const readDevice = (device: ProtoDeviceInfo): DeviceInfo => ({
  name: device.name,
  fingerprint: device.fingerprint,
  addedAt: device.addedAt,
  isThis: device.isThis,
  isRevoked: device.isRevoked,
})

interface Refusal {
  error: ProtoIdentityErrorCode
  detail: string
}

const readAnswer = <T, U>(
  answer: Refusal,
  { value, read, missing }: { value: U | undefined; read: (value: U) => T; missing: string },
): Result<T, IdentityFailure> => {
  const error = readErrorCode(answer.error)
  if (error !== undefined) {
    return asFailure<IdentityFailure>({ error, message: answer.detail })
  }
  if (value === undefined) {
    return asFailure<IdentityFailure>({ error: 'keyStoreFailed', message: missing })
  }
  return asValue(read(value))
}

const readStartAnswer = (answer: Refusal): Result<undefined, IdentityFailure> => {
  const error = readErrorCode(answer.error)
  return error === undefined ? asValue(undefined) : asFailure({ error, message: answer.detail })
}

const readAccount = (fingerprint: string): string => fingerprint

const readKit = (kit: ProtoRecoveryKit): RecoveryKit => ({
  id: kit.kitId,
  accountFingerprint: kit.accountFingerprint,
  sheets: kit.sheets.map((sheet) => [...sheet.words]),
})

const writeSheets = (sheets: readonly (readonly string[])[]) =>
  sheets.map((words) => ({ words: [...words] }))

const writeProof = (proof: KitProof) =>
  'words' in proof
    ? { case: 'words' as const, value: { words: [...proof.words] } }
    : { case: 'sheets' as const, value: { sheets: writeSheets(proof.sheets) } }

const readLinkAnswer = (
  answer: Refusal & { link?: ProtoLinkInfo | undefined },
): Result<LinkInfo, IdentityFailure> => {
  const error = readErrorCode(answer.error)
  if (error !== undefined) {
    return asFailure({ error, message: answer.detail })
  }
  if (answer.link === undefined) {
    return asFailure({ error: 'noLink', message: 'The core returned no link.' })
  }
  return asValue(readLink(answer.link))
}

/** The calls that keep the identity of this device, over an IdentityService client. */
export const createAccountClient = (rpc: AccountRpc): AccountClient => ({
  get: async () => {
    const { identity } = await rpc.getIdentity({})
    return identity === undefined ? undefined : readInfo(identity)
  },
  generateWords: async () => [...(await rpc.generateRecoveryWords({})).words],
  create: async (words) => {
    const answer = await rpc.createIdentity({ words: [...words] })
    return readAnswer(answer, {
      value: answer.identity,
      read: readInfo,
      missing: 'The core returned no identity.',
    })
  },
  startRestore: async (words) => readStartAnswer(await rpc.startRestore({ words: [...words] })),
  delete: async () => {
    await rpc.deleteIdentity({})
  },
  createKit: async (proof) => {
    const answer = await rpc.createRecoveryKit({ proof: writeProof(proof) })
    return readAnswer(answer, {
      value: answer.kit,
      read: readKit,
      missing: 'The core returned no kit.',
    })
  },
  checkKit: async (sheets) => {
    const answer = await rpc.checkRecoveryKit({ sheets: writeSheets(sheets) })
    const account = answer.accountFingerprint === '' ? undefined : answer.accountFingerprint
    return readAnswer(answer, {
      value: account,
      read: readAccount,
      missing: 'The core returned no account.',
    })
  },
  startRestoreFromKit: async (sheets) =>
    readStartAnswer(await rpc.startRestoreFromKit({ sheets: writeSheets(sheets) })),
})

/** The calls of the link of this device to an account, over an IdentityService client. */
export const createLinkClient = (rpc: LinkRpc): LinkClient => ({
  startLink: async (role, mode, scannedText) =>
    readLinkAnswer(
      await rpc.startLink({
        role: role === 'host' ? ProtoLinkRole.HOST : ProtoLinkRole.JOINER,
        mode: mode === 'scan' ? ProtoLinkMode.SCAN : ProtoLinkMode.SHOW,
        scannedText,
      }),
    ),
  getLink: async () => readLinkAnswer(await rpc.getLink({})),
  decideLink: async (approve) => readStartAnswer(await rpc.decideLink({ approve })),
  enterLinkCode: async (code) => readStartAnswer(await rpc.enterLinkCode({ code })),
  confirmJoin: async (approve) => readStartAnswer(await rpc.confirmJoin({ approve })),
  cancelLink: async () => readStartAnswer(await rpc.cancelLink({})),
})

/** The calls of the devices of the account, over an IdentityService client. */
export const createDevicesClient = (rpc: DevicesRpc): DevicesClient => ({
  listDevices: async () => {
    const answer = await rpc.listDevices({})
    const error = readErrorCode(answer.error)
    return error === undefined
      ? asValue(answer.devices.map(readDevice))
      : asFailure({ error, message: answer.detail })
  },
  revokeDevice: async (fingerprint) => readStartAnswer(await rpc.revokeDevice({ fingerprint })),
})

/** The identity managed through an IdentityService client. */
export const createIdentityClient = (rpc: IdentityRpc): IdentityClient => ({
  ...createAccountClient(rpc),
  ...createLinkClient(rpc),
  ...createDevicesClient(rpc),
})
