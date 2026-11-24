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
      default:
        return failure.message
    }
  },
}
