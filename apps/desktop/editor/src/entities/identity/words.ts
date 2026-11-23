/** What the identity store reports when the core refuses an identity call or a read fails. */
// Stryker disable StringLiteral: interface copy
import type { IdentityErrorCode } from './lib/types'

export const IDENTITY_WORDS = {
  loadFailed: 'The identity could not be read.',
  describeFailure: (
    failure: { readonly error: IdentityErrorCode; readonly message: string },
    proves: 'words' | 'sheets' = 'sheets',
  ): string => {
    switch (failure.error) {
      case 'invalidWords':
        return 'Some words are not valid.'
      case 'identityExists':
        return 'This device already has an identity.'
      case 'keyStoreFailed':
        return 'The identity could not be saved. Check the system keychain.'
      case 'notEnoughSheets':
        return 'One more sheet is needed.'
      case 'invalidSheets':
        return 'Some sheets are not valid.'
      case 'otherAccount':
        return proves === 'words'
          ? 'The words belong to another account.'
          : 'The sheets belong to another account.'
      case 'noIdentity':
        return 'This device has no identity to make a kit for.'
      default:
        return failure.message
    }
  },
}
