/**
 * The letters that open the vaults on the welcome screen.
 *
 * The screen shows the letter on the row and the window handles the keypress.
 * Both windows take their letters from here, so a displayed letter always
 * works.
 */

import { isTyping } from '@numen/ui'

/**
 * The letters the vaults are picked by, in the order they are listed. They are
 * capitals, which is how every letter on a key cap is drawn.
 */
export const VAULT_LETTERS = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ'

/**
 * The letter that selects one vault of the list; positions past the alphabet
 * have none. With thirty vaults, the last four are opened with the pointer.
 */
export const vaultLetter = (pos: number): string => VAULT_LETTERS[pos] ?? ''

/**
 * Which vault of the list a keystroke opens, counting from the top, and nothing
 * where it opens none.
 *
 * A key held down repeats and a key pressed with a modifier is the machine's
 * own shortcut; a letter typed into a field is text. None of the three is a
 * person picking a vault.
 */
export function getVaultForKey(press: KeyboardEvent, vaults: number): number | null {
  if (press.repeat || press.altKey || press.ctrlKey || press.metaKey) return null
  if (isTyping(press)) return null
  if (press.key.length !== 1) return null

  const index = VAULT_LETTERS.indexOf(press.key.toUpperCase())
  return index >= 0 && index < vaults ? index : null
}
