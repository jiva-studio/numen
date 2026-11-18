/**
 * What the license store reports when the core refuses an activation or deactivation, or a read
 * fails.
 */
// Stryker disable StringLiteral: interface copy
import type { LicenseErrorCode } from './lib/types'

export const LICENSE_WORDS = {
  loadFailed: 'The licenses could not be read.',
  describeFailure: (failure: {
    readonly errorCode: LicenseErrorCode
    readonly detail: string
  }): string => {
    switch (failure.errorCode) {
      case 'invalid_key':
        return 'This license key was not recognized.'
      case 'activation_limit_reached':
        return 'This license key has reached its machine limit. Deactivate another machine first.'
      case 'expired':
        return 'This license has expired.'
      case 'disabled':
        return 'This license has been disabled.'
      case 'product_mismatch':
        return 'This license key is for a different product.'
      case 'provider_unavailable':
        return failure.detail === ''
          ? 'The license server could not be reached. Check your network connection and try again.'
          : failure.detail
      case 'key_store_failed':
        return 'The license key could not be saved or removed. Check the system keychain.'
      case 'not_found':
        return 'This license is no longer declared by any installed plugin.'
      default:
        return failure.detail
    }
  },
}
