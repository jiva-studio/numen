/**
 * UI error messages for note operations and vault operations.
 */
import type { ErrorCode } from '../note'
import type { VaultErrorCode } from '../vaults'

/** What the vault reports as error for a command, in words a person reads. */
export const ERRORS: Record<ErrorCode, string> = {
  missing: 'that note is not in the vault',
  notANote: 'that file is not a note',
  notText: 'that file is not text',
  tooLarge: 'that note is longer than this writes',
  bodyRefused: 'that text cannot be written into a note',
  unreadable: 'the frontmatter of that note cannot be read',
  occupied: 'a note of that name is filed there, so the note was renamed and its file was not',
  unnameable: 'a note cannot be called that',
  notAStencil: 'that note is not a stencil',
  notADeck: 'that note is not a deck',
  deckTooLarge: 'that deck is longer than this reads',
  notAPreset: 'that note is not a preset',
}

/** What the list of vaults reports as error for a command, in words a person reads. */
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
