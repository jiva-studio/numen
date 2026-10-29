/** Error messages the vault list reports for a command, in user-readable text. */
// Stryker disable StringLiteral: interface copy
import type { VaultErrorCode } from './types'

export const VAULT_ERRORS: Record<VaultErrorCode, string> = {
  unreadable: 'folder missing or unreadable',
  copy: 'folder is a copy of a vault already added',
  overlaps: 'folder is inside a vault already added, or contains one',
  nameTaken: 'a vault with this name exists',
  lastVault: 'the only vault cannot be removed',
  showing: 'the open vault cannot be removed',
  unknown: 'vault not in the list',
  noTrash: 'no trash on this system',
  asking: 'a tab has an unresolved file conflict',
}
