/**
 * The account identity of this device: its two fingerprints and the name of the device. No secret
 * key reaches the window; the recovery words cross only in generating words and in creating the
 * identity.
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

/** Why creating the identity was refused. */
type IdentityErrorCode = 'invalidWords' | 'identityExists' | 'keyStoreFailed'

/** A served refusal with the reason the core gives. */
export type IdentityFailure = Failure<IdentityErrorCode>

/** The calls of IdentityService the identity is managed through. */
type IdentityRpc = Pick<
  Client<typeof IdentityService>,
  'getIdentity' | 'generateRecoveryWords' | 'createIdentity' | 'startRestore' | 'deleteIdentity'
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
  }
}

const readAnswer = (answer: {
  error: ProtoIdentityErrorCode
  detail: string
  identity?: ProtoIdentityInfo | undefined
}): Result<IdentityInfo, IdentityFailure> => {
  const error = readErrorCode(answer.error)
  if (error !== undefined) {
    return asFailure({ error, message: answer.detail })
  }
  if (answer.identity === undefined) {
    return asFailure({ error: 'keyStoreFailed', message: 'The core returned no identity.' })
  }
  return asValue(readInfo(answer.identity))
}

const readStartAnswer = (answer: {
  error: ProtoIdentityErrorCode
  detail: string
}): Result<undefined, IdentityFailure> => {
  const error = readErrorCode(answer.error)
  return error === undefined ? asValue(undefined) : asFailure({ error, message: answer.detail })
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
})
