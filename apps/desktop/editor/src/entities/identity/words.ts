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
        return 'Some words are not valid.'
      case 'identity_exists':
        return 'This device already has an identity.'
      case 'key_store_failed':
        return 'The identity could not be saved. Check the system keychain.'
      case 'last_device':
        return 'The last device of the account cannot be revoked.'
      case 'revoke_raced':
        return 'A device was added while you revoked. Check the list.'
      case 'not_allowed_this_device':
        return 'This device cannot revoke itself.'
      case 'link_text_invalid':
        return 'This text is not the link of a QR code.'
      case 'link_active':
        return 'Another link is in progress.'
      case 'code_invalid':
        return 'The code has six digits.'
      default:
        return failure.detail
    }
  },
}
