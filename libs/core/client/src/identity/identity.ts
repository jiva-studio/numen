/**
 * The account identity of this device: its two fingerprints and the name of the device, the link of
 * a device to the account by a QR code and the devices of the account. No secret key reaches the
 * window; the recovery words cross only in generating words and in creating the identity.
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
} from '@numen/protocol'
import { asFailure, asValue, type Result } from '@numen/result'

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
export type IdentityErrorCode =
  | 'invalid_words'
  | 'identity_exists'
  | 'key_store_failed'
  | 'link_active'
  | 'no_link'
  | 'link_text_invalid'
  | 'last_device'
  | 'revoke_raced'
  | 'not_allowed_this_device'
  | 'no_identity'
  | 'device_revoked'
  | 'unknown_device'
  | 'code_invalid'

/** A served refusal with the reason the core gives. */
export interface IdentityFailure {
  readonly errorCode: IdentityErrorCode
  readonly detail: string
}

/** The device that has the account is the host, a device with none is the joiner. */
export type LinkRole = 'host' | 'joiner'

/** The device that shows the QR code, or the one that reads it. */
export type LinkMode = 'show' | 'scan'

/** Where the link stands. */
export type LinkPhase =
  | 'waiting_for_channel'
  | 'waiting_for_peer'
  | 'confirm_peer'
  | 'waiting_for_accept'
  | 'enter_code'
  | 'publishing'
  | 'confirm_account'
  | 'linked'
  | 'failed'

/** Why a link failed. */
export type LinkReason =
  | 'declined'
  | 'device_has_identity'
  | 'wrong_code'
  | 'too_many_wrong_codes'
  | 'stale_chain'
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

/** The calls of IdentityService the identity is managed through. */
export type IdentityRpc = Pick<
  Client<typeof IdentityService>,
  | 'getIdentity'
  | 'generateRecoveryWords'
  | 'createIdentity'
  | 'startRestore'
  | 'deleteIdentity'
  | 'startLink'
  | 'getLink'
  | 'decideLink'
  | 'enterLinkCode'
  | 'confirmJoin'
  | 'cancelLink'
  | 'listDevices'
  | 'revokeDevice'
>

/** The identity of this device. */
export interface IdentityClient {
  /** The identity of this device, or undefined when it has none. */
  get(): Promise<IdentityInfo | undefined>
  generateWords(): Promise<readonly string[]>
  create(words: readonly string[]): Promise<Result<IdentityInfo, IdentityFailure>>
  /** Starts to restore the account of the words; the identity exists once the core has completed it. */
  startRestore(words: readonly string[]): Promise<Result<undefined, IdentityFailure>>
  delete(): Promise<void>
  /** Starts a link; scannedText is the text of the QR code in mode scan. */
  startLink(
    role: LinkRole,
    mode: LinkMode,
    scannedText: string,
  ): Promise<Result<LinkInfo, IdentityFailure>>
  /** The link in progress; the failure no_link when there is none. */
  getLink(): Promise<Result<LinkInfo, IdentityFailure>>
  /** The answer of the person on the host to the new device. */
  decideLink(approve: boolean): Promise<Result<undefined, IdentityFailure>>
  enterLinkCode(code: string): Promise<Result<undefined, IdentityFailure>>
  /** The answer of the person on the new device to the account it is offered. */
  confirmJoin(approve: boolean): Promise<Result<undefined, IdentityFailure>>
  cancelLink(): Promise<Result<undefined, IdentityFailure>>
  /** The devices of the account: the active ones, then the revoked ones. */
  listDevices(): Promise<Result<readonly DeviceInfo[], IdentityFailure>>
  revokeDevice(fingerprint: string): Promise<Result<undefined, IdentityFailure>>
}

const readInfo = (info: ProtoIdentityInfo): IdentityInfo => ({
  accountFingerprint: info.accountFingerprint,
  deviceFingerprint: info.deviceFingerprint,
  deviceName: info.deviceName,
  deviceAddedAt: info.deviceAddedAt,
  isRevoked: info.isRevoked,
  keyEpoch: info.keyEpoch,
})

