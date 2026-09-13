/** What the list of vaults reports as error for a command, in words a person reads. */
import type { VaultErrorCode } from './types'

export const VAULT_ERRORS: Record<VaultErrorCode, string> = {
  unreadable: 'that folder is not there, or cannot be read',
  copy: 'that folder is a copy of a vault this installation already holds',
  overlaps: 'that folder is inside a vault already added, or holds one',
  nameTaken: 'a vault is already called that',
  lastVault: 'that is the only vault this installation has',
  showing: 'that is the vault in front of you',
  unknown: 'that vault is not on the list',
  noTrash: 'this machine has nowhere to put what is deleted',
  asking: 'a tab is holding text you have to answer for, so the window stayed where it was',
}
