/**
 * What is offered over what is in front, and what a command carries out of it.
 *
 * The table is read here without a palette: which group a command stands in,
 * and which of them another command's row reaches.
 */
import { describe, expect, it } from 'vitest'
import { commandsOf, overNote } from './commands'
import { asksCommands } from './target'
import { WORDS as words } from '../../shared/words'

describe('the character that means the commands', () => {
  it('is one typed into a field holding nothing', () => {
    expect(asksCommands('', '>')).toBe(true)
  })

  it('is not one typed into a field holding words, so a search for one stands', () => {
    expect(asksCommands('foo', '>foo')).toBe(false)
    expect(asksCommands('', '>foo')).toBe(false)
    expect(asksCommands('>', '>>')).toBe(false)
  })
})

describe('the commands over a note', () => {
  it('leave out the ones another command reaches on its own row', () => {
    const over = overNote(commandsOf(words)).map((one) => one.id)

    expect(over).not.toContain('beside')
    expect(over).not.toContain('destroy')
  })
})

describe('the commands over the vaults an installation holds', () => {
  it('are offered in the group of the vault in front', () => {
    const over = commandsOf(words)
      .filter((one) => one.group === 'vault')
      .map((one) => one.id)

    expect(over).toStrictEqual([
      'first',
      'goto',
      'openVault',
      'newVault',
      'renameVault',
      'forgetVault',
      'eraseVault',
    ])
  })
})
