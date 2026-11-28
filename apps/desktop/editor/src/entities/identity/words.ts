/** What the identity store reports when the core refuses an identity call or a read fails. */
import type { IdentityErrorCode } from './lib/types'

const FAILURE_TEXTS: Readonly<Partial<Record<IdentityErrorCode, string>>> = {
  invalidWords: 'Some words are not valid.',
  identityExists: 'This device already has an identity.',
  keyStoreFailed: 'The identity could not be saved. Check the system keychain.',
  notEnoughSheets: 'One more sheet is needed.',
  invalidSheets: 'Some sheets are not valid.',
  noIdentity: 'This device has no identity to make a kit for.',
  lastDevice: 'The last device of the identity cannot be revoked.',
  revokeRaced: 'A device was added while you revoked. Check the list.',
  notAllowedThisDevice: 'This device cannot revoke itself.',
  linkTextInvalid: 'This text is not the link of a QR code.',
  linkActive: 'Another link is in progress.',
}

export const IDENTITY_WORDS = {
  loadFailed: 'The identity could not be read.',
  describeFailure: (
    failure: { readonly error: IdentityErrorCode; readonly message: string },
    proves: 'words' | 'sheets' = 'sheets',
  ): string => {
    if (failure.error === 'otherAccount') return `The ${proves} belong to another identity.`
    return FAILURE_TEXTS[failure.error] ?? failure.message
  },
}
