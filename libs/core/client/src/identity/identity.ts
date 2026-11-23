/**
 * The account identity of this device: its two fingerprints and the name of the device. No secret
 * key reaches the window; the recovery words and the sheets of a kit cross only in the calls
 * that make or take them.
 */
import type { Client } from '@connectrpc/connect'
import {
  IdentityErrorCode as ProtoIdentityErrorCode,
  type IdentityInfo as ProtoIdentityInfo,
  type IdentityService,
  type RecoveryKit as ProtoRecoveryKit,
} from '@numen/protocol'
import { asFailure, asValue, type Failure, type Result } from '@numen/result'

/** What a person can see of the identity of this device. */
export interface IdentityInfo {
  readonly accountFingerprint: string
  readonly deviceFingerprint: string
  readonly deviceName: string
  readonly deviceAddedAt: string
}

/** Why an identity call was refused. */
type IdentityErrorCode =
  | 'invalidWords'
  | 'identityExists'
  | 'keyStoreFailed'
  | 'invalidSheets'
  | 'notEnoughSheets'
  | 'otherAccount'
  | 'noIdentity'

/** A served refusal with the reason the core gives. */
export type IdentityFailure = Failure<IdentityErrorCode>

/** The calls of IdentityService the identity is managed through. */
export type IdentityRpc = Pick<
  Client<typeof IdentityService>,
  | 'getIdentity'
  | 'generateRecoveryWords'
  | 'createIdentity'
  | 'deleteIdentity'
  | 'createRecoveryKit'
  | 'checkRecoveryKit'
  | 'restoreIdentityFromKit'
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

/** The identity of this device. */
export interface IdentityClient {
  /** The identity of this device, or undefined when it has none. */
  get(): Promise<IdentityInfo | undefined>
  generateWords(): Promise<readonly string[]>
  create(words: readonly string[]): Promise<Result<IdentityInfo, IdentityFailure>>
  delete(): Promise<void>
  /** Makes a new kit of three sheets once the proof fits the account of this device. */
  createKit(proof: KitProof): Promise<Result<RecoveryKit, IdentityFailure>>
  /** The account fingerprint when two sheets fit the account of this device. */
  checkKit(sheets: readonly (readonly string[])[]): Promise<Result<string, IdentityFailure>>
  /** Keeps an identity on this device from two sheets of a kit. */
  restoreFromKit(
    sheets: readonly (readonly string[])[],
  ): Promise<Result<IdentityInfo, IdentityFailure>>
}

const readInfo = (info: ProtoIdentityInfo): IdentityInfo => ({
  accountFingerprint: info.accountFingerprint,
  deviceFingerprint: info.deviceFingerprint,
  deviceName: info.deviceName,
  deviceAddedAt: info.deviceAddedAt,
})

const readErrorCode = (code: ProtoIdentityErrorCode): IdentityErrorCode | undefined => {
  switch (code) {
    case ProtoIdentityErrorCode.UNSPECIFIED:
      return undefined
    case ProtoIdentityErrorCode.INVALID_WORDS:
      return 'invalidWords'
    case ProtoIdentityErrorCode.IDENTITY_EXISTS:
      return 'identityExists'
    case ProtoIdentityErrorCode.KEY_STORE_FAILED:
      return 'keyStoreFailed'
    case ProtoIdentityErrorCode.INVALID_SHEETS:
      return 'invalidSheets'
    case ProtoIdentityErrorCode.NOT_ENOUGH_SHEETS:
      return 'notEnoughSheets'
    case ProtoIdentityErrorCode.OTHER_ACCOUNT:
      return 'otherAccount'
    case ProtoIdentityErrorCode.NO_IDENTITY:
      return 'noIdentity'
  }
}

const readAnswer = <T, U>(
  answer: { error: ProtoIdentityErrorCode; detail: string },
  { value, read, missing }: { value: U | undefined; read: (value: U) => T; missing: string },
): Result<T, IdentityFailure> => {
  const error = readErrorCode(answer.error)
  if (error !== undefined) {
    return asFailure({ error, message: answer.detail })
  }
  if (value === undefined) {
    return asFailure({ error: 'keyStoreFailed', message: missing })
  }
  return asValue(read(value))
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

/** The identity managed through an IdentityService client. */
export const createIdentityClient = (rpc: IdentityRpc): IdentityClient => ({
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
  restoreFromKit: async (sheets) => {
    const answer = await rpc.restoreIdentityFromKit({ sheets: writeSheets(sheets) })
    return readAnswer(answer, {
      value: answer.identity,
      read: readInfo,
      missing: 'The core returned no identity.',
    })
  },
})
