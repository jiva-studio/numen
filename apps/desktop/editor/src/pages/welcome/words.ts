/**
 * The text of the welcome screen that no other part of the window uses.
 */
// Stryker disable StringLiteral: interface copy
import { createKeyChord } from '@numen/ui'

export const WELCOME_WORDS = {
  commandsKeys: createKeyChord('p', navigator.userAgent),
  /** The row that creates a vault. */
  newVaultDetail: 'Choose a folder',
}
