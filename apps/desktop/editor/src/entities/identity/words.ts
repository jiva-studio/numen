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
      case 'invalidRevokeReason':
        return 'Say whether the device was lost or stolen, and since when for a stolen one.'
      case 'invalidChainMark':
        return 'The chain mark is a number, a dash and 12 digits or letters, as printed on the kit.'
      case 'chainOutdated':
        return 'The server holds an older history of your account than your kit names. Nothing was restored. Try again later, or restore without the chain mark if you know the server lost your account.'
      default:
        return failure.message
    }
  },
}
