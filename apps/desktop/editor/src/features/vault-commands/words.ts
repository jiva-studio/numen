/**
 * What the commands over the vaults say: the text of their rows, and the answers they report.
 */
// Stryker disable StringLiteral: interface copy
import { VAULT_ERRORS } from '@/entities/vault'
import { WORDS } from '@/shared/ui/words'

export const VAULT_WORDS = {
  goto: WORDS.goto,
  openVault: WORDS.openVault,
  newVault: WORDS.newVault,
  renameVault: WORDS.renameVault,
  removeVault: WORDS.removeVault,
  eraseVault: WORDS.eraseVault,
  emptyTrash: WORDS.emptyTrash,
  retainedNotice: WORDS.retainedNotice,
  keepsVault: WORDS.keepsVault,
  keepsTrash: WORDS.keepsTrash,
  emptiesTrash: WORDS.emptiesTrash,
  removeVaultAction: WORDS.removeVaultAction,
  stays: WORDS.stays,
  exactly: WORDS.exactly,
  erases: WORDS.erases,
  binned: WORDS.binned,
  typeVaultBack: WORDS.typeVaultBack,
  folder: WORDS.folder,
  undeleted: WORDS.undeleted,
  vaultErrors: VAULT_ERRORS,
}
