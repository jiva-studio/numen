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

/** Why creating the identity was refused. */
export type IdentityErrorCode = 'invalidWords' | 'identityExists' | 'keyStoreFailed'

/** A served refusal with the reason the core gives. */
export interface IdentityFailure {
  readonly error: IdentityErrorCode
  readonly message: string
}
