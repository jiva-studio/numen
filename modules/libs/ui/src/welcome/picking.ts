/**
 * The letters the vaults on the welcome screen are opened by.
 *
 * The screen draws the letter on the row and the window presses it, and both
 * windows take theirs from here, so a letter a person sees is a letter that
 * works.
 */

/**
 * The letters the vaults are picked by, in the order they are listed. They are
 * capitals, which is how every letter on a key cap is drawn.
 */
export const VAULT_LETTERS = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ'

/**
 * The letter one vault of the list is picked by, and nothing past the alphabet:
 * an installation of thirty vaults is one where the last four are opened with
 * the hand.
 */
export const vaultLetter = (at: number): string => VAULT_LETTERS[at] ?? ''

/**
 * Which vault of the list a keystroke opens, counting from the top, and nothing
 * where it opens none.
 *
 * A key held down repeats and a key pressed with a modifier is the machine's
 * own shortcut; a letter typed into a field is text. None of the three is a
 * person picking a vault.
 */
export function opensVault(press: KeyboardEvent, vaults: number): number | null {
  if (press.repeat || press.altKey || press.ctrlKey || press.metaKey) return null
  if (typing(press)) return null
  if (press.key.length !== 1) return null

  const at = VAULT_LETTERS.indexOf(press.key.toUpperCase())
  return at >= 0 && at < vaults ? at : null
}

/** Whether the key was pressed into something being written in. */
const typing = (press: KeyboardEvent): boolean => {
  const at = press.target as HTMLElement | null
  if (!at) return false
  return at.tagName === 'INPUT' || at.tagName === 'TEXTAREA' || at.isContentEditable === true
}
