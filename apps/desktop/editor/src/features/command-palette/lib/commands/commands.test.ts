/**
 * What is offered over what is in front, and what a command carries out of it.
 *
 * and which of them another command's row reaches.
 */
import { describe, expect, it } from 'vitest'
import { createCommands, isCommandPrefixTyped, getNoteCommands } from './commands'
import { WORDS as words } from '@/shared/words'

describe('the character that means the commands', () => {
  it('is one typed into a field holding nothing', () => {
    expect(isCommandPrefixTyped('', '>')).toBe(true)
  })

  it('is not one typed into a field holding words, so a search for one remains', () => {
    expect(isCommandPrefixTyped('foo', '>foo')).toBe(false)
    expect(isCommandPrefixTyped('', '>foo')).toBe(false)
    expect(isCommandPrefixTyped('foo', '>')).toBe(false)
    expect(isCommandPrefixTyped('>', '>>')).toBe(false)
  })
})

describe('the commands over a note', () => {
  it('leave out the ones another command reaches on its own row', () => {
    const commands = getNoteCommands(createCommands(words)).map((one) => one.id)

    expect(commands).not.toContain('beside')
    expect(commands).not.toContain('destroy')
  })
})

describe('the commands over the vaults an installation holds', () => {
  it('are offered in the group of the vault in front', () => {
    const commands = createCommands(words)
      .filter((one) => one.group === 'vault')
      .map((one) => one.id)

    expect(commands).toStrictEqual([
      'first',
      'goto',
      'openVault',
      'newVault',
      'renameVault',
      'removeVault',
      'eraseVault',
    ])
  })
})

describe('the commands that ask for a choice among a few', () => {
  it('are the ones that set how the window is drawn and how it works', () => {
    const choosing = createCommands(words)
      .filter((one) => one.needs === 'choosing')
      .map((one) => one.id)

    expect(choosing).toStrictEqual([
      'appearance',
      'mode',
      'interfaceScale',
      'textScale',
      'syncing',
      'nesting',
      'headings',
      'performanceProfile',
    ])
  })
})