const readErrorCode = (code: ProtoIdentityErrorCode): IdentityErrorCode | undefined => {
  switch (code) {
    case ProtoIdentityErrorCode.UNSPECIFIED:
      return undefined
    case ProtoIdentityErrorCode.INVALID_WORDS:
      return 'invalid_words'
    case ProtoIdentityErrorCode.IDENTITY_EXISTS:
      return 'identity_exists'
    case ProtoIdentityErrorCode.KEY_STORE_FAILED:
      return 'key_store_failed'
    case ProtoIdentityErrorCode.LINK_ACTIVE:
      return 'link_active'
    case ProtoIdentityErrorCode.NO_LINK:
      return 'no_link'
    case ProtoIdentityErrorCode.LINK_TEXT_INVALID:
      return 'link_text_invalid'
    case ProtoIdentityErrorCode.LAST_DEVICE:
      return 'last_device'
    case ProtoIdentityErrorCode.REVOKE_RACED:
      return 'revoke_raced'
    case ProtoIdentityErrorCode.NOT_ALLOWED_THIS_DEVICE:
      return 'not_allowed_this_device'
    case ProtoIdentityErrorCode.NO_IDENTITY:
      return 'no_identity'
    case ProtoIdentityErrorCode.DEVICE_REVOKED:
      return 'device_revoked'
    case ProtoIdentityErrorCode.UNKNOWN_DEVICE:
      return 'unknown_device'
    case ProtoIdentityErrorCode.CODE_INVALID:
      return 'code_invalid'
  }
}

const readPhase = (phase: ProtoLinkPhase): LinkPhase => {
  switch (phase) {
    case ProtoLinkPhase.WAITING_FOR_CHANNEL:
      return 'waiting_for_channel'
    case ProtoLinkPhase.WAITING_FOR_PEER:
      return 'waiting_for_peer'
    case ProtoLinkPhase.CONFIRM_PEER:
      return 'confirm_peer'
    case ProtoLinkPhase.WAITING_FOR_ACCEPT:
      return 'waiting_for_accept'
    case ProtoLinkPhase.ENTER_CODE:
      return 'enter_code'
    case ProtoLinkPhase.PUBLISHING:
      return 'publishing'
    case ProtoLinkPhase.CONFIRM_ACCOUNT:
      return 'confirm_account'
    case ProtoLinkPhase.LINKED:
      return 'linked'
    case ProtoLinkPhase.UNSPECIFIED:
    case ProtoLinkPhase.FAILED:
      return 'failed'
  }
}

const readReason = (reason: ProtoLinkReason): LinkReason | undefined => {
  switch (reason) {
    case ProtoLinkReason.UNSPECIFIED:
      return undefined
    case ProtoLinkReason.DECLINED:
      return 'declined'
    case ProtoLinkReason.DEVICE_HAS_IDENTITY:
      return 'device_has_identity'
    case ProtoLinkReason.WRONG_CODE:
      return 'wrong_code'
    case ProtoLinkReason.TOO_MANY_WRONG_CODES:
      return 'too_many_wrong_codes'
    case ProtoLinkReason.STALE_CHAIN:
      return 'stale_chain'
    case ProtoLinkReason.CANCELLED:
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

const readAnswer = (answer: Refusal & { identity?: ProtoIdentityInfo | undefined }) => {
  const errorCode = readErrorCode(answer.error)
  if (errorCode !== undefined) {
    return asFailure<IdentityFailure>({ errorCode, detail: answer.detail })
  }
  if (answer.identity === undefined) {
    return asFailure<IdentityFailure>({
      errorCode: 'key_store_failed',
      detail: 'The core returned no identity.',
    })
  }
  return asValue(readInfo(answer.identity))
}

const readStartAnswer = (answer: Refusal): Result<undefined, IdentityFailure> => {
  const errorCode = readErrorCode(answer.error)
  return errorCode === undefined
    ? asValue(undefined)
    : asFailure({ errorCode, detail: answer.detail })
}

const readLinkAnswer = (
  answer: Refusal & { link?: ProtoLinkInfo | undefined },
): Result<LinkInfo, IdentityFailure> => {
  const errorCode = readErrorCode(answer.error)
  if (errorCode !== undefined) {
    return asFailure({ errorCode, detail: answer.detail })
  }
  if (answer.link === undefined) {
    return asFailure({ errorCode: 'no_link', detail: 'The core returned no link.' })
  }
  return asValue(readLink(answer.link))
}

/** The identity managed through an IdentityService client. */
export const createIdentityClient = (rpc: IdentityRpc): IdentityClient => ({
  get: async () => {
    const { identity } = await rpc.getIdentity({})
    return identity === undefined ? undefined : readInfo(identity)
  },
  generateWords: async () => [...(await rpc.generateRecoveryWords({})).words],
  create: async (words) => readAnswer(await rpc.createIdentity({ words: [...words] })),
  startRestore: async (words) => readStartAnswer(await rpc.startRestore({ words: [...words] })),
  delete: async () => {
    await rpc.deleteIdentity({})
  },
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
  listDevices: async () => {
    const answer = await rpc.listDevices({})
    const errorCode = readErrorCode(answer.error)
    return errorCode === undefined
      ? asValue(answer.devices.map(readDevice))
      : asFailure({ errorCode, detail: answer.detail })
  },
  revokeDevice: async (fingerprint) => readStartAnswer(await rpc.revokeDevice({ fingerprint })),
})
