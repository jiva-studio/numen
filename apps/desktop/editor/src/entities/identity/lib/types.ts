/**
 * What the identity of this device is, as the window holds it: two fingerprints and the name of the
 * device. No key and no recovery word is part of it.
 */

/** What a person can see of the identity of this device. */
export interface IdentityInfo {
  readonly accountFingerprint: string
  readonly deviceFingerprint: string
  readonly deviceName: string
  /** When this device joined the account, RFC 3339 in UTC. */
  readonly deviceAddedAt: string
}

/** Why creating the identity was refused. */
export type IdentityErrorCode = 'invalid_words' | 'identity_exists' | 'key_store_failed'

/** A served refusal with the reason the core gives. */
export interface IdentityFailure {
  readonly errorCode: IdentityErrorCode
  readonly detail: string
}
