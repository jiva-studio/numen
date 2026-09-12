/**
 * The letters the vaults are opened by, asked without a screen.
 *
 * The negatives are the ones worth having: a letter no vault stands at opens
 * nothing, and a letter typed into a field is text.
 */
import { describe, expect, it } from 'vitest'
import { getVaultForKey, vaultLetter, VAULT_LETTERS } from './letters'

/** pressed is one keystroke, as the window meets it. */
const createPress = (key: string, more: Partial<KeyboardEvent> = {}) =>
  ({
    key,
    repeat: false,
    altKey: false,
    ctrlKey: false,
    metaKey: false,
    shiftKey: false,
    ...more,
  }) as KeyboardEvent

/** into is a keystroke that landed in something being written in. */
const into = (tag: string, written = false): Partial<KeyboardEvent> => ({
  target: { tagName: tag, isContentEditable: written } as unknown as EventTarget,
})

describe('the letter a vault is drawn with', () => {
  it('is the alphabet in capitals, from the top of the list down', () => {
    expect(vaultLetter(0)).toBe('A')
    expect(vaultLetter(1)).toBe('B')
    expect(vaultLetter(25)).toBe('Z')
  })

  it('is nothing at all past the alphabet', () => {
    expect(vaultLetter(VAULT_LETTERS.length)).toBe('')
    expect(vaultLetter(99)).toBe('')
  })
})

describe('a letter pressed over the list', () => {
  it('opens the vault standing at it', () => {
    expect(getVaultForKey(createPress('a'), 2)).toBe(0)
    expect(getVaultForKey(createPress('b'), 2)).toBe(1)
  })

  it('opens it whichever case the letter is typed in', () => {
    expect(getVaultForKey(createPress('B'), 2)).toBe(1)
    expect(getVaultForKey(createPress('B', { shiftKey: true }), 2)).toBe(1)
  })

  it('opens nothing where the list is shorter than the letter', () => {
    expect(getVaultForKey(createPress('c'), 2)).toBeNull()
    expect(getVaultForKey(createPress('a'), 0)).toBeNull()
  })

  it('opens nothing for a key that is no letter of the alphabet', () => {
    expect(getVaultForKey(createPress('1'), 2)).toBeNull()
    expect(getVaultForKey(createPress('Enter'), 2)).toBeNull()
    expect(getVaultForKey(createPress('Escape'), 2)).toBeNull()
    expect(getVaultForKey(createPress(' '), 2)).toBeNull()
  })

  it('opens nothing while the key is held down', () => {
    expect(getVaultForKey(createPress('a', { repeat: true }), 2)).toBeNull()
  })

  it('opens nothing where the letter is held with a modifier', () => {
    expect(getVaultForKey(createPress('a', { ctrlKey: true }), 2)).toBeNull()
    expect(getVaultForKey(createPress('a', { metaKey: true }), 2)).toBeNull()
    expect(getVaultForKey(createPress('a', { altKey: true }), 2)).toBeNull()
  })

  it('opens nothing where the letter was typed into a field', () => {
    expect(getVaultForKey(createPress('a', into('INPUT')), 2)).toBeNull()
    expect(getVaultForKey(createPress('a', into('TEXTAREA')), 2)).toBeNull()
    expect(getVaultForKey(createPress('a', into('DIV', true)), 2)).toBeNull()
  })

  it('opens the vault where the letter landed on something written in nowhere', () => {
    expect(getVaultForKey(createPress('a', into('DIV')), 2)).toBe(0)
    expect(getVaultForKey(createPress('a', { target: null }), 2)).toBe(0)
  })
})
