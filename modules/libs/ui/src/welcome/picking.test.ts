/**
 * The letters the vaults are opened by, asked without a screen.
 *
 * The negatives are the ones worth having: a letter no vault stands at opens
 * nothing, and a letter typed into a field is text.
 */
import { describe, expect, it } from 'vitest'
import { opensVault, vaultLetter, VAULT_LETTERS } from './picking'

/** pressed is one keystroke, as the window meets it. */
const pressed = (key: string, more: Partial<KeyboardEvent> = {}) =>
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
  it('is the alphabet, from the top of the list down', () => {
    expect(vaultLetter(0)).toBe('a')
    expect(vaultLetter(1)).toBe('b')
    expect(vaultLetter(25)).toBe('z')
  })

  it('is nothing at all past the alphabet', () => {
    expect(vaultLetter(VAULT_LETTERS.length)).toBe('')
    expect(vaultLetter(99)).toBe('')
  })
})

describe('a letter pressed over the list', () => {
  it('opens the vault standing at it', () => {
    expect(opensVault(pressed('a'), 2)).toBe(0)
    expect(opensVault(pressed('b'), 2)).toBe(1)
  })

  it('opens it whichever case the letter is typed in', () => {
    expect(opensVault(pressed('B'), 2)).toBe(1)
    expect(opensVault(pressed('B', { shiftKey: true }), 2)).toBe(1)
  })

  it('opens nothing where the list is shorter than the letter', () => {
    expect(opensVault(pressed('c'), 2)).toBeNull()
    expect(opensVault(pressed('a'), 0)).toBeNull()
  })

  it('opens nothing for a key that is no letter of the alphabet', () => {
    expect(opensVault(pressed('1'), 2)).toBeNull()
    expect(opensVault(pressed('Enter'), 2)).toBeNull()
    expect(opensVault(pressed('Escape'), 2)).toBeNull()
    expect(opensVault(pressed(' '), 2)).toBeNull()
  })

  it('opens nothing while the key is held down', () => {
    expect(opensVault(pressed('a', { repeat: true }), 2)).toBeNull()
  })

  it('opens nothing where the letter is held with a modifier', () => {
    expect(opensVault(pressed('a', { ctrlKey: true }), 2)).toBeNull()
    expect(opensVault(pressed('a', { metaKey: true }), 2)).toBeNull()
    expect(opensVault(pressed('a', { altKey: true }), 2)).toBeNull()
  })

  it('opens nothing where the letter was typed into a field', () => {
    expect(opensVault(pressed('a', into('INPUT')), 2)).toBeNull()
    expect(opensVault(pressed('a', into('TEXTAREA')), 2)).toBeNull()
    expect(opensVault(pressed('a', into('DIV', true)), 2)).toBeNull()
  })

  it('opens the vault where the letter landed on something written in nowhere', () => {
    expect(opensVault(pressed('a', into('DIV')), 2)).toBe(0)
    expect(opensVault(pressed('a', { target: null }), 2)).toBe(0)
  })
})
