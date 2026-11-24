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
export type IdentityRpc = Pick<
  Client<typeof IdentityService>,
  'getIdentity' | 'generateRecoveryWords' | 'createIdentity' | 'deleteIdentity'
>

/** The identity of this device. */
export interface IdentityClient {
  /** The identity of this device, or undefined when it has none. */
  get(): Promise<IdentityInfo | undefined>
  generateWords(): Promise<readonly string[]>
  create(words: readonly string[]): Promise<Result<IdentityInfo, IdentityFailure>>
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

/** The identity managed through an IdentityService client. */
export const createIdentityClient = (rpc: IdentityRpc): IdentityClient => ({
  get: async () => {
    const { identity } = await rpc.getIdentity({})
    return identity === undefined ? undefined : readInfo(identity)
  },
  generateWords: async () => [...(await rpc.generateRecoveryWords({})).words],
  create: async (words) => readAnswer(await rpc.createIdentity({ words: [...words] })),
  delete: async () => {
    await rpc.deleteIdentity({})
  },
})
