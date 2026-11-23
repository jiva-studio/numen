/**
 * What the identity of this device is, as the window holds it: two fingerprints and the name of the
 * device, the link of a device to the account and the devices of the account. No key and no recovery
 * word is part of it.
 */

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
type LinkReason =
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
