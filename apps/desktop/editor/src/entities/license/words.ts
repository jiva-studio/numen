/**
 * What the license store reports when the core refuses an activation or deactivation, or a read
 * fails.
 */
// Stryker disable StringLiteral: interface copy
import type { LicenseErrorCode } from './lib/types'

export const LICENSE_WORDS = {
  loadFailed: 'The licenses could not be read.',
  describeFailure: (failure: {
    readonly error: LicenseErrorCode
    readonly message: string
  }): string => {
    switch (failure.error) {
      case 'invalidKey':
        return 'This license key was not recognized.'
      case 'activationLimitReached':
        return 'This license key has reached its machine limit. Deactivate another machine first.'
      case 'expired':
        return 'This license has expired.'
      case 'disabled':
        return 'This license has been disabled.'
      case 'productMismatch':
        return 'This license key is for a different product.'
      case 'providerUnavailable':
        return failure.message === ''
          ? 'The license server could not be reached. Check your network connection and try again.'
          : failure.message
      case 'keyStoreFailed':
        return 'The license key could not be saved or removed. Check the system keychain.'
      case 'notFound':
        return 'This license is no longer declared by any installed plugin.'
      default:
        return failure.message
    }
  },
}
