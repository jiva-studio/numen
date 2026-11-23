/**
 * The text of the welcome screen that no other part of the window uses.
 */
// Stryker disable StringLiteral: interface copy
import { getKeyChord } from '@numen/ui'

export const WELCOME_WORDS = {
  commandsKeys: getKeyChord('p', navigator.userAgent),
  /** The row that creates a vault. */
  newVaultDetail: 'Choose a folder',
}
