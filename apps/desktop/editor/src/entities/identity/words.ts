/** What the identity store reports when the core refuses to create the identity or a read fails. */
// Stryker disable StringLiteral: interface copy
import type { IdentityErrorCode } from './lib/types'

export const IDENTITY_WORDS = {
  loadFailed: 'The identity could not be read.',
  describeFailure: (failure: {
    readonly error: IdentityErrorCode
    readonly message: string
  }): string => {
    switch (failure.error) {
      case 'invalidWords':
        return 'Some words are not valid.'
      case 'identityExists':
        return 'This device already has an identity.'
      case 'keyStoreFailed':
        return 'The identity could not be saved. Check the system keychain.'
      case 'lastDevice':
        return 'The last device of the account cannot be revoked.'
      case 'revokeRaced':
        return 'A device was added while you revoked. Check the list.'
      case 'notAllowedThisDevice':
        return 'This device cannot revoke itself.'
      case 'linkTextInvalid':
        return 'This text is not the link of a QR code.'
      case 'linkActive':
        return 'Another link is in progress.'
      case 'codeInvalid':
        return 'The code has six digits.'
      default:
        return failure.message
    }
  },
}
