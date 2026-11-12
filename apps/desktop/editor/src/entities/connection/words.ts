/**
 * What the connection store reports when the core refuses a change or a read fails.
 */
// Stryker disable StringLiteral: interface copy

const toUpperFirst = (text: string): string => text.charAt(0).toUpperCase() + text.slice(1)

export const CONNECTION_WORDS = {
  loadFailed: 'The connections could not be read.',
  describeFailure: (failure: { readonly errorCode: string; readonly detail: string }): string => {
    switch (failure.errorCode) {
      case 'name_taken':
        return 'Another connection has this name.'
      case 'invalid_field':
        return failure.detail === ''
          ? 'A value is not accepted.'
          : `${toUpperFirst(failure.detail)}.`
      case 'type_unavailable':
        return 'No installed plugin provides this type of connection.'
      case 'key_store_failed':
        return 'The key could not be stored or removed. Check the system keychain.'
      case 'stale_revision':
        return 'Another window changed this connection. Close this dialog and open it again.'
      case 'not_found':
        return 'This connection no longer exists.'
      default:
        return failure.detail
    }
  },
}
