/** What the identity store reports when the core refuses an identity call or a read fails. */
// Stryker disable StringLiteral: interface copy
import type { IdentityErrorCode } from './lib/types'

export const IDENTITY_WORDS = {
  loadFailed: 'The identity could not be read.',
  describeFailure: (
    failure: { readonly errorCode: IdentityErrorCode; readonly detail: string },
    proves: 'words' | 'sheets' = 'sheets',
  ): string => {
    switch (failure.errorCode) {
      case 'invalid_words':
        return 'Some words are not valid.'
      case 'identity_exists':
        return 'This device already has an identity.'
      case 'key_store_failed':
        return 'The identity could not be saved. Check the system keychain.'
      case 'not_enough_sheets':
        return 'One more sheet is needed.'
      case 'invalid_sheets':
        return 'These are sheets that are not valid.'
      case 'other_account':
        return proves === 'words'
          ? 'These are words of another account.'
          : 'These are sheets of another account.'
      case 'no_identity':
        return 'This device has no identity to make a kit for.'
      default:
        return failure.detail
    }
  },
}
