/** What the identity store reports when the core refuses to create the identity or a read fails. */
// Stryker disable StringLiteral: interface copy
import type { IdentityErrorCode } from './lib/types'

export const IDENTITY_WORDS = {
  loadFailed: 'The identity could not be read.',
  describeFailure: (failure: {
    readonly errorCode: IdentityErrorCode
    readonly detail: string
  }): string => {
    switch (failure.errorCode) {
      case 'invalid_words':
        return 'These are words that are not valid.'
      case 'identity_exists':
        return 'This device already has an identity.'
      case 'key_store_failed':
        return 'The identity could not be saved. Check the system keychain.'
      default:
        return failure.detail
    }
  },
}
