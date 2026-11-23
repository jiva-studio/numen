/**
 * What the identity of this device is, as the window holds it: two fingerprints and the name of the
 * device. No key and no recovery word is part of it.
 */

/** What a person can see of the identity of this device. */
export interface IdentityInfo {
  readonly accountFingerprint: string
  readonly deviceFingerprint: string
  readonly deviceName: string
  readonly deviceAddedAt: string
}

/** Why an identity call was refused. */
export type IdentityErrorCode =
  | 'invalidWords'
  | 'identityExists'
  | 'keyStoreFailed'
  | 'invalidSheets'
  | 'notEnoughSheets'
  | 'otherAccount'
  | 'noIdentity'

/** Three sheets of one recovery kit, each the words of a sheet in order. */
export interface RecoveryKit {
  readonly id: string
  readonly accountFingerprint: string
  readonly sheets: readonly (readonly string[])[]
}

/** What proves the account when a kit is made: the 12 words or two sheets of an earlier kit. */
export type KitProof =
  { readonly words: readonly string[] } | { readonly sheets: readonly (readonly string[])[] }

/** A served refusal with the reason the core gives. */
export interface IdentityFailure {
  readonly error: IdentityErrorCode
  readonly message: string
}
