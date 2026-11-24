/**
 * What the identity of this device is, as the window holds it: two fingerprints and the name of the
 * device, the link of a device to the identity and the devices of the identity. No key and no recovery
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

/** What proves the account when a kit is saved or the identity is restored: the 12 words or two sheets of an earlier kit. */
export type KitProof =
  { readonly words: readonly string[] } | { readonly sheets: readonly (readonly string[])[] }

/** A served refusal with the reason the core gives. */
export interface IdentityFailure {
  readonly error: IdentityErrorCode
  readonly message: string
}

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
  | 'publishing'
  | 'confirmAccount'
  | 'linked'
  | 'failed'

/** Why a link failed. */
type LinkReason =
  | 'declined'
  | 'deviceHasIdentity'
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
  readonly peerName: string
  readonly peerFingerprint: string
  readonly accountFingerprint: string
  readonly reason: LinkReason | undefined
  readonly deviceName: string
  readonly deviceFingerprint: string
}

/** One device of the identity. */
export interface DeviceInfo {
  readonly name: string
  readonly fingerprint: string
  readonly addedAt: string
  readonly isThis: boolean
  readonly isRevoked: boolean
}
