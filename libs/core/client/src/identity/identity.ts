/**
 * The account identity of this device: its two fingerprints and the name of the device. No secret
 * key reaches the window; the recovery words and the sheets of a kit cross only in the calls
 * that take them, and the sheets of a new kit never do: the core saves them as a PDF.
 */
import type { Client } from '@connectrpc/connect'
import {
  IdentityErrorCode as ProtoIdentityErrorCode,
  type IdentityInfo as ProtoIdentityInfo,
  type IdentityService,
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
type IdentityRpc = Pick<
  Client<typeof IdentityService>,
  | 'getIdentity'
  | 'generateRecoveryWords'
  | 'createIdentity'
  | 'startRestore'
  | 'deleteIdentity'
  | 'saveRecoveryKit'
  | 'startRestoreFromKit'
>

/** What proves the account when a kit is saved: the 12 words or two sheets of an earlier kit. */
export type KitProof =
  { readonly words: readonly string[] } | { readonly sheets: readonly (readonly string[])[] }

/** The identity of this device. */
export interface IdentityClient {
  /** The identity of this device, or undefined when it has none. */
  get(): Promise<IdentityInfo | undefined>
  generateWords(): Promise<readonly string[]>
  create(words: readonly string[]): Promise<Result<IdentityInfo, IdentityFailure>>
  /** Starts to restore the account of the words; the identity exists once the core has completed it. */
  startRestore(words: readonly string[]): Promise<Result<undefined, IdentityFailure>>
  delete(): Promise<void>
  /**
   * Makes a new kit of three sheets once the proof fits the account of this device and saves it as a
   * PDF where the person chooses. True when it was saved, false when the person chose no file.
   */
  saveKit(proof: KitProof): Promise<Result<boolean, IdentityFailure>>
  /** Starts to restore the account of two sheets of a kit, like startRestore does for the words. */
  startRestoreFromKit(
    sheets: readonly (readonly string[])[],
  ): Promise<Result<undefined, IdentityFailure>>
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

const readStartAnswer = (answer: {
  error: ProtoIdentityErrorCode
  detail: string
}): Result<undefined, IdentityFailure> => {
  const error = readErrorCode(answer.error)
  return error === undefined ? asValue(undefined) : asFailure({ error, message: answer.detail })
}

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
  startRestore: async (words) => readStartAnswer(await rpc.startRestore({ words: [...words] })),
  delete: async () => {
    await rpc.deleteIdentity({})
  },
  saveKit: async (proof) => {
    const answer = await rpc.saveRecoveryKit({ proof: writeProof(proof) })
    const error = readErrorCode(answer.error)
    return error === undefined
      ? asValue(answer.isSaved)
      : asFailure({ error, message: answer.detail })
  },
  startRestoreFromKit: async (sheets) =>
    readStartAnswer(await rpc.startRestoreFromKit({ sheets: writeSheets(sheets) })),
})
